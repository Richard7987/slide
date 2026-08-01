package code_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maaslalani/slides/internal/code"
)

func TestExecute(t *testing.T) {
	tt := []struct {
		block    code.Block
		expected code.Result
	}{
		{
			block: code.Block{
				Code: `
package main

import "fmt"

func main() {
  fmt.Print("Hello, go!")
}
        `,
				Language: "go",
			},
			expected: code.Result{
				Out:      "Hello, go!",
				ExitCode: 0,
			},
		},
		{
			block: code.Block{
				Code:     `echo "Hello, bash!"`,
				Language: "bash",
			},
			expected: code.Result{
				Out:      "Hello, bash!\n",
				ExitCode: 0,
			},
		},
		{
			block: code.Block{
				Code:     `Invalid Code`,
				Language: "bash",
			},
			expected: code.Result{
				Out:      "exit status 127",
				ExitCode: 127,
			},
		},
		{
			block: code.Block{
				Code:     `Invalid Code`,
				Language: "invalid",
			},
			expected: code.Result{
				Out:      "Error: unsupported language",
				ExitCode: code.ExitCodeInternalError,
			},
		},
	}

	for _, tc := range tt {
		r := code.Execute(tc.block)
		if r.Out != tc.expected.Out {
			t.Fatalf("invalid output for lang %s, got %s, want %s | %+v",
				tc.block.Language, r.Out, tc.expected.Out, r)
		}

		if r.ExitCode != tc.expected.ExitCode {
			t.Fatalf("unexpected exit code, got %d, want %d", r.ExitCode, tc.expected.ExitCode)
		}
	}
}

func TestExecuteCapturesImages(t *testing.T) {
	r := code.Execute(code.Block{
		Code:     `touch plot.png b.txt z.jpg`,
		Language: "bash",
	})
	defer os.RemoveAll(r.WorkDir)

	if r.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d, out=%q", r.ExitCode, r.Out)
	}
	if r.WorkDir == "" {
		t.Fatal("expected a non-empty WorkDir")
	}

	want := []string{
		filepath.Join(r.WorkDir, "plot.png"),
		filepath.Join(r.WorkDir, "z.jpg"),
	}
	if len(r.Images) != len(want) {
		t.Fatalf("got %d images, want %d: %v", len(r.Images), len(want), r.Images)
	}
	for i := range want {
		if r.Images[i] != want[i] {
			t.Errorf("image %d: got %s, want %s", i, r.Images[i], want[i])
		}
	}
}

func TestExecuteEnv(t *testing.T) {
	r := code.Execute(code.Block{
		Code:     `echo "$FOO"`,
		Language: "bash",
		Env:      map[string]string{"FOO": "bar"},
	})
	defer os.RemoveAll(r.WorkDir)

	if r.Out != "bar\n" {
		t.Fatalf("got %q, want %q", r.Out, "bar\n")
	}
}
