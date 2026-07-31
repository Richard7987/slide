// Package image renders markdown images (![alt](src "caption")) as real
// terminal graphics via the Kitty Unicode Placeholder technique, with the
// caption shown below. Sentinel lines used during pre-processing are
// wrapped in Private Use Area characters so they can't collide with real
// slide content and glamour's word-wrap never splits them (no spaces).
package image

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Ref is a single ![alt](src "caption") reference found in a slide.
type Ref struct {
	Raw     string
	Alt     string
	Src     string
	Caption string
}

// Placement pairs a resolved Ref with its cache Entry, in sentinel order.
type Placement struct {
	Ref   Ref
	Entry Entry
}

var refRe = regexp.MustCompile(`!\[([^\]]*)\]\(\s*(\S+?)(?:\s+"([^"]*)")?\s*\)`)

// Parse extracts image references from raw slide markdown, in the order
// they appear.
func Parse(markdown string) []Ref {
	var refs []Ref
	for _, m := range refRe.FindAllStringSubmatch(markdown, -1) {
		refs = append(refs, Ref{Raw: m[0], Alt: m[1], Src: m[2], Caption: m[3]})
	}
	return refs
}

const sentinelOpen = "IMG"
const sentinelClose = ""

var sentinelRe = regexp.MustCompile(sentinelOpen + `(\d+)` + sentinelClose)

func sentinel(index int) string {
	return fmt.Sprintf("%s%d%s", sentinelOpen, index, sentinelClose)
}

// Preprocess replaces each image reference whose cache entry is already
// resolved (lookup returns ok) with a sentinel line, plus a caption
// paragraph if present. References not yet resolved are left untouched, so
// glamour falls back to its default "Image: alt -> src" rendering until
// they become ready. The returned placements are indexed exactly as the
// sentinels appear, for Expand to consume after markdown rendering.
func Preprocess(markdown string, lookup func(Ref) (Entry, bool)) (string, []Placement) {
	var placements []Placement
	for _, ref := range Parse(markdown) {
		entry, ok := lookup(ref)
		if !ok {
			continue
		}
		replacement := sentinel(len(placements))
		if ref.Caption != "" {
			replacement += "\n\n*" + ref.Caption + "*"
		}
		markdown = strings.Replace(markdown, ref.Raw, replacement, 1)
		placements = append(placements, Placement{Ref: ref, Entry: entry})
	}
	return markdown, placements
}

// Expand scans already-rendered text (post markdown, post padding) line by
// line and replaces each sentinel line with the real placeholder block for
// its placement, expanding it to as many rows as the image needs.
func Expand(rendered string, placements []Placement) string {
	if len(placements) == 0 {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		m := sentinelRe.FindStringSubmatch(line)
		if m == nil {
			out = append(out, line)
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil || idx < 0 || idx >= len(placements) {
			out = append(out, line)
			continue
		}
		e := placements[idx].Entry
		out = append(out, strings.Split(PlaceholderBlock(e.ID, e.Cols, e.Rows), "\n")...)
	}
	return strings.Join(out, "\n")
}
