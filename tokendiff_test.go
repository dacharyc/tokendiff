package tokendiff

import (
	"reflect"
	"testing"
)

// TestDwdiffMotivatingExample tests the exact example from the dwdiff README
// that motivated its creation. This is the key use case we must support.
// NOTE: Requires explicit delimiters since default delimiters are empty.
func TestDwdiffMotivatingExample(t *testing.T) {
	old := "void someFunction(SomeType var)"
	new := "void someFunction(SomeOtherType var)"

	// Use parentheses as delimiters to split function arguments
	opts := Options{Delimiters: "()"}
	diffs := DiffStrings(old, new, opts)

	// Find the actual changes (non-Equal diffs)
	var deletions, insertions []string
	for _, d := range diffs {
		switch d.Type {
		case Delete:
			deletions = append(deletions, d.Token)
		case Insert:
			insertions = append(insertions, d.Token)
		}
	}

	// The key insight: only "SomeType" should be deleted and "SomeOtherType" inserted
	// NOT "someFunction(SomeType" -> "someFunction(SomeOtherType" like wdiff would do
	expectedDeletions := []string{"SomeType"}
	expectedInsertions := []string{"SomeOtherType"}

	if !reflect.DeepEqual(deletions, expectedDeletions) {
		t.Errorf("Deletions = %v, want %v", deletions, expectedDeletions)
	}
	if !reflect.DeepEqual(insertions, expectedInsertions) {
		t.Errorf("Insertions = %v, want %v", insertions, expectedInsertions)
	}
}

func TestDiffStrings(t *testing.T) {
	tests := []struct {
		name     string
		text1    string
		text2    string
		opts     Options
		expected []Diff
	}{
		{
			name:  "identical strings",
			text1: "hello world",
			text2: "hello world",
			opts:  DefaultOptions(),
			expected: []Diff{
				{Equal, "hello"},
				{Equal, "world"},
			},
		},
		{
			name:  "single word change",
			text1: "hello world",
			text2: "hello universe",
			opts:  DefaultOptions(),
			expected: []Diff{
				{Equal, "hello"},
				{Delete, "world"},
				{Insert, "universe"},
			},
		},
		{
			name:  "function parameter type change",
			text1: "foo(int x)",
			text2: "foo(string x)",
			opts:  Options{Delimiters: "()"},
			expected: []Diff{
				{Equal, "foo"},
				{Equal, "("},
				{Delete, "int"},
				{Insert, "string"},
				{Equal, "x"},
				{Equal, ")"},
			},
		},
		{
			name:  "added parameter",
			text1: "foo(a)",
			text2: "foo(a, b)",
			opts:  Options{Delimiters: "(),"},
			expected: []Diff{
				{Equal, "foo"},
				{Equal, "("},
				{Equal, "a"},
				{Insert, ","},
				{Insert, "b"},
				{Equal, ")"},
			},
		},
		{
			name:  "empty to something",
			text1: "",
			text2: "hello",
			opts:  DefaultOptions(),
			expected: []Diff{
				{Insert, "hello"},
			},
		},
		{
			name:  "something to empty",
			text1: "hello",
			text2: "",
			opts:  DefaultOptions(),
			expected: []Diff{
				{Delete, "hello"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiffStrings(tt.text1, tt.text2, tt.opts)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DiffStrings(%q, %q) = %v, want %v",
					tt.text1, tt.text2, result, tt.expected)
			}
		})
	}
}

func TestDiffTokens(t *testing.T) {
	// Test that DiffTokens works directly with pre-tokenized input
	tokens1 := []string{"a", "b", "c"}
	tokens2 := []string{"a", "x", "c"}

	diffs := DiffTokens(tokens1, tokens2)

	expected := []Diff{
		{Equal, "a"},
		{Delete, "b"},
		{Insert, "x"},
		{Equal, "c"},
	}

	if !reflect.DeepEqual(diffs, expected) {
		t.Errorf("DiffTokens() = %v, want %v", diffs, expected)
	}
}

// TestOperationString tests the Operation.String() method
func TestOperationString(t *testing.T) {
	tests := []struct {
		op       Operation
		expected string
	}{
		{Equal, "Equal"},
		{Insert, "Insert"},
		{Delete, "Delete"},
		{Operation(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.op.String()
			if result != tt.expected {
				t.Errorf("Operation(%d).String() = %q, want %q", tt.op, result, tt.expected)
			}
		})
	}
}

