package tokendiff

import (
	"strings"
	"testing"
)

func TestFormatDiff(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []Diff
		expected string
	}{
		{
			name: "simple change",
			diffs: []Diff{
				{Equal, "hello"},
				{Delete, "world"},
				{Insert, "universe"},
			},
			expected: "hello [-world-]{+universe+}",
		},
		{
			name: "function type change",
			diffs: []Diff{
				{Equal, "foo"},
				{Equal, "("},
				{Delete, "int"},
				{Insert, "string"},
				{Equal, ")"},
			},
			expected: "foo([-int-]{+string+})",
		},
		{
			name: "the motivating example formatted",
			diffs: []Diff{
				{Equal, "void"},
				{Equal, "someFunction"},
				{Equal, "("},
				{Delete, "SomeType"},
				{Insert, "SomeOtherType"},
				{Equal, "var"},
				{Equal, ")"},
			},
			expected: "void someFunction([-SomeType-]{+SomeOtherType+} var)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiff(tt.diffs)
			if result != tt.expected {
				t.Errorf("FormatDiff() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestNeedsSpaceBefore tests the exported spacing function
func TestNeedsSpaceBefore(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		{"hello", true},
		{"(", false},
		{")", false},
		{"{", false},
		{"}", false},
		{",", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := NeedsSpaceBefore(tt.token)
			if result != tt.expected {
				t.Errorf("NeedsSpaceBefore(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

// TestNeedsSpaceAfter tests the exported spacing function
func TestNeedsSpaceAfter(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		{"hello", true},
		{"(", false},
		{"{", false},
		{")", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := NeedsSpaceAfter(tt.token)
			if result != tt.expected {
				t.Errorf("NeedsSpaceAfter(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestFormatDiffWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []Diff
		opts     FormatOptions
		expected string
	}{
		{
			name: "default markers",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts:     DefaultFormatOptions(),
			expected: "[-old-]{+new+}",
		},
		{
			name: "custom markers",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				StartDelete: "<del>",
				StopDelete:  "</del>",
				StartInsert: "<ins>",
				StopInsert:  "</ins>",
			},
			expected: "<del>old</del><ins>new</ins>",
		},
		{
			name: "suppress deleted",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoDeleted:   true,
			},
			expected: "{+new+}",
		},
		{
			name: "suppress inserted",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoInserted:  true,
			},
			expected: "[-old-]",
		},
		{
			name: "suppress common",
			diffs: []Diff{
				{Type: Equal, Token: "same"},
				{Type: Delete, Token: "old"},
			},
			opts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoCommon:    true,
			},
			expected: "[-old-]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiffWithOptions(tt.diffs, tt.opts)
			if result != tt.expected {
				t.Errorf("FormatDiffWithOptions() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorNames(t *testing.T) {
	names := ColorNames()

	// Should return a non-empty list
	if len(names) == 0 {
		t.Error("ColorNames() returned empty list")
	}

	// Should contain standard colors
	expectedColors := []string{"red", "green", "blue", "black", "white"}
	for _, expected := range expectedColors {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ColorNames() missing expected color: %s", expected)
		}
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:      "empty string",
			spec:      "",
			wantErr:   false,
			wantEmpty: true,
		},
		{
			name:    "valid foreground color",
			spec:    "red",
			wantErr: false,
		},
		{
			name:    "valid foreground with background",
			spec:    "red:white",
			wantErr: false,
		},
		{
			name:    "invalid color",
			spec:    "notacolor",
			wantErr: true,
		},
		{
			name:    "invalid background color",
			spec:    "red:notacolor",
			wantErr: true,
		},
		{
			name:    "case insensitive",
			spec:    "RED",
			wantErr: false,
		},
		{
			name:    "with whitespace",
			spec:    "  red  ",
			wantErr: false,
		},
		{
			name:    "bright color",
			spec:    "brightred",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseColor(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseColor(%q) expected error, got nil", tt.spec)
				}
			} else {
				if err != nil {
					t.Errorf("ParseColor(%q) unexpected error: %v", tt.spec, err)
				}
				if tt.wantEmpty && result != "" {
					t.Errorf("ParseColor(%q) = %q, want empty", tt.spec, result)
				}
				if !tt.wantEmpty && result == "" {
					t.Errorf("ParseColor(%q) = empty, want non-empty", tt.spec)
				}
			}
		})
	}
}

func TestParseColorSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name:    "single color for delete",
			spec:    "red",
			wantErr: false,
		},
		{
			name:    "both colors specified",
			spec:    "red,green",
			wantErr: false,
		},
		{
			name:    "colors with backgrounds",
			spec:    "red:white,green:black",
			wantErr: false,
		},
		{
			name:    "invalid delete color",
			spec:    "notacolor,green",
			wantErr: true,
		},
		{
			name:    "invalid insert color",
			spec:    "red,notacolor",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteColor, insertColor, err := ParseColorSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseColorSpec(%q) expected error, got nil", tt.spec)
				}
			} else {
				if err != nil {
					t.Errorf("ParseColorSpec(%q) unexpected error: %v", tt.spec, err)
				}
				if deleteColor == "" {
					t.Errorf("ParseColorSpec(%q) returned empty delete color", tt.spec)
				}
				if insertColor == "" {
					t.Errorf("ParseColorSpec(%q) returned empty insert color", tt.spec)
				}
			}
		})
	}
}

