package tokendiff

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ProcessUnifiedDiff reads a unified diff from input and applies word-level
// diffing to each hunk. The result is written to output with diff headers
// preserved and hunk content replaced with word-level diff output.
//
// This is useful for enhancing the output of tools like "git diff" to show
// exactly which words changed within each line, rather than showing entire
// lines as changed.
func ProcessUnifiedDiff(input io.Reader, output io.Writer, opts Options, fmtOpts FormatOptions) error {
	scanner := bufio.NewScanner(input)
	var oldLines, newLines []string
	inHunk := false

	flushHunk := func() {
		if len(oldLines) == 0 && len(newLines) == 0 {
			return
		}

		oldText := strings.Join(oldLines, "\n")
		newText := strings.Join(newLines, "\n")

		result := DiffWholeFiles(oldText, newText, opts, fmtOpts)
		fmt.Fprintln(output, result.Formatted)

		oldLines = nil
		newLines = nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		// File headers
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			flushHunk()
			inHunk = false
			fmt.Fprintln(output, line)
			continue
		}

		// Hunk header
		if strings.HasPrefix(line, "@@") {
			flushHunk()
			inHunk = true
			fmt.Fprintln(output, line)
			continue
		}

		// Git extended headers
		if strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") ||
			strings.HasPrefix(line, "similarity") ||
			strings.HasPrefix(line, "rename") ||
			strings.HasPrefix(line, "Binary") {
			flushHunk()
			inHunk = false
			fmt.Fprintln(output, line)
			continue
		}

		if !inHunk {
			fmt.Fprintln(output, line)
			continue
		}

		// Inside a hunk: collect old/new lines
		if strings.HasPrefix(line, "-") {
			oldLines = append(oldLines, line[1:])
		} else if strings.HasPrefix(line, "+") {
			newLines = append(newLines, line[1:])
		} else if strings.HasPrefix(line, " ") || line == "" {
			// Context line or empty line - flush accumulated changes first
			flushHunk()
			if strings.HasPrefix(line, " ") {
				fmt.Fprintln(output, line[1:])
			} else {
				fmt.Fprintln(output, line)
			}
		} else {
			// Unknown line format - pass through
			flushHunk()
			fmt.Fprintln(output, line)
		}
	}

	flushHunk()

	return scanner.Err()
}
