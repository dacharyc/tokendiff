package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dacharyc/tokendiff"
)

func TestFormatDiffsAdvanced(t *testing.T) {
	defaultOpts := tokendiff.FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
		UseColor:    true,
		DeleteColor: defaultDeleteColor,
		InsertColor: defaultInsertColor,
		ColorReset:  tokendiff.ANSIReset,
	}

	tests := []struct {
		name     string
		diffs    []tokendiff.Diff
		opts     tokendiff.FormatOptions
		contains []string
	}{
		{
			name: "deletions are red (no markers with color)",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "old"},
			},
			opts:     defaultOpts,
			contains: []string{defaultDeleteColor, "old", tokendiff.ANSIReset},
		},
		{
			name: "insertions are green (no markers with color)",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Insert, Token: "new"},
			},
			opts:     defaultOpts,
			contains: []string{defaultInsertColor, "new", tokendiff.ANSIReset},
		},
		{
			name: "equal tokens have no color",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Equal, Token: "same"},
			},
			opts:     defaultOpts,
			contains: []string{"same"},
		},
		{
			name: "custom markers",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "old"},
				{Type: tokendiff.Insert, Token: "new"},
			},
			opts: tokendiff.FormatOptions{
				StartDelete: "<del>",
				StopDelete:  "</del>",
				StartInsert: "<ins>",
				StopInsert:  "</ins>",
				UseColor:    false,
			},
			contains: []string{"<del>old</del>", "<ins>new</ins>"},
		},
		{
			name: "suppress deleted",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "old"},
				{Type: tokendiff.Insert, Token: "new"},
			},
			opts: tokendiff.FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoDeleted:   true,
				UseColor:    false,
			},
			contains: []string{"{+new+}"},
		},
		{
			name: "suppress inserted",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "old"},
				{Type: tokendiff.Insert, Token: "new"},
			},
			opts: tokendiff.FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoInserted:  true,
				UseColor:    false,
			},
			contains: []string{"[-old-]"},
		},
		{
			name: "suppress common",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Equal, Token: "same"},
				{Type: tokendiff.Delete, Token: "old"},
			},
			opts: tokendiff.FormatOptions{
				StartDelete: "[-",
				StopDelete:  "-]",
				StartInsert: "{+",
				StopInsert:  "+}",
				NoCommon:    true,
				UseColor:    false,
			},
			contains: []string{"[-old-]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokendiff.FormatDiffsAdvanced(tt.diffs, tt.opts)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("FormatDiffsAdvanced() = %q, want it to contain %q", result, s)
				}
			}
		})
	}
}

func TestSuppressOutput(t *testing.T) {
	diffs := []tokendiff.Diff{
		{Type: tokendiff.Equal, Token: "common"},
		{Type: tokendiff.Delete, Token: "deleted"},
		{Type: tokendiff.Insert, Token: "inserted"},
	}

	// Test noDeleted
	opts := tokendiff.FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
		NoDeleted:   true,
	}
	result := tokendiff.FormatDiffsAdvanced(diffs, opts)
	if strings.Contains(result, "deleted") {
		t.Error("NoDeleted should suppress deleted tokens")
	}
	if !strings.Contains(result, "common") || !strings.Contains(result, "inserted") {
		t.Error("NoDeleted should not affect common or inserted tokens")
	}

	// Test noInserted
	opts = tokendiff.FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
		NoInserted:  true,
	}
	result = tokendiff.FormatDiffsAdvanced(diffs, opts)
	if strings.Contains(result, "inserted") {
		t.Error("NoInserted should suppress inserted tokens")
	}

	// Test noCommon
	opts = tokendiff.FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
		NoCommon:    true,
	}
	result = tokendiff.FormatDiffsAdvanced(diffs, opts)
	if strings.Contains(result, "common") {
		t.Error("NoCommon should suppress common tokens")
	}
}