func TestColorCode(t *testing.T) {
	tests := []struct {
		name    string
		fg      string
		bg      string
		bold    bool
		wantErr bool
	}{
		{
			name:    "foreground only",
			fg:      "red",
			bg:      "",
			bold:    false,
			wantErr: false,
		},
		{
			name:    "background only",
			fg:      "",
			bg:      "white",
			bold:    false,
			wantErr: false,
		},
		{
			name:    "both fg and bg",
			fg:      "red",
			bg:      "white",
			bold:    false,
			wantErr: false,
		},
		{
			name:    "with bold",
			fg:      "red",
			bg:      "",
			bold:    true,
			wantErr: false,
		},
		{
			name:    "invalid foreground",
			fg:      "notacolor",
			bg:      "",
			bold:    false,
			wantErr: true,
		},
		{
			name:    "invalid background",
			fg:      "red",
			bg:      "notacolor",
			bold:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ColorCode(tt.fg, tt.bg, tt.bold)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ColorCode(%q, %q, %v) expected error", tt.fg, tt.bg, tt.bold)
				}
			} else {
				if err != nil {
					t.Errorf("ColorCode(%q, %q, %v) unexpected error: %v", tt.fg, tt.bg, tt.bold, err)
				}
				// Result should be non-empty for valid colors
				if (tt.fg != "" || tt.bg != "" || tt.bold) && result == "" {
					t.Errorf("ColorCode(%q, %q, %v) = empty, want non-empty", tt.fg, tt.bg, tt.bold)
				}
			}
		})
	}
}

func TestOverstrikeUnderline(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "a",
			expected: "_\ba",
		},
		{
			input:    "ab",
			expected: "_\ba_\bb",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := OverstrikeUnderline(tt.input)
			if result != tt.expected {
				t.Errorf("OverstrikeUnderline(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOverstrikeBold(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "a",
			expected: "a\ba",
		},
		{
			input:    "ab",
			expected: "a\bab\bb",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := OverstrikeBold(tt.input)
			if result != tt.expected {
				t.Errorf("OverstrikeBold(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatDiffsAdvanced(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []Diff
		opts     FormatOptions
		contains []string
	}{
		{
			name: "basic with default options",
			diffs: []Diff{
				{Type: Equal, Token: "hello"},
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts:     DefaultFormatOptions(),
			contains: []string{"hello", "old", "new"},
		},
		{
			name: "with color",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				UseColor:    true,
				DeleteColor: ANSIDeleteColor,
				InsertColor: ANSIInsertColor,
				ColorReset:  ANSIReset,
			},
			contains: []string{ANSIDeleteColor, ANSIInsertColor, ANSIReset},
		},
		{
			name: "with match context",
			diffs: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Equal, Token: "x"},
				{Type: Insert, Token: "b"},
			},
			opts: FormatOptions{
				StartDelete:  "[-",
				StopDelete:   "-]",
				StartInsert:  "{+",
				StopInsert:   "+}",
				MatchContext: 2, // x should be converted since only 1 equal token
			},
			contains: []string{"[-", "-]", "{+", "+}"},
		},
		{
			name: "with aggregation",
			diffs: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "b"},
				{Type: Insert, Token: "x"},
			},
			opts: FormatOptions{
				StartDelete:      "[-",
				StopDelete:       "-]",
				StartInsert:      "{+",
				StopInsert:       "+}",
				AggregateChanges: true,
			},
			contains: []string{"a b"}, // aggregated
		},
		{
			name: "less mode with overstrike",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				LessMode: true,
			},
			contains: []string{"_\bo", "n\bn"}, // overstrike patterns
		},
		{
			name: "printer mode",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				PrinterMode: true,
			},
			contains: []string{"_\bo", "n\bn"},
		},
		{
			name: "suppress deleted",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoDeleted:   true,
			},
			contains: []string{"new"},
		},
		{
			name: "suppress inserted",
			diffs: []Diff{
				{Type: Delete, Token: "old"},
				{Type: Insert, Token: "new"},
			},
			opts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoInserted:  true,
			},
			contains: []string{"old"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiffsAdvanced(tt.diffs, tt.opts)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatDiffsAdvanced() = %q, missing %q", result, want)
				}
			}
		})
	}
}

