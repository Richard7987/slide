package latex

import "strings"

// symbols maps no-argument macros to their Unicode glyph. Not exhaustive --
// best-effort coverage of the common Greek letters, operators, relations,
// set/logic notation, and arrows used in typical slide math.
var symbols = map[string]rune{
	"alpha": 'α', "beta": 'β', "gamma": 'γ', "delta": 'δ',
	"epsilon": 'ε', "varepsilon": 'ε', "zeta": 'ζ', "eta": 'η',
	"theta": 'θ', "vartheta": 'ϑ', "iota": 'ι', "kappa": 'κ',
	"lambda": 'λ', "mu": 'μ', "nu": 'ν', "xi": 'ξ', "omicron": 'ο',
	"pi": 'π', "varpi": 'ϖ', "rho": 'ρ', "varrho": 'ϱ',
	"sigma": 'σ', "varsigma": 'ς', "tau": 'τ', "upsilon": 'υ',
	"phi": 'φ', "varphi": 'ϕ', "chi": 'χ', "psi": 'ψ', "omega": 'ω',

	"Gamma": 'Γ', "Delta": 'Δ', "Theta": 'Θ', "Lambda": 'Λ',
	"Xi": 'Ξ', "Pi": 'Π', "Sigma": 'Σ', "Upsilon": 'Υ',
	"Phi": 'Φ', "Psi": 'Ψ', "Omega": 'Ω',

	"times": '×', "div": '÷', "pm": '±', "mp": '∓', "cdot": '·',
	"ast": '∗', "star": '⋆', "circ": '∘', "bullet": '•',
	"oplus": '⊕', "ominus": '⊖', "otimes": '⊗', "oslash": '⊘', "odot": '⊙',

	"leq": '≤', "geq": '≥', "neq": '≠', "approx": '≈', "equiv": '≡',
	"sim": '∼', "simeq": '≃', "cong": '≅', "propto": '∝',
	"ll": '≪', "gg": '≫',

	"in": '∈', "notin": '∉', "ni": '∋', "subset": '⊂', "supset": '⊃',
	"subseteq": '⊆', "supseteq": '⊇', "cup": '∪', "cap": '∩',
	"setminus": '∖', "emptyset": '∅', "varnothing": '∅',

	"forall": '∀', "exists": '∃', "nexists": '∄', "neg": '¬',
	"land": '∧', "lor": '∨', "wedge": '∧', "vee": '∨',

	"infty": '∞', "partial": '∂', "nabla": '∇', "hbar": 'ℏ', "ell": 'ℓ',
	"sum": '∑', "prod": '∏', "coprod": '∐',
	"int": '∫', "oint": '∮', "iint": '∬', "iiint": '∭',

	"to": '→', "rightarrow": '→', "leftarrow": '←',
	"leftrightarrow": '↔', "Rightarrow": '⇒', "Leftarrow": '⇐',
	"Leftrightarrow": '⇔', "mapsto": '↦', "uparrow": '↑', "downarrow": '↓',

	"therefore": '∴', "because": '∵', "perp": '⊥', "parallel": '∥',
	"angle": '∠', "triangle": '△', "aleph": 'ℵ',
	"dots": '…', "ldots": '…', "cdots": '⋯', "vdots": '⋮', "ddots": '⋱',
}

// spacingCommands produce no visible output of their own.
var spacingCommands = map[string]bool{
	"left": true, "right": true, ",": true, "!": true, ";": true,
	"quad": true, "qquad": true,
}

var superscripts = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴', '5': '⁵',
	'6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹', '+': '⁺', '-': '⁻',
	'=': '⁼', '(': '⁽', ')': '⁾', 'n': 'ⁿ', 'i': 'ⁱ',
	'a': 'ᵃ', 'b': 'ᵇ', 'c': 'ᶜ', 'd': 'ᵈ', 'e': 'ᵉ', 'f': 'ᶠ',
	'g': 'ᵍ', 'h': 'ʰ', 'j': 'ʲ', 'k': 'ᵏ', 'l': 'ˡ', 'm': 'ᵐ',
	'o': 'ᵒ', 'p': 'ᵖ', 'r': 'ʳ', 's': 'ˢ', 't': 'ᵗ', 'u': 'ᵘ',
	'v': 'ᵛ', 'w': 'ʷ', 'x': 'ˣ', 'y': 'ʸ', 'z': 'ᶻ',
}

