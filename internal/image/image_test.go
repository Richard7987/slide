package image_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/maaslalani/slides/internal/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []image.Ref
	}{
		{
			name:     "no images",
			markdown: "# Just a title\n\nSome text.",
			want:     nil,
		},
		{
			name:     "image without caption",
			markdown: "![a diagram](./diagram.png)",
			want: []image.Ref{
				{Raw: "![a diagram](./diagram.png)", Alt: "a diagram", Src: "./diagram.png"},
			},
		},
		{
			name:     "image with caption",
			markdown: `![a diagram](./diagram.png "Figure 1: the diagram")`,
			want: []image.Ref{
				{
					Raw:     `![a diagram](./diagram.png "Figure 1: the diagram")`,
					Alt:     "a diagram",
					Src:     "./diagram.png",
					Caption: "Figure 1: the diagram",
				},
			},
		},
		{
			name:     "two images on the same slide",
			markdown: "![a](a.png)\n\n![b](b.png)",
			want: []image.Ref{
				{Raw: "![a](a.png)", Alt: "a", Src: "a.png"},
				{Raw: "![b](b.png)", Alt: "b", Src: "b.png"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, image.Parse(tt.markdown))
		})
	}
}

func TestPreprocessAndExpand(t *testing.T) {
	md := "# Title\n\n![a chart](chart.png \"Figure 1\")\n\nSome text after."

	lookup := func(ref image.Ref) (image.Entry, bool) {
		if ref.Src == "chart.png" {
			return image.Entry{ID: 5, Cols: 10, Rows: 3}, true
		}
		return image.Entry{}, false
	}

	pre, placements := image.Preprocess(md, lookup)
	require.Len(t, placements, 1)
	assert.Equal(t, image.Entry{ID: 5, Cols: 10, Rows: 3}, placements[0].Entry)
	assert.NotContains(t, pre, "![a chart]")
	assert.Contains(t, pre, "Figure 1")

	expanded := image.Expand(pre, placements)
	placeholderLines := 0
	for _, l := range strings.Split(expanded, "\n") {
		if strings.Contains(l, string(kitty.Placeholder)) {
			placeholderLines++
		}
	}
	assert.Equal(t, 3, placeholderLines)
}

func TestPreprocess_SkipsUnresolvedRefs(t *testing.T) {
	md := "![missing](nope.png)"
	pre, placements := image.Preprocess(md, func(image.Ref) (image.Entry, bool) {
		return image.Entry{}, false
	})
	assert.Equal(t, md, pre)
	assert.Empty(t, placements)
}