func TestFormatDiffsAdvancedWithLineNumbers(t *testing.T) {
	diffs := []Diff{
		{Type: Equal, Token: "line1"},
		{Type: Equal, Token: "\n"},
		{Type: Delete, Token: "old"},
		{Type: Insert, Token: "new"},
		{Type: Equal, Token: "\n"},
		{Type: Equal, Token: "line3"},
	}

	opts := FormatOptions{
		StartDelete:     "[-",
		StopDelete:      "-]",
		StartInsert:     "{+",
		StopInsert:      "+}",
		ShowLineNumbers: true,
		LineNumWidth:    3,
	}

	result := FormatDiffsAdvanced(diffs, opts)

	// Should contain line number formatting
	if !strings.Contains(result, ":") {
		t.Errorf("FormatDiffsAdvanced with line numbers should contain ':' separator, got %q", result)
	}
}

func TestFormatDiffResultAdvanced(t *testing.T) {
	// Create a DiffResult with position info
	text1 := "hello old world"
	text2 := "hello new world"
	opts := DefaultOptions()
	result := DiffStringsWithPositions(text1, text2, opts)

	tests := []struct {
		name     string
		fmtOpts  FormatOptions
		contains []string
		excludes []string
	}{
		{
			name:     "default formatting",
			fmtOpts:  DefaultFormatOptions(),
			contains: []string{"hello", "old", "new", "world"},
		},
		{
			name: "with colors",
			fmtOpts: FormatOptions{
				UseColor:    true,
				DeleteColor: ANSIDeleteColor,
				InsertColor: ANSIInsertColor,
			},
			contains: []string{ANSIDeleteColor, ANSIInsertColor},
		},
		{
			name: "suppress deleted",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoDeleted:   true,
			},
			contains: []string{"new"},
			excludes: []string{"[-"},
		},
		{
			name: "suppress inserted",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoInserted:  true,
			},
			contains: []string{"old"},
			excludes: []string{"{+"},
		},
		{
			name: "suppress common",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoCommon:    true,
			},
			contains: []string{"old", "new"},
		},
		{
			name: "with line numbers",
			fmtOpts: FormatOptions{
				StartDelete:     "[-",
				StopDelete:      "-]",
				StartInsert:     "{+",
				StopInsert:      "+}",
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
			contains: []string{":"},
		},
		{
			name: "with line numbers and colors",
			fmtOpts: FormatOptions{
				UseColor:        true,
				DeleteColor:     ANSIDeleteColor,
				InsertColor:     ANSIInsertColor,
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
			contains: []string{":", ANSIDeleteColor},
		},
		{
			name: "less mode",
			fmtOpts: FormatOptions{
				LessMode: true,
			},
			contains: []string{"_\b"}, // overstrike pattern
		},
		{
			name: "printer mode",
			fmtOpts: FormatOptions{
				PrinterMode: true,
			},
			contains: []string{"\b"}, // overstrike pattern
		},
		{
			name: "with match context",
			fmtOpts: FormatOptions{
				StartDelete:  "[-",
				StopDelete:   "-]",
				StartInsert:  "{+",
				StopInsert:   "+}",
				MatchContext: 3,
			},
			contains: []string{"old", "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatDiffResultAdvanced(result, tt.fmtOpts)

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("FormatDiffResultAdvanced() missing %q in output: %q", want, output)
				}
			}
			for _, exclude := range tt.excludes {
				if strings.Contains(output, exclude) {
					t.Errorf("FormatDiffResultAdvanced() should not contain %q in output: %q", exclude, output)
				}
			}
		})
	}
}

