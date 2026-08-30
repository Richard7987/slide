package code

import (
	"os/exec"
	"strings"
)

// Warm primes slow interpreters used by auto-run (<!-- run -->) slides so
// the first slide that executes doesn't stall on a cold start. Currently
// only Julia benefits (its first `using` of a plotting package can take
// tens of seconds to precompile); the warm-up is a single throwaway
// process kicked off in the background. Best effort: failures are ignored.
func Warm(slides []string) {
	for _, s := range slides {
		if !strings.Contains(s, RunMarker) {
			continue
		}
		blocks, err := Parse(s)
		if err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Language == Julia {
				go func() {
					_ = exec.Command("julia", "--startup-file=no", "-e",
						"try; using UnicodePlots; catch; end").Run()
				}()
				return
			}
		}
	}
}
