package image

import (
	stdimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// cellWidthOverHeight approximates a terminal cell's pixel aspect ratio
// (cells are roughly twice as tall as wide in most monospace fonts). There
// is no cheap way to query the real value without an extra async
// round-trip to the terminal, so v1 uses this fixed assumption.
const cellWidthOverHeight = 0.5

// Entry is a resolved, ready-to-place image.
type Entry struct {
	ID      int
	Cols    int
	Rows    int
	PadLeft int // spaces to prepend to each row, to center it within maxCols
}

type cacheKey struct {
	path    string
	maxCols int
	maxRows int
}

// Cache decodes and transmits images over the Kitty Graphics Protocol,
// keyed by resolved path and target size budget, and assigns each a
// stable ID for the process's lifetime.
type Cache struct {
	mu      sync.RWMutex
	entries map[cacheKey]Entry
	nextID  int
}

func NewCache() *Cache {
	return &Cache{entries: make(map[cacheKey]Entry), nextID: 1}
}

// Get returns the entry for path/maxCols/maxRows if it has already been
// loaded. It does no I/O and is safe to call from View() on every frame.
func (c *Cache) Get(path string, maxCols, maxRows int) (Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[cacheKey{path, maxCols, maxRows}]
	return e, ok
}

// Load decodes and encodes the image at path if it isn't cached yet,
// returning the entry and the Kitty transmit sequence to send for it. It
// does file I/O and image decoding, so callers must only invoke it from a
// tea.Cmd, never from View(). If the entry is already cached, transmitSeq
// is empty and there is nothing new to send.
func (c *Cache) Load(path string, maxCols, maxRows int) (entry Entry, transmitSeq string, err error) {
	if e, ok := c.Get(path, maxCols, maxRows); ok {
		return e, "", nil
	}

	var (
		frames []stdimage.Image
		delays []time.Duration
		bounds stdimage.Rectangle
	)

	if strings.EqualFold(filepath.Ext(path), ".gif") {
		frames, delays, err = decodeGIFFrames(path)
		if err != nil {
			return Entry{}, "", err
		}
	}
	if len(frames) == 0 {
		f, err := os.Open(path)
		if err != nil {
			return Entry{}, "", err
		}
		img, _, err := stdimage.Decode(f)
		f.Close()
		if err != nil {
			return Entry{}, "", err
		}
		frames = []stdimage.Image{img}
	}
	bounds = frames[0].Bounds()

	cols, rows := fitSize(bounds, maxCols, maxRows)
	padLeft := (maxCols - cols) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	c.mu.Lock()
	id := c.nextID
	c.nextID++
	e := Entry{ID: id, Cols: cols, Rows: rows, PadLeft: padLeft}
	c.entries[cacheKey{path, maxCols, maxRows}] = e
	c.mu.Unlock()

	// A GIF with only one real frame (common: a still image saved with a
	// .gif extension) takes the plain, single-frame path -- no point
	// paying for animation control codes that would never fire.
	var seq string
	if len(frames) > 1 {
		seq, err = TransmitAnimSeq(frames, delays, id, cols, rows)
	} else {
		seq, err = TransmitSeq(frames[0], id, cols, rows)
	}
	if err != nil {
		return Entry{}, "", err
	}
	return e, seq, nil
}

// fitSize picks a cols x rows grid that preserves the image's aspect
// ratio, filling as much of the maxCols x maxRows budget as it can.
func fitSize(bounds stdimage.Rectangle, maxCols, maxRows int) (cols, rows int) {
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return 1, 1
	}
	if maxCols < 1 {
		maxCols = 1
	}
	if maxRows < 1 {
		maxRows = 1
	}

	cols = maxCols
	rows = int(float64(cols) * cellWidthOverHeight * float64(h) / float64(w))
	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows) / cellWidthOverHeight * float64(w) / float64(h))
	}
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	if cols > maxCols {
		cols = maxCols
	}
	return cols, rows
}