func TestFormatDiffResultAdvancedMultiline(t *testing.T) {
	// Test multiline content with various options
	text1 := "line1\nold line\nline3"
	text2 := "line1\nnew line\nline3"
	opts := DefaultOptions()
	result := DiffStringsWithPositions(text1, text2, opts)

	tests := []struct {
		name    string
		fmtOpts FormatOptions
	}{
		{
			name: "multiline with line numbers",
			fmtOpts: FormatOptions{
				StartDelete:     "[-",
				StopDelete:      "-]",
				StartInsert:     "{+",
				StopInsert:      "+}",
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
		},
		{
			name: "multiline with colors and line numbers",
			fmtOpts: FormatOptions{
				UseColor:        true,
				DeleteColor:     ANSIDeleteColor,
				InsertColor:     ANSIInsertColor,
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
		},
		{
			name: "multiline with repeat markers",
			fmtOpts: FormatOptions{
				StartDelete:   "[-",
				StopDelete:    "-]",
				StartInsert:   "{+",
				StopInsert:    "+}",
				RepeatMarkers: true,
			},
		},
		{
			name: "multiline with colors and repeat markers",
			fmtOpts: FormatOptions{
				UseColor:      true,
				DeleteColor:   ANSIDeleteColor,
				InsertColor:   ANSIInsertColor,
				RepeatMarkers: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatDiffResultAdvanced(result, tt.fmtOpts)
			// Just verify it doesn't panic and produces output
			if output == "" {
				t.Error("FormatDiffResultAdvanced() returned empty string")
			}
			t.Logf("Output: %q", output)
		})
	}
}

func TestFormatDiffResultAdvancedEdgeCases(t *testing.T) {
	// Test edge cases
	tests := []struct {
		name    string
		text1   string
		text2   string
		fmtOpts FormatOptions
	}{
		{
			name:    "empty texts",
			text1:   "",
			text2:   "",
			fmtOpts: DefaultFormatOptions(),
		},
		{
			name:    "only insertions",
			text1:   "",
			text2:   "new content",
			fmtOpts: DefaultFormatOptions(),
		},
		{
			name:    "only deletions",
			text1:   "old content",
			text2:   "",
			fmtOpts: DefaultFormatOptions(),
		},
		{
			name:  "newline token only",
			text1: "a\nb",
			text2: "a\nc",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			result := DiffStringsWithPositions(tt.text1, tt.text2, opts)
			output := FormatDiffResultAdvanced(result, tt.fmtOpts)
			// Just verify no panic
			t.Logf("Output: %q", output)
		})
	}
}

func TestFormatDiffResultAdvancedMultilineWithAllModes(t *testing.T) {
	// Comprehensive multiline tests
	text1 := "line1\nold line\nline3\nline4"
	text2 := "line1\nnew line\nline3\nline5"
	opts := DefaultOptions()
	result := DiffStringsWithPositions(text1, text2, opts)

	tests := []struct {
		name    string
		fmtOpts FormatOptions
	}{
		{
			name: "with NoDeleted",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoDeleted:   true,
			},
		},
		{
			name: "with NoInserted",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoInserted:  true,
			},
		},
		{
			name: "with NoCommon and line numbers",
			fmtOpts: FormatOptions{
				StartDelete:     "[-",
				StopDelete:      "-]",
				StartInsert:     "{+",
				StopInsert:      "+}",
				NoCommon:        true,
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
		},
		{
			name: "with colors line numbers and repeat markers",
			fmtOpts: FormatOptions{
				UseColor:        true,
				DeleteColor:     ANSIDeleteColor,
				InsertColor:     ANSIInsertColor,
				ColorReset:      ANSIReset,
				ShowLineNumbers: true,
				LineNumWidth:    2,
				RepeatMarkers:   true,
			},
		},
		{
			name: "less mode with line numbers",
			fmtOpts: FormatOptions{
				LessMode:        true,
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
		},
		{
			name: "printer mode with line numbers",
			fmtOpts: FormatOptions{
				PrinterMode:     true,
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatDiffResultAdvanced(result, tt.fmtOpts)
			if output == "" && (text1 != "" || text2 != "") {
				t.Error("Expected non-empty output for non-empty inputs")
			}
			t.Logf("Output:\n%s", output)
		})
	}
}

func TestFormatDiffResultAdvancedWithGaps(t *testing.T) {
	// Test handling of gaps between tokens (whitespace preservation)
	text1 := "hello   world"    // multiple spaces
	text2 := "hello   universe" // multiple spaces
	opts := DefaultOptions()
	result := DiffStringsWithPositions(text1, text2, opts)

	fmtOpts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}

	output := FormatDiffResultAdvanced(result, fmtOpts)

	// Original spacing should be somewhat preserved
	if !strings.Contains(output, "hello") {
		t.Errorf("Output should contain 'hello', got %q", output)
	}
}

func TestFormatDiffResultAdvancedConsecutiveRuns(t *testing.T) {
	// Test consecutive runs of same operation type
	text1 := "a b c old1 old2 old3 d e f"
	text2 := "a b c new1 new2 new3 d e f"
	opts := DefaultOptions()
	result := DiffStringsWithPositions(text1, text2, opts)

	tests := []struct {
		name    string
		fmtOpts FormatOptions
	}{
		{
			name: "basic markers",
			fmtOpts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
			},
		},
		{
			name: "with line numbers",
			fmtOpts: FormatOptions{
				StartDelete:     "[-",
				StopDelete:      "-]",
				StartInsert:     "{+",
				StopInsert:      "+}",
				ShowLineNumbers: true,
				LineNumWidth:    2,
			},
		},
		{
			name: "with colors",
			fmtOpts: FormatOptions{
				UseColor:    true,
				DeleteColor: ANSIDeleteColor,
				InsertColor: ANSIInsertColor,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatDiffResultAdvanced(result, tt.fmtOpts)
			t.Logf("Output: %q", output)
		})
	}
}

