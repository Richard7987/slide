package model

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/maaslalani/slides/internal/file"
	"github.com/maaslalani/slides/internal/navigation"
	"github.com/maaslalani/slides/internal/process"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/maaslalani/slides/internal/bib"
	"github.com/maaslalani/slides/internal/code"
	"github.com/maaslalani/slides/internal/image"
	"github.com/maaslalani/slides/internal/latex"
	"github.com/maaslalani/slides/internal/meta"
	"github.com/maaslalani/slides/internal/reveal"
	"github.com/maaslalani/slides/styles"
	"github.com/muesli/termenv"
)

var (
	//go:embed tutorial.md
	slidesTutorial []byte
	tabSpaces      = strings.Repeat(" ", 4)
)

const (
	delimiter = "\n---\n"
)

// Model represents the model of this presentation, which contains all the
// state related to the current slides.
type Model struct {
	Slides   []string
	Page     int
	Author   string
	Date     string
	Theme    glamour.TermRendererOption
	Paging   string
	FileName string
	viewport viewport.Model
	buffer   string
	// VirtualText is used for additional information that is not part of the
	// original slides, it will be displayed on a slide and reset on page change
	VirtualText string
	Search      navigation.Search

	images     *image.Cache
	graphicsOK bool
	darkBG     bool
	// mediaRows is the per-item row budget for the current slide's images
	// and $$ blocks. Recomputed (from the measured height of the slide's
	// prose) whenever the page or the viewport size changes, so figures
	// shrink to leave room for whatever text shares the slide and follow
	// Kitty's font-size / zoom live.
	mediaRows int
	spinner   spinner.Model
	// step is the current slide's reveal progress, an index into
	// reveal.Steps(current slide).
	step int

	// executing is true while a ctrl+e code execution is in flight.
	executing bool
	// execDirs are the temporary directories the current slide's last
	// ctrl+e execution ran in, owned by Model until the page changes.
	execDirs []string
	// execImages are the image files (plots, etc.) that execution left
	// behind, in display order.
	execImages []string
}

// nextKeys/prevKeys are the single-keystroke navigation keys (as opposed to
// a numeric buffer or the "g"/"G" jump keys) -- pressing one of these is
// what advances/retracts a reveal step before it pages, mirroring
// navigation.Navigate's own key sets.
var (
	nextKeys = map[string]bool{" ": true, "down": true, "j": true, "right": true, "l": true, "enter": true, "n": true, "pgdown": true}
	prevKeys = map[string]bool{"up": true, "k": true, "left": true, "h": true, "p": true, "pgup": true, "N": true}
)

func bufferIsNumeric(buffer string) bool {
	_, err := strconv.Atoi(buffer)
	return err == nil
}

type fileWatchMsg struct{}

// imagesLoadedMsg carries the Kitty transmit sequences for images that
// finished decoding since the last time it was requested.
type imagesLoadedMsg struct{ seqs []string }

var fileInfo os.FileInfo

// Init initializes the model, begins watching the slides file for changes
// if it exists, and probes whether the terminal supports the Kitty
// graphics protocol.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.Raw(image.ProbeSeq()), m.spinner.Tick}
	if m.FileName != "" {
		fileInfo, _ = os.Stat(m.FileName)
		cmds = append(cmds, fileWatchCmd())
	}
	return tea.Batch(cmds...)
}

// imageBaseDir returns the directory image sources are resolved relative
// to: the slides file's directory, or the working directory for stdin.
func (m *Model) imageBaseDir() string {
	if m.FileName == "" {
		return "."
	}
	return filepath.Dir(m.FileName)
}

// presentation margin reserved on each side of the viewport, so slides
// (text, images, everything) never run edge-to-edge.
const hMarginFrac = 0.04
const vMarginFrac = 0.03

// contentSize returns the viewport size minus the presentation margin --
// the width/height everything on the slide is laid out within.
func (m *Model) contentSize() (width, height int) {
	width = m.viewport.Width() - 2*int(float64(m.viewport.Width())*hMarginFrac)
	height = m.viewport.Height() - 2*int(float64(m.viewport.Height())*vMarginFrac)
	if width < 1 {
		width = m.viewport.Width()
	}
	if height < 1 {
		height = m.viewport.Height()
	}
	return width, height
}

// leftMargin is how many columns to indent the whole rendered slide by.
// Since its content is contentSize()-wide, this centers it in the
// viewport, giving an equal margin on both sides.
func (m *Model) leftMargin() int {
	contentWidth, _ := m.contentSize()
	margin := (m.viewport.Width() - contentWidth) / 2
	if margin < 0 {
		return 0
	}
	return margin
}

