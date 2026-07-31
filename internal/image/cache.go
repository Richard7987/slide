package image

import (
	stdimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sync"
)

// cellWidthOverHeight approximates a terminal cell's pixel aspect ratio
// (cells are roughly twice as tall as wide in most monospace fonts). There
// is no cheap way to query the real value without an extra async
// round-trip to the terminal, so v1 uses this fixed assumption.
const cellWidthOverHeight = 0.5

// defaultMaxCols caps how wide a single image gets by default, so one
// picture doesn't consume the whole slide width on a very wide terminal.
const defaultMaxCols = 60

// maxRows caps how tall a single image gets, so a narrow/tall image
// doesn't consume the whole slide height.
const maxRows = 20

// Entry is a resolved, ready-to-place image.
type Entry struct {
	ID   int
	Cols int
	Rows int
}

type cacheKey struct {
	path    string
	maxCols int
}

// Cache decodes and transmits images over the Kitty Graphics Protocol,
// keyed by resolved path and target width, and assigns each a stable ID
// for the process's lifetime.
type Cache struct {
	mu      sync.RWMutex
	entries map[cacheKey]Entry
	nextID  int
}

func NewCache() *Cache {
	return &Cache{entries: make(map[cacheKey]Entry), nextID: 1}
}

// Get returns the entry for path/maxCols if it has already been loaded.
// It does no I/O and is safe to call from View() on every frame.
func (c *Cache) Get(path string, maxCols int) (Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[cacheKey{path, maxCols}]
	return e, ok
}

// Load decodes and encodes the image at path if it isn't cached yet,
// returning the entry and the Kitty transmit sequence to send for it. It
// does file I/O and image decoding, so callers must only invoke it from a
// tea.Cmd, never from View(). If the entry is already cached, transmitSeq
// is empty and there is nothing new to send.
func (c *Cache) Load(path string, maxCols int) (entry Entry, transmitSeq string, err error) {
	if e, ok := c.Get(path, maxCols); ok {
		return e, "", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return Entry{}, "", err
	}
	defer f.Close()

	img, _, err := stdimage.Decode(f)
	if err != nil {
		return Entry{}, "", err
	}

	cols, rows := fitSize(img.Bounds(), maxCols)

	c.mu.Lock()
	id := c.nextID
	c.nextID++
	e := Entry{ID: id, Cols: cols, Rows: rows}
	c.entries[cacheKey{path, maxCols}] = e
	c.mu.Unlock()

	seq, err := TransmitSeq(img, id, cols, rows)
	if err != nil {
		return Entry{}, "", err
	}
	return e, seq, nil
}

// fitSize picks a cols x rows grid that preserves the image's aspect
// ratio within the column/row caps.
func fitSize(bounds stdimage.Rectangle, maxCols int) (cols, rows int) {
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return 1, 1
	}

	cols = maxCols
	if cols > defaultMaxCols {
		cols = defaultMaxCols
	}
	if cols < 1 {
		cols = 1
	}

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
	return cols, rows
}