func TestFormatDiffResultAdvancedPositionEdgeCases(t *testing.T) {
	// Test edge cases in position handling
	tests := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "single token change",
			text1: "old",
			text2: "new",
		},
		{
			name:  "change at start",
			text1: "old word word",
			text2: "new word word",
		},
		{
			name:  "change at end",
			text1: "word word old",
			text2: "word word new",
		},
		{
			name:  "change in middle",
			text1: "word old word",
			text2: "word new word",
		},
		{
			name:  "multiple separate changes",
			text1: "a old1 b old2 c",
			text2: "a new1 b new2 c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			result := DiffStringsWithPositions(tt.text1, tt.text2, opts)

			fmtOpts := FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
			}

			output := FormatDiffResultAdvanced(result, fmtOpts)
			t.Logf("%s: %q", tt.name, output)
		})
	}
}

func TestFormatDiffsAdvancedNewlineTokens(t *testing.T) {
	// Test handling of pure newline tokens
	diffs := []Diff{
		{Type: Equal, Token: "line1"},
		{Type: Delete, Token: "\n"},
		{Type: Insert, Token: "\n"},
		{Type: Equal, Token: "line2"},
	}

	tests := []struct {
		name string
		opts FormatOptions
	}{
		{
			name: "with colors",
			opts: FormatOptions{
				UseColor:    true,
				DeleteColor: ANSIDeleteColor,
				InsertColor: ANSIInsertColor,
			},
		},
		{
			name: "with less mode",
			opts: FormatOptions{
				LessMode: true,
			},
		},
		{
			name: "with printer mode",
			opts: FormatOptions{
				PrinterMode: true,
			},
		},
		{
			name: "with text markers",
			opts: FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiffsAdvanced(diffs, tt.opts)
			// Newline tokens should be preserved
			if !strings.Contains(result, "\n") {
				t.Errorf("FormatDiffsAdvanced() should contain newlines, got %q", result)
			}
		})
	}
}

func TestFormatDiffsAdvancedHeuristicSpacing(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []Diff
		opts     FormatOptions
		contains string
	}{
		{
			name: "spacing between same type deletes",
			diffs: []Diff{
				{Type: Delete, Token: "foo"},
				{Type: Delete, Token: "bar"},
			},
			opts: FormatOptions{
				StartDelete:      "[-",
				StopDelete:       "-]",
				StartInsert:      "{+",
				StopInsert:       "+}",
				HeuristicSpacing: true,
			},
			contains: " ", // space between foo and bar
		},
		{
			name: "spacing between same type inserts",
			diffs: []Diff{
				{Type: Insert, Token: "foo"},
				{Type: Insert, Token: "bar"},
			},
			opts: FormatOptions{
				StartDelete:      "[-",
				StopDelete:       "-]",
				StartInsert:      "{+",
				StopInsert:       "+}",
				HeuristicSpacing: true,
			},
			contains: " ",
		},
		{
			name: "spacing between equal tokens - no extra space",
			diffs: []Diff{
				{Type: Equal, Token: "foo"},
				{Type: Equal, Token: "bar"},
			},
			opts: FormatOptions{
				HeuristicSpacing: true,
			},
			contains: "foobar", // no space added between equals
		},
		{
			name: "spacing on type transition",
			diffs: []Diff{
				{Type: Equal, Token: "foo"},
				{Type: Delete, Token: "bar"},
			},
			opts: FormatOptions{
				StartDelete:      "[-",
				StopDelete:       "-]",
				HeuristicSpacing: true,
			},
			contains: "foo [-bar-]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiffsAdvanced(tt.diffs, tt.opts)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("FormatDiffsAdvanced() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestFormatDiffsAdvancedLineNumbersWithMultipleTypes(t *testing.T) {
	// Test line number tracking with different operation types
	diffs := []Diff{
		{Type: Equal, Token: "line1"},
		{Type: Equal, Token: "\n"},
		{Type: Delete, Token: "deleted"},
		{Type: Delete, Token: "\n"},
		{Type: Insert, Token: "inserted"},
		{Type: Insert, Token: "\n"},
		{Type: Equal, Token: "line3"},
	}

	opts := FormatOptions{
		StartDelete:     "[-",
		StopDelete:      "-]",
		StartInsert:     "{+",
		StopInsert:      "+}",
		ShowLineNumbers: true,
		LineNumWidth:    2,
	}

	result := FormatDiffsAdvanced(diffs, opts)

	// Should have multiple lines with line number prefixes
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines with line numbers, got %d", len(lines))
	}

	// Each line should start with a line number pattern
	for i, line := range lines {
		if line != "" && !strings.Contains(line, ":") {
			t.Errorf("Line %d should contain ':' for line numbers: %q", i, line)
		}
	}
}

func TestFormatDiffsAdvancedNoCommon(t *testing.T) {
	diffs := []Diff{
		{Type: Equal, Token: "same"},
		{Type: Delete, Token: "old"},
		{Type: Insert, Token: "new"},
	}

	opts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
		NoCommon:    true,
	}

	result := FormatDiffsAdvanced(diffs, opts)

	if strings.Contains(result, "same") {
		t.Errorf("FormatDiffsAdvanced() with NoCommon should not contain 'same', got %q", result)
	}
	if !strings.Contains(result, "old") || !strings.Contains(result, "new") {
		t.Errorf("FormatDiffsAdvanced() should contain 'old' and 'new', got %q", result)
	}
}