// mediaSize returns the column/row budget for one media item on the
// current slide: the full content width, and the per-item row budget
// computed by refreshMediaBudget (with a conservative fallback for the
// first frame, before any WindowSizeMsg has arrived).
func (m *Model) mediaSize() (maxCols, maxRows int) {
	contentWidth, contentHeight := m.contentSize()
	maxRows = m.mediaRows
	if maxRows < 3 {
		maxRows = contentHeight - 10
		if maxRows < 3 {
			maxRows = 3
		}
	}
	return contentWidth, maxRows
}

// refreshMediaBudget recomputes mediaRows for the current slide. Call it
// whenever the page or viewport size changes, before mediaSize is read.
func (m *Model) refreshMediaBudget() {
	if m.Page < 0 || m.Page >= len(m.Slides) {
		return
	}
	m.mediaRows = m.computeMediaBudget(m.Slides[m.Page])
}

// computeMediaBudget returns how many rows one image or $$ block on slide
// may use. It renders the slide's prose alone -- every image and display
// block removed -- to measure how many rows the surrounding text really
// occupies at the current width, then splits what's left evenly across
// the media items. This is what keeps a figure from overrunning the
// caption or paragraph beneath it, and it re-measures on every resize so
// the fit tracks Kitty's zoom.
func (m *Model) computeMediaBudget(slide string) int {
	contentWidth, contentHeight := m.contentSize()

	n := len(image.Parse(slide))
	if blocks, _ := latex.ParseDisplay(slide); len(blocks) > 0 {
		n += len(blocks)
	}
	if n < 1 {
		return contentHeight
	}

	textRows := 3
	if r, err := glamour.NewTermRenderer(m.Theme, glamour.WithWordWrap(contentWidth)); err == nil {
		if out, rerr := r.Render(code.HideComments(stripMedia(slide))); rerr == nil {
			textRows = lipgloss.Height(strings.TrimRight(out, "\n"))
		}
	}

	// styles.Slide's Padding(1) (2 rows) + the status bar and its padding
	// (~3 rows) + one row of separation per media item + a small cushion so
	// a slightly-off prose measurement never spills past the last row.
	reserved := textRows + 7 + n
	budget := contentHeight - reserved
	if budget < 3*n {
		budget = 3 * n
	}
	return budget / n
}

// stripMedia removes a slide's images and $$ blocks but keeps each
// image's caption as a line of text, so computeMediaBudget measures the
// prose that will actually surround the figures.
func stripMedia(slide string) string {
	_, slide = latex.ParseDisplay(slide) // display blocks -> equal-length spaces
	for _, ref := range image.Parse(slide) {
		repl := ""
		if ref.Caption != "" {
			repl = "*" + ref.Caption + "*"
		}
		slide = strings.Replace(slide, ref.Raw, repl, 1)
	}
	return slide
}

// execMediaSize is the column/row budget for a code execution's plot: half
// of mediaSize's row budget, so the code's stdout still has room next to
// the image it produced.
func (m *Model) execMediaSize() (maxCols, maxRows int) {
	maxCols, maxRows = m.mediaSize()
	maxRows /= 2
	if maxRows < 2 {
		maxRows = 2
	}
	return maxCols, maxRows
}

// loadPathSeq loads path into images if it isn't cached yet, returning its
// Kitty transmit sequence. ok is false both on failure and when the entry
// was already cached (nothing new to send).
func loadPathSeq(images *image.Cache, path string, maxCols, maxRows int) (seq string, ok bool) {
	_, seq, err := images.Load(path, maxCols, maxRows)
	if err != nil || seq == "" {
		return "", false
	}
	return seq, true
}

// loadPathsCmd decodes and transmits any of paths not already cached. It
// does file I/O, so it must only run inside a tea.Cmd.
func loadPathsCmd(images *image.Cache, paths []string, maxCols, maxRows int) tea.Cmd {
	if images == nil || len(paths) == 0 {
		return nil
	}
	return func() tea.Msg {
		var seqs []string
		for _, path := range paths {
			if seq, ok := loadPathSeq(images, path, maxCols, maxRows); ok {
				seqs = append(seqs, seq)
			}
		}
		if len(seqs) == 0 {
			return nil
		}
		return imagesLoadedMsg{seqs: seqs}
	}
}

