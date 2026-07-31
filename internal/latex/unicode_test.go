package latex_test

import (
	"testing"

	"github.com/maaslalani/slides/internal/latex"
	"github.com/stretchr/testify/assert"
)

func TestToUnicode(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"plain text", "x + y", "x + y"},
		{"greek letter", `\alpha`, "α"},
		{"superscript digit", "x^2", "x²"},
		{"superscript group", "x^{23}", "x²³"},
		{"subscript digit", "x_1", "x₁"},
		{"subscript group", "a_{ij}", "aᵢⱼ"},
		{"unmapped superscript falls back to raw char", "x^q", "xq"},
		{"frac simple", `\frac{a}{b}`, "(a)/(b)"},
		{"frac nested macro", `\frac{\alpha}{2}`, "(α)/(2)"},
		{"sqrt simple", `\sqrt{x}`, "√(x)"},
		{"sqrt nested", `\sqrt{x^2}`, "√(x²)"},
		{"unrecognized macro left as-is", `\foobar`, `\foobar`},
		{"spacing commands dropped", `a\,b\;c\!d`, "abcd"},
		{"left right dropped", `\left(x\right)`, "(x)"},
		{"braces alone are stripped", "{x}", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, latex.ToUnicode(tt.source))
		})
	}
}

func TestToUnicode_GreekInExpression(t *testing.T) {
	got := latex.ToUnicode(`\alpha^2 + \beta \leq \gamma`)
	assert.Equal(t, "α² + β ≤ γ", got)
}
