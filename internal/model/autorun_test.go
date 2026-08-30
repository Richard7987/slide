package model

import (
	"strings"
	"testing"
)

func TestCodeFenceRe(t *testing.T) {
	in := "title\n\nintro line\n\n```julia\nusing UnicodePlots\nprintln(1)\n```\n\ntail\n"
	got := codeFenceRe.ReplaceAllString(in, "")
	if strings.Contains(got, "using UnicodePlots") || strings.Contains(got, "```") {
		t.Fatalf("fence not stripped: %q", got)
	}
	if !strings.Contains(got, "intro line") || !strings.Contains(got, "tail") {
		t.Fatalf("prose lost: %q", got)
	}

	// tilde fences and a missing language tag both count
	in2 := "~~~\ncode here\n~~~\n"
	if codeFenceRe.ReplaceAllString(in2, "X") != "X" {
		t.Errorf("tilde/no-lang fence not matched: %q", codeFenceRe.ReplaceAllString(in2, "X"))
	}
}
