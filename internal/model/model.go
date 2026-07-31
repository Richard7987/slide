package model

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/maaslalani/slides/internal/file"
	"github.com/maaslalani/slides/internal/navigation"
	"github.com/maaslalani/slides/internal/process"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/charmbracelet/glamour"
	"github.com/maaslalani/slides/internal/code"
	"github.com/maaslalani/slides/internal/image"
	"github.com/maaslalani/slides/internal/meta"
	"github.com/maaslalani/slides/styles"
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
	cmds := []tea.Cmd{tea.Raw(image.ProbeSeq())}
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

// loadImagesCmd decodes and encodes any not-yet-cached images referenced
// by slide, off the render path. It does file I/O, so it must only run
// inside a tea.Cmd.
func loadImagesCmd(images *image.Cache, baseDir, slide string, maxCols int) tea.Cmd {
	refs := image.Parse(slide)
	if images == nil || len(refs) == 0 {
		return nil
	}
	return func() tea.Msg {
		var seqs []string
		for _, ref := range refs {
			path := filepath.Join(baseDir, ref.Src)
			if _, ok := images.Get(path, maxCols); ok {
				continue
			}
			_, seq, err := images.Load(path, maxCols)
			if err != nil || seq == "" {
				continue
			}
			seqs = append(seqs, seq)
		}
		if len(seqs) == 0 {
			return nil
		}
		return imagesLoadedMsg{seqs: seqs}
	}
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
		m.Theme = styles.SelectTheme(metaData.Theme)
	}
	if m.images == nil {
		m.images = image.NewCache()
	}

	return nil
}

// Update updates the presentation model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height)
		if !m.graphicsOK || len(m.Slides) == 0 {
			return m, nil
		}
		return m, loadImagesCmd(m.images, m.imageBaseDir(), m.Slides[m.Page], m.viewport.Width())

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
			// Run code blocks
			blocks, err := code.Parse(m.Slides[m.Page])
			if err != nil {
				// We couldn't parse the code block on the screen
				m.VirtualText = "\n" + err.Error()
				return m, nil
			}
			var outs []string
			for _, block := range blocks {
				res := code.Execute(block)
				outs = append(outs, res.Out)
			}
			m.VirtualText = strings.Join(outs, "\n")
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
			newState := navigation.Navigate(navigation.State{
				Buffer:      m.buffer,
				Page:        m.Page,
				TotalSlides: len(m.Slides),
			}, keyPress)
			m.buffer = newState.Buffer
			m.SetPage(newState.Page)
			if !m.graphicsOK {
				return m, nil
			}
			return m, loadImagesCmd(m.images, m.imageBaseDir(), m.Slides[m.Page], m.viewport.Width())
		}

	case uv.KittyGraphicsEvent:
		if msg.Options.ID == image.ProbeID && string(msg.Payload) == "OK" {
			m.graphicsOK = true
			return m, loadImagesCmd(m.images, m.imageBaseDir(), m.Slides[m.Page], m.viewport.Width())
		}
		return m, nil

	case imagesLoadedMsg:
		return m, tea.Raw(strings.Join(msg.seqs, ""))

	case fileWatchMsg:
		newFileInfo, err := os.Stat(m.FileName)
		if err == nil && newFileInfo.ModTime() != fileInfo.ModTime() {
			fileInfo = newFileInfo
			_ = m.Load()
			if m.Page >= len(m.Slides) {
				m.Page = len(m.Slides) - 1
			}
		}
		return m, fileWatchCmd()
	}
	return m, nil
}

// View renders the current slide in the presentation and the status bar which
// contains the author, date, and pagination information.
func (m Model) View() tea.View {
	r, _ := glamour.NewTermRenderer(m.Theme, glamour.WithWordWrap(m.viewport.Width()))
	slide := m.Slides[m.Page]
	slide = code.HideComments(slide)

	var placements []image.Placement
	if m.graphicsOK && m.images != nil {
		baseDir, maxCols := m.imageBaseDir(), m.viewport.Width()
		slide, placements = image.Preprocess(slide, func(ref image.Ref) (image.Entry, bool) {
			return m.images.Get(filepath.Join(baseDir, ref.Src), maxCols)
		})
	}

	slide, err := r.Render(slide)
	slide = strings.ReplaceAll(slide, "\t", tabSpaces)
	slide += m.VirtualText
	if err != nil {
		slide = fmt.Sprintf("Error: Could not render markdown! (%v)", err)
	}
	slide = styles.Slide.Render(slide)
	if len(placements) > 0 {
		slide = image.Expand(slide, placements)
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

	m.VirtualText = ""
	m.Page = page
}

// Pages returns all the slides in the presentation.
func (m *Model) Pages() []string {
	return m.Slides
}
