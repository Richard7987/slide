package bib

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var citeRe = regexp.MustCompile(`\[@([A-Za-z0-9_:.-]+)\]`)

// Process scans slides for [@key] citations, numbers the cited entries
// 1..N in alphabetical order of the reference (first author's family
// name, then year, then title), rewrites each [@key] into a [n] markdown
// link to a #key anchor (it never resolves anywhere -- it's just how
// glamour picks up the theme's link_text color without hardcoding one),
// and appends a References slide listing every cited entry in that same
// alphabetical order. Citation keys not found in entries are left as
// literal [@key] text.
func Process(slides []string, entries map[string]Entry) []string {
	seen := make(map[string]bool)
	var cited []string
	for _, slide := range slides {
		for _, m := range citeRe.FindAllStringSubmatch(slide, -1) {
			key := m[1]
			if _, ok := entries[key]; !ok {
				continue
			}
			if !seen[key] {
				seen[key] = true
				cited = append(cited, key)
			}
		}
	}
	if len(cited) == 0 {
		return slides
	}

	order := append([]string(nil), cited...)
	sort.SliceStable(order, func(i, j int) bool {
		return sortKey(entries[order[i]]) < sortKey(entries[order[j]])
	})
	numbers := make(map[string]int, len(order))
	for i, key := range order {
		numbers[key] = i + 1
	}

	out := make([]string, len(slides))
	for i, slide := range slides {
		out[i] = citeRe.ReplaceAllStringFunc(slide, func(match string) string {
			key := citeRe.FindStringSubmatch(match)[1]
			n, ok := numbers[key]
			if !ok {
				return match
			}
			return fmt.Sprintf("[[%d](#%s)]", n, key)
		})
	}
	out = append(out, referencesSlides(order, entries)...)
	return out
}

// sortKey is the alphabetical sort key for an entry: the first author's
// family name (lowercased), then year, then title, with NUL separators so
// shorter names sort before longer ones sharing a prefix. Entries with no
// author fall back to their title.
func sortKey(e Entry) string {
	f := e.Fields
	var name string
	if as := splitAuthors(f["author"]); len(as) > 0 {
		if name = strings.ToLower(as[0].Last); name == "" {
			name = strings.ToLower(as[0].First)
		}
	}
	if name == "" {
		name = strings.ToLower(strings.TrimLeft(f["title"], `"'`))
	}
	return name + "\x00" + f["year"] + "\x00" + strings.ToLower(f["title"])
}

// refsPerSlide caps how many entries land on one References slide. A
// terminal slide can't scroll, so a long bibliography is broken across
// several slides rather than running off the bottom.
const refsPerSlide = 6

// referencesSlides renders the cited entries (already in final numbered
// order) as one or more "## References" slides of at most refsPerSlide
// entries each.
func referencesSlides(order []string, entries map[string]Entry) []string {
	var slides []string
	for start := 0; start < len(order); start += refsPerSlide {
		end := start + refsPerSlide
		if end > len(order) {
			end = len(order)
		}
		var b strings.Builder
		if start == 0 {
			b.WriteString("## References\n\n")
		} else {
			b.WriteString("## References (cont.)\n\n")
		}
		for i := start; i < end; i++ {
			b.WriteString(FormatReference(i+1, entries[order[i]]))
			b.WriteString("\n\n")
		}
		slides = append(slides, strings.TrimRight(b.String(), "\n"))
	}
	return slides
}
