package tokendiff

import (
	"strings"
	"testing"
)

func TestProcessUnifiedDiff(t *testing.T) {
	input := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 context line
-old word here
+new word here
 another context
`

	expected := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
context line
[-old-]{+new+} word here
another context
`

	opts := DefaultOptions()
	fmtOpts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}

	var output strings.Builder
	err := ProcessUnifiedDiff(strings.NewReader(input), &output, opts, fmtOpts)
	if err != nil {
		t.Fatalf("ProcessUnifiedDiff error: %v", err)
	}

	if output.String() != expected {
		t.Errorf("ProcessUnifiedDiff:\ngot:\n%s\nwant:\n%s", output.String(), expected)
	}
}

func TestProcessUnifiedDiffMultipleChanges(t *testing.T) {
	input := `--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,2 @@
-hello world
+hello universe
@@ -10,2 +10,2 @@
-foo bar
+foo baz
`

	opts := DefaultOptions()
	fmtOpts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}

	var output strings.Builder
	err := ProcessUnifiedDiff(strings.NewReader(input), &output, opts, fmtOpts)
	if err != nil {
		t.Fatalf("ProcessUnifiedDiff error: %v", err)
	}

	result := output.String()
	// Space between Delete and Insert because both "world" and "universe" are
	// preceded by a space in their respective texts
	if !strings.Contains(result, "hello [-world-] {+universe+}") {
		t.Errorf("expected word diff for first hunk, got: %s", result)
	}
	if !strings.Contains(result, "foo [-bar-] {+baz+}") {
		t.Errorf("expected word diff for second hunk, got: %s", result)
	}
}

func TestProcessUnifiedDiffEmptyLines(t *testing.T) {
	// Test handling of empty lines within hunks
	input := `--- a/file.txt
+++ b/file.txt
@@ -1,4 +1,4 @@
 first line

-old line
+new line
`

	opts := DefaultOptions()
	fmtOpts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}

	var output strings.Builder
	err := ProcessUnifiedDiff(strings.NewReader(input), &output, opts, fmtOpts)
	if err != nil {
		t.Fatalf("ProcessUnifiedDiff error: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "first line") {
		t.Errorf("expected 'first line' in output, got: %s", result)
	}
	if !strings.Contains(result, "[-old-]{+new+} line") {
		t.Errorf("expected word diff, got: %s", result)
	}
}

func TestProcessUnifiedDiffUnknownLineFormat(t *testing.T) {
	// Test handling of lines with unknown format (no prefix)
	input := `--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,2 @@
-old
+new
unprefixed line
`

	opts := DefaultOptions()
	fmtOpts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}

	var output strings.Builder
	err := ProcessUnifiedDiff(strings.NewReader(input), &output, opts, fmtOpts)
	if err != nil {
		t.Fatalf("ProcessUnifiedDiff error: %v", err)
	}

	result := output.String()
	// The unknown line should be passed through
	if !strings.Contains(result, "unprefixed line") {
		t.Errorf("expected unknown line to be passed through, got: %s", result)
	}
}

func TestProcessUnifiedDiffGitExtendedHeaders(t *testing.T) {
	// Test handling of various git extended headers
	input := `diff --git a/file.txt b/file.txt
index abc123..def456 100644
new file mode 100644
deleted file mode 100644
similarity index 95%
rename from old.txt
Binary files differ
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old
+new
`

	opts := DefaultOptions()
	fmtOpts := FormatOptions{
		StartDelete: "[-",
		StopDelete:  "-]",
		StartInsert: "{+",
		StopInsert:  "+}",
	}

	var output strings.Builder
	err := ProcessUnifiedDiff(strings.NewReader(input), &output, opts, fmtOpts)
	if err != nil {
		t.Fatalf("ProcessUnifiedDiff error: %v", err)
	}

	result := output.String()
	// All git headers should be passed through
	headers := []string{
		"diff --git",
		"index abc123",
		"new file mode",
		"deleted file mode",
		"similarity index",
		"rename from",
		"Binary files",
	}
	for _, h := range headers {
		if !strings.Contains(result, h) {
			t.Errorf("expected header %q in output, got: %s", h, result)
		}
	}
}
