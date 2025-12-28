// Package tokendiff provides token-level diffing with delimiter support.
//
// Unlike traditional line-based diff tools, tokendiff operates at the token level
// and treats configurable delimiter characters as separate tokens. This allows
// for more precise diffs when comparing code or structured text.
//
// For example, when comparing:
//
//	someFunction(SomeType var)
//	someFunction(SomeOtherType var)
//
// A line-based diff would show the entire line changed. A word-based diff
// without delimiter awareness might show "someFunction(SomeType" changed to
// "someFunction(SomeOtherType". But tokendiff correctly identifies that only
// "SomeType" changed to "SomeOtherType" because it treats "(" as a delimiter.
//
// This package uses github.com/dacharyc/diffx to provide this functionality.
package tokendiff

import (
	"strings"

	"github.com/dacharyc/diffx"
)

// DefaultDelimiters contains the default set of delimiter characters.
// These are characters that are treated as separate tokens even when
// not surrounded by whitespace.
// NOTE: Original dwdiff has NO default delimiters (empty string).
// Words are split only on whitespace unless -d or -P is specified.
const DefaultDelimiters = ""

// DefaultWhitespace contains the default set of whitespace characters.
const DefaultWhitespace = " \t\n\r"

// Operation represents a diff operation type.
type Operation int

const (
	// Equal indicates the token is unchanged.
	Equal Operation = iota
	// Insert indicates the token was added.
	Insert
	// Delete indicates the token was removed.
	Delete
)

