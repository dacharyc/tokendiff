package tokendiff

import (
	"reflect"
	"testing"
)

func TestParseUnifiedDiff(t *testing.T) {
	input := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 context
-old line
+new line
 more context
`

	diffs, err := ParseUnifiedDiff(input)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff() error = %v", err)
	}

	if len(diffs) != 1 {
		t.Fatalf("ParseUnifiedDiff() got %d diffs, want 1", len(diffs))
	}

	diff := diffs[0]
	if diff.OldFile != "a/file.txt" {
		t.Errorf("OldFile = %q, want %q", diff.OldFile, "a/file.txt")
	}
	if diff.NewFile != "b/file.txt" {
		t.Errorf("NewFile = %q, want %q", diff.NewFile, "b/file.txt")
	}
	if len(diff.Hunks) != 1 {
		t.Fatalf("Hunks count = %d, want 1", len(diff.Hunks))
	}

	hunk := diff.Hunks[0]
	if hunk.OldStart != 1 || hunk.OldCount != 3 {
		t.Errorf("Old range = %d,%d, want 1,3", hunk.OldStart, hunk.OldCount)
	}
	if hunk.NewStart != 1 || hunk.NewCount != 3 {
		t.Errorf("New range = %d,%d, want 1,3", hunk.NewStart, hunk.NewCount)
	}
	if len(hunk.OldLines) != 1 || hunk.OldLines[0] != "old line" {
		t.Errorf("OldLines = %v, want [old line]", hunk.OldLines)
	}
	if len(hunk.NewLines) != 1 || hunk.NewLines[0] != "new line" {
		t.Errorf("NewLines = %v, want [new line]", hunk.NewLines)
	}
}

func TestApplyWordDiff(t *testing.T) {
	tests := []struct {
		name     string
		hunk     DiffHunk
		expected []Diff
	}{
		{
			name: "single word change",
			hunk: DiffHunk{
				OldLines: []string{"hello world"},
				NewLines: []string{"hello universe"},
			},
			expected: []Diff{
				{Type: Equal, Token: "hello"},
				{Type: Delete, Token: "world"},
				{Type: Insert, Token: "universe"},
			},
		},
		{
			name: "multiline hunk",
			hunk: DiffHunk{
				OldLines: []string{"line one", "line two"},
				NewLines: []string{"line one", "line three"},
			},
			expected: []Diff{
				{Type: Equal, Token: "line"},
				{Type: Equal, Token: "one"},
				{Type: Equal, Token: "line"},
				{Type: Delete, Token: "two"},
				{Type: Insert, Token: "three"},
			},
		},
		{
			name: "empty hunk",
			hunk: DiffHunk{
				OldLines: []string{},
				NewLines: []string{},
			},
			expected: nil,
		},
		{
			name: "addition only",
			hunk: DiffHunk{
				OldLines: []string{},
				NewLines: []string{"new text"},
			},
			expected: []Diff{
				{Type: Insert, Token: "new"},
				{Type: Insert, Token: "text"},
			},
		},
		{
			name: "deletion only",
			hunk: DiffHunk{
				OldLines: []string{"old text"},
				NewLines: []string{},
			},
			expected: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Delete, Token: "text"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyWordDiff(tt.hunk, DefaultOptions())
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ApplyWordDiff() = %v, want %v", result, tt.expected)
			}
		})
	}
}
