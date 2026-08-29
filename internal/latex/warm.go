package latex

import (
	"runtime"
	"sync"
)

// Warm pre-renders every unique $$…$$ block across slides, concurrently,
// populating the on-disk PNG cache so the first navigation to a slide with
// equations is instant instead of paying tectonic's cold start. Best
// effort: individual render failures are ignored, and a block whose PNG
// already exists is skipped cheaply by Render itself.
func Warm(slides []string, dark bool) {
	seen := make(map[string]struct{})
	var sources []string
	for _, s := range slides {
		blocks, _ := ParseDisplay(s)
		for _, b := range blocks {
			if _, ok := seen[b.Source]; ok {
				continue
			}
			seen[b.Source] = struct{}{}
			sources = append(sources, b.Source)
		}
	}
	if len(sources) == 0 {
		return
	}

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}

	ch := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for src := range ch {
				_, _ = Render(src, dark)
			}
		}()
	}
	for _, src := range sources {
		ch <- src
	}
	close(ch)
	wg.Wait()
}