// TestHasChangesFromLibrary verifies CLI uses library's HasChanges correctly
func TestHasChangesFromLibrary(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []tokendiff.Diff
		expected bool
	}{
		{
			name: "all equal",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Equal, Token: "a"},
				{Type: tokendiff.Equal, Token: "b"},
			},
			expected: false,
		},
		{
			name: "has delete",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Equal, Token: "a"},
				{Type: tokendiff.Delete, Token: "b"},
			},
			expected: true,
		},
		{
			name: "has insert",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Insert, Token: "a"},
			},
			expected: true,
		},
		{
			name:     "empty",
			diffs:    []tokendiff.Diff{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokendiff.HasChanges(tt.diffs)
			if result != tt.expected {
				t.Errorf("tokendiff.HasChanges() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	// Create a temp file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello\nworld\n"

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Test reading it
	result, err := readFile(path)
	if err != nil {
		t.Fatalf("readFile() error = %v", err)
	}
	if result != content {
		t.Errorf("readFile() = %q, want %q", result, content)
	}

	// Test reading non-existent file
	_, err = readFile(filepath.Join(dir, "nonexistent.txt"))
	if err == nil {
		t.Error("readFile() expected error for non-existent file")
	}
}

// TestSpacingFromLibrary verifies CLI uses library's spacing functions correctly
func TestSpacingFromLibrary(t *testing.T) {
	// Just verify the library functions are accessible and work
	if !tokendiff.NeedsSpaceBefore("hello") {
		t.Error("NeedsSpaceBefore(\"hello\") should be true")
	}
	if tokendiff.NeedsSpaceBefore("(") {
		t.Error("NeedsSpaceBefore(\"(\") should be false")
	}
	if !tokendiff.NeedsSpaceAfter("hello") {
		t.Error("NeedsSpaceAfter(\"hello\") should be true")
	}
	if tokendiff.NeedsSpaceAfter("(") {
		t.Error("NeedsSpaceAfter(\"(\") should be false")
	}
}

// Integration test using temp files
func TestDiffWholeIntegration(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	err := os.WriteFile(oldPath, []byte("void someFunction(SomeType var)"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(newPath, []byte("void someFunction(SomeOtherType var)"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Read files
	text1, err := readFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	text2, err := readFile(newPath)
	if err != nil {
		t.Fatal(err)
	}

	// Diff them - use explicit delimiters to parse function arguments
	opts := tokendiff.Options{Delimiters: "()"}
	diffs := tokendiff.DiffStrings(text1, text2, opts)

	// Verify the motivating example works correctly
	var deletions, insertions []string
	for _, d := range diffs {
		switch d.Type {
		case tokendiff.Delete:
			deletions = append(deletions, d.Token)
		case tokendiff.Insert:
			insertions = append(insertions, d.Token)
		}
	}

	if len(deletions) != 1 || deletions[0] != "SomeType" {
		t.Errorf("Expected deletion of 'SomeType', got %v", deletions)
	}
	if len(insertions) != 1 || insertions[0] != "SomeOtherType" {
		t.Errorf("Expected insertion of 'SomeOtherType', got %v", insertions)
	}
}

func TestMatchContext(t *testing.T) {
	tests := []struct {
		name         string
		diffs        []tokendiff.Diff
		matchContext int
		expected     string
	}{
		{
			name: "no match context",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Equal, Token: "x"},
				{Type: tokendiff.Insert, Token: "b"},
			},
			matchContext: 0,
			expected:     "[-a-] x {+b+}",
		},
		{
			name: "match context 1 - single equal is enough",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Equal, Token: "x"},
				{Type: tokendiff.Insert, Token: "b"},
			},
			matchContext: 1,
			expected:     "[-a-] x {+b+}",
		},
		{
			name: "match context 2 - single equal not enough",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Equal, Token: "x"},
				{Type: tokendiff.Insert, Token: "b"},
			},
			matchContext: 2,
			expected:     "[-a-] [-x-] {+x+} {+b+}",
		},
		{
			name: "match context preserves leading equals",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Equal, Token: "start"},
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Insert, Token: "b"},
			},
			matchContext: 2,
			expected:     "start [-a-] {+b+}",
		},
		{
			name: "match context preserves trailing equals",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Insert, Token: "b"},
				{Type: tokendiff.Equal, Token: "end"},
			},
			matchContext: 2,
			expected:     "[-a-] {+b+} end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tokendiff.FormatOptions{
				StartDelete:      "[-",
				StopDelete:       "-]",
				StartInsert:      "{+",
				StopInsert:       "+}",
				MatchContext:     tt.matchContext,
				HeuristicSpacing: true,
			}
			result := tokendiff.FormatDiffsAdvanced(tt.diffs, opts)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOverstrikeModes(t *testing.T) {
	diffs := []tokendiff.Diff{
		{Type: tokendiff.Equal, Token: "hello"},
		{Type: tokendiff.Delete, Token: "ab"},
		{Type: tokendiff.Insert, Token: "xy"},
	}

	// Test less mode
	t.Run("less mode underlines deleted, bolds inserted", func(t *testing.T) {
		opts := tokendiff.FormatOptions{
			StartDelete: "[-",
			StopDelete:  "-]",
			StartInsert: "{+",
			StopInsert:  "+}",
			LessMode:    true,
		}
		result := tokendiff.FormatDiffsAdvanced(diffs, opts)
		// Deleted "ab" should be underlined: _\ba_\bb
		expectedDelete := "_\ba_\bb"
		// Inserted "xy" should be bold: x\bxy\by
		expectedInsert := "x\bxy\by"
		if !strings.Contains(result, expectedDelete) {
			t.Errorf("less mode: expected underlined delete %q in %q", expectedDelete, result)
		}
		if !strings.Contains(result, expectedInsert) {
			t.Errorf("less mode: expected bold insert %q in %q", expectedInsert, result)
		}
		// Should not contain markers
		if strings.Contains(result, "[-") || strings.Contains(result, "{+") {
			t.Error("less mode should not include markers")
		}
	})

	// Test printer mode
	t.Run("printer mode underlines deleted, bolds inserted", func(t *testing.T) {
		opts := tokendiff.FormatOptions{
			StartDelete: "[-",
			StopDelete:  "-]",
			StartInsert: "{+",
			StopInsert:  "+}",
			PrinterMode: true,
		}
		result := tokendiff.FormatDiffsAdvanced(diffs, opts)
		// Same behavior as less mode
		expectedDelete := "_\ba_\bb"
		expectedInsert := "x\bxy\by"
		if !strings.Contains(result, expectedDelete) {
			t.Errorf("printer mode: expected underlined delete %q in %q", expectedDelete, result)
		}
		if !strings.Contains(result, expectedInsert) {
			t.Errorf("printer mode: expected bold insert %q in %q", expectedInsert, result)
		}
	})
}