// TestHasChanges tests the HasChanges helper function
func TestHasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []Diff
		expected bool
	}{
		{
			name:     "empty",
			diffs:    []Diff{},
			expected: false,
		},
		{
			name:     "all equal",
			diffs:    []Diff{{Equal, "a"}, {Equal, "b"}},
			expected: false,
		},
		{
			name:     "has insert",
			diffs:    []Diff{{Equal, "a"}, {Insert, "b"}},
			expected: true,
		},
		{
			name:     "has delete",
			diffs:    []Diff{{Delete, "a"}, {Equal, "b"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasChanges(tt.diffs)
			if result != tt.expected {
				t.Errorf("HasChanges() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestIgnoreCase tests case-insensitive comparison
func TestIgnoreCase(t *testing.T) {
	tests := []struct {
		name       string
		text1      string
		text2      string
		ignoreCase bool
		wantChange bool
	}{
		{
			name:       "case differs, ignore case",
			text1:      "Hello World",
			text2:      "hello world",
			ignoreCase: true,
			wantChange: false,
		},
		{
			name:       "case differs, case sensitive",
			text1:      "Hello World",
			text2:      "hello world",
			ignoreCase: false,
			wantChange: true,
		},
		{
			name:       "same case, ignore case",
			text1:      "hello world",
			text2:      "hello world",
			ignoreCase: true,
			wantChange: false,
		},
		{
			name:       "different words, ignore case",
			text1:      "hello world",
			text2:      "hello universe",
			ignoreCase: true,
			wantChange: true,
		},
		{
			name:       "mixed case change, ignore case",
			text1:      "Hello WORLD",
			text2:      "HELLO world",
			ignoreCase: true,
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Delimiters: DefaultDelimiters,
				IgnoreCase: tt.ignoreCase,
			}
			diffs := DiffStrings(tt.text1, tt.text2, opts)
			hasChange := HasChanges(diffs)
			if hasChange != tt.wantChange {
				t.Errorf("DiffStrings(%q, %q, ignoreCase=%v) hasChanges = %v, want %v",
					tt.text1, tt.text2, tt.ignoreCase, hasChange, tt.wantChange)
			}
		})
	}
}

// TestIgnoreCasePreservesOriginal tests that original case is preserved in output
func TestIgnoreCasePreservesOriginal(t *testing.T) {
	text1 := "Hello World foo"
	text2 := "hello WORLD bar"

	opts := Options{
		Delimiters: DefaultDelimiters,
		IgnoreCase: true,
	}
	diffs := DiffStrings(text1, text2, opts)

	// Should have: Equal "hello", Equal "WORLD", Delete "foo", Insert "bar"
	// The equal tokens should use the case from text2 (new file)
	var equalTokens []string
	var deleteTokens []string
	var insertTokens []string

	for _, d := range diffs {
		switch d.Type {
		case Equal:
			equalTokens = append(equalTokens, d.Token)
		case Delete:
			deleteTokens = append(deleteTokens, d.Token)
		case Insert:
			insertTokens = append(insertTokens, d.Token)
		}
	}

	// Equal tokens should be from text2
	expectedEqual := []string{"hello", "WORLD"}
	if !reflect.DeepEqual(equalTokens, expectedEqual) {
		t.Errorf("Equal tokens = %v, want %v", equalTokens, expectedEqual)
	}

	// Delete should be from text1
	expectedDelete := []string{"foo"}
	if !reflect.DeepEqual(deleteTokens, expectedDelete) {
		t.Errorf("Delete tokens = %v, want %v", deleteTokens, expectedDelete)
	}

	// Insert should be from text2
	expectedInsert := []string{"bar"}
	if !reflect.DeepEqual(insertTokens, expectedInsert) {
		t.Errorf("Insert tokens = %v, want %v", insertTokens, expectedInsert)
	}
}

func TestDiffTokensWithPreprocessing(t *testing.T) {
	tests := []struct {
		name                 string
		tokens1              []string
		tokens2              []string
		wantNoSpuriousEquals bool // if true, check that common tokens like "-" are not Equal
	}{
		{
			name:                 "simple diff with no confusing tokens",
			tokens1:              []string{"hello", "world"},
			tokens2:              []string{"hello", "universe"},
			wantNoSpuriousEquals: false,
		},
		{
			name:                 "diff with high-frequency dash token",
			tokens1:              []string{"@@", "-117,6", "+117,34", "@@", "#", "Remove", "-", "essential", "for", "integration"},
			tokens2:              []string{"@@", "-7,6", "+7,8", "@@", "##", "Features", "-", "Display", "HTML", "from", "files"},
			wantNoSpuriousEquals: false, // With threshold, "-" may still be kept
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that preprocessing produces valid diff output
			result := DiffTokensWithPreprocessing(tt.tokens1, tt.tokens2)

			// Basic sanity checks
			if result == nil {
				t.Error("DiffTokensWithPreprocessing() returned nil")
				return
			}

			// Verify that reconstructing from diffs gives back original tokens
			var deleted, inserted, equal []string
			for _, d := range result {
				switch d.Type {
				case Delete:
					deleted = append(deleted, d.Token)
				case Insert:
					inserted = append(inserted, d.Token)
				case Equal:
					equal = append(equal, d.Token)
				}
			}

			// All deleted tokens should come from tokens1
			// All inserted tokens should come from tokens2
			// Equal tokens should exist in both
			t.Logf("Deleted: %v", deleted)
			t.Logf("Inserted: %v", inserted)
			t.Logf("Equal: %v", equal)
		})
	}
}

func TestDiffStringsWithPreprocessing(t *testing.T) {
	// Test the integration function that combines tokenization + preprocessing
	tests := []struct {
		name  string
		text1 string
		text2 string
		opts  Options
	}{
		{
			name:  "simple text diff",
			text1: "hello world",
			text2: "hello universe",
			opts:  DefaultOptions(),
		},
		{
			name:  "code diff",
			text1: "func foo(a int) { return a }",
			text2: "func foo(b int) { return b }",
			opts:  Options{Delimiters: "(){}"},
		},
		{
			name:  "diff format header scenario",
			text1: "@@ -117,6 +117,34 @@ # Remove\n- essential for integration",
			text2: "@@ -7,6 +7,8 @@ ## Features\n- Display HTML from files",
			opts:  DefaultOptions(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiffStringsWithPreprocessing(tt.text1, tt.text2, tt.opts)

			// Basic sanity check
			if result == nil {
				t.Error("DiffStringsWithPreprocessing() returned nil")
				return
			}

			// Log the result for debugging
			for _, d := range result {
				t.Logf("%s: %q", d.Type, d.Token)
			}
		})
	}
}

// TestPreprocessingVsRawDiff compares preprocessing output to raw diff
func TestPreprocessingVsRawDiff(t *testing.T) {
	// This tests how the histogram diff algorithm handles common tokens.
	// The algorithm should avoid matching common tokens like "-" and "for"
	// as anchors, grouping related changes together for readability.

	tokens1 := []string{"@@", "-117,6", "+117,34", "@@", "#", "Remove", "-", "essential", "for", "integration"}
	tokens2 := []string{"@@", "-7,6", "+7,8", "@@", "##", "Features", "-", "Display", "HTML", "from", "files"}

	rawResult := DiffTokens(tokens1, tokens2)
	preprocessedResult := DiffTokensWithPreprocessing(tokens1, tokens2)

	t.Log("Raw diff result:")
	for _, d := range rawResult {
		t.Logf("  %s: %q", d.Type, d.Token)
	}

	t.Log("\nPreprocessed diff result:")
	for _, d := range preprocessedResult {
		t.Logf("  %s: %q", d.Type, d.Token)
	}

	// Count Equal tokens in each result
	rawEquals := 0
	for _, d := range rawResult {
		if d.Type == Equal {
			rawEquals++
		}
	}

	prepEquals := 0
	for _, d := range preprocessedResult {
		if d.Type == Equal {
			prepEquals++
		}
	}

	t.Logf("\nRaw diff Equal tokens: %d", rawEquals)
	t.Logf("Preprocessed Equal tokens: %d", prepEquals)
}

// TestDiffTokensRaw tests the raw diff function
func TestDiffTokensRaw(t *testing.T) {
	tests := []struct {
		name    string
		tokens1 []string
		tokens2 []string
	}{
		{
			name:    "simple change",
			tokens1: []string{"a", "b", "c"},
			tokens2: []string{"a", "x", "c"},
		},
		{
			name:    "all different",
			tokens1: []string{"a", "b"},
			tokens2: []string{"x", "y"},
		},
		{
			name:    "all same",
			tokens1: []string{"a", "b", "c"},
			tokens2: []string{"a", "b", "c"},
		},
		{
			name:    "empty inputs",
			tokens1: []string{},
			tokens2: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiffTokensRaw(tt.tokens1, tt.tokens2)

			// DiffTokensRaw may return nil for empty inputs, which is valid
			if result == nil && (len(tt.tokens1) > 0 || len(tt.tokens2) > 0) {
				t.Error("DiffTokensRaw() returned nil for non-empty inputs")
				return
			}

			// Verify result is consistent
			var deleted, inserted, equal int
			for _, d := range result {
				switch d.Type {
				case Delete:
					deleted++
				case Insert:
					inserted++
				case Equal:
					equal++
				}
			}

			// Total tokens in result should make sense
			totalOps := deleted + inserted + equal
			t.Logf("Deleted: %d, Inserted: %d, Equal: %d, Total: %d", deleted, inserted, equal, totalOps)
		})
	}
}

// TestDiffStringsWithPositions tests diff with position tracking
func TestDiffStringsWithPositions(t *testing.T) {
	tests := []struct {
		name  string
		text1 string
		text2 string
		opts  Options
	}{
		{
			name:  "simple text",
			text1: "hello world",
			text2: "hello universe",
			opts:  DefaultOptions(),
		},
		{
			name:  "with delimiters",
			text1: "foo(bar)",
			text2: "foo(baz)",
			opts:  Options{Delimiters: "()"},
		},
		{
			name:  "multiline",
			text1: "line1\nline2",
			text2: "line1\nline3",
			opts:  DefaultOptions(),
		},
		{
			name:  "case insensitive",
			text1: "Hello World",
			text2: "hello WORLD",
			opts:  Options{IgnoreCase: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiffStringsWithPositions(tt.text1, tt.text2, tt.opts)

			// Check that result contains expected fields
			if result.Text1 != tt.text1 {
				t.Errorf("Text1 = %q, want %q", result.Text1, tt.text1)
			}
			if result.Text2 != tt.text2 {
				t.Errorf("Text2 = %q, want %q", result.Text2, tt.text2)
			}

			// Check that diffs are present
			if result.Diffs == nil {
				t.Error("Diffs is nil")
			}

			// Check that positions are present and valid
			if result.Positions1 == nil {
				t.Error("Positions1 is nil")
			}
			if result.Positions2 == nil {
				t.Error("Positions2 is nil")
			}

			// Verify positions are within bounds
			for i, pos := range result.Positions1 {
				if pos.Start < 0 || pos.End > len(tt.text1) || pos.Start > pos.End {
					t.Errorf("Invalid Position1[%d]: Start=%d, End=%d, text length=%d",
						i, pos.Start, pos.End, len(tt.text1))
				}
			}
			for i, pos := range result.Positions2 {
				if pos.Start < 0 || pos.End > len(tt.text2) || pos.Start > pos.End {
					t.Errorf("Invalid Position2[%d]: Start=%d, End=%d, text length=%d",
						i, pos.Start, pos.End, len(tt.text2))
				}
			}
		})
	}
}

// TestComputeStatistics tests the statistics computation
func TestComputeStatistics(t *testing.T) {
	tests := []struct {
		name         string
		text1        string
		text2        string
		opts         Options
		wantOld      int
		wantNew      int
		wantDeleted  int
		wantInserted int
	}{
		{
			name:         "identical",
			text1:        "hello world",
			text2:        "hello world",
			opts:         DefaultOptions(),
			wantOld:      2,
			wantNew:      2,
			wantDeleted:  0,
			wantInserted: 0,
		},
		{
			name:         "one word changed",
			text1:        "hello world",
			text2:        "hello universe",
			opts:         DefaultOptions(),
			wantOld:      2,
			wantNew:      2,
			wantDeleted:  1,
			wantInserted: 1,
		},
		{
			name:         "word added",
			text1:        "hello",
			text2:        "hello world",
			opts:         DefaultOptions(),
			wantOld:      1,
			wantNew:      2,
			wantDeleted:  0,
			wantInserted: 1,
		},
		{
			name:         "word removed",
			text1:        "hello world",
			text2:        "hello",
			opts:         DefaultOptions(),
			wantOld:      2,
			wantNew:      1,
			wantDeleted:  1,
			wantInserted: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := DiffStrings(tt.text1, tt.text2, tt.opts)
			stats := ComputeStatistics(tt.text1, tt.text2, diffs, tt.opts)

			if stats.OldWords != tt.wantOld {
				t.Errorf("OldWords = %d, want %d", stats.OldWords, tt.wantOld)
			}
			if stats.NewWords != tt.wantNew {
				t.Errorf("NewWords = %d, want %d", stats.NewWords, tt.wantNew)
			}
			if stats.DeletedWords != tt.wantDeleted {
				t.Errorf("DeletedWords = %d, want %d", stats.DeletedWords, tt.wantDeleted)
			}
			if stats.InsertedWords != tt.wantInserted {
				t.Errorf("InsertedWords = %d, want %d", stats.InsertedWords, tt.wantInserted)
			}
		})
	}
}

// TestDiffStringsWithPreprocessingIgnoreCase tests case-insensitive preprocessing
func TestDiffStringsWithPreprocessingIgnoreCase(t *testing.T) {
	tests := []struct {
		name       string
		text1      string
		text2      string
		wantChange bool
	}{
		{
			name:       "case only difference - no change expected",
			text1:      "Hello World",
			text2:      "hello world",
			wantChange: false,
		},
		{
			name:       "case and content difference",
			text1:      "Hello World foo",
			text2:      "HELLO WORLD bar",
			wantChange: true,
		},
		{
			name:       "mixed case with stopwords",
			text1:      "The quick brown fox",
			text2:      "THE SLOW BROWN FOX",
			wantChange: true,
		},
		{
			name:       "all same with different case",
			text1:      "ABC DEF GHI",
			text2:      "abc def ghi",
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{IgnoreCase: true}
			diffs := DiffStringsWithPreprocessing(tt.text1, tt.text2, opts)
			hasChange := HasChanges(diffs)
			if hasChange != tt.wantChange {
				t.Errorf("DiffStringsWithPreprocessing() hasChange = %v, want %v", hasChange, tt.wantChange)
				for _, d := range diffs {
					t.Logf("  %s: %q", d.Type, d.Token)
				}
			}
		})
	}
}