// String returns a human-readable representation of the operation.
func (o Operation) String() string {
	switch o {
	case Equal:
		return "Equal"
	case Insert:
		return "Insert"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}

// Diff represents a single diff operation on a token.
type Diff struct {
	Type  Operation
	Token string
}

// Options configures the diff behavior.
type Options struct {
	// Delimiters is the set of characters to treat as separate tokens.
	// If empty, DefaultDelimiters is used.
	// This is ignored if UsePunctuation is true.
	Delimiters string

	// Whitespace is the set of characters to treat as whitespace (word separators).
	// If empty, DefaultWhitespace is used.
	Whitespace string

	// UsePunctuation, when true, uses Unicode punctuation characters as
	// delimiters instead of the Delimiters string. This matches dwdiff's
	// -P/--punctuation flag behavior.
	UsePunctuation bool

	// PreserveWhitespace, when true, includes whitespace as separate tokens.
	// When false (default), whitespace is used only to separate words and
	// is not included in the diff output.
	PreserveWhitespace bool

	// IgnoreCase, when true, performs case-insensitive comparison.
	// The original case is preserved in the output.
	IgnoreCase bool
}

// DefaultOptions returns Options with default settings.
func DefaultOptions() Options {
	return Options{
		Delimiters:         DefaultDelimiters,
		PreserveWhitespace: false,
	}
}

// DiffResult contains diff output along with position information
// needed to reconstruct original spacing for Equal content.
type DiffResult struct {
	Diffs      []Diff
	Text1      string     // original old text
	Text2      string     // original new text
	Positions1 []TokenPos // token positions in text1
	Positions2 []TokenPos // token positions in text2
}

// DiffTokens computes the diff between two token slices.
// It uses the Myers diff algorithm via diffx.
func DiffTokens(tokens1, tokens2 []string) []Diff {
	return diffTokensWithDiffx(tokens1, tokens2)
}

// diffTokensWithDiffx uses the diffx library for diffing.
// Uses histogram-style diff which produces cleaner output by avoiding
// spurious matches on common words like "the", "for", "in".
func diffTokensWithDiffx(tokens1, tokens2 []string) []Diff {
	ops := diffx.DiffHistogram(tokens1, tokens2)
	return diffxOpsToDiffs(ops, tokens1, tokens2)
}

// diffxOpsToDiffs converts diffx DiffOps to tokendiff Diffs.
func diffxOpsToDiffs(ops []diffx.DiffOp, tokens1, tokens2 []string) []Diff {
	var result []Diff

	for _, op := range ops {
		switch op.Type {
		case diffx.Equal:
			for i := op.AStart; i < op.AEnd; i++ {
				result = append(result, Diff{Type: Equal, Token: tokens1[i]})
			}
		case diffx.Delete:
			for i := op.AStart; i < op.AEnd; i++ {
				result = append(result, Diff{Type: Delete, Token: tokens1[i]})
			}
		case diffx.Insert:
			for i := op.BStart; i < op.BEnd; i++ {
				result = append(result, Diff{Type: Insert, Token: tokens2[i]})
			}
		}
	}

	return result
}

// DiffTokensRaw computes the diff without semantic cleanup.
// Use this when you need the raw Myers diff output.
func DiffTokensRaw(tokens1, tokens2 []string) []Diff {
	return diffTokensWithDiffx(tokens1, tokens2)
}

// DiffStrings tokenizes both strings and computes their diff.
func DiffStrings(text1, text2 string, opts Options) []Diff {
	tokens1 := Tokenize(text1, opts)
	tokens2 := Tokenize(text2, opts)

	if opts.IgnoreCase {
		return diffTokensIgnoreCase(tokens1, tokens2)
	}
	return DiffTokens(tokens1, tokens2)
}

// DiffStringsWithPositions tokenizes and diffs strings, returning position info.
// This allows formatters to preserve original spacing for Equal content.
func DiffStringsWithPositions(text1, text2 string, opts Options) DiffResult {
	tokens1, pos1 := TokenizeWithPositions(text1, opts)
	tokens2, pos2 := TokenizeWithPositions(text2, opts)

	var diffs []Diff
	if opts.IgnoreCase {
		diffs = diffTokensIgnoreCase(tokens1, tokens2)
	} else {
		diffs = DiffTokens(tokens1, tokens2)
	}

	return DiffResult{
		Diffs:      diffs,
		Text1:      text1,
		Text2:      text2,
		Positions1: pos1,
		Positions2: pos2,
	}
}

// diffTokensIgnoreCase computes diff with case-insensitive comparison,
// preserving original case in output.
func diffTokensIgnoreCase(tokens1, tokens2 []string) []Diff {
	// Create lowercased versions for comparison
	lower1 := make([]string, len(tokens1))
	lower2 := make([]string, len(tokens2))
	for i, t := range tokens1 {
		lower1[i] = strings.ToLower(t)
	}
	for i, t := range tokens2 {
		lower2[i] = strings.ToLower(t)
	}

	// Use diffx on lowercased tokens
	ops := diffx.DiffHistogram(lower1, lower2)

	// Convert back to diffs using original tokens
	var result []Diff
	for _, op := range ops {
		switch op.Type {
		case diffx.Equal:
			// For equal tokens, prefer the original from tokens2 (new file)
			for i := op.BStart; i < op.BEnd; i++ {
				result = append(result, Diff{Type: Equal, Token: tokens2[i]})
			}
		case diffx.Delete:
			for i := op.AStart; i < op.AEnd; i++ {
				result = append(result, Diff{Type: Delete, Token: tokens1[i]})
			}
		case diffx.Insert:
			for i := op.BStart; i < op.BEnd; i++ {
				result = append(result, Diff{Type: Insert, Token: tokens2[i]})
			}
		}
	}
	return result
}

// HasChanges returns true if the diff slice contains any non-Equal operations.
func HasChanges(diffs []Diff) bool {
	for _, d := range diffs {
		if d.Type != Equal {
			return true
		}
	}
	return false
}

// DiffStatistics holds statistics about a diff operation.
type DiffStatistics struct {
	OldWords      int // total words in old text
	NewWords      int // total words in new text
	DeletedWords  int // words deleted (present in old but not new)
	InsertedWords int // words inserted (present in new but not old)
	CommonWords   int // words common to both texts
}

// ComputeStatistics calculates statistics for a diff.
func ComputeStatistics(text1, text2 string, diffs []Diff, opts Options) DiffStatistics {
	tokens1 := Tokenize(text1, opts)
	tokens2 := Tokenize(text2, opts)

	var st DiffStatistics
	st.OldWords = len(tokens1)
	st.NewWords = len(tokens2)

	for _, d := range diffs {
		switch d.Type {
		case Equal:
			st.CommonWords++
		case Delete:
			st.DeletedWords++
		case Insert:
			st.InsertedWords++
		}
	}

	return st
}

// DiffTokensWithPreprocessing computes the diff using histogram-style preprocessing.
// This uses diffx's histogram diff algorithm which:
// 1. Filters stopwords (common words like "the", "for", "in") from anchor selection
// 2. Uses low-frequency tokens as anchors for divide-and-conquer
// 3. Produces cleaner output without spurious matches on common words
//
// This produces readable output that groups semantically related changes together.
func DiffTokensWithPreprocessing(tokens1, tokens2 []string) []Diff {
	// Use diffx histogram diff which handles stopword filtering internally
	return DiffTokens(tokens1, tokens2)
}

// DiffStringsWithPreprocessing tokenizes both strings and computes their diff
// using histogram-based preprocessing that filters confusing tokens.
func DiffStringsWithPreprocessing(text1, text2 string, opts Options) []Diff {
	tokens1 := Tokenize(text1, opts)
	tokens2 := Tokenize(text2, opts)

	if opts.IgnoreCase {
		// For case-insensitive, use lowercased tokens for comparison
		lower1 := make([]string, len(tokens1))
		lower2 := make([]string, len(tokens2))
		for i, t := range tokens1 {
			lower1[i] = strings.ToLower(t)
		}
		for i, t := range tokens2 {
			lower2[i] = strings.ToLower(t)
		}
		// Use preprocessing on lowercased tokens, then map back to original case
		return diffTokensIgnoreCaseWithPreprocessing(tokens1, tokens2, lower1, lower2)
	}
	return DiffTokensWithPreprocessing(tokens1, tokens2)
}

// DiffStringsWithPositionsAndPreprocessing tokenizes and diffs strings using
// histogram-based preprocessing, returning position info for formatting.
// This allows formatters to preserve original spacing for Equal content.
func DiffStringsWithPositionsAndPreprocessing(text1, text2 string, opts Options) DiffResult {
	tokens1, pos1 := TokenizeWithPositions(text1, opts)
	tokens2, pos2 := TokenizeWithPositions(text2, opts)

	var diffs []Diff
	if opts.IgnoreCase {
		// For case-insensitive, use lowercased tokens for comparison
		lower1 := make([]string, len(tokens1))
		lower2 := make([]string, len(tokens2))
		for i, t := range tokens1 {
			lower1[i] = strings.ToLower(t)
		}
		for i, t := range tokens2 {
			lower2[i] = strings.ToLower(t)
		}
		diffs = diffTokensIgnoreCaseWithPreprocessing(tokens1, tokens2, lower1, lower2)
	} else {
		diffs = DiffTokensWithPreprocessing(tokens1, tokens2)
	}

	return DiffResult{
		Diffs:      diffs,
		Text1:      text1,
		Text2:      text2,
		Positions1: pos1,
		Positions2: pos2,
	}
}

// diffTokensIgnoreCaseWithPreprocessing handles case-insensitive diff with preprocessing.
func diffTokensIgnoreCaseWithPreprocessing(tokens1, tokens2, lower1, lower2 []string) []Diff {
	// Filter using lowercase versions
	filtered1, filtered2, map1, map2 := DiscardConfusingTokens(lower1, lower2)

	if len(filtered1) == 0 && len(filtered2) == 0 {
		return diffTokensIgnoreCase(tokens1, tokens2)
	}

	// Diff filtered lowercase tokens
	filteredDiffs := DiffTokens(filtered1, filtered2)

	// Expand back using original case tokens
	expandedDiffs := expandFilteredDiffsWithCase(filteredDiffs, tokens1, tokens2, lower1, lower2, map1, map2)

	return ShiftBoundaries(expandedDiffs)
}

// expandFilteredDiffsWithCase expands filtered diffs preserving original case.
func expandFilteredDiffsWithCase(filteredDiffs []Diff, tokens1, tokens2, lower1, lower2 []string, map1, map2 []int) []Diff {
	result := make([]Diff, 0, len(tokens1)+len(tokens2))

	origIdx1 := 0
	origIdx2 := 0
	filtIdx1 := 0
	filtIdx2 := 0

	for _, d := range filteredDiffs {
		switch d.Type {
		case Equal:
			targetOrig1 := map1[filtIdx1]
			targetOrig2 := map2[filtIdx2]

			for origIdx1 < targetOrig1 {
				result = append(result, Diff{Type: Delete, Token: tokens1[origIdx1]})
				origIdx1++
			}
			for origIdx2 < targetOrig2 {
				result = append(result, Diff{Type: Insert, Token: tokens2[origIdx2]})
				origIdx2++
			}

			// Use token from tokens2 (new file) for Equal
			result = append(result, Diff{Type: Equal, Token: tokens2[origIdx2]})
			origIdx1++
			origIdx2++
			filtIdx1++
			filtIdx2++

		case Delete:
			targetOrig1 := map1[filtIdx1]
			for origIdx1 < targetOrig1 {
				result = append(result, Diff{Type: Delete, Token: tokens1[origIdx1]})
				origIdx1++
			}
			result = append(result, Diff{Type: Delete, Token: tokens1[origIdx1]})
			origIdx1++
			filtIdx1++

		case Insert:
			targetOrig2 := map2[filtIdx2]
			for origIdx2 < targetOrig2 {
				result = append(result, Diff{Type: Insert, Token: tokens2[origIdx2]})
				origIdx2++
			}
			result = append(result, Diff{Type: Insert, Token: tokens2[origIdx2]})
			origIdx2++
			filtIdx2++
		}
	}

	for origIdx1 < len(tokens1) {
		result = append(result, Diff{Type: Delete, Token: tokens1[origIdx1]})
		origIdx1++
	}
	for origIdx2 < len(tokens2) {
		result = append(result, Diff{Type: Insert, Token: tokens2[origIdx2]})
		origIdx2++
	}

	return result
}
