package styles_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/maaslalani/slides/styles"
	"github.com/stretchr/testify/assert"
)

func TestJoinVertical_ZeroHeightDoesNotPanic(t *testing.T) {
	// Regression: the very first render can happen before the initial
	// tea.WindowSizeMsg arrives, with height still 0 -- must not panic.
	assert.NotPanics(t, func() {
		styles.JoinVertical("slide content\nmore lines", "status", 0)
	})
}

func TestJoinVertical_BottomTallerThanHeightDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		styles.JoinVertical("top", "line1\nline2\nline3", 1)
	})
}

func TestJoinVertical_ClipsOverflowingTop(t *testing.T) {
	top := "1\n2\n3\n4\n5"
	got := styles.JoinVertical(top, "status", 3)
	// height=3, bottom is 1 line -> top's budget is 2 lines; the last of
	// those is swapped for a blank one so bottom merges into it cleanly
	// instead of appending onto real content ("2status").
	assert.Equal(t, "1\nstatus", got)
}

func TestJoinVertical_ClipsOverflowingTop_PaddedBottomStaysClean(t *testing.T) {
	// bottom shaped like the real status bar: Padding(1) gives it a
	// blank line above and below its actual content.
	top := "1\n2\n3\n4\n5"
	bottom := " \nSTATUS\n "
	got := styles.JoinVertical(top, bottom, 5)
	assert.Equal(t, "1\n \nSTATUS\n ", got)
	for _, line := range strings.Split(got, "\n") {
		assert.NotContains(t, line, "2STATUS", "clipped content must not merge into status text")
	}
}

func TestSelectTheme(t *testing.T) {
	tests := []struct {
		name    string
		theme   string
		want    ansi.StyleConfig
		wantErr bool
	}{
		{name: "Select dark theme", theme: "dark", want: glamourstyles.DarkStyleConfig, wantErr: false},
		{name: "Select light theme", theme: "light", want: glamourstyles.LightStyleConfig, wantErr: false},
		{name: "Select ascii theme", theme: "ascii", want: glamourstyles.ASCIIStyleConfig, wantErr: false},
		{name: "Select notty theme", theme: "notty", want: glamourstyles.NoTTYStyleConfig, wantErr: false},
		{name: "Select theme with error", theme: "notty", want: glamourstyles.DarkStyleConfig, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute the theme selection and ensure
			// it returns a non-nil theme
			selectedTheme := styles.SelectTheme(tt.theme)
			assert.NotNil(t, selectedTheme)

			// Initialize renderers to compare output
			gotRenderer, _ := glamour.NewTermRenderer(selectedTheme)
			wantRenderer, _ := glamour.NewTermRenderer(glamour.WithStyles(tt.want))

			// Render a the same string with two different
			// renderers
			gotOutput, _ := gotRenderer.Render(tt.name)
			wantOutput, _ := wantRenderer.Render(tt.name)

			// Inject exception to ensure a style that doesn't match
			// it's associated string
			if tt.wantErr {
				assert.NotEqual(t, wantOutput, gotOutput)
				return
			}

			// Ensure they both match
			assert.Equal(t, wantOutput, gotOutput)
		})
	}
}

func TestSelectTheme_file(t *testing.T) {
	tests := []struct {
		name       string
		theme      string
		fileExists bool
	}{
		{name: "Select custom theme json", theme: "./theme.json", fileExists: true},
		{name: "Use an invalid filepath", theme: "./someinvalidfile.toml", fileExists: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Successfully return a theme if a file exists
			assert.NotNil(t, styles.SelectTheme(tt.theme))

			// Successfully return a theme if a file doesn't exist
			if !tt.fileExists {
				assert.NotNil(t, styles.SelectTheme(tt.theme))
			}
		})
	}
}
