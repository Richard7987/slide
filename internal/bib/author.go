package bib

import "strings"

// Author is a single name split into given (First) and family (Last)
// parts. Others marks the "and others" sentinel in a BibTeX author field.
type Author struct {
	First, Last string
	Others      bool
}

// splitAuthors parses a BibTeX author/editor field ("First Last and
// First2 Last2 and others"). It handles the two common per-author forms:
// "Last, First" (comma present) and "First Last" (no comma, last word is
// the family name). It doesn't handle "von"/"Jr" particles.
func splitAuthors(field string) []Author {
	if field == "" {
		return nil
	}
	parts := strings.Split(field, " and ")
	authors := make([]Author, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.EqualFold(p, "others") {
			authors = append(authors, Author{Others: true})
			continue
		}
		authors = append(authors, splitAuthor(p))
	}
	return authors
}

func splitAuthor(s string) Author {
	if comma := strings.Index(s, ","); comma >= 0 {
		return Author{
			Last:  strings.TrimSpace(s[:comma]),
			First: strings.TrimSpace(s[comma+1:]),
		}
	}
	words := strings.Fields(s)
	switch len(words) {
	case 0:
		return Author{}
	case 1:
		return Author{Last: words[0]}
	default:
		return Author{
			First: strings.Join(words[:len(words)-1], " "),
			Last:  words[len(words)-1],
		}
	}
}

// formatAuthors renders authors in numeric/IEEE-ish style: one author is
// just "F. Last"; two are joined with "and"; three to six are a comma
// list with "and" before the last; more than six (or a trailing "and
// others") collapses to the first author plus "et al."
func formatAuthors(authors []Author) string {
	if len(authors) == 0 {
		return ""
	}

	others := false
	if authors[len(authors)-1].Others {
		others = true
		authors = authors[:len(authors)-1]
	}
	if len(authors) == 0 {
		return ""
	}
	if len(authors) > 6 {
		others = true
		authors = authors[:1]
	}

	formatted := make([]string, len(authors))
	for i, a := range authors {
		formatted[i] = formatOneAuthor(a)
	}

	if others {
		return formatted[0] + " et al."
	}

	switch len(formatted) {
	case 1:
		return formatted[0]
	case 2:
		return formatted[0] + " and " + formatted[1]
	default:
		return strings.Join(formatted[:len(formatted)-1], ", ") + ", and " + formatted[len(formatted)-1]
	}
}

// formatOneAuthor renders a single author as initials + last name, e.g.
// First "Given Names" -> "G. N. Last".
func formatOneAuthor(a Author) string {
	var initials []string
	for _, w := range strings.Fields(a.First) {
		r := []rune(w)
		if len(r) == 0 {
			continue
		}
		initials = append(initials, string(r[0])+".")
	}
	switch {
	case a.Last == "":
		return strings.Join(initials, " ")
	case len(initials) == 0:
		return a.Last
	default:
		return strings.Join(initials, " ") + " " + a.Last
	}
}
