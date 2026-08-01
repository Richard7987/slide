package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	stdimage "image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

// decodeGIFFrames decodes every frame of an animated GIF into full-canvas
// RGBA images (GIF frames can cover only part of the logical screen and
// rely on the previous frame's pixels showing through, or a disposal
// method, for the rest -- composing here means every returned frame is
// already a complete, self-contained image). A single-frame GIF returns
// one frame, same as decoding it as a plain static image would.
func decodeGIFFrames(path string) ([]stdimage.Image, []time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, nil, err
	}

	screen := stdimage.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvas := stdimage.NewRGBA(screen)

	frames := make([]stdimage.Image, len(g.Image))
	delays := make([]time.Duration, len(g.Image))
	var previous *stdimage.RGBA

	for i, src := range g.Image {
		var disposal byte
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			snapshot := stdimage.NewRGBA(screen)
			draw.Draw(snapshot, screen, canvas, screen.Min, draw.Src)
			previous = snapshot
		}

		draw.Draw(canvas, src.Bounds(), src, src.Bounds().Min, draw.Over)

		frame := stdimage.NewRGBA(screen)
		draw.Draw(frame, screen, canvas, screen.Min, draw.Src)
		frames[i] = frame

		// GIF delay is in 100ths of a second; encoders commonly leave it
		// at 0 meaning "as fast as possible", which every major viewer
		// instead treats as ~100ms to avoid pegging a CPU core.
		ms := 10
		if i < len(g.Delay) {
			ms = g.Delay[i] * 10
		}
		if ms <= 10 {
			ms = 100
		}
		delays[i] = time.Duration(ms) * time.Millisecond

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, src.Bounds(), &stdimage.Uniform{C: color.Transparent}, stdimage.Point{}, draw.Src)
		case gif.DisposalPrevious:
			if previous != nil {
				draw.Draw(canvas, screen, previous, screen.Min, draw.Src)
			}
		}
	}

	return frames, delays, nil
}

// TransmitAnimSeq transmits frames as a Kitty native animation under a
// single image id: the root frame goes through the same virtual-placement
// path as any static image (TransmitSeq), the rest are appended with a=f,
// and a final a=a control starts an infinite, terminal-driven loop -- so
// once sent, playback needs no further involvement from this process.
func TransmitAnimSeq(frames []stdimage.Image, delays []time.Duration, id, cols, rows int) (string, error) {
	var out strings.Builder

	seq, err := TransmitSeq(frames[0], id, cols, rows)
	if err != nil {
		return "", err
	}
	out.WriteString(seq)

	// The root frame's gap can't be set at transmit time, so it's set
	// separately, same as the protocol documents.
	out.WriteString(ansi.KittyGraphics(nil, "a=a", fmt.Sprintf("i=%d", id), "r=1", fmt.Sprintf("z=%d", delays[0].Milliseconds()), "q=2"))

	for i := 1; i < len(frames); i++ {
		var buf bytes.Buffer
		if err := png.Encode(&buf, frames[i]); err != nil {
			return "", err
		}
		out.WriteString(transmitAnimFrame(id, delays[i].Milliseconds(), buf.Bytes()))
	}

	// s=3: run, looping back to the first frame at the end. v=1: loop
	// forever.
	out.WriteString(ansi.KittyGraphics(nil, "a=a", fmt.Sprintf("i=%d", id), "s=3", "v=1", "q=2"))

	return out.String(), nil
}

// transmitAnimFrame base64-encodes and, if needed, chunks a single frame's
// PNG payload across multiple escape codes (mirroring the chunking
// convention x/ansi/kitty's own EncodeGraphics uses for a normal image
// transmission: continuation chunks repeat a= and q=, never i=).
func transmitAnimFrame(id int, gapMs int64, png []byte) string {
	b64 := []byte(base64.StdEncoding.EncodeToString(png))

	var out strings.Builder
	for i := 0; i < len(b64); i += kitty.MaxChunkSize {
		end := i + kitty.MaxChunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		isFirst := i == 0
		isLast := end == len(b64)

		var opts []string
		if isFirst {
			opts = []string{"a=f", fmt.Sprintf("i=%d", id), "f=100", fmt.Sprintf("z=%d", gapMs), "q=2"}
		} else {
			opts = []string{"a=f", "q=2"}
		}
		if !isFirst || !isLast {
			if isLast {
				opts = append(opts, "m=0")
			} else {
				opts = append(opts, "m=1")
			}
		}
		out.WriteString(ansi.KittyGraphics(chunk, opts...))
	}
	return out.String()
}
