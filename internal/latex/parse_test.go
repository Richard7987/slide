package latex_test

import (
	"testing"

	"github.com/maaslalani/slides/internal/latex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDisplay(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []latex.Block
	}{
		{
			name:     "no math",
			markdown: "just text",
			want:     nil,
		},
		{
			name:     "single block",
			markdown: "before\n\n$$ x^2 $$\n\nafter",
			want: []latex.Block{
				{Raw: "$$ x^2 $$", Source: "x^2"},
			},
		},
		{
			name:     "two blocks",
			markdown: "$$a$$ and $$b$$",
			want: []latex.Block{
				{Raw: "$$a$$", Source: "a"},
				{Raw: "$$b$$", Source: "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, _ := latex.ParseDisplay(tt.markdown)
			assert.Equal(t, tt.want, blocks)
		})
	}
}

func TestParseDisplay_Masking(t *testing.T) {
	_, masked := latex.ParseDisplay("$$ x^2 $$")
	assert.NotContains(t, masked, "$")
}

func TestParseInline(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []latex.Block
	}{
		{"no math", "just text", nil},
		{"single inline", "the value $x^2$ here", []latex.Block{{Raw: "$x^2$", Source: "x^2"}}},
		{"two inline", "$a$ and $b$", []latex.Block{{Raw: "$a$", Source: "a"}, {Raw: "$b$", Source: "b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, latex.ParseInline(tt.markdown))
		})
	}
}

func TestParseInline_IgnoresDisplayBlocks(t *testing.T) {
	md := "before $$ x^2 $$ after $y$ end"
	display, masked := latex.ParseDisplay(md)
	require.Len(t, display, 1)

	inline := latex.ParseInline(masked)
	assert.Equal(t, []latex.Block{{Raw: "$y$", Source: "y"}}, inline)
}