func TestAggregateDiffs(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []tokendiff.Diff
		expected string
	}{
		{
			name: "adjacent deletions combined",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Delete, Token: "b"},
			},
			expected: "[-a b-]",
		},
		{
			name: "adjacent insertions combined",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Insert, Token: "a"},
				{Type: tokendiff.Insert, Token: "b"},
			},
			expected: "{+a b+}",
		},
		{
			name: "adjacent changes combined separately",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Delete, Token: "b"},
				{Type: tokendiff.Insert, Token: "c"},
				{Type: tokendiff.Insert, Token: "d"},
			},
			expected: "[-a b-] {+c d+}",
		},
		{
			name: "equal tokens separate groups",
			diffs: []tokendiff.Diff{
				{Type: tokendiff.Delete, Token: "a"},
				{Type: tokendiff.Equal, Token: "x"},
				{Type: tokendiff.Delete, Token: "b"},
			},
			expected: "[-a-] x [-b-]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tokendiff.FormatOptions{
				StartDelete:      "[-",
				StopDelete:       "-]",
				StartInsert:      "{+",
				StopInsert:       "+}",
				AggregateChanges: true,
				HeuristicSpacing: true,
			}
			result := tokendiff.FormatDiffsAdvanced(tt.diffs, opts)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Test parsing a config string
	configContent := `# Comment line
delimiters=(){}
ignore-case
color=red,green
match-context=3
`
	// Write to temp file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "testconfig")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	if cfg.delimiters != "()" + "{}" {
		t.Errorf("delimiters = %q, want %q", cfg.delimiters, "(){}")
	}
	if !cfg.ignoreCase {
		t.Error("ignoreCase should be true")
	}
	if cfg.colorSpec != "red,green" {
		t.Errorf("colorSpec = %q, want %q", cfg.colorSpec, "red,green")
	}
	if cfg.matchContext != 3 {
		t.Errorf("matchContext = %d, want 3", cfg.matchContext)
	}
}