func TestFormatDiffsAdvancedRepeatMarkers(t *testing.T) {
	// Test repeat markers with multiline content using FormatDiffsAdvanced
	diffs := []Diff{
		{Type: Delete, Token: "line1\nline2"},
		{Type: Insert, Token: "new1\nnew2"},
	}

	tests := []struct {
		name     string
		opts     FormatOptions
		contains string
	}{
		{
			name: "repeat markers without color",
			opts: FormatOptions{
				StartDelete:   "[-",
				StopDelete:    "-]",
				StartInsert:   "{+",
				StopInsert:    "+}",
				RepeatMarkers: true,
			},
			contains: "-]\n[-", // marker repeated at line break
		},
		{
			name: "repeat markers with color",
			opts: FormatOptions{
				UseColor:      true,
				DeleteColor:   ANSIDeleteColor,
				InsertColor:   ANSIInsertColor,
				ColorReset:    ANSIReset,
				RepeatMarkers: true,
			},
			contains: ANSIReset + "\n" + ANSIDeleteColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDiffsAdvanced(diffs, tt.opts)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("FormatDiffsAdvanced() missing %q, got %q", tt.contains, result)
			}
		})
	}
}

// TestFormatDiffResultAdvancedNoDeletedNoInserted tests NoDeleted and NoInserted options
func TestFormatDiffResultAdvancedNoDeletedNoInserted(t *testing.T) {
	result := DiffResult{
		Text1: "hello world",
		Text2: "hello universe",
		Diffs: []Diff{
			{Type: Equal, Token: "hello"},
			{Type: Delete, Token: "world"},
			{Type: Insert, Token: "universe"},
		},
		Positions1: []TokenPos{{0, 5}, {6, 11}},
		Positions2: []TokenPos{{0, 5}, {6, 14}},
	}

	t.Run("NoDeleted hides deletions", func(t *testing.T) {
		opts := FormatOptions{
			StartDelete: "[-",
			StopDelete:  "-]",
			StartInsert: "{+",
			StopInsert:  "+}",
			NoDeleted:   true,
		}
		output := FormatDiffResultAdvanced(result, opts)
		if strings.Contains(output, "world") {
			t.Errorf("NoDeleted should hide deleted token, got %q", output)
		}
		if !strings.Contains(output, "universe") {
			t.Errorf("NoDeleted should show inserted token, got %q", output)
		}
	})

	t.Run("NoInserted hides insertions", func(t *testing.T) {
		opts := FormatOptions{
			StartDelete: "[-",
			StopDelete:  "-]",
			StartInsert: "{+",
			StopInsert:  "+}",
			NoInserted:  true,
		}
		output := FormatDiffResultAdvanced(result, opts)
		if strings.Contains(output, "universe") {
			t.Errorf("NoInserted should hide inserted token, got %q", output)
		}
		if !strings.Contains(output, "world") {
			t.Errorf("NoInserted should show deleted token, got %q", output)
		}
	})
}

// TestFormatDiffResultAdvancedLessPrinterMode tests LessMode and PrinterMode
func TestFormatDiffResultAdvancedLessPrinterMode(t *testing.T) {
	result := DiffResult{
		Text1: "old text",
		Text2: "new text",
		Diffs: []Diff{
			{Type: Delete, Token: "old"},
			{Type: Insert, Token: "new"},
			{Type: Equal, Token: "text"},
		},
		Positions1: []TokenPos{{0, 3}, {4, 8}},
		Positions2: []TokenPos{{0, 3}, {4, 8}},
	}

	t.Run("LessMode uses underline overstrike", func(t *testing.T) {
		opts := FormatOptions{LessMode: true}
		output := FormatDiffResultAdvanced(result, opts)
		// LessMode uses backspace-based underlining for deleted
		if !strings.Contains(output, "_\bo") {
			t.Errorf("LessMode should use overstrike underline for deletions, got %q", output)
		}
	})

	t.Run("PrinterMode uses bold overstrike", func(t *testing.T) {
		opts := FormatOptions{PrinterMode: true}
		output := FormatDiffResultAdvanced(result, opts)
		// PrinterMode uses backspace-based bolding for inserted
		if !strings.Contains(output, "n\bn") {
			t.Errorf("PrinterMode should use overstrike bold for insertions, got %q", output)
		}
	})
}