// loadImagesCmd decodes and encodes any not-yet-cached images or display
// math referenced by slide, off the render path. It does file I/O (and,
// for display math, shells out to tectonic/gs), so it must only run
// inside a tea.Cmd.
func loadImagesCmd(images *image.Cache, baseDir, slide string, maxCols, maxRows int, darkBG bool) tea.Cmd {
	refs := image.Parse(slide)
	display, _ := latex.ParseDisplay(slide)
	if images == nil || (len(refs) == 0 && len(display) == 0) {
		return nil
	}
	return func() tea.Msg {
		var seqs []string
		for _, ref := range refs {
			path := filepath.Join(baseDir, ref.Src)
			if seq, ok := loadPathSeq(images, path, maxCols, maxRows); ok {
				seqs = append(seqs, seq)
			}
		}
		// Rasterize this slide's $$…$$ blocks concurrently -- tectonic's
		// cold start dominates, so a slide with two or three equations
		// would otherwise stall on them one after another.
		rendered := make([]string, len(display))
		var wg sync.WaitGroup
		for i, b := range display {
			wg.Add(1)
			go func(i int, src string) {
				defer wg.Done()
				if p, err := latex.Render(src, darkBG); err == nil {
					rendered[i] = p
				}
			}(i, b.Source)
		}
		wg.Wait()
		for _, p := range rendered {
			if p == "" {
				continue
			}
			if seq, ok := loadPathSeq(images, p, maxCols, maxRows); ok {
				seqs = append(seqs, seq)
			}
		}
		if len(seqs) == 0 {
			return nil
		}
		return imagesLoadedMsg{seqs: seqs}
	}
}

// codeExecutedMsg carries the result of an async ctrl+e code execution.
// page pins it to the slide it was launched from, so a slide change while
// execution is still in flight doesn't apply a stale result.
type codeExecutedMsg struct {
	page   int
	text   string
	dirs   []string
	images []string
}

// executeCodeCmd runs blocks (potentially slow: Julia's first run compiles
// its dependencies) off the render path, in its own goroutine managed by
// bubbletea.
func executeCodeCmd(page int, blocks []code.Block) tea.Cmd {
	return func() tea.Msg {
		var outs []string
		var dirs []string
		var images []string
		for _, block := range blocks {
			res := code.Execute(block)
			outs = append(outs, res.Out)
			if res.WorkDir != "" {
				dirs = append(dirs, res.WorkDir)
			}
			images = append(images, res.Images...)
		}
		return codeExecutedMsg{page: page, text: strings.Join(outs, "\n"), dirs: dirs, images: images}
	}
}

// juliaEnv tunes Julia's runtime for headless, non-interactive execution:
// GKSwstype=100 keeps GR.jl/Plots.jl's GR backend from trying to open a
// window it can't (it would otherwise hang waiting for a display), and
// COLUMNS/LINES give UnicodePlots.jl a real canvas size to draw into --
// without a TTY, Julia's own displaysize() falls back to a fixed 80x24.
func juliaEnv(maxCols, maxRows int) map[string]string {
	return map[string]string{
		"GKSwstype": "100",
		"COLUMNS":   strconv.Itoa(maxCols),
		"LINES":     strconv.Itoa(maxRows),
	}
}

// centerMarker on its own line makes renderSlide horizontally center the
// slide's text.
const centerMarker = "<!-- center -->"

// centerLines horizontally centers every non-blank text line of s within
// width. Lines carrying a Kitty graphics placeholder are left untouched --
// their own PadLeft already centers the image.
func centerLines(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.ContainsRune(line, kitty.Placeholder) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		w := lipgloss.Width(trimmed)
		if w == 0 || w >= width {
			continue
		}
		lines[i] = strings.Repeat(" ", (width-w)/2) + trimmed
	}
	return strings.Join(lines, "\n")
}

// resolveThemeRef makes a relative `theme:` path in the frontmatter
// resolve against the presentation file's directory (as bibliography and
// images already do), not the process's working directory. Builtin
// keywords, URLs and absolute paths pass through untouched.
func (m *Model) resolveThemeRef(ref string) string {
	switch ref {
	case "", "ascii", "light", "dark", "notty":
		return ref
	}
	if strings.HasPrefix(ref, "http") || filepath.IsAbs(ref) {
		return ref
	}
	return filepath.Join(m.imageBaseDir(), ref)
}

func fileWatchCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

