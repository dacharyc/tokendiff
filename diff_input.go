package tokendiff

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// isGitExtendedHeader returns true if the line is a git extended header.
func isGitExtendedHeader(line string) bool {
	prefixes := []string{"diff ", "index ", "new file", "deleted file", "similarity", "rename", "Binary"}
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// isDiffHeader returns true if the line is a file header.
func isDiffHeader(line string) bool {
	return strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++")
}

// isHunkHeader returns true if the line is a hunk header.
func isHunkHeader(line string) bool {
	return strings.HasPrefix(line, "@@")
}

// diffProcessor handles processing unified diff input.
type diffProcessor struct {
	output   io.Writer
	opts     Options
	fmtOpts  FormatOptions
	oldLines []string
	newLines []string
	inHunk   bool
}

// flushHunk outputs accumulated changes as word-level diff.
func (p *diffProcessor) flushHunk() {
	if len(p.oldLines) == 0 && len(p.newLines) == 0 {
		return
	}

	oldText := strings.Join(p.oldLines, "\n")
	newText := strings.Join(p.newLines, "\n")

	result := DiffWholeFiles(oldText, newText, p.opts, p.fmtOpts)
	fmt.Fprintln(p.output, result.Formatted)

	p.oldLines = nil
	p.newLines = nil
}

// processHunkLine handles a line inside a hunk.
func (p *diffProcessor) processHunkLine(line string) {
	switch {
	case strings.HasPrefix(line, "-"):
		p.oldLines = append(p.oldLines, line[1:])
	case strings.HasPrefix(line, "+"):
		p.newLines = append(p.newLines, line[1:])
	case strings.HasPrefix(line, " "):
		p.flushHunk()
		fmt.Fprintln(p.output, line[1:])
	case line == "":
		p.flushHunk()
		fmt.Fprintln(p.output, line)
	default:
		p.flushHunk()
		fmt.Fprintln(p.output, line)
	}
}

// processLine handles a single line of diff input.
func (p *diffProcessor) processLine(line string) {
	switch {
	case isDiffHeader(line), isGitExtendedHeader(line):
		p.flushHunk()
		p.inHunk = false
		fmt.Fprintln(p.output, line)
	case isHunkHeader(line):
		p.flushHunk()
		p.inHunk = true
		fmt.Fprintln(p.output, line)
	case p.inHunk:
		p.processHunkLine(line)
	default:
		fmt.Fprintln(p.output, line)
	}
}

// ProcessUnifiedDiff reads a unified diff from input and applies word-level
// diffing to each hunk. The result is written to output with diff headers
// preserved and hunk content replaced with word-level diff output.
func ProcessUnifiedDiff(input io.Reader, output io.Writer, opts Options, fmtOpts FormatOptions) error {
	scanner := bufio.NewScanner(input)
	p := &diffProcessor{output: output, opts: opts, fmtOpts: fmtOpts}

	for scanner.Scan() {
		p.processLine(scanner.Text())
	}

	p.flushHunk()
	return scanner.Err()
}