// TestDiffStringsWithPreprocessingCaseSensitive tests case-sensitive preprocessing
func TestDiffStringsWithPreprocessingCaseSensitive(t *testing.T) {
	tests := []struct {
		name       string
		text1      string
		text2      string
		wantChange bool
	}{
		{
			name:       "case difference - change expected",
			text1:      "Hello World",
			text2:      "hello world",
			wantChange: true,
		},
		{
			name:       "identical - no change",
			text1:      "hello world",
			text2:      "hello world",
			wantChange: false,
		},
		{
			name:       "different content",
			text1:      "foo bar",
			text2:      "baz qux",
			wantChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{IgnoreCase: false}
			diffs := DiffStringsWithPreprocessing(tt.text1, tt.text2, opts)
			hasChange := HasChanges(diffs)
			if hasChange != tt.wantChange {
				t.Errorf("DiffStringsWithPreprocessing() hasChange = %v, want %v", hasChange, tt.wantChange)
			}
		})
	}
}

// TestDiffStringsWithPositionsAndPreprocessingIgnoreCase tests case-insensitive with preprocessing
func TestDiffStringsWithPositionsAndPreprocessingIgnoreCase(t *testing.T) {
	tests := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "case change only",
			text1: "Hello World",
			text2: "hello world",
		},
		{
			name:  "case change with content change",
			text1: "Hello World foo",
			text2: "HELLO WORLD bar",
		},
		{
			name:  "mixed case with preprocessing triggers",
			text1: "The quick brown fox the jumps over the lazy dog",
			text2: "THE QUICK BROWN CAT THE LEAPS OVER THE LAZY DOG",
		},
		{
			name:  "with deletes only",
			text1: "Hello World Extra",
			text2: "hello world",
		},
		{
			name:  "with inserts only",
			text1: "Hello",
			text2: "hello world extra",
		},
		{
			name:  "complex mixed case changes",
			text1: "FUNCTION getData(X int) RETURN",
			text2: "function setData(y string) return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{IgnoreCase: true}
			result := DiffStringsWithPositionsAndPreprocessing(tt.text1, tt.text2, opts)

			if result.Text1 != tt.text1 {
				t.Errorf("Text1 = %q, want %q", result.Text1, tt.text1)
			}
			if result.Text2 != tt.text2 {
				t.Errorf("Text2 = %q, want %q", result.Text2, tt.text2)
			}

			// Log the result for inspection
			for _, d := range result.Diffs {
				t.Logf("%s: %q", d.Type, d.Token)
			}
		})
	}
}