var subscripts = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄', '5': '₅',
	'6': '₆', '7': '₇', '8': '₈', '9': '₉', '+': '₊', '-': '₋',
	'=': '₌', '(': '₍', ')': '₎',
	'a': 'ₐ', 'e': 'ₑ', 'h': 'ₕ', 'i': 'ᵢ', 'j': 'ⱼ', 'k': 'ₖ',
	'l': 'ₗ', 'm': 'ₘ', 'n': 'ₙ', 'o': 'ₒ', 'p': 'ₚ', 'r': 'ᵣ',
	's': 'ₛ', 't': 'ₜ', 'u': 'ᵤ', 'v': 'ᵥ', 'x': 'ₓ',
}

// ToUnicode approximates LaTeX math as plain Unicode text: symbol macros
// become their glyph, ^ and _ become super/subscripts where a mapping
// exists, \frac and \sqrt get a plain-text form. This is best-effort, not
// a full TeX parser -- anything it doesn't recognize is left as-is so
// nothing silently disappears.
func ToUnicode(source string) string {
	var out strings.Builder
	runes := []rune(source)
	i := 0
	for i < len(runes) {
		switch runes[i] {
		case '\\':
			i = writeMacro(&out, runes, i)
		case '^':
			i = writeScript(&out, runes, i+1, superscripts)
		case '_':
			i = writeScript(&out, runes, i+1, subscripts)
		case '{', '}':
			i++ // grouping braces carry no text of their own
		default:
			out.WriteRune(runes[i])
			i++
		}
	}
	return out.String()
}

// writeMacro handles a \command starting at i (runes[i] == '\\'), writing
// its Unicode form (or the raw macro if unrecognized) and returning the
// index just past it.
func writeMacro(out *strings.Builder, runes []rune, i int) int {
	start := i
	i++ // skip the backslash
	if i >= len(runes) {
		out.WriteRune('\\')
		return i
	}
	if !isLetter(runes[i]) {
		name := string(runes[i])
		i++
		if !spacingCommands[name] {
			out.WriteRune('\\')
			out.WriteRune(runes[start+1])
		}
		return i
	}

	j := i
	for j < len(runes) && isLetter(runes[j]) {
		j++
	}
	name := string(runes[i:j])

	switch name {
	case "frac":
		a, next := readGroupOrChar(runes, skipSpace(runes, j))
		b, next2 := readGroupOrChar(runes, skipSpace(runes, next))
		out.WriteString("(" + ToUnicode(a) + ")/(" + ToUnicode(b) + ")")
		return next2
	case "sqrt":
		a, next := readGroupOrChar(runes, skipSpace(runes, j))
		out.WriteString("√(" + ToUnicode(a) + ")")
		return next
	}

	if spacingCommands[name] {
		return j
	}
	if r, ok := symbols[name]; ok {
		out.WriteRune(r)
		return j
	}
	out.WriteString(string(runes[start:j]))
	return j
}

// writeScript writes the super/subscript form of the token starting at i
// (a single character or a {...} group), using table for the mapping.
// Runes without a mapping are written unchanged rather than dropped.
func writeScript(out *strings.Builder, runes []rune, i int, table map[rune]rune) int {
	token, next := readGroupOrChar(runes, i)
	for _, r := range token {
		if mapped, ok := table[r]; ok {
			out.WriteRune(mapped)
		} else {
			out.WriteRune(r)
		}
	}
	return next
}

// readGroupOrChar reads a {...} balanced group (returning its inner
// content) or, if the next rune isn't '{', a single character.
func readGroupOrChar(runes []rune, i int) (string, int) {
	if i >= len(runes) {
		return "", i
	}
	if runes[i] != '{' {
		return string(runes[i]), i + 1
	}
	depth := 1
	start := i + 1
	j := start
	for j < len(runes) && depth > 0 {
		switch runes[j] {
		case '{':
			depth++
		case '}':
			depth--
		}
		j++
	}
	end := j - 1
	if end < start {
		end = start
	}
	return string(runes[start:end]), j
}

func skipSpace(runes []rune, i int) int {
	for i < len(runes) && runes[i] == ' ' {
		i++
	}
	return i
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
