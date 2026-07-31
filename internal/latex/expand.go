package latex

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/maaslalani/slides/internal/image"
)

// Own Private Use Area marker (TEX instead of image's IMG), so sentinel
// numbering never needs to be coordinated with internal/image's.
const sentinelOpen = "TEX"
const sentinelClose = ""

var sentinelRe = regexp.MustCompile(sentinelOpen + `(\d+)` + sentinelClose)

func sentinel(i int) string {
	return fmt.Sprintf("%s%d%s", sentinelOpen, i, sentinelClose)
}

// Preprocess finds display ($$...$$) and inline ($...$) math in markdown.
// Display blocks become a real image (sentinel + placement) when
// graphicsOK and lookup resolves their cache entry; when graphics are
// supported but the entry isn't ready yet, they get loading (e.g. a
// spinner frame) instead. Without graphics support at all -- and always
// for inline math, which never renders as an image -- they're replaced
// with a Unicode approximation so they still read as text.
func Preprocess(markdown string, dark, graphicsOK bool, loading string, lookup func(path string) (image.Entry, bool)) (string, []image.Placement) {
	display, masked := ParseDisplay(markdown)
	inline := ParseInline(masked)

	var placements []image.Placement
	for _, b := range display {
		if graphicsOK {
			if entry, ok := lookup(CachePath(b.Source, dark)); ok {
				markdown = strings.Replace(markdown, b.Raw, sentinel(len(placements)), 1)
				placements = append(placements, image.Placement{Entry: entry})
				continue
			}
			markdown = strings.Replace(markdown, b.Raw, loading, 1)
			continue
		}
		markdown = strings.Replace(markdown, b.Raw, ToUnicode(b.Source), 1)
	}
	for _, b := range inline {
		markdown = strings.Replace(markdown, b.Raw, ToUnicode(b.Source), 1)
	}
	return markdown, placements
}

// Expand mirrors image.Expand's line-splicing logic, matched against
// latex's own sentinel marker.
func Expand(rendered string, placements []image.Placement) string {
	if len(placements) == 0 {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		m := sentinelRe.FindStringSubmatch(line)
		if m == nil {
			out = append(out, line)
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil || idx < 0 || idx >= len(placements) {
			out = append(out, line)
			continue
		}
		e := placements[idx].Entry
		pad := strings.Repeat(" ", e.PadLeft)
		for _, row := range strings.Split(image.PlaceholderBlock(e.ID, e.Cols, e.Rows), "\n") {
			out = append(out, pad+row)
		}
	}
	return strings.Join(out, "\n")
}
