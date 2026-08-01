package code

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Block represents a code block.
type Block struct {
	Code     string
	Language string
	// Env holds extra environment variables to set for this block's
	// execution, on top of the process's own environment. Used to tune
	// language-specific runtime behavior (e.g. terminal width for
	// UnicodePlots.jl) without changing Execute's signature.
	Env map[string]string
}

// Result represents the output for an executed code block.
type Result struct {
	Out           string
	ExitCode      int
	ExecutionTime time.Duration
	// WorkDir is the temporary directory the code ran in. The caller owns
	// its cleanup (os.RemoveAll) once Result is no longer needed.
	WorkDir string
	// Images holds the absolute paths of any image files (plots, etc.)
	// the code left behind in WorkDir, in name order.
	Images []string
}

// ?: means non-capture group
var re = regexp.MustCompile("(?s)(?:```|~~~)(\\w+)\n(.*?)\n(?:```|~~~)\\s?")

var (
	// ErrParse is the returned error when we cannot parse the code block (i.e.
	// there is no code block on the current slide) or the code block is
	// incorrectly written.
	ErrParse = errors.New("Error: could not parse code block")
)

// Parse takes a block of markdown and returns an array of Block's with code
// and associated languages
func Parse(markdown string) ([]Block, error) {
	matches := re.FindAllStringSubmatch(markdown, -1)

	var rv []Block
	for _, match := range matches {
		// There was either no language specified or no code block
		// Either way, we cannot execute the expression
		if len(match) < 3 {
			continue
		}
		rv = append(rv, Block{
			Language: match[1],
			Code:     RemoveComments(match[2]),
		})

	}

	if len(rv) == 0 {
		return nil, ErrParse
	}

	return rv, nil
}

const (
	// ExitCodeInternalError represents the exit code in which the code
	// executing the code didn't work.
	ExitCodeInternalError = -1
)

// imageExtensions are the file extensions Execute looks for, in WorkDir,
// once a code block finishes running -- anything a plotting library might
// have saved there. SVG is deliberately excluded: Go's image package can't
// decode it without a new dependency.
var imageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
}

// Execute takes a code.Block and returns the output of the executed code.
// It runs the code in its own temporary directory (Result.WorkDir), so a
// relative path the code saves a file to (e.g. a plot) lands somewhere
// predictable; the caller is responsible for removing that directory once
// done with the result.
func Execute(code Block) Result {
	// Check supported language
	language, ok := Languages[code.Language]
	if !ok {
		return Result{
			Out:      "Error: unsupported language",
			ExitCode: ExitCodeInternalError,
		}
	}

	workDir, err := os.MkdirTemp(os.TempDir(), "slides-*")
	if err != nil {
		return Result{
			Out:      "Error: could not create directory",
			ExitCode: ExitCodeInternalError,
		}
	}

	sourcePath := filepath.Join(workDir, "code."+language.Extension)
	if err := os.WriteFile(sourcePath, []byte(code.Code), 0o600); err != nil {
		os.RemoveAll(workDir)
		return Result{
			Out:      "Error: could not write to file",
			ExitCode: ExitCodeInternalError,
		}
	}

	var (
		output   strings.Builder
		exitCode int
	)

	// replacer for commands
	repl := strings.NewReplacer(
		"<file>", sourcePath,
		// <name>: file name without extension and without path
		"<name>", strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath)),
		"<path>", workDir,
	)

	var env []string
	if len(code.Env) > 0 {
		env = os.Environ()
		for k, v := range code.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// For accuracy of program execution speed, we can't put anything after
	// recording the start time or before recording the end time.
	start := time.Now()

	for _, c := range language.Commands {
		var command []string
		// replace <file>, <name> and <path> in commands
		for _, v := range c {
			command = append(command, repl.Replace(v))
		}
		// execute and write output
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Dir = workDir
		if env != nil {
			cmd.Env = env
		}
		out, err := cmd.Output()
		if err != nil {
			output.Write([]byte(err.Error()))
		} else {
			output.Write(out)
		}

		// update status code
		if err != nil {
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			} else {
				exitCode = 1 // non-zero
			}
		}
	}

	end := time.Now()

	return Result{
		Out:           output.String(),
		ExitCode:      exitCode,
		ExecutionTime: end.Sub(start),
		WorkDir:       workDir,
		Images:        findImages(workDir, sourcePath),
	}
}

// findImages lists the image files a code block's execution left behind in
// dir, sorted by name, skipping the source file itself.
func findImages(dir, sourcePath string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if path == sourcePath {
			continue
		}
		if !imageExtensions[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	images := make([]string, len(names))
	for i, name := range names {
		images[i] = filepath.Join(dir, name)
	}
	return images
}