// Load loads all of the content and metadata for the presentation.
func (m *Model) Load() error {
	var content string
	var err error

	if m.FileName != "" {
		content, err = readFile(m.FileName)
	} else {
		content, err = readStdin()
	}

	if err != nil {
		return err
	}

	content = strings.ReplaceAll(content, "\r", "")

	content = strings.TrimPrefix(content, strings.TrimPrefix(delimiter, "\n"))
	slides := strings.Split(content, delimiter)

	metaData, exists := meta.New().Parse(slides[0])
	// If the user specifies a custom configuration options
	// skip the first "slide" since this is all configuration
	if exists && len(slides) > 1 {
		slides = slides[1:]
	}

	m.Slides = slides
	m.Author = metaData.Author
	m.Date = metaData.Date
	m.Paging = metaData.Paging
	if m.Theme == nil {
		m.Theme = styles.SelectTheme(m.resolveThemeRef(metaData.Theme))
		// Computed here, once, before the tea.Program takes over the
		// terminal: HasDarkBackground does its own OSC 11 query, which
		// would race with Bubble Tea's input reader if called later from
		// a tea.Cmd while the program is running.
		m.darkBG = termenv.HasDarkBackground()
		m.spinner = spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	if m.images == nil {
		m.images = image.NewCache()
	}
	if metaData.Bibliography != "" {
		if entries, err := bib.ParseFile(filepath.Join(m.imageBaseDir(), metaData.Bibliography)); err == nil {
			m.Slides = bib.Process(m.Slides, entries)
		}
	}

	// Pre-render every $$…$$ block in the deck, in parallel, in the
	// background: navigating to a slide with equations then costs a cache
	// hit instead of tectonic's cold start one block at a time. A copy of
	// the slice is passed so a concurrent file-watch reload can't race the
	// warmer.
	go latex.Warm(append([]string(nil), m.Slides...), m.darkBG)

	return nil
}

// Update updates the presentation model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height)
		m.refreshMediaBudget()
		if !m.graphicsOK || len(m.Slides) == 0 {
			return m, nil
		}
		maxCols, maxRows := m.mediaSize()
		return m, loadImagesCmd(m.images, m.imageBaseDir(), m.Slides[m.Page], maxCols, maxRows, m.darkBG)

	case tea.KeyPressMsg:
		keyPress := msg.String()

		if m.Search.Active {
			switch msg.String() {
			case "enter":
				// execute current buffer
				if m.Search.Query() != "" {
					m.Search.Execute(&m)
				} else {
					m.Search.Done()
				}
				// cancel search
				return m, nil
			case "ctrl+c", "esc":
				// quit command mode
				m.Search.SetQuery("")
				m.Search.Done()
				return m, nil
			}

			var cmd tea.Cmd
			m.Search.SearchTextInput, cmd = m.Search.SearchTextInput.Update(msg)
			return m, cmd
		}

		switch keyPress {
		case "/":
			// Begin search
			m.Search.Begin()
			m.Search.SearchTextInput.Focus()
			return m, nil
		case "ctrl+n":
			// Go to next occurrence
			m.Search.Execute(&m)
		case "ctrl+e":
			// Run code blocks. Execution happens off the render path
			// (codeExecutedMsg/executeCodeCmd below): some languages
			// (Julia, first run) take long enough to compile that running
			// them inline here would freeze the whole presentation.
			blocks, err := code.Parse(m.Slides[m.Page])
			if err != nil {
				// We couldn't parse the code block on the screen
				m.VirtualText = "\n" + err.Error()
				return m, nil
			}
			maxCols, maxRows := m.mediaSize()
			for i := range blocks {
				if blocks[i].Language == code.Julia {
					blocks[i].Env = juliaEnv(maxCols, maxRows)
				}
			}
			for _, d := range m.execDirs {
				os.RemoveAll(d)
			}
			m.execDirs = nil
			m.execImages = nil
			m.VirtualText = ""
			m.executing = true
			return m, executeCodeCmd(m.Page, blocks)
		case "y":
			blocks, err := code.Parse(m.Slides[m.Page])
			if err != nil {
				return m, nil
			}
			for _, b := range blocks {
				_ = clipboard.WriteAll(b.Code)
			}
			return m, nil
		case "ctrl+c", "q":
			return m, tea.Quit
		default:
			oldBuffer := m.buffer
			newState := navigation.Navigate(navigation.State{
				Buffer:      m.buffer,
				Page:        m.Page,
				TotalSlides: len(m.Slides),
			}, keyPress)
			m.buffer = newState.Buffer

			lastStep := len(reveal.Steps(m.Slides[m.Page])) - 1
			switch {
			case nextKeys[keyPress] && !bufferIsNumeric(oldBuffer) && m.step < lastStep:
				// Reveal the next step of the current slide instead of
				// paging forward.
				m.step++
			case prevKeys[keyPress] && !bufferIsNumeric(oldBuffer) && m.step > 0:
				// Retract the current slide's own reveal before leaving it.
				m.step--
			case newState.Page != m.Page:
				m.SetPage(newState.Page)
				if prevKeys[keyPress] && !bufferIsNumeric(oldBuffer) {
					// Arriving on a previous slide via a single "previous"
					// keystroke shows it fully revealed, matching how it
					// looked when it was last on screen; any jump (3j, 5G,
					// gg) always lands at step zero.
					m.step = len(reveal.Steps(m.Slides[m.Page])) - 1
				} else {
					m.step = 0
				}
			}

			m.refreshMediaBudget()
			if !m.graphicsOK {
				return m, nil
			}
			maxCols, maxRows := m.mediaSize()
			return m, loadImagesCmd(m.images, m.imageBaseDir(), m.Slides[m.Page], maxCols, maxRows, m.darkBG)
		}

	case uv.KittyGraphicsEvent:
		if msg.Options.ID == image.ProbeID && string(msg.Payload) == "OK" {
			m.graphicsOK = true
			m.refreshMediaBudget()
			maxCols, maxRows := m.mediaSize()
			return m, loadImagesCmd(m.images, m.imageBaseDir(), m.Slides[m.Page], maxCols, maxRows, m.darkBG)
		}
		return m, nil

	case imagesLoadedMsg:
		return m, tea.Raw(strings.Join(msg.seqs, ""))

	case codeExecutedMsg:
		if msg.page != m.Page {
			// The user navigated away while this execution was still
			// running; its output belongs to a slide that's no longer
			// current, so just clean up instead of displaying it.
			for _, d := range msg.dirs {
				os.RemoveAll(d)
			}
			return m, nil
		}
		m.executing = false
		m.VirtualText = msg.text
		m.execDirs = msg.dirs
		m.execImages = msg.images
		if !m.graphicsOK || len(msg.images) == 0 {
			return m, nil
		}
		maxCols, maxRows := m.execMediaSize()
		return m, loadPathsCmd(m.images, msg.images, maxCols, maxRows)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case fileWatchMsg:
		newFileInfo, err := os.Stat(m.FileName)
		if err == nil && newFileInfo.ModTime() != fileInfo.ModTime() {
			fileInfo = newFileInfo
			_ = m.Load()
			if m.Page >= len(m.Slides) {
				m.Page = len(m.Slides) - 1
			}
			m.refreshMediaBudget()
		}
		return m, fileWatchCmd()
	}
	return m, nil
}

