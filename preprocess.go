package tokendiff

// Discard status constants for token filtering
const (
	discardKeep        = 0 // Token should be used for matching
	discardDefinitely  = 1 // Token has no matches in other file - definitely discard
	discardProvisional = 2 // Token has too many matches - provisionally discard
)

// DiscardConfusingTokens filters tokens that appear too frequently, which would
// create many spurious match points during diff calculation.
// Returns filtered token slices and index maps back to original positions.
//
// Algorithm (inspired by GNU diff's discard_confusing_lines):
// 1. Count occurrences of each token in the OTHER file (equiv_count)
// 2. Mark tokens:
//   - equiv_count == 0: definitely discard (can't match anything)
//   - equiv_count > √n: provisionally discard (too many potential matches)
//   - else: keep for matching
//
// 3. Apply provisional discard rules:
//   - Provisional tokens are kept only if they form runs with non-provisional
//     endpoints AND at least 25% of the run is provisional
//
// Note: Filtered tokens are still included in the final diff output - they're
// just excluded from the LCS matching to prevent spurious anchoring.
func DiscardConfusingTokens(tokens1, tokens2 []string) (filtered1, filtered2 []string, map1, map2 []int) {
	// Count occurrences of each token in each file separately
	counts1 := make(map[string]int)
	for _, t := range tokens1 {
		counts1[t]++
	}
	counts2 := make(map[string]int)
	for _, t := range tokens2 {
		counts2[t]++
	}

	// Threshold: tokens appearing more than √(total) times are "confusing"
	// Minimum of 2 to avoid filtering everything in small files
	total := len(tokens1) + len(tokens2)
	many := int(sqrt(float64(total)))
	if many < 2 {
		many = 2
	}

	// Mark each token in file1 based on its occurrences in file2
	discard1 := markTokensForDiscard(tokens1, counts2, many)

	// Mark each token in file2 based on its occurrences in file1
	discard2 := markTokensForDiscard(tokens2, counts1, many)

	// Apply provisional discard rules to refine the discard decisions
	applyProvisionalRules(discard1)
	applyProvisionalRules(discard2)

	// Build filtered token lists and index maps
	filtered1 = make([]string, 0, len(tokens1))
	map1 = make([]int, 0, len(tokens1))
	for i, t := range tokens1 {
		if discard1[i] == discardKeep {
			filtered1 = append(filtered1, t)
			map1 = append(map1, i)
		}
	}

	filtered2 = make([]string, 0, len(tokens2))
	map2 = make([]int, 0, len(tokens2))
	for i, t := range tokens2 {
		if discard2[i] == discardKeep {
			filtered2 = append(filtered2, t)
			map2 = append(map2, i)
		}
	}

	return filtered1, filtered2, map1, map2
}

// markTokensForDiscard marks each token based on its occurrence count in the other file.
// Returns a slice of discard status values (discardKeep, discardDefinitely, discardProvisional).
//
// Note: We do NOT discard tokens with zero count in the other file.
// Those tokens can only be delete/insert anyway (not matches), and removing them from
// the filtered list can cause the diff algorithm to find different anchor points.
func markTokensForDiscard(tokens []string, otherCounts map[string]int, threshold int) []int {
	discard := make([]int, len(tokens))
	for i, t := range tokens {
		equivCount := otherCounts[t]
		if equivCount > threshold {
			// Token appears too many times in the other file - provisionally discard
			discard[i] = discardProvisional
		} else {
			// Keep token - either it has a reasonable count or doesn't exist in other file
			// (tokens with 0 count can't match anyway, but keeping them preserves order)
			discard[i] = discardKeep
		}
	}
	return discard
}

// applyProvisionalRules refines provisional discard decisions based on run context.
// A provisional token is converted to keep if it's part of a run where:
// - The run has non-provisional (kept) endpoints on both sides
// - At least 25% of the run consists of provisional tokens
//
// This prevents discarding provisional tokens that are "anchored" by kept tokens,
// which would create spurious match points.
func applyProvisionalRules(discard []int) {
	n := len(discard)
	if n == 0 {
		return
	}

	// Find runs of discardable tokens (both definite and provisional)
	// A run starts after a kept token and ends before a kept token
	i := 0
	for i < n {
		// Skip kept tokens
		if discard[i] == discardKeep {
			i++
			continue
		}

		// Found start of a discardable run
		runStart := i
		provisionalCount := 0

		// Find the end of this run
		for i < n && discard[i] != discardKeep {
			if discard[i] == discardProvisional {
				provisionalCount++
			}
			i++
		}
		runEnd := i // exclusive

		runLen := runEnd - runStart

		// Check if this run has kept endpoints on both sides
		hasLeftEndpoint := runStart > 0 // there's a kept token before
		hasRightEndpoint := runEnd < n  // there's a kept token after

		// If the run has anchors on both sides and enough provisional tokens,
		// convert provisional tokens back to keep
		if hasLeftEndpoint && hasRightEndpoint && runLen > 0 {
			// At least 25% of the run should be provisional to keep them
			provisionalRatio := float64(provisionalCount) / float64(runLen)
			if provisionalRatio >= 0.25 {
				// Convert provisional tokens in this run back to keep
				for j := runStart; j < runEnd; j++ {
					if discard[j] == discardProvisional {
						discard[j] = discardKeep
					}
				}
			}
		}
	}
}

// sqrt returns the integer square root (avoiding math import for this simple case)
func sqrt(n float64) float64 {
	if n <= 0 {
		return 0
	}
	x := n
	for i := 0; i < 100; i++ {
		next := 0.5 * (x + n/x)
		if next >= x-0.5 && next <= x+0.5 {
			break
		}
		x = next
	}
	return x
}
