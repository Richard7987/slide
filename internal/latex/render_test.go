package latex_test

import (
	"image"
	_ "image/png"
	"os"
	"testing"

	"github.com/maaslalani/slides/internal/latex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	if testing.Short() {
		t.Skip("shells out to tectonic and gs")
	}

	path, err := latex.Render(`\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}`, true)
	require.NoError(t, err)

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	img, format, err := image.Decode(f)
	require.NoError(t, err)
	assert.Equal(t, "png", format)
	assert.Greater(t, img.Bounds().Dx(), 0)
	assert.Greater(t, img.Bounds().Dy(), 0)
}

func TestRender_CachesOnDisk(t *testing.T) {
	if testing.Short() {
		t.Skip("shells out to tectonic and gs")
	}

	source := `y = mx + b`
	path1, err := latex.Render(source, false)
	require.NoError(t, err)

	path2, err := latex.Render(source, false)
	require.NoError(t, err)

	assert.Equal(t, path1, path2)
}

func TestCachePath_DeterministicAndColorSensitive(t *testing.T) {
	p1 := latex.CachePath("x^2", true)
	p2 := latex.CachePath("x^2", true)
	p3 := latex.CachePath("x^2", false)

	assert.Equal(t, p1, p2)
	assert.NotEqual(t, p1, p3)
}
