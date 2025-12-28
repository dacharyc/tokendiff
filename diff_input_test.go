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
