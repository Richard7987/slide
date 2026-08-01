package image_test

import (
	stdimage "image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maaslalani/slides/internal/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestGIF(t *testing.T, dir, name string, delays []int) string {
	t.Helper()
	palette := []color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}}

	g := &gif.GIF{}
	for i, d := range delays {
		frame := stdimage.NewPaletted(stdimage.Rect(0, 0, 20, 10), palette)
		fill := uint8(i % len(palette))
		for y := 0; y < 10; y++ {
			for x := 0; x < 20; x++ {
				frame.SetColorIndex(x, y, fill)
			}
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, d)
		g.Disposal = append(g.Disposal, gif.DisposalNone)
	}

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, gif.EncodeAll(f, g))
	return path
}

func TestCache_LoadAnimatedGIF(t *testing.T) {
	dir := t.TempDir()
	path := writeTestGIF(t, dir, "anim.gif", []int{10, 20, 0})

	c := image.NewCache()
	entry, seq, err := c.Load(path, 40, 20)
	require.NoError(t, err)
	require.NotEmpty(t, seq)
	assert.Equal(t, 1, entry.ID)

	// Root frame, then two more appended, then playback control.
	assert.Equal(t, 2, strings.Count(seq, "a=f"))
	assert.Contains(t, seq, "a=a")
	assert.Contains(t, seq, "s=3")
	assert.Contains(t, seq, "v=1")
	// Frame with Delay=0 falls back to the ~100ms convention, not 0ms.
	assert.NotContains(t, seq, "z=0")
}

func TestCache_LoadSingleFrameGIFIsPlain(t *testing.T) {
	dir := t.TempDir()
	path := writeTestGIF(t, dir, "still.gif", []int{10})

	c := image.NewCache()
	_, seq, err := c.Load(path, 40, 20)
	require.NoError(t, err)

	assert.NotContains(t, seq, "a=f", "a single-frame GIF shouldn't pay for animation control codes")
	assert.NotContains(t, seq, "a=a")
}

func TestDeleteSeq(t *testing.T) {
	seq := image.DeleteSeq(42)
	assert.Contains(t, seq, "a=d")
	assert.Contains(t, seq, "d=I")
	assert.Contains(t, seq, "i=42")
}
