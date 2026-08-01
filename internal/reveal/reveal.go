// Package reveal splits a slide into progressively revealed steps, marked
// by an HTML comment (<!-- pause -->) that stays invisible in any other
// markdown renderer.
package reveal

import "regexp"

var pauseRe = regexp.MustCompile(`(?m)^[ \t]*<!--[ \t]*pause[ \t]*-->[ \t]*$`)

// Steps splits slide into cumulative steps: step i contains everything up
// to and including the i-th marker. A slide with no marker always yields
// exactly one step, equal to the whole slide.
func Steps(slide string) []string {
	parts := pauseRe.Split(slide, -1)
	steps := make([]string, len(parts))
	acc := ""
	for i, part := range parts {
		acc += part
		steps[i] = acc
	}
	return steps
}
