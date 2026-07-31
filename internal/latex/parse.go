// Package latex renders LaTeX math ($...$ inline, $$...$$ block) found in
// slide markdown. Block math becomes a real image via internal/image and
// the Kitty Graphics Protocol; inline math is approximated as Unicode text
// so it can sit on the same line as surrounding prose.
package latex

import (
	"regexp"
	"strings"
)

// Block is a single math region found in a slide.
type Block struct {
	Raw    string
	Source string
}

var displayRe = regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)
var inlineRe = regexp.MustCompile(`\$([^$\n]+?)\$`)

// ParseDisplay extracts $$...$$ blocks, in order of appearance. It also
// returns markdown with every display span blanked out to spaces (masked),
// for ParseInline to run over -- otherwise inline's regex would treat a
// display block's interior as its own match.
func ParseDisplay(markdown string) (blocks []Block, masked string) {
	masked = markdown
	for _, m := range displayRe.FindAllStringSubmatch(markdown, -1) {
		blocks = append(blocks, Block{Raw: m[0], Source: strings.TrimSpace(m[1])})
		masked = strings.Replace(masked, m[0], strings.Repeat(" ", len(m[0])), 1)
	}
	return blocks, masked
}

// ParseInline extracts $...$ spans from already-masked markdown (see
// ParseDisplay).
func ParseInline(masked string) []Block {
	var blocks []Block
	for _, m := range inlineRe.FindAllStringSubmatch(masked, -1) {
		blocks = append(blocks, Block{Raw: m[0], Source: strings.TrimSpace(m[1])})
	}
	return blocks
}