// TestFormatDiffResultAdvancedRepeatMarkersNoColor tests RepeatMarkers with text markers (no color)
func TestFormatDiffResultAdvancedRepeatMarkersNoColor(t *testing.T) {
	result := DiffResult{
		Text1: "line1\nline2",
		Text2: "changed1\nchanged2",
		Diffs: []Diff{
			{Type: Delete, Token: "line1\nline2"},
			{Type: Insert, Token: "changed1\nchanged2"},
		},
		Positions1: []TokenPos{{0, 11}},
		Positions2: []TokenPos{{0, 16}},
	}

	t.Run("RepeatMarkers repeats text markers at newlines", func(t *testing.T) {
		opts := FormatOptions{
			StartDelete:   "[-",
			StopDelete:    "-]",
			StartInsert:   "{+",
			StopInsert:    "+}",
			RepeatMarkers: true,
		}
		output := FormatDiffResultAdvanced(result, opts)
		// Marker should repeat after newline
		if !strings.Contains(output, "-]\n[-") {
			t.Errorf("RepeatMarkers should repeat delete markers, got %q", output)
		}
		if !strings.Contains(output, "+}\n{+") {
			t.Errorf("RepeatMarkers should repeat insert markers, got %q", output)
		}
	})
}

// TestFormatDiffResultAdvancedNoCommonWithLineNumbers tests NoCommon with line number tracking
func TestFormatDiffResultAdvancedNoCommonWithLineNumbers(t *testing.T) {
	result := DiffResult{
		Text1: "aaa\nbbb\nccc",
		Text2: "aaa\nxxx\nccc",
		Diffs: []Diff{
			{Type: Equal, Token: "aaa"},
			{Type: Equal, Token: "\n"},
			{Type: Delete, Token: "bbb"},
			{Type: Insert, Token: "xxx"},
			{Type: Equal, Token: "\n"},
			{Type: Equal, Token: "ccc"},
		},
		Positions1: []TokenPos{{0, 3}, {3, 4}, {4, 7}, {7, 8}, {8, 11}},
		Positions2: []TokenPos{{0, 3}, {3, 4}, {4, 7}, {7, 8}, {8, 11}},
	}

	t.Run("NoCommon with line numbers tracks lines correctly", func(t *testing.T) {
		opts := FormatOptions{
			StartDelete:     "[-",
			StopDelete:      "-]",
			StartInsert:     "{+",
			StopInsert:      "+}",
			NoCommon:        true,
			ShowLineNumbers: true,
			LineNumWidth:    3,
		}
		output := FormatDiffResultAdvanced(result, opts)
		// Common text hidden but line numbers should still advance
		if !strings.Contains(output, "bbb") || !strings.Contains(output, "xxx") {
			t.Errorf("NoCommon should show changes, got %q", output)
		}
	})
}

// TestFormatDiffResultAdvancedFallbackFormatting tests fallback when positions are invalid
func TestFormatDiffResultAdvancedFallbackFormatting(t *testing.T) {
	// Create a result with no positions to trigger fallback path
	result := DiffResult{
		Text1: "hello world",
		Text2: "hello universe",
		Diffs: []Diff{
			{Type: Equal, Token: "hello"},
			{Type: Delete, Token: "world"},
			{Type: Insert, Token: "universe"},
		},
		Positions1: nil, // No positions forces fallback
		Positions2: nil,
	}

	t.Run("Fallback uses heuristic spacing", func(t *testing.T) {
		opts := FormatOptions{
			StartDelete: "[-",
			StopDelete:  "-]",
			StartInsert: "{+",
			StopInsert:  "+}",
		}
		output := FormatDiffResultAdvanced(result, opts)
		// Should still produce valid output with heuristic spacing
		if !strings.Contains(output, "hello") {
			t.Errorf("Fallback should output Equal tokens, got %q", output)
		}
		if !strings.Contains(output, "[-world-]") {
			t.Errorf("Fallback should format Delete, got %q", output)
		}
		if !strings.Contains(output, "{+universe+}") {
			t.Errorf("Fallback should format Insert, got %q", output)
		}
	})

	t.Run("Fallback with consecutive tokens", func(t *testing.T) {
		result := DiffResult{
			Diffs: []Diff{
				{Type: Delete, Token: "a"},
				{Type: Delete, Token: "b"},
				{Type: Insert, Token: "x"},
				{Type: Insert, Token: "y"},
			},
			Positions1: nil,
			Positions2: nil,
		}
		opts := FormatOptions{
			StartDelete: "[-",
			StopDelete:  "-]",
			StartInsert: "{+",
			StopInsert:  "+}",
		}
		output := FormatDiffResultAdvanced(result, opts)
		// Should handle consecutive changes
		if !strings.Contains(output, "a") && !strings.Contains(output, "b") {
			t.Errorf("Fallback should show deletions, got %q", output)
		}
	})
}

