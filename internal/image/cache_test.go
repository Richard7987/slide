package image_test

import (
	stdimage "image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/maaslalani/slides/internal/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestPNG(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, png.Encode(f, img))
	return path
}

func TestCache_LoadAndGet(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPNG(t, dir, "wide.png", 200, 100)

	c := image.NewCache()

	_, ok := c.Get(path, 40, 20)
	assert.False(t, ok, "not loaded yet")

	entry, seq, err := c.Load(path, 40, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, seq)
	assert.Equal(t, 1, entry.ID)
	assert.Equal(t, 40, entry.Cols)

	got, ok := c.Get(path, 40, 20)
	assert.True(t, ok)
	assert.Equal(t, entry, got)

	entry2, seq2, err := c.Load(path, 40, 20)
	require.NoError(t, err)
	assert.Equal(t, entry, entry2)
	assert.Empty(t, seq2, "reloading a cached entry should not retransmit")
}

func TestCache_DifferentBudgetGetsDifferentID(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPNG(t, dir, "img.png", 100, 100)

	c := image.NewCache()
	e1, _, err := c.Load(path, 40, 20)
	require.NoError(t, err)
	e2, _, err := c.Load(path, 20, 20)
	require.NoError(t, err)
	e3, _, err := c.Load(path, 40, 5)
	require.NoError(t, err)

	assert.NotEqual(t, e1.ID, e2.ID, "different maxCols should be a different entry")
	assert.NotEqual(t, e1.ID, e3.ID, "different maxRows should be a different entry")
}

func TestCache_TallImageCapsRows(t *testing.T) {
	dir := t.TempDir()
	path := writeTestPNG(t, dir, "tall.png", 10, 1000)

	c := image.NewCache()
	entry, _, err := c.Load(path, 60, 20)
	require.NoError(t, err)

	assert.LessOrEqual(t, entry.Rows, 20)
	assert.Greater(t, entry.Cols, 0)
}

func TestCache_MissingFile(t *testing.T) {
	c := image.NewCache()
	_, _, err := c.Load("/no/such/file.png", 40, 20)
	assert.Error(t, err)
}

func TestCache_CentersNarrowerThanBudget(t *testing.T) {
	dir := t.TempDir()
	// A very tall, narrow image: the row budget forces cols well under
	// maxCols, so it should end up padded to sit centered.
	path := writeTestPNG(t, dir, "narrow.png", 10, 1000)

	c := image.NewCache()
	entry, _, err := c.Load(path, 100, 20)
	require.NoError(t, err)

	require.Less(t, entry.Cols, 100)
	assert.Equal(t, (100-entry.Cols)/2, entry.PadLeft)
}

func TestCache_FillsBudgetGetsNoPadding(t *testing.T) {
	dir := t.TempDir()
	// A very wide image at a tight row budget: cols == maxCols, so it
	// should need no centering.
	path := writeTestPNG(t, dir, "wide2.png", 2000, 10)

	c := image.NewCache()
	entry, _, err := c.Load(path, 80, 5)
	require.NoError(t, err)

	assert.Equal(t, 80, entry.Cols)
	assert.Equal(t, 0, entry.PadLeft)
}