// renderSlide runs one slide's markdown (or reveal step) through the full
// render pipeline: image/LaTeX preprocessing, glamour, expansion of the
// resulting placeholders, and the left margin. includeVirtualText controls
// whether code-execution output is appended -- View's full-height
// measurement pass leaves it out, since it's ephemeral and not part of the
// slide's own content.
func (m Model) renderSlide(md string, includeVirtualText bool) string {
	contentWidth, _ := m.contentSize()
	r, _ := glamour.NewTermRenderer(m.Theme, glamour.WithWordWrap(contentWidth))
	slide := code.HideComments(md)

	// A slide carrying `<!-- center -->` on its own line has every text
	// line horizontally centered after rendering (images keep their own
	// centering). Strip the marker before glamour sees it.
	centered := strings.Contains(slide, centerMarker)
	if centered {
		slide = strings.ReplaceAll(slide, centerMarker+"\n", "")
		slide = strings.ReplaceAll(slide, centerMarker, "")
	}

	maxCols, maxRows := m.mediaSize()
	spin := m.spinner.View()

	var placements []image.Placement
	if m.graphicsOK && m.images != nil {
		baseDir := m.imageBaseDir()
		slide, placements = image.Preprocess(slide, spin+" cargando imagen…", func(ref image.Ref) (image.Entry, bool) {
			return m.images.Get(filepath.Join(baseDir, ref.Src), maxCols, maxRows)
		})
	}

	var latexPlacements []image.Placement
	if m.images != nil {
		slide, latexPlacements = latex.Preprocess(slide, m.darkBG, m.graphicsOK, spin+" renderizando fórmula…", func(path string) (image.Entry, bool) {
			return m.images.Get(path, maxCols, maxRows)
		})
	}

	slide, err := r.Render(slide)
	slide = strings.ReplaceAll(slide, "\t", tabSpaces)
	if includeVirtualText {
		if m.VirtualText != "" {
			// Centered the same way images are (via their own PadLeft),
			// so text-only output -- UnicodePlots' terminal-drawn charts,
			// in particular -- lines up with the image-based plots
			// instead of hugging the left edge.
			slide += lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, m.VirtualText)
		}
		if m.executing {
			slide += "\n" + spin + " ejecutando…"
		}
		if m.images != nil && len(m.execImages) > 0 {
			execCols, execRows := m.execMediaSize()
			for _, path := range m.execImages {
				if entry, ok := m.images.Get(path, execCols, execRows); ok {
					idx := len(placements)
					slide += "\n" + image.Sentinel(idx)
					placements = append(placements, image.Placement{Entry: entry})
				} else {
					slide += "\n" + spin + " generando gráfico…"
				}
			}
		}
	}
	if err != nil {
		slide = fmt.Sprintf("Error: Could not render markdown! (%v)", err)
	}
	slide = styles.Slide.Render(slide)
	if len(placements) > 0 {
		slide = image.Expand(slide, placements)
	}
	if len(latexPlacements) > 0 {
		slide = latex.Expand(slide, latexPlacements)
	}
	if centered {
		slide = centerLines(slide, contentWidth)
	}
	if margin := m.leftMargin(); margin > 0 {
		pad := strings.Repeat(" ", margin)
		lines := strings.Split(slide, "\n")
		for i := range lines {
			lines[i] = pad + lines[i]
		}
		slide = strings.Join(lines, "\n")
	}
	return slide
}

