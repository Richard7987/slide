// Package bib parses BibTeX files and formats numbered references for
// citations found in slide markdown.
package bib

import (
	"io"
	"os"
	"strings"
)

// Entry is a single @type{key, ...} BibTeX entry. Fields are already
// stripped of braces/quotes and accent-converted.
type Entry struct {
	Type   string
	Key    string
	Fields map[string]string
}

// ParseFile reads and parses a .bib file.
func ParseFile(path string) (map[string]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads a .bib document from r. Entries it can't make sense of are
// skipped rather than causing the whole file to fail -- a malformed one
// shouldn't take down a presentation.
func Parse(r io.Reader) (map[string]Entry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parseString(string(data)), nil
}

func parseString(s string) map[string]Entry {
	entries := make(map[string]Entry)
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		for i < len(runes) && runes[i] != '@' {
			i++
		}
		if i >= len(runes) {
			break
		}
		i++ // skip '@'

		typeStart := i
		for i < len(runes) && isLetter(runes[i]) {
			i++
		}
		typ := strings.ToLower(string(runes[typeStart:i]))

		i = skipSpace(runes, i)
		if i >= len(runes) || runes[i] != '{' {
			continue
		}
		bodyEnd := matchBrace(runes, i)
		if bodyEnd < 0 {
			break // unbalanced braces from here on; nothing more to parse
		}
		body := runes[i+1 : bodyEnd]
		i = bodyEnd + 1

		if typ == "comment" || typ == "string" || typ == "preamble" {
			continue
		}

		key, fields := parseEntryBody(body)
		if key == "" {
			continue
		}
		entries[key] = Entry{Type: typ, Key: key, Fields: fields}
	}
	return entries
}

func parseEntryBody(body []rune) (string, map[string]string) {
	segments := splitTopLevel(body, ',')
	if len(segments) == 0 {
		return "", nil
	}
	key := strings.TrimSpace(string(segments[0]))
	if key == "" {
		return "", nil
	}

	fields := make(map[string]string)
	for _, seg := range segments[1:] {
		s := strings.TrimSpace(string(seg))
		if s == "" {
			continue
		}
		eq := strings.IndexRune(s, '=')
		if eq < 0 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(s[:eq]))
		fields[name] = parseValue(strings.TrimSpace(s[eq+1:]))
	}
	return key, fields
}

// parseValue strips the {...}/"..." delimiter (bare words/numbers pass
// through as-is) and converts LaTeX accents in the result.
func parseValue(raw string) string {
	runes := []rune(raw)
	if len(runes) == 0 {
		return ""
	}

	var inner string
	switch runes[0] {
	case '{':
		if end := matchBrace(runes, 0); end >= 0 {
			inner = string(runes[1:end])
		} else {
			inner = string(runes[1:])
		}
	case '"':
		if end := len(runes) - 1; end > 0 && runes[end] == '"' {
			inner = string(runes[1:end])
		} else {
			inner = string(runes[1:])
		}
	default:
		inner = raw
	}
	return stripAccents(inner)
}

// splitTopLevel splits runes on sep, ignoring separators nested inside
// {...} or "...".
func splitTopLevel(runes []rune, sep rune) [][]rune {
	var parts [][]rune
	depth := 0
	inQuote := false
	start := 0
	for i, r := range runes {
		switch r {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case '"':
			if depth == 0 {
				inQuote = !inQuote
			}
		case sep:
			if depth == 0 && !inQuote {
				parts = append(parts, runes[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, runes[start:])
	return parts
}

// matchBrace returns the index of the '}' matching runes[open] (which must
// be '{'), or -1 if unbalanced.
func matchBrace(runes []rune, open int) int {
	depth := 0
	for i := open; i < len(runes); i++ {
		switch runes[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func skipSpace(runes []rune, i int) int {
	for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t' || runes[i] == '\n' || runes[i] == '\r') {
		i++
	}
	return i
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
