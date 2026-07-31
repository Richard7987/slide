package bib

import "strings"

// Accent tables covering the accented Latin letters most likely to appear
// in author names and titles. Not exhaustive.
var (
	acute = map[rune]rune{
		'a': 'á', 'e': 'é', 'i': 'í', 'o': 'ó', 'u': 'ú', 'y': 'ý',
		'A': 'Á', 'E': 'É', 'I': 'Í', 'O': 'Ó', 'U': 'Ú', 'Y': 'Ý',
		'c': 'ć', 'n': 'ń', 's': 'ś', 'z': 'ź',
		'C': 'Ć', 'N': 'Ń', 'S': 'Ś', 'Z': 'Ź',
	}
	grave = map[rune]rune{
		'a': 'à', 'e': 'è', 'i': 'ì', 'o': 'ò', 'u': 'ù',
		'A': 'À', 'E': 'È', 'I': 'Ì', 'O': 'Ò', 'U': 'Ù',
	}
	umlaut = map[rune]rune{
		'a': 'ä', 'e': 'ë', 'i': 'ï', 'o': 'ö', 'u': 'ü', 'y': 'ÿ',
		'A': 'Ä', 'E': 'Ë', 'I': 'Ï', 'O': 'Ö', 'U': 'Ü',
	}
	tilde = map[rune]rune{
		'a': 'ã', 'n': 'ñ', 'o': 'õ',
		'A': 'Ã', 'N': 'Ñ', 'O': 'Õ',
	}
	circumflex = map[rune]rune{
		'a': 'â', 'e': 'ê', 'i': 'î', 'o': 'ô', 'u': 'û',
		'A': 'Â', 'E': 'Ê', 'I': 'Î', 'O': 'Ô', 'U': 'Û',
	}
	cedilla = map[rune]rune{'c': 'ç', 'C': 'Ç', 's': 'ş', 'S': 'Ş'}
	caron   = map[rune]rune{
		'c': 'č', 's': 'š', 'z': 'ž', 'e': 'ě', 'r': 'ř',
		'C': 'Č', 'S': 'Š', 'Z': 'Ž',
	}
)

func accentTable(cmd rune) map[rune]rune {
	switch cmd {
	case '\'':
		return acute
	case '`':
		return grave
	case '"':
		return umlaut
	case '~':
		return tilde
	case '^':
		return circumflex
	case 'c':
		return cedilla
	case 'v':
		return caron
	}
	return nil
}

// stripAccents converts recognized LaTeX accent commands (\'e, \"{o},
// \c{c}, ...) to their precomposed Unicode letter, and drops any
// remaining brace characters (typographic protection, e.g. {IEEE}, with
// no textual meaning of their own). Unrecognized backslash commands are
// left as-is.
func stripAccents(s string) string {
	var out strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		switch runes[i] {
		case '\\':
			i = writeAccent(&out, runes, i)
		case '{', '}':
			i++
		default:
			out.WriteRune(runes[i])
			i++
		}
	}
	return out.String()
}

func writeAccent(out *strings.Builder, runes []rune, i int) int {
	i++ // skip backslash
	if i >= len(runes) {
		out.WriteRune('\\')
		return i
	}
	cmd := runes[i]
	i++

	table := accentTable(cmd)
	if table == nil {
		// Not a command we recognize: leave it untouched, including
		// whatever follows, so we never eat a space that wasn't ours.
		out.WriteRune('\\')
		out.WriteRune(cmd)
		return i
	}

	// Letter-based commands (\c, \v, ...) need a space or braces to
	// separate them from the base letter, per TeX control-word lexing;
	// that separating space carries no output of its own -- but if the
	// base letter turns out unmapped, restore it in the fallback below.
	spacedCmd := i < len(runes) && runes[i] == ' '
	if spacedCmd {
		i++
	}
	braced := i < len(runes) && runes[i] == '{'
	if braced {
		i++
	}
	if i >= len(runes) {
		out.WriteRune('\\')
		out.WriteRune(cmd)
		return i
	}

	base := runes[i]
	if r, ok := table[base]; ok {
		out.WriteRune(r)
		i++
		if braced && i < len(runes) && runes[i] == '}' {
			i++
		}
		return i
	}

	out.WriteRune('\\')
	out.WriteRune(cmd)
	if spacedCmd {
		out.WriteRune(' ')
	}
	return i
}
