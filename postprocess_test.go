package tokendiff

import (
	"reflect"
	"testing"
)

func TestAggregateDiffs(t *testing.T) {
	tests := []struct {
		name     string
		input    []Diff
		expected []Diff
	}{
		{
			name:     "empty",
			input:    []Diff{},
			expected: []Diff{},
		},
		{
			name: "single token",
			input: []Diff{
				{Type: Delete, Token: "a"},
			},
			expected: []Diff{
				{Type: Delete, Token: "a"},
			},
		},
		{
			name: "adjacent deletes combined",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "b"},
			},
			expected: []Diff{
				{Type: Delete, Token: "a b"},
			},
		},
		{
			name: "adjacent inserts combined",
			input: []Diff{
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "y"},
			},
			expected: []Diff{
				{Type: Insert, Token: "x y"},
			},
		},
		{
			name: "different types not combined",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
			},
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
			},
		},
		{
			name: "equals separate groups",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Delete, Token: "b"},
			},
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Delete, Token: "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateDiffs(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("AggregateDiffs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApplyMatchContext(t *testing.T) {
	tests := []struct {
		name       string
		input      []Diff
		minContext int
		expected   []Diff
	}{
		{
			name:       "zero context returns unchanged",
			input:      []Diff{{Type: Delete, Token: "a"}},
			minContext: 0,
			expected:   []Diff{{Type: Delete, Token: "a"}},
		},
		{
			name:       "negative context returns unchanged",
			input:      []Diff{{Type: Delete, Token: "a"}},
			minContext: -1,
			expected:   []Diff{{Type: Delete, Token: "a"}},
		},
		{
			name:       "empty input returns empty",
			input:      []Diff{},
			minContext: 2,
			expected:   []Diff{},
		},
		{
			name: "sufficient context preserved",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Equal, Token: "y"},
				{Type: Insert, Token: "b"},
			},
			minContext: 2,
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Equal, Token: "y"},
				{Type: Insert, Token: "b"},
			},
		},
		{
			name: "insufficient context converted to delete+insert",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Insert, Token: "b"},
			},
			minContext: 2,
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "x"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "b"},
			},
		},
		{
			name: "leading equals preserved",
			input: []Diff{
				{Type: Equal, Token: "start"},
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
			},
			minContext: 5,
			expected: []Diff{
				{Type: Equal, Token: "start"},
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
			},
		},
		{
			name: "trailing equals preserved",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
				{Type: Equal, Token: "end"},
			},
			minContext: 5,
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
				{Type: Equal, Token: "end"},
			},
		},
		{
			name: "multiple insufficient runs converted",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Insert, Token: "b"},
				{Type: Equal, Token: "y"},
				{Type: Delete, Token: "c"},
			},
			minContext: 2,
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "x"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "b"},
				{Type: Delete, Token: "y"},
				{Type: Insert, Token: "y"},
				{Type: Delete, Token: "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyMatchContext(tt.input, tt.minContext)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ApplyMatchContext() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComputeTokenSimilarity(t *testing.T) {
	tests := []struct {
		name   string
		text1  string
		text2  string
		opts   Options
		minSim float64 // minimum expected similarity
		maxSim float64 // maximum expected similarity
	}{
		{
			name:   "identical strings",
			text1:  "hello world",
			text2:  "hello world",
			opts:   DefaultOptions(),
			minSim: 1.0,
			maxSim: 1.0,
		},
		{
			name:   "completely different",
			text1:  "hello world",
			text2:  "foo bar baz",
			opts:   DefaultOptions(),
			minSim: 0.0,
			maxSim: 0.0,
		},
		{
			name:   "partial overlap - half tokens match",
			text1:  "hello world",
			text2:  "hello foo",
			opts:   DefaultOptions(),
			minSim: 0.3, // At least some similarity
			maxSim: 0.6, // But not too much
		},
		{
			name:   "empty text1",
			text1:  "",
			text2:  "hello world",
			opts:   DefaultOptions(),
			minSim: 0.0,
			maxSim: 0.0,
		},
		{
			name:   "empty text2",
			text1:  "hello world",
			text2:  "",
			opts:   DefaultOptions(),
			minSim: 0.0,
			maxSim: 0.0,
		},
		{
			name:   "both empty",
			text1:  "",
			text2:  "",
			opts:   DefaultOptions(),
			minSim: 1.0, // Two identical strings (both empty) = 100% similarity
			maxSim: 1.0,
		},
		{
			name:   "code with one token changed",
			text1:  "func getData(x int)",
			text2:  "func getData(y int)",
			opts:   Options{Delimiters: "()"},
			minSim: 0.7, // Most tokens match
			maxSim: 0.9,
		},
		{
			name:   "whitespace only text",
			text1:  "   ",
			text2:  "hello",
			opts:   DefaultOptions(),
			minSim: 0.0,
			maxSim: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := ComputeTokenSimilarity(tt.text1, tt.text2, tt.opts)
			if sim < tt.minSim || sim > tt.maxSim {
				t.Errorf("ComputeTokenSimilarity(%q, %q) = %v, want between %v and %v",
					tt.text1, tt.text2, sim, tt.minSim, tt.maxSim)
			}
		})
	}
}

func TestShiftBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    []Diff
		expected []Diff
	}{
		{
			name:     "empty input",
			input:    []Diff{},
			expected: []Diff{},
		},
		{
			name: "no changes to make",
			input: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Delete, Token: "b"},
				{Type: Insert, Token: "c"},
				{Type: Equal, Token: "d"},
			},
			expected: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Delete, Token: "b"},
				{Type: Insert, Token: "c"},
				{Type: Equal, Token: "d"},
			},
		},
		{
			name: "common prefix shifted to equal",
			input: []Diff{
				{Type: Delete, Token: "x"},
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "b"},
			},
			// DELETE[x a] INSERT[x b] -> EQUAL[x] DELETE[a] INSERT[b]
			expected: []Diff{
				{Type: Equal, Token: "x"},
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
			},
		},
		{
			name: "common suffix shifted to equal",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "x"},
				{Type: Insert, Token: "b"},
				{Type: Insert, Token: "x"},
			},
			// DELETE[a x] INSERT[b x] -> DELETE[a] INSERT[b] EQUAL[x]
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
				{Type: Equal, Token: "x"},
			},
		},
		{
			name: "both prefix and suffix shifted",
			input: []Diff{
				{Type: Delete, Token: "x"},
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "y"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "b"},
				{Type: Insert, Token: "y"},
			},
			// DELETE[x a y] INSERT[x b y] -> EQUAL[x] DELETE[a] INSERT[b] EQUAL[y]
			expected: []Diff{
				{Type: Equal, Token: "x"},
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "b"},
				{Type: Equal, Token: "y"},
			},
		},
		{
			name: "backward shift from preceding equal",
			input: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Delete, Token: "x"},
				{Type: Delete, Token: "b"},
				{Type: Insert, Token: "c"},
			},
			// EQUAL[a x] DELETE[x b] INSERT[c] -> EQUAL[a] DELETE[b x] INSERT[c]
			// The x gets shifted from start of delete to end
			expected: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Delete, Token: "b"},
				{Type: Delete, Token: "x"},
				{Type: Insert, Token: "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShiftBoundaries(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ShiftBoundaries() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInterleaveDiffs(t *testing.T) {
	tests := []struct {
		name     string
		input    []Diff
		expected []Diff
	}{
		{
			name:     "empty input",
			input:    []Diff{},
			expected: []Diff{},
		},
		{
			name: "only equals pass through",
			input: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Equal, Token: "b"},
			},
			expected: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Equal, Token: "b"},
			},
		},
		{
			name: "single delete-insert pair interleaved",
			input: []Diff{
				{Type: Delete, Token: "old1"},
				{Type: Delete, Token: "old2"},
				{Type: Insert, Token: "new1"},
				{Type: Insert, Token: "new2"},
			},
			expected: []Diff{
				{Type: Delete, Token: "old1"},
				{Type: Insert, Token: "new1"},
				{Type: Delete, Token: "old2"},
				{Type: Insert, Token: "new2"},
			},
		},
		{
			name: "more deletes than inserts",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "b"},
				{Type: Delete, Token: "c"},
				{Type: Insert, Token: "x"},
			},
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "x"},
				{Type: Delete, Token: "b"},
				{Type: Delete, Token: "c"},
			},
		},
		{
			name: "more inserts than deletes",
			input: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "y"},
				{Type: Insert, Token: "z"},
			},
			expected: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "y"},
				{Type: Insert, Token: "z"},
			},
		},
		{
			name: "insert without preceding delete",
			input: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Insert, Token: "new"},
				{Type: Equal, Token: "b"},
			},
			expected: []Diff{
				{Type: Equal, Token: "a"},
				{Type: Insert, Token: "new"},
				{Type: Equal, Token: "b"},
			},
		},
		{
			name: "equals between changes",
			input: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
				{Type: Equal, Token: "same"},
				{Type: Delete, Token: "old2"},
				{Type: Insert, Token: "new2"},
			},
			expected: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
				{Type: Equal, Token: "same"},
				{Type: Delete, Token: "old2"},
				{Type: Insert, Token: "new2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InterleaveDiffs(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("InterleaveDiffs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEliminateStopwordAnchors(t *testing.T) {
	tests := []struct {
		name     string
		input    []Diff
		expected []Diff
	}{
		{
			name:     "empty input",
			input:    []Diff{},
			expected: []Diff{},
		},
		{
			name: "no stopwords - unchanged",
			input: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Equal, Token: "important"},
				{Type: Insert, Token: "new"},
			},
			expected: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Equal, Token: "important"},
				{Type: Insert, Token: "new"},
			},
		},
		{
			name: "single stopword between changes - converted",
			input: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Equal, Token: "the"},
				{Type: Insert, Token: "new"},
			},
			expected: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Delete, Token: "the"},
				{Type: Insert, Token: "the"},
				{Type: Insert, Token: "new"},
			},
		},
		{
			name: "stopword not between changes - preserved",
			input: []Diff{
				{Type: Equal, Token: "start"},
				{Type: Equal, Token: "the"},
				{Type: Delete, Token: "old"},
			},
			expected: []Diff{
				{Type: Equal, Token: "start"},
				{Type: Equal, Token: "the"},
				{Type: Delete, Token: "old"},
			},
		},
		{
			name: "multiple equals including stopword - preserved",
			input: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Equal, Token: "the"},
				{Type: Equal, Token: "word"},
				{Type: Insert, Token: "new"},
			},
			expected: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Equal, Token: "the"},
				{Type: Equal, Token: "word"},
				{Type: Insert, Token: "new"},
			},
		},
		{
			name: "punctuation stopword between changes",
			input: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Equal, Token: "-"},
				{Type: Insert, Token: "new"},
			},
			expected: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Delete, Token: "-"},
				{Type: Insert, Token: "-"},
				{Type: Insert, Token: "new"},
			},
		},
		{
			name: "insert before delete - stopword reordered",
			input: []Diff{
				{Type: Insert, Token: "new"},
				{Type: Equal, Token: "the"},
				{Type: Delete, Token: "old"},
			},
			expected: []Diff{
				{Type: Insert, Token: "new"},
				{Type: Insert, Token: "the"},
				{Type: Delete, Token: "the"},
				{Type: Delete, Token: "old"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EliminateStopwordAnchors(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("EliminateStopwordAnchors() = %v, want %v", result, tt.expected)
			}
		})
	}
}
