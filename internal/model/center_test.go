package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi/kitty"
)

func TestCenterLines(t *testing.T) {
	const width = 20

	got := centerLines("hello\n\n  spaced  ", width)
	lines := strings.Split(got, "\n")
	if lines[0] != strings.Repeat(" ", 7)+"hello" { // (20-5)/2 = 7
		t.Errorf("line 0 not centered: %q", lines[0])
	}
	if lines[1] != "" {
		t.Errorf("blank line changed: %q", lines[1])
	}
	if lines[2] != strings.Repeat(" ", 7)+"spaced" { // trimmed len 6, (20-6)/2 = 7
		t.Errorf("line 2 not centered/trimmed: %q", lines[2])
	}
}

func TestCenterLinesLeavesPlaceholdersAndAnsi(t *testing.T) {
	ph := string(kitty.Placeholder) + "xxxx"
	got := centerLines(ph, 40)
	if got != ph {
		t.Errorf("placeholder line was modified: %q", got)
	}

	ansi := "\x1b[1mbold\x1b[0m" // visible width 4
	got = centerLines(ansi, 10)
	if got != strings.Repeat(" ", 3)+ansi { // (10-4)/2 = 3
		t.Errorf("ansi width miscounted: %q", got)
	}

	long := "this line is wider than the width"
	if centerLines(long, 10) != long {
		t.Errorf("over-wide line should be untouched")
	}
}