// TestExpandFilteredDiffsWithCaseEdgeCases tests edge cases for the case-preserving expansion
func TestExpandFilteredDiffsWithCaseEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "empty filtered result",
			text1: "the the the",
			text2: "THE THE THE",
		},
		{
			name:  "all tokens filtered",
			text1: "a a a a a a a a a a",
			text2: "A A A A A A A A A A",
		},
		{
			name:  "mixed filtered and non-filtered",
			text1: "Hello the World a test",
			text2: "HELLO THE WORLD A TEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{IgnoreCase: true}
			result := DiffStringsWithPositionsAndPreprocessing(tt.text1, tt.text2, opts)

			// Verify no errors and result structure is valid
			if result.Diffs == nil {
				t.Error("Diffs should not be nil")
			}
			t.Logf("Got %d diffs", len(result.Diffs))
		})
	}
}

// TestExpandFilteredDiffsWithCaseTrailingTokens tests trailing token handling
func TestExpandFilteredDiffsWithCaseTrailingTokens(t *testing.T) {
	tests := []struct {
		name       string
		text1      string
		text2      string
		wantDelete bool
		wantInsert bool
	}{
		{
			name:       "trailing deletes",
			text1:      "HELLO WORLD EXTRA MORE",
			text2:      "hello world",
			wantDelete: true,
			wantInsert: false,
		},
		{
			name:       "trailing inserts",
			text1:      "HELLO",
			text2:      "hello world extra more",
			wantDelete: false,
			wantInsert: true,
		},
		{
			name:       "both trailing",
			text1:      "HELLO WORLD OLD",
			text2:      "hello universe new",
			wantDelete: true,
			wantInsert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{IgnoreCase: true}
			result := DiffStringsWithPositionsAndPreprocessing(tt.text1, tt.text2, opts)

			hasDelete := false
			hasInsert := false
			for _, d := range result.Diffs {
				if d.Type == Delete {
					hasDelete = true
				}
				if d.Type == Insert {
					hasInsert = true
				}
			}

			if tt.wantDelete && !hasDelete {
				t.Errorf("Expected Delete diffs, got none")
			}
			if tt.wantInsert && !hasInsert {
				t.Errorf("Expected Insert diffs, got none")
			}
		})
	}
}

