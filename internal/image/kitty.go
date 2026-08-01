package image

import (
	"bytes"
	"fmt"
	stdimage "image"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

// ProbeID is a reserved image ID used only to check whether the terminal
// understands the Kitty graphics protocol; never placed on screen.
const ProbeID = 999999

// ProbeSeq transmits a throwaway 1x1 image with quiet mode off, so the
// terminal answers with an OK/error response that Update can watch for.
func ProbeSeq() string {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	_ = kitty.EncodeGraphics(&buf, img, &kitty.Options{
		ID:           ProbeID,
		Action:       kitty.Transmit,
		Transmission: kitty.Direct,
		Format:       kitty.RGBA,
		ImageWidth:   1,
		ImageHeight:  1,
		Quite:        0,
	})
	return buf.String()
}

// TransmitSeq encodes img as PNG (letting the terminal decode and scale it
// itself, so we never need to resize pixels ourselves) and returns the
// escape sequence that transmits it and registers a virtual placement
// sized to cols x rows.
func TransmitSeq(img stdimage.Image, id, cols, rows int) (string, error) {
	var buf bytes.Buffer
	err := kitty.EncodeGraphics(&buf, img, &kitty.Options{
		ID:               id,
		Action:           kitty.TransmitAndPut,
		Transmission:     kitty.Direct,
		Format:           kitty.PNG,
		Columns:          cols,
		Rows:             rows,
		VirtualPlacement: true,
		Chunk:            true,
		Quite:            2,
	})
	return buf.String(), err
}

// DeleteSeq returns the escape sequence that frees an image and all its
// frames/placements (id, uppercase D value = also free the pixel data, not
// just the placements) from the terminal's memory -- needed for animated
// images once they scroll off screen, since the terminal keeps decoding
// and redrawing their frames for as long as their placement exists.
func DeleteSeq(id int) string {
	return ansi.KittyGraphics(nil, "a=d", "d=I", fmt.Sprintf("i=%d", id), "q=2")
}

// fgSGR encodes id into a foreground color escape, replicating
// ultraviolet's own reference encoding (id bytes -> indexed color when the
// two high bytes are zero, else 24-bit truecolor).
func fgSGR(id int) string {
	r := (id >> 16) & 0xff
	g := (id >> 8) & 0xff
	b := id & 0xff
	if r == 0 && g == 0 {
		return "\x1b[38;5;" + strconv.Itoa(b) + "m"
	}
	return "\x1b[38;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b) + "m"
}

// PlaceholderBlock returns the Unicode-placeholder text for a placed
// image: only the first cell of each row carries the row+column
// diacritics, the terminal infers the rest from position.
func PlaceholderBlock(id, cols, rows int) string {
	fg := fgSGR(id)
	const reset = "\x1b[0m"
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		var sb strings.Builder
		sb.WriteString(fg)
		sb.WriteRune(kitty.Placeholder)
		sb.WriteRune(kitty.Diacritic(y))
		sb.WriteRune(kitty.Diacritic(0))
		for x := 1; x < cols; x++ {
			sb.WriteRune(kitty.Placeholder)
		}
		sb.WriteString(reset)
		lines[y] = sb.String()
	}
	return strings.Join(lines, "\n")
}