func TestLoadConfigEmpty(t *testing.T) {
	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	// Should return defaults
	if cfg.startDelete != "[-" {
		t.Errorf("startDelete = %q, want %q", cfg.startDelete, "[-")
	}
	if cfg.stopInsert != "+}" {
		t.Errorf("stopInsert = %q, want %q", cfg.stopInsert, "+}")
	}
}

func TestApplyConfigOption(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		checkFn func(cfg config) bool
		wantErr bool
	}{
		{"ignore-case", "true", func(cfg config) bool { return cfg.ignoreCase }, false},
		{"i", "yes", func(cfg config) bool { return cfg.ignoreCase }, false},
		{"statistics", "", func(cfg config) bool { return cfg.statistics }, false},
		{"match-context", "5", func(cfg config) bool { return cfg.matchContext == 5 }, false},
		{"unknown-option", "value", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			cfg := defaultConfig()
			err := applyConfigOption(&cfg, tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkFn != nil && !tt.checkFn(cfg) {
				t.Error("config check failed")
			}
		})
	}
}

// Test escape sequence parsing for delimiters
func TestParseEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hex escape newline",
			input:    `\x0A`,
			expected: "\n",
		},
		{
			name:     "hex escape tab",
			input:    `\x09`,
			expected: "\t",
		},
		{
			name:     "named escape newline",
			input:    `\n`,
			expected: "\n",
		},
		{
			name:     "named escape tab",
			input:    `\t`,
			expected: "\t",
		},
		{
			name:     "named escape carriage return",
			input:    `\r`,
			expected: "\r",
		},
		{
			name:     "escaped backslash",
			input:    `\\`,
			expected: `\`,
		},
		{
			name:     "escaped exclamation",
			input:    `\!`,
			expected: "!",
		},
		{
			name:     "mixed escapes with regular chars",
			input:    `abc\x0Adef\nghi`,
			expected: "abc\ndef\nghi",
		},
		{
			name:     "original dwdiff delimiter string",
			input:    `\x0A%,;/:._{}[]()'|`,
			expected: "\n%,;/:._{}[]()'|",
		},
		{
			name:     "no escapes passthrough",
			input:    "(){}[]<>,.;:",
			expected: "(){}[]<>,.;:",
		},
		{
			name:     "invalid hex escape kept as-is",
			input:    `\xZZ`,
			expected: `\xZZ`,
		},
		{
			name:     "uppercase hex",
			input:    `\X0A`,
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEscapeSequences(tt.input)
			if result != tt.expected {
				t.Errorf("parseEscapeSequences(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test line-by-line diff detection
func TestLineByLineDiffDetection(t *testing.T) {
	text1 := "line1\nline2\nline3"
	text2 := "line1\nchanged\nline3"

	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")

	opts := tokendiff.DefaultOptions()

	// Line 0: no changes
	diffs0 := tokendiff.DiffStrings(lines1[0], lines2[0], opts)
	if tokendiff.HasChanges(diffs0) {
		t.Error("Line 0 should have no changes")
	}

	// Line 1: has changes
	diffs1 := tokendiff.DiffStrings(lines1[1], lines2[1], opts)
	if !tokendiff.HasChanges(diffs1) {
		t.Error("Line 1 should have changes")
	}

	// Line 2: no changes
	diffs2 := tokendiff.DiffStrings(lines1[2], lines2[2], opts)
	if tokendiff.HasChanges(diffs2) {
		t.Error("Line 2 should have no changes")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Create a temp directory to use as home
	tmpHome, err := os.MkdirTemp("", "tokendiff-test-home")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Save and restore HOME
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Save and restore XDG_CONFIG_HOME
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	t.Run("no config file exists", func(t *testing.T) {
		result := findConfigFile("")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("finds .tokendiffrc in home", func(t *testing.T) {
		configPath := filepath.Join(tmpHome, ".tokendiffrc")
		if err := os.WriteFile(configPath, []byte("# config"), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}
		defer os.Remove(configPath)

		result := findConfigFile("")
		if result != configPath {
			t.Errorf("expected %q, got %q", configPath, result)
		}
	})

	t.Run("finds XDG config", func(t *testing.T) {
		xdgDir := filepath.Join(tmpHome, ".config", "tokendiff")
		if err := os.MkdirAll(xdgDir, 0755); err != nil {
			t.Fatalf("failed to create XDG dir: %v", err)
		}
		configPath := filepath.Join(xdgDir, "config")
		if err := os.WriteFile(configPath, []byte("# config"), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}
		defer os.RemoveAll(filepath.Join(tmpHome, ".config"))

		result := findConfigFile("")
		if result != configPath {
			t.Errorf("expected %q, got %q", configPath, result)
		}
	})

	t.Run("prefers .tokendiffrc over XDG", func(t *testing.T) {
		// Create both files
		homeConfig := filepath.Join(tmpHome, ".tokendiffrc")
		if err := os.WriteFile(homeConfig, []byte("# home config"), 0644); err != nil {
			t.Fatalf("failed to create home config: %v", err)
		}
		defer os.Remove(homeConfig)

		xdgDir := filepath.Join(tmpHome, ".config", "tokendiff")
		if err := os.MkdirAll(xdgDir, 0755); err != nil {
			t.Fatalf("failed to create XDG dir: %v", err)
		}
		xdgConfig := filepath.Join(xdgDir, "config")
		if err := os.WriteFile(xdgConfig, []byte("# xdg config"), 0644); err != nil {
			t.Fatalf("failed to create XDG config: %v", err)
		}
		defer os.RemoveAll(filepath.Join(tmpHome, ".config"))

		result := findConfigFile("")
		if result != homeConfig {
			t.Errorf("expected home config %q, got %q", homeConfig, result)
		}
	})

	t.Run("finds profile-specific config", func(t *testing.T) {
		profileConfig := filepath.Join(tmpHome, ".tokendiffrc.myprofile")
		if err := os.WriteFile(profileConfig, []byte("# profile config"), 0644); err != nil {
			t.Fatalf("failed to create profile config: %v", err)
		}
		defer os.Remove(profileConfig)

		result := findConfigFile("myprofile")
		if result != profileConfig {
			t.Errorf("expected %q, got %q", profileConfig, result)
		}
	})

	t.Run("profile not found returns empty", func(t *testing.T) {
		result := findConfigFile("nonexistent")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("respects XDG_CONFIG_HOME", func(t *testing.T) {
		customXDG := filepath.Join(tmpHome, "custom-xdg")
		xdgDir := filepath.Join(customXDG, "tokendiff")
		if err := os.MkdirAll(xdgDir, 0755); err != nil {
			t.Fatalf("failed to create custom XDG dir: %v", err)
		}
		configPath := filepath.Join(xdgDir, "config")
		if err := os.WriteFile(configPath, []byte("# custom xdg config"), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}
		defer os.RemoveAll(customXDG)

		os.Setenv("XDG_CONFIG_HOME", customXDG)
		defer os.Unsetenv("XDG_CONFIG_HOME")

		result := findConfigFile("")
		if result != configPath {
			t.Errorf("expected %q, got %q", configPath, result)
		}
	})
}

func TestPercent(t *testing.T) {
	tests := []struct {
		name     string
		part     int
		total    int
		expected int
	}{
		{"zero total returns zero", 5, 0, 0},
		{"zero part returns zero", 0, 100, 0},
		{"50 percent", 50, 100, 50},
		{"100 percent", 100, 100, 100},
		{"rounding down", 1, 3, 33},
		{"small numbers", 1, 10, 10},
		{"large numbers", 999, 1000, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percent(tt.part, tt.total)
			if result != tt.expected {
				t.Errorf("percent(%d, %d) = %d, want %d", tt.part, tt.total, result, tt.expected)
			}
		})
	}
}
