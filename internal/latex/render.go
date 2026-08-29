package latex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dpi is a fixed rasterization resolution. The PNG is always downscaled to
// a handful of terminal cells by internal/image, so 200 is visually
// indistinguishable from 300 on screen while roughly halving Ghostscript's
// work and the PNG's decode/transmit cost -- the first run-through of a
// deck with many equations is noticeably quicker.
const dpi = 200

// preview, not the default box-register backend, is required: standalone's
// default backend fails to detect an equation* (or \[...\]) as the
// document's content and errors with "Missing $ inserted" -- confirmed by
// hand against this tectonic bundle before wiring this in.
const texTemplate = `\documentclass[preview,border=2pt]{standalone}
\usepackage{amsmath,amssymb}
\usepackage{xcolor}
\begin{document}
\color{%s}
\[
%s
\]
\end{document}
`

func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "slides", "latex")
}

// CachePath returns the deterministic on-disk location for source's
// rendered PNG. It does no I/O, so it's safe to call from View().
func CachePath(source string, dark bool) string {
	h := sha256.New()
	h.Write([]byte(source))
	if dark {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	return filepath.Join(cacheDir(), hex.EncodeToString(h.Sum(nil))+".png")
}

// Render compiles source (a LaTeX math expression) to a tightly-cropped
// transparent PNG and returns its path, reusing a previous render if
// CachePath already exists on disk. It shells out to tectonic and
// ghostscript, so it must only run inside a tea.Cmd, never from View().
func Render(source string, dark bool) (string, error) {
	path := CachePath(source, dark)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		return "", err
	}

	tmp, err := os.MkdirTemp("", "slides-latex-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	color := "black"
	if dark {
		color = "white"
	}
	tex := fmt.Sprintf(texTemplate, color, source)

	tectonic := exec.Command("tectonic", "--outdir", tmp, "--outfmt", "pdf", "--untrusted", "-")
	tectonic.Stdin = strings.NewReader(tex)
	if out, err := tectonic.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tectonic: %w: %s", err, out)
	}

	// standalone crops the PDF page to the content bounding box already,
	// so rasterizing gives an already-tight PNG -- no trim step needed.
	pdf := filepath.Join(tmp, "texput.pdf")
	png := filepath.Join(tmp, "out.png")
	gs := exec.Command("gs",
		"-dSAFER", "-dBATCH", "-dNOPAUSE", "-dQUIET",
		"-dTextAlphaBits=4", "-dGraphicsAlphaBits=4",
		"-sDEVICE=pngalpha", fmt.Sprintf("-r%d", dpi),
		"-sOutputFile="+png, pdf,
	)
	if out, err := gs.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gs: %w: %s", err, out)
	}

	if err := os.Rename(png, path); err != nil {
		data, rerr := os.ReadFile(png)
		if rerr != nil {
			return "", rerr
		}
		if werr := os.WriteFile(path, data, 0o644); werr != nil {
			return "", werr
		}
	}

	return path, nil
}
