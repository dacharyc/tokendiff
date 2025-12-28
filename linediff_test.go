package tokendiff

import (
	"strings"
	"testing"
)

func TestFindPositionalPairings(t *testing.T) {
	tests := []struct {
		name     string
		deletes  []string
		inserts  []string
		expected int // expected number of pairings
	}{
		{
			name:     "empty inputs",
			deletes:  []string{},
			inserts:  []string{},
			expected: 0,
		},
		{
			name:     "equal length",
			deletes:  []string{"a", "b", "c"},
			inserts:  []string{"x", "y", "z"},
			expected: 3,
		},
		{
			name:     "more deletes than inserts",
			deletes:  []string{"a", "b", "c"},
			inserts:  []string{"x"},
			expected: 1,
		},
		{
			name:     "more inserts than deletes",
			deletes:  []string{"a"},
			inserts:  []string{"x", "y", "z"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairings := FindPositionalPairings(tt.deletes, tt.inserts)
			if len(pairings) != tt.expected {
				t.Errorf("FindPositionalPairings() returned %d pairings, want %d",
					len(pairings), tt.expected)
			}

			// Verify pairings are positional (index i pairs with index i)
			for i, p := range pairings {
				if p.DeleteIndex != i || p.InsertIndex != i {
					t.Errorf("Pairing %d: got DeleteIndex=%d, InsertIndex=%d, want both=%d",
						i, p.DeleteIndex, p.InsertIndex, i)
				}
				if p.Similarity != 1.0 {
					t.Errorf("Pairing %d: got Similarity=%v, want 1.0", i, p.Similarity)
				}
			}
		})
	}
}

func TestFindSimilarityPairings(t *testing.T) {
	opts := Options{Delimiters: "()"}

	tests := []struct {
		name      string
		deletes   []string
		inserts   []string
		threshold float64
		wantPairs int
	}{
		{
			name:      "identical lines pair up",
			deletes:   []string{"hello world", "foo bar"},
			inserts:   []string{"foo bar", "hello world"},
			threshold: 0.5,
			wantPairs: 2,
		},
		{
			name:      "similar lines pair up",
			deletes:   []string{"func getData(x int)"},
			inserts:   []string{"func getData(y int)"},
			threshold: 0.5,
			wantPairs: 1,
		},
		{
			name:      "dissimilar lines don't pair",
			deletes:   []string{"hello world"},
			inserts:   []string{"completely different"},
			threshold: 0.9,
			wantPairs: 0,
		},
		{
			name:      "empty inputs",
			deletes:   []string{},
			inserts:   []string{},
			threshold: 0.5,
			wantPairs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairings := FindSimilarityPairings(tt.deletes, tt.inserts, opts, tt.threshold)
			if len(pairings) != tt.wantPairs {
				t.Errorf("FindSimilarityPairings() returned %d pairings, want %d",
					len(pairings), tt.wantPairs)
			}
		})
	}
}

func TestDiffWholeFiles(t *testing.T) {
	text1 := "hello world\nfoo bar"
	text2 := "hello universe\nfoo bar"
	opts := DefaultOptions()
	fmtOpts := DefaultFormatOptions()

	result := DiffWholeFiles(text1, text2, opts, fmtOpts)

	if !result.HasChanges {
		t.Error("DiffWholeFiles() expected HasChanges=true")
	}

	if result.Statistics.DeletedWords == 0 {
		t.Error("DiffWholeFiles() expected some deleted words")
	}

	if result.Statistics.InsertedWords == 0 {
		t.Error("DiffWholeFiles() expected some inserted words")
	}

	// Check formatted output contains the change markers
	if !strings.Contains(result.Formatted, "world") || !strings.Contains(result.Formatted, "universe") {
		t.Errorf("DiffWholeFiles() formatted output missing expected content: %s", result.Formatted)
	}
}

func TestDiffLineByLine(t *testing.T) {
	tests := []struct {
		name       string
		text1      string
		text2      string
		algorithm  string
		wantChange bool
	}{
		{
			name:       "identical files",
			text1:      "line1\nline2\nline3",
			text2:      "line1\nline2\nline3",
			algorithm:  "normal",
			wantChange: false,
		},
		{
			name:       "one line changed - normal algorithm",
			text1:      "line1\nold line\nline3",
			text2:      "line1\nnew line\nline3",
			algorithm:  "normal",
			wantChange: true,
		},
		{
			name:       "one line changed - best algorithm",
			text1:      "line1\nold line\nline3",
			text2:      "line1\nnew line\nline3",
			algorithm:  "best",
			wantChange: true,
		},
		{
			name:       "line added",
			text1:      "line1\nline2",
			text2:      "line1\nline2\nline3",
			algorithm:  "normal",
			wantChange: true,
		},
		{
			name:       "line deleted",
			text1:      "line1\nline2\nline3",
			text2:      "line1\nline3",
			algorithm:  "normal",
			wantChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			fmtOpts := DefaultFormatOptions()

			result := DiffLineByLine(tt.text1, tt.text2, opts, fmtOpts, tt.algorithm, 0.5)

			if result.HasChanges != tt.wantChange {
				t.Errorf("DiffLineByLine() HasChanges=%v, want %v", result.HasChanges, tt.wantChange)
			}
		})
	}
}

func TestFilterWithContext(t *testing.T) {
	// Create a set of line results with some changes
	lines := []LineDiffResult{
		{OldLineNum: 1, NewLineNum: 1, HasChanges: false, Output: "line1"},
		{OldLineNum: 2, NewLineNum: 2, HasChanges: false, Output: "line2"},
		{OldLineNum: 3, NewLineNum: 3, HasChanges: true, Output: "changed"},
		{OldLineNum: 4, NewLineNum: 4, HasChanges: false, Output: "line4"},
		{OldLineNum: 5, NewLineNum: 5, HasChanges: false, Output: "line5"},
		{OldLineNum: 6, NewLineNum: 6, HasChanges: false, Output: "line6"},
	}

	tests := []struct {
		name         string
		contextLines int
		wantCount    int
	}{
		{
			name:         "zero context returns all",
			contextLines: 0,
			wantCount:    6,
		},
		{
			name:         "context of 1",
			contextLines: 1,
			wantCount:    3, // lines 2, 3, 4
		},
		{
			name:         "context of 2",
			contextLines: 2,
			wantCount:    5, // lines 1, 2, 3, 4, 5
		},
		{
			name:         "large context includes all",
			contextLines: 10,
			wantCount:    6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterWithContext(lines, tt.contextLines)
			if len(result) != tt.wantCount {
				t.Errorf("FilterWithContext() returned %d lines, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestDiffLineByLineWithPairing(t *testing.T) {
	// Test that line pairing works correctly for changed blocks
	text1 := "func old1()\nfunc old2()"
	text2 := "func new1()\nfunc new2()"

	opts := Options{Delimiters: "()"}
	fmtOpts := DefaultFormatOptions()

	result := DiffLineByLine(text1, text2, opts, fmtOpts, "normal", 0.0)

	if !result.HasChanges {
		t.Error("Expected changes in diff output")
	}

	// Should have 2 lines of output
	if len(result.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result.Lines))
	}

	// Both lines should be marked as changed
	for i, line := range result.Lines {
		if !line.HasChanges {
			t.Errorf("Line %d should be marked as changed", i)
		}
	}
}
