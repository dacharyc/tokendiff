package tokendiff

import "strings"

// AggregateDiffs combines adjacent diffs of the same type into single tokens.
// For example, consecutive Delete operations are merged into one Delete
// with tokens joined appropriately (spaces between words, no spaces between
// punctuation/delimiters).
func AggregateDiffs(diffs []Diff) []Diff {
	if len(diffs) == 0 {
		return diffs
	}

	var result []Diff
	var currentType Operation = -1
	var currentTokens []string

	flush := func() {
		if len(currentTokens) > 0 && currentType >= 0 {
			// Join tokens using spacing heuristics
			var sb strings.Builder
			for i, token := range currentTokens {
				if i > 0 {
					prevToken := currentTokens[i-1]
					// Only add space if both tokens support spacing
					if NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(token) {
						sb.WriteString(" ")
					}
				}
				sb.WriteString(token)
			}
			result = append(result, Diff{
				Type:  currentType,
				Token: sb.String(),
			})
			currentTokens = nil
		}
	}

	for _, d := range diffs {
		if d.Type != currentType {
			flush()
			currentType = d.Type
		}
		currentTokens = append(currentTokens, d.Token)
	}
	flush()

	return result
}

// InterleaveDiffs reorders diffs so that Delete/Insert pairs are interleaved.
// When there's a sequence of Deletes followed by Inserts, this function pairs them
// positionally: Delete[0] Insert[0] Delete[1] Insert[1], etc.
// Excess Deletes or Inserts (if the counts don't match) are output at the end.
func InterleaveDiffs(diffs []Diff) []Diff {
	if len(diffs) == 0 {
		return diffs
	}

	var result []Diff
	i := 0

	for i < len(diffs) {
		d := diffs[i]

		if d.Type == Equal {
			// Equal tokens pass through unchanged
			result = append(result, d)
			i++
			continue
		}

		if d.Type == Delete {
			// Collect all consecutive Deletes
			deleteStart := i
			for i < len(diffs) && diffs[i].Type == Delete {
				i++
			}
			deletes := diffs[deleteStart:i]

			// Collect all consecutive Inserts that follow
			insertStart := i
			for i < len(diffs) && diffs[i].Type == Insert {
				i++
			}
			inserts := diffs[insertStart:i]

			// Interleave deletes and inserts
			maxLen := len(deletes)
			if len(inserts) > maxLen {
				maxLen = len(inserts)
			}

			for j := 0; j < maxLen; j++ {
				if j < len(deletes) {
					result = append(result, deletes[j])
				}
				if j < len(inserts) {
					result = append(result, inserts[j])
				}
			}
			continue
		}

		if d.Type == Insert {
			// Insert without preceding Delete - output as-is
			result = append(result, d)
			i++
			continue
		}
	}

	return result
}

// ApplyMatchContext processes diffs to require minimum context between changes.
// Equal tokens that appear between changes with fewer than minContext matches
// are converted to both Delete and Insert operations. This reduces noise from
// coincidental short matches between larger changes.
//
// If minContext is 0 or negative, the diffs are returned unchanged.
func ApplyMatchContext(diffs []Diff, minContext int) []Diff {
	if minContext <= 0 || len(diffs) == 0 {
		return diffs
	}

	var result []Diff

	// Find runs of Equal tokens between changes
	i := 0
	for i < len(diffs) {
		d := diffs[i]

		if d.Type != Equal {
			result = append(result, d)
			i++
			continue
		}

		// Found an Equal - collect consecutive Equals
		equalStart := i
		for i < len(diffs) && diffs[i].Type == Equal {
			i++
		}
		equalEnd := i
		equalCount := equalEnd - equalStart

		// Check if these Equals are between changes
		hasPrevChange := false
		hasNextChange := false

		if equalStart > 0 {
			prev := diffs[equalStart-1]
			hasPrevChange = prev.Type == Delete || prev.Type == Insert
		}
		if equalEnd < len(diffs) {
			next := diffs[equalEnd]
			hasNextChange = next.Type == Delete || next.Type == Insert
		}

		// If sandwiched between changes and below threshold, convert to changes
		if hasPrevChange && hasNextChange && equalCount < minContext {
			for j := equalStart; j < equalEnd; j++ {
				token := diffs[j].Token
				result = append(result, Diff{Type: Delete, Token: token})
				result = append(result, Diff{Type: Insert, Token: token})
			}
		} else {
			// Keep as Equal
			for j := equalStart; j < equalEnd; j++ {
				result = append(result, diffs[j])
			}
		}
	}

	return result
}

