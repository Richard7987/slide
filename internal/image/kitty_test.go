package image_test

import (
	stdimage "image"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/maaslalani/slides/internal/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceholderBlock(t *testing.T) {
	got := image.PlaceholderBlock(1, 2, 2)
	lines := strings.Split(got, "\n")
	require.Len(t, lines, 2)

	const fg = "\x1b[38;5;1m"
	const reset = "\x1b[0m"
	ph := string(kitty.Placeholder)

	assert.Equal(t, fg+ph+string(kitty.Diacritic(0))+string(kitty.Diacritic(0))+ph+reset, lines[0])
	assert.Equal(t, fg+ph+string(kitty.Diacritic(1))+string(kitty.Diacritic(0))+ph+reset, lines[1])
}

func TestPlaceholderBlock_TruecolorID(t *testing.T) {
	// A non-zero green byte forces the 24-bit branch instead of indexed color.
	got := image.PlaceholderBlock(0x00010203, 1, 1)
	assert.Contains(t, got, "\x1b[38;2;1;2;3m")
}

func TestProbeSeq(t *testing.T) {
	seq := image.ProbeSeq()
	assert.True(t, strings.HasPrefix(seq, "\x1b_G"))
	assert.Contains(t, seq, "i=999999")
}

func TestTransmitSeq(t *testing.T) {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 4, 4))
	seq, err := image.TransmitSeq(img, 7, 10, 5)
	require.NoError(t, err)
	assert.Contains(t, seq, "i=7")
	assert.Contains(t, seq, "c=10")
	assert.Contains(t, seq, "r=5")
	assert.Contains(t, seq, "f=100")
}