// View renders the current slide in the presentation and the status bar which
// contains the author, date, and pagination information.
func (m Model) View() tea.View {
	steps := reveal.Steps(m.Slides[m.Page])
	step := m.step
	if step > len(steps)-1 {
		step = len(steps) - 1
	}
	if step < 0 {
		step = 0
	}

	slide := m.renderSlide(steps[step], true)
	if len(steps) > 1 {
		// Pad the current (possibly partial) step to the height of the
		// slide's final, fully revealed step, so JoinVertical's centering
		// doesn't shift already-visible lines as more steps are revealed.
		full := m.renderSlide(steps[len(steps)-1], false)
		if diff := lipgloss.Height(full) - lipgloss.Height(slide); diff > 0 {
			slide += strings.Repeat("\n", diff)
		}
	}

	var left string
	if m.Search.Active {
		// render search bar
		left = m.Search.SearchTextInput.View()
	} else {
		// render author and date
		left = styles.Author.Render(m.Author) + styles.Date.Render(m.Date)
	}

	right := styles.Page.Render(m.paging())
	status := styles.Status.Render(styles.JoinHorizontal(left, right, m.viewport.Width()))
	view := tea.NewView(styles.JoinVertical(slide, status, m.viewport.Height()))
	view.AltScreen = true
	return view
}

func (m *Model) paging() string {
	switch strings.Count(m.Paging, "%d") {
	case 2:
		return fmt.Sprintf(m.Paging, m.Page+1, len(m.Slides))
	case 1:
		return fmt.Sprintf(m.Paging, m.Page+1)
	default:
		return m.Paging
	}
}

func readFile(path string) (string, error) {
	s, err := os.Stat(path)
	if err != nil {
		return "", errors.New("could not read file")
	}
	if s.IsDir() {
		return "", errors.New("can not read directory")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(b)

	// Pre-process slides if the file is executable to avoid
	// unintentional code execution when presenting slides
	if file.IsExecutable(s) {
		// Remove shebang if file has one
		if strings.HasPrefix(content, "#!") {
			content = strings.Join(strings.SplitN(content, "\n", 2)[1:], "\n")
		}

		content = process.Pre(content)
	}

	return content, err
}

func readStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		return string(slidesTutorial), nil
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		_, err = b.WriteRune(r)
		if err != nil {
			return "", err
		}
	}

	return b.String(), nil
}

// CurrentPage returns the current page the presentation is on.
func (m *Model) CurrentPage() int {
	return m.Page
}

// SetPage sets which page the presentation should render.
func (m *Model) SetPage(page int) {
	if m.Page == page {
		return
	}

	for _, d := range m.execDirs {
		os.RemoveAll(d)
	}
	m.execDirs = nil
	m.execImages = nil
	m.executing = false
	m.VirtualText = ""
	m.Page = page
}

// Pages returns all the slides in the presentation.
func (m *Model) Pages() []string {
	return m.Slides
}
