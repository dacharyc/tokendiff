package tokendiff

import (
	"fmt"
	"strings"
)

// DiffHunk represents a single hunk from a unified diff.
type DiffHunk struct {
	// OldStart is the starting line number in the old file.
	OldStart int
	// OldCount is the number of lines from the old file.
	OldCount int
	// NewStart is the starting line number in the new file.
	NewStart int
	// NewCount is the number of lines in the new file.
	NewCount int
	// OldLines contains the removed lines (without the leading "-").
	OldLines []string
	// NewLines contains the added lines (without the leading "+").
	NewLines []string
	// ContextBefore contains context lines before the change.
	ContextBefore []string
	// ContextAfter contains context lines after the change.
	ContextAfter []string
}

// UnifiedDiff represents a parsed unified diff.
type UnifiedDiff struct {
	// OldFile is the name of the old file (from "---" line).
	OldFile string
	// NewFile is the name of the new file (from "+++" line).
	NewFile string
	// Hunks contains all the diff hunks.
	Hunks []DiffHunk
}

// ParseUnifiedDiff parses a unified diff string into structured data.
// It handles standard unified diff format as produced by diff -u or git diff.
func ParseUnifiedDiff(input string) ([]UnifiedDiff, error) {
	var results []UnifiedDiff
	var current *UnifiedDiff
	var currentHunk *DiffHunk
	inHunk := false

	lines := strings.Split(input, "\n")

	flushHunk := func() {
		if currentHunk != nil && current != nil {
			current.Hunks = append(current.Hunks, *currentHunk)
			currentHunk = nil
		}
		inHunk = false
	}

	for _, line := range lines {
		// New file diff
		if strings.HasPrefix(line, "--- ") {
			flushHunk()
			if current != nil {
				results = append(results, *current)
			}
			current = &UnifiedDiff{
				OldFile: strings.TrimPrefix(line, "--- "),
			}
			continue
		}

		if strings.HasPrefix(line, "+++ ") && current != nil {
			current.NewFile = strings.TrimPrefix(line, "+++ ")
			continue
		}

		// Hunk header
		if strings.HasPrefix(line, "@@") && current != nil {
			flushHunk()
			currentHunk = &DiffHunk{}
			inHunk = true

			// Parse @@ -start,count +start,count @@
			var oldStart, oldCount, newStart, newCount int
			// Try parsing with counts
			n, _ := fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@",
				&oldStart, &oldCount, &newStart, &newCount)
			if n < 4 {
				// Try without counts (single line changes)
				fmt.Sscanf(line, "@@ -%d +%d @@", &oldStart, &newStart)
				oldCount = 1
				newCount = 1
			}
			currentHunk.OldStart = oldStart
			currentHunk.OldCount = oldCount
			currentHunk.NewStart = newStart
			currentHunk.NewCount = newCount
			continue
		}

		if !inHunk || currentHunk == nil {
			continue
		}

		// Process hunk content
		if strings.HasPrefix(line, "-") {
			currentHunk.OldLines = append(currentHunk.OldLines, line[1:])
		} else if strings.HasPrefix(line, "+") {
			currentHunk.NewLines = append(currentHunk.NewLines, line[1:])
		} else if strings.HasPrefix(line, " ") {
			// Context line
			if len(currentHunk.OldLines) == 0 && len(currentHunk.NewLines) == 0 {
				currentHunk.ContextBefore = append(currentHunk.ContextBefore, line[1:])
			} else {
				currentHunk.ContextAfter = append(currentHunk.ContextAfter, line[1:])
			}
		}
	}

	// Flush remaining
	flushHunk()
	if current != nil {
		results = append(results, *current)
	}

	return results, nil
}

// ApplyWordDiff applies word-level diffing to a unified diff hunk.
// It returns the word-level diff result for the changed lines.
func ApplyWordDiff(hunk DiffHunk, opts Options) []Diff {
	oldText := strings.Join(hunk.OldLines, "\n")
	newText := strings.Join(hunk.NewLines, "\n")
	return DiffStrings(oldText, newText, opts)
}