// stopwords contains common short tokens that often create spurious Equal anchors
// when they appear in different semantic contexts. These are converted to Delete+Insert
// when sandwiched between changes.
var stopwords = map[string]bool{
	// Single-character punctuation
	"-": true, ".": true, ",": true, ":": true, ";": true,
	"(": true, ")": true, "[": true, "]": true,
	// Common short words
	"a": true, "an": true, "the": true,
	"to": true, "of": true, "in": true, "on": true, "at": true,
	"for": true, "and": true, "or": true, "is": true, "it": true,
	"with": true, "as": true, "by": true, "be": true,
}

// EliminateStopwordAnchors converts stopword Equal tokens to Delete+Insert
// when they appear as single tokens sandwiched between changes.
// Unlike ApplyMatchContext, this only affects specific stopwords, preserving
// meaningful single-token Equals like "support", "config", etc.
//
// The stopword is added to both the preceding Delete run and the following
// Insert run, so they merge together during formatting instead of appearing
// as separate `[---] {+-+}` markers.
func EliminateStopwordAnchors(diffs []Diff) []Diff {
	if len(diffs) == 0 {
		return diffs
	}

	var result []Diff

	i := 0
	for i < len(diffs) {
		d := diffs[i]

		if d.Type != Equal {
			result = append(result, d)
			i++
			continue
		}

		// Found an Equal - collect consecutive Equals
		equalStart := i
		for i < len(diffs) && diffs[i].Type == Equal {
			i++
		}
		equalEnd := i
		equalCount := equalEnd - equalStart

		// Check what comes before and after
		var prevType, nextType Operation = -1, -1
		if equalStart > 0 {
			prevType = diffs[equalStart-1].Type
		}
		if equalEnd < len(diffs) {
			nextType = diffs[equalEnd].Type
		}

		// Only convert if:
		// 1. Single token Equal run
		// 2. Sandwiched between changes (Delete or Insert on both sides)
		// 3. Token is a stopword
		if equalCount == 1 && (prevType == Delete || prevType == Insert) && (nextType == Delete || nextType == Insert) {
			token := diffs[equalStart].Token
			if stopwords[token] {
				// Determine how to merge:
				// - If prev is Insert and next is Delete: add to end of prev Insert, start of next Delete
				// - If prev is Delete and next is Insert: add to end of prev Delete, start of next Insert
				// For simplicity, we'll create Delete first, then Insert, so the formatting
				// can group them with adjacent same-type tokens.
				//
				// Actually, to properly merge, we need to reorder:
				// If previous was Insert and next is Delete, we want:
				//   Insert(token) Delete(token)
				// If previous was Delete and next is Insert, we want:
				//   Delete(token) Insert(token)
				if prevType == Insert {
					// Previous was Insert, so add Insert first (merges with prev)
					result = append(result, Diff{Type: Insert, Token: token})
					result = append(result, Diff{Type: Delete, Token: token})
				} else {
					// Previous was Delete, so add Delete first (merges with prev)
					result = append(result, Diff{Type: Delete, Token: token})
					result = append(result, Diff{Type: Insert, Token: token})
				}
				continue
			}
		}

		// Keep as Equal
		for j := equalStart; j < equalEnd; j++ {
			result = append(result, diffs[j])
		}
	}

	return result
}

// ComputeTokenSimilarity calculates similarity between two strings based on shared tokens.
// Returns a value between 0.0 (no similarity) and 1.0 (identical).
// Similarity is computed as the ratio of Equal tokens to total diff operations.
func ComputeTokenSimilarity(text1, text2 string, opts Options) float64 {
	// Handle edge cases
	if text1 == text2 {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}

	tokens1 := Tokenize(text1, opts)
	tokens2 := Tokenize(text2, opts)

	// If either has no tokens, no similarity
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	diffs := DiffTokens(tokens1, tokens2)

	var equalCount, totalCount int
	for _, d := range diffs {
		totalCount++
		if d.Type == Equal {
			equalCount++
		}
	}

	if totalCount == 0 {
		return 0.0
	}

	return float64(equalCount) / float64(totalCount)
}