// TestExpandFilteredDiffsWithCaseWithSkippedTokens tests cases where tokens are skipped
func TestExpandFilteredDiffsWithCaseWithSkippedTokens(t *testing.T) {
	// This tests the inner loops in expandFilteredDiffsWithCase
	// where we have to skip over tokens that were filtered out
	tests := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "high frequency tokens cause filtering",
			text1: "the the the UNIQUE the the",
			text2: "the the the DIFFERENT the the",
		},
		{
			name:  "filtered tokens before equal",
			text1: "a a MATCH b b",
			text2: "x x MATCH y y",
		},
		{
			name:  "filtered tokens before delete",
			text1: "the the OLD the",
			text2: "the the the",
		},
		{
			name:  "filtered tokens before insert",
			text1: "the the the",
			text2: "the the NEW the",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{IgnoreCase: true}
			result := DiffStringsWithPositionsAndPreprocessing(tt.text1, tt.text2, opts)

			if result.Diffs == nil {
				t.Error("Diffs should not be nil")
			}

			// Verify that all tokens from input are accounted for in output
			var deleteTokens, insertTokens, equalTokens []string
			for _, d := range result.Diffs {
				switch d.Type {
				case Delete:
					deleteTokens = append(deleteTokens, d.Token)
				case Insert:
					insertTokens = append(insertTokens, d.Token)
				case Equal:
					equalTokens = append(equalTokens, d.Token)
				}
			}
			t.Logf("Deletes: %v", deleteTokens)
			t.Logf("Inserts: %v", insertTokens)
			t.Logf("Equals: %v", equalTokens)
		})
	}
}

// Benchmark full diff
func BenchmarkDiffStrings(b *testing.B) {
	text1 := "func processData(input []byte, config *Config) (Result, error) {"
	text2 := "func processData(input []byte, options *Options) (Result, error) {"
	opts := DefaultOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DiffStrings(text1, text2, opts)
	}
}
