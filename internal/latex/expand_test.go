package latex_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/maaslalani/slides/internal/image"
	"github.com/maaslalani/slides/internal/latex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocess_DisplayBecomesImage(t *testing.T) {
	md := "before\n\n$$ x^2 $$\n\nafter " + `$\alpha$` + " end"

	lookup := func(path string) (image.Entry, bool) {
		return image.Entry{ID: 3, Cols: 5, Rows: 2}, true
	}

	pre, placements := latex.Preprocess(md, true, true, lookup)
	require.Len(t, placements, 1)
	assert.Equal(t, image.Entry{ID: 3, Cols: 5, Rows: 2}, placements[0].Entry)
	assert.NotContains(t, pre, "$$")
	assert.Contains(t, pre, "α")
}

func TestPreprocess_FallsBackToUnicodeWhenNotReady(t *testing.T) {
	md := "$$ x^2 $$"
	pre, placements := latex.Preprocess(md, true, true, func(string) (image.Entry, bool) {
		return image.Entry{}, false
	})
	assert.Empty(t, placements)
	assert.Contains(t, pre, "x²")
}

func TestPreprocess_FallsBackToUnicodeWithoutGraphics(t *testing.T) {
	md := "$$ x^2 $$"
	pre, placements := latex.Preprocess(md, true, false, func(string) (image.Entry, bool) {
		t.Fatal("lookup should not be called when graphicsOK is false")
		return image.Entry{}, false
	})
	assert.Empty(t, placements)
	assert.Contains(t, pre, "x²")
}

func TestExpand(t *testing.T) {
	md := "$$ x^2 $$"
	pre, placements := latex.Preprocess(md, true, true, func(string) (image.Entry, bool) {
		return image.Entry{ID: 1, Cols: 4, Rows: 3}, true
	})

	expanded := latex.Expand(pre, placements)
	placeholderLines := 0
	for _, l := range strings.Split(expanded, "\n") {
		if strings.Contains(l, string(kitty.Placeholder)) {
			placeholderLines++
		}
	}
	assert.Equal(t, 3, placeholderLines)
}