// collectTokensOfType collects consecutive tokens of a given type starting at index.
// Returns the tokens and the new index after the run.
func collectTokensOfType(diffs []Diff, start int, tokenType Operation) ([]string, int) {
	i := start
	for i < len(diffs) && diffs[i].Type == tokenType {
		i++
	}
	tokens := make([]string, 0, i-start)
	for j := start; j < i; j++ {
		tokens = append(tokens, diffs[j].Token)
	}
	return tokens, i
}

// findCommonPrefix returns the length of the common prefix between two token slices.
func findCommonPrefix(a, b []string) int {
	count := 0
	for count < len(a) && count < len(b) && a[count] == b[count] {
		count++
	}
	return count
}

// findCommonSuffix returns the length of the common suffix between two token slices.
func findCommonSuffix(a, b []string) int {
	count := 0
	for count < len(a) && count < len(b) &&
		a[len(a)-1-count] == b[len(b)-1-count] {
		count++
	}
	return count
}

// appendTokensAsDiffs appends tokens as Diffs of the given type.
func appendTokensAsDiffs(result []Diff, tokens []string, tokenType Operation) []Diff {
	for _, t := range tokens {
		result = append(result, Diff{Type: tokenType, Token: t})
	}
	return result
}

// processDeleteInsertPair handles a Delete/Insert pair and applies boundary shifting.
func processDeleteInsertPair(result []Diff, deleteTokens, insertTokens []string) []Diff {
	// Check backward shift: if previous Equal ends with same token as Delete starts
	for len(result) > 0 && result[len(result)-1].Type == Equal &&
		len(deleteTokens) > 0 &&
		result[len(result)-1].Token == deleteTokens[0] {
		shiftedToken := deleteTokens[0]
		deleteTokens = deleteTokens[1:]
		deleteTokens = append(deleteTokens, shiftedToken)
		result = result[:len(result)-1]
	}

	// Check forward shift: common prefix in Delete and Insert
	commonPrefix := findCommonPrefix(deleteTokens, insertTokens)
	if commonPrefix > 0 {
		result = appendTokensAsDiffs(result, deleteTokens[:commonPrefix], Equal)
		deleteTokens = deleteTokens[commonPrefix:]
		insertTokens = insertTokens[commonPrefix:]
	}

	// Check common suffix in Delete and Insert
	commonSuffix := findCommonSuffix(deleteTokens, insertTokens)
	var suffixTokens []string
	if commonSuffix > 0 {
		suffixTokens = make([]string, commonSuffix)
		for j := 0; j < commonSuffix; j++ {
			suffixTokens[j] = deleteTokens[len(deleteTokens)-commonSuffix+j]
		}
		deleteTokens = deleteTokens[:len(deleteTokens)-commonSuffix]
		insertTokens = insertTokens[:len(insertTokens)-commonSuffix]
	}

	// Output remaining deletes, inserts, and common suffix as Equal
	result = appendTokensAsDiffs(result, deleteTokens, Delete)
	result = appendTokensAsDiffs(result, insertTokens, Insert)
	result = appendTokensAsDiffs(result, suffixTokens, Equal)

	return result
}

// ShiftBoundaries adjusts diff boundaries to create cleaner output.
// When a deleted token matches an adjacent equal token, shift the boundary.
//
// This is a standard diff post-processing step (similar to GNU diff's shift_boundaries).
//
// Patterns detected and shifted:
//   - EQUAL[...x] DELETE[x] INSERT[y] → EQUAL[...x] INSERT[y] (shift delete into equal)
//   - DELETE[x] INSERT[x...] EQUAL[...] → INSERT[...] EQUAL[x...] (shift common prefix)
func ShiftBoundaries(diffs []Diff) []Diff {
	if len(diffs) == 0 {
		return diffs
	}

	result := make([]Diff, 0, len(diffs))

	i := 0
	for i < len(diffs) {
		if diffs[i].Type == Delete {
			deleteTokens, nextIdx := collectTokensOfType(diffs, i, Delete)
			insertTokens, nextIdx := collectTokensOfType(diffs, nextIdx, Insert)
			i = nextIdx

			result = processDeleteInsertPair(result, deleteTokens, insertTokens)
			continue
		}

		result = append(result, diffs[i])
		i++
	}

	return result
}