// TestFormatDiffResultAdvancedColorStateTransitions tests color state management
func TestFormatDiffResultAdvancedColorStateTransitions(t *testing.T) {
	result := DiffResult{
		Text1: "old middle old2",
		Text2: "new middle new2",
		Diffs: []Diff{
			{Type: Delete, Token: "old"},
			{Type: Insert, Token: "new"},
			{Type: Equal, Token: "middle"},
			{Type: Delete, Token: "old2"},
			{Type: Insert, Token: "new2"},
		},
		Positions1: []TokenPos{{0, 3}, {4, 10}, {11, 15}},
		Positions2: []TokenPos{{0, 3}, {4, 10}, {11, 15}},
	}

	t.Run("Color resets between state transitions", func(t *testing.T) {
		opts := FormatOptions{
			UseColor:        true,
			DeleteColor:     ANSIDeleteColor,
			InsertColor:     ANSIInsertColor,
			ColorReset:      ANSIReset,
			ShowLineNumbers: true,
			LineNumWidth:    3,
		}
		output := FormatDiffResultAdvanced(result, opts)
		// Should have color reset when transitioning from colored to equal
		if !strings.Contains(output, ANSIReset) {
			t.Errorf("Should contain color reset codes, got %q", output)
		}
	})
}

// TestFormatDiffResultAdvancedGapInText2 tests gap handling in text2 positions
func TestFormatDiffResultAdvancedGapInText2(t *testing.T) {
	// Simulate a case where there's whitespace between tokens
	result := DiffResult{
		Text1: "a b",
		Text2: "a   b", // extra spaces
		Diffs: []Diff{
			{Type: Equal, Token: "a"},
			{Type: Equal, Token: "b"},
		},
		Positions1: []TokenPos{{0, 1}, {2, 3}},
		Positions2: []TokenPos{{0, 1}, {4, 5}}, // Gap from 1-4
	}

	opts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}
	output := FormatDiffResultAdvanced(result, opts)
	// Gap should be included in output
	if !strings.Contains(output, "a") || !strings.Contains(output, "b") {
		t.Errorf("Should output tokens, got %q", output)
	}
}

// TestLeadingWhitespaceInInsert tests that leading whitespace is preserved for Insert operations
func TestLeadingWhitespaceInInsert(t *testing.T) {
	// Scenario: deleting "old" and inserting "    new" (with leading spaces)
	result := DiffResult{
		Text1: "old",
		Text2: "    new", // 4 spaces before "new"
		Diffs: []Diff{
			{Type: Delete, Token: "old"},
			{Type: Insert, Token: "new"},
		},
		Positions1: []TokenPos{{0, 3}},
		Positions2: []TokenPos{{4, 7}}, // "new" starts at position 4
	}

	opts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}
	output := FormatDiffResultAdvanced(result, opts)

	// The 4 leading spaces should be preserved before the insert marker
	if !strings.Contains(output, "    {+new+}") {
		t.Errorf("Leading whitespace should be preserved for Insert, got %q", output)
	}
}

// TestLeadingWhitespaceInDelete tests that leading whitespace is preserved for Delete operations
func TestLeadingWhitespaceInDelete(t *testing.T) {
	// Scenario: deleting "    old" (with leading spaces) and inserting "new"
	result := DiffResult{
		Text1: "    old", // 4 spaces before "old"
		Text2: "new",
		Diffs: []Diff{
			{Type: Delete, Token: "old"},
			{Type: Insert, Token: "new"},
		},
		Positions1: []TokenPos{{4, 7}}, // "old" starts at position 4
		Positions2: []TokenPos{{0, 3}},
	}

	opts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}
	output := FormatDiffResultAdvanced(result, opts)

	// The 4 leading spaces should be preserved before the delete marker
	if !strings.Contains(output, "    [-old-]") {
		t.Errorf("Leading whitespace should be preserved for Delete, got %q", output)
	}
}

// TestMultiSpacePreservationInDelete tests that multiple spaces between tokens are preserved
func TestMultiSpacePreservationInDelete(t *testing.T) {
	// Scenario: deleting "hello    world" with multiple spaces
	result := DiffResult{
		Text1: "hello    world", // 4 spaces between
		Text2: "goodbye",
		Diffs: []Diff{
			{Type: Delete, Token: "hello"},
			{Type: Delete, Token: "world"},
			{Type: Insert, Token: "goodbye"},
		},
		Positions1: []TokenPos{{0, 5}, {9, 14}}, // Gap from 5-9 (4 spaces)
		Positions2: []TokenPos{{0, 7}},
	}

	opts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}
	output := FormatDiffResultAdvanced(result, opts)

	// The original spacing between deleted tokens should be preserved
	if !strings.Contains(output, "[-hello    world-]") {
		t.Errorf("Multiple spaces should be preserved in Delete, got %q", output)
	}
}
