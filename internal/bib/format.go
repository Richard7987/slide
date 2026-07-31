package bib

import (
	"fmt"
	"strings"
)

// refBuilder joins non-empty reference fragments with ", ", except that a
// fragment immediately following a quoted title gets a plain space -- the
// comma is already inside the closing quote, matching the numeric/IEEE
// convention of `"Title," Journal, ...`.
type refBuilder struct {
	parts []string
}

func (r *refBuilder) add(s string) {
	if s != "" {
		r.parts = append(r.parts, s)
	}
}

func (r *refBuilder) addTitle(title string) {
	if title != "" {
		r.parts = append(r.parts, `"`+title+`,"`)
	}
}

func (r *refBuilder) String() string {
	var b strings.Builder
	for i, p := range r.parts {
		if i > 0 {
			if strings.HasSuffix(r.parts[i-1], `,"`) {
				b.WriteString(" ")
			} else {
				b.WriteString(", ")
			}
		}
		b.WriteString(p)
	}
	return strings.TrimRight(b.String(), ".,") + "."
}

// FormatReference renders entry e as the n-th numbered reference, in a
// plain numeric/IEEE-ish style. Field presence varies by entry type;
// missing fields are simply omitted rather than leaving gaps.
func FormatReference(n int, e Entry) string {
	var body string
	switch e.Type {
	case "book":
		body = formatBook(e.Fields)
	case "inproceedings", "conference":
		body = formatInproceedings(e.Fields)
	case "misc":
		body = formatMisc(e.Fields)
	case "techreport":
		body = formatTechreport(e.Fields)
	case "phdthesis":
		body = formatThesis(e.Fields, "Ph.D. dissertation")
	case "mastersthesis":
		body = formatThesis(e.Fields, "M.S. thesis")
	default: // "article" and anything unrecognized use the article template
		body = formatArticle(e.Fields)
	}
	return fmt.Sprintf("[%d] %s", n, body)
}

func formatArticle(f map[string]string) string {
	var r refBuilder
	r.add(formatAuthors(splitAuthors(f["author"])))
	r.addTitle(f["title"])
	r.add(emphasize(f["journal"]))
	r.add(formatVolNo(f))
	r.add(formatPages(f["pages"]))
	r.add(formatMonthYear(f))
	return r.String()
}

func formatBook(f map[string]string) string {
	var r refBuilder
	r.add(formatAuthors(splitAuthors(f["author"])))
	r.add(emphasize(f["title"]))
	if ed := f["edition"]; ed != "" {
		r.add(ed + " ed.")
	}
	r.add(formatCityPublisher(f))
	r.add(f["year"])
	return r.String()
}

func formatInproceedings(f map[string]string) string {
	var r refBuilder
	r.add(formatAuthors(splitAuthors(f["author"])))
	r.addTitle(f["title"])
	if bt := f["booktitle"]; bt != "" {
		r.add("in " + emphasize(bt))
	}
	r.add(f["address"])
	r.add(f["year"])
	r.add(formatPages(f["pages"]))
	return r.String()
}

func formatMisc(f map[string]string) string {
	var r refBuilder
	r.add(formatAuthors(splitAuthors(f["author"])))
	r.addTitle(f["title"])
	r.add(f["year"])
	if url := entryURL(f); url != "" {
		r.add("[Online]. Available: " + url)
	}
	return r.String()
}

func formatTechreport(f map[string]string) string {
	var r refBuilder
	r.add(formatAuthors(splitAuthors(f["author"])))
	r.addTitle(f["title"])
	r.add(f["institution"])
	r.add(f["address"])
	if num := f["number"]; num != "" {
		r.add("Rep. " + num)
	}
	r.add(f["year"])
	return r.String()
}

func formatThesis(f map[string]string, kind string) string {
	var r refBuilder
	r.add(formatAuthors(splitAuthors(f["author"])))
	r.addTitle(f["title"])
	r.add(kind)
	r.add(f["school"])
	r.add(f["address"])
	r.add(f["year"])
	return r.String()
}

func emphasize(s string) string {
	if s == "" {
		return ""
	}
	return "*" + s + "*"
}

func formatCityPublisher(f map[string]string) string {
	city, publisher := f["address"], f["publisher"]
	switch {
	case city != "" && publisher != "":
		return city + ": " + publisher
	case publisher != "":
		return publisher
	default:
		return city
	}
}

func formatVolNo(f map[string]string) string {
	var parts []string
	if v := f["volume"]; v != "" {
		parts = append(parts, "vol. "+v)
	}
	if n := f["number"]; n != "" {
		parts = append(parts, "no. "+n)
	}
	return strings.Join(parts, ", ")
}

func formatPages(raw string) string {
	if raw == "" {
		return ""
	}
	return "pp. " + strings.ReplaceAll(raw, "--", "–")
}

func formatMonthYear(f map[string]string) string {
	month, year := f["month"], f["year"]
	switch {
	case month != "" && year != "":
		return month + " " + year
	case year != "":
		return year
	default:
		return month
	}
}

// entryURL returns a link target for an entry, preferring an explicit url
// field and falling back to doi, for use as a citation hyperlink.
func entryURL(f map[string]string) string {
	if u := f["url"]; u != "" {
		return u
	}
	if doi := f["doi"]; doi != "" {
		return "https://doi.org/" + doi
	}
	return ""
}
