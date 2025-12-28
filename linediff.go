package tokendiff

import "strings"

// LinePairing represents a pairing between a deleted line and an inserted line.
type LinePairing struct {
	DeleteIndex int     // index in deletes slice
	InsertIndex int     // index in inserts slice
	Similarity  float64 // similarity score (0.0-1.0)
}

// FindPositionalPairings pairs deleted and inserted lines by position.
// Delete[0] pairs with Insert[0], Delete[1] with Insert[1], etc.
// Returns pairings only up to min(len(deletes), len(inserts)).
func FindPositionalPairings(deletes, inserts []string) []LinePairing {
	minLen := len(deletes)
	if len(inserts) < minLen {
		minLen = len(inserts)
	}

	pairings := make([]LinePairing, minLen)
	for i := 0; i < minLen; i++ {
		pairings[i] = LinePairing{
			DeleteIndex: i,
			InsertIndex: i,
			Similarity:  1.0, // Positional pairing doesn't compute similarity
		}
	}
	return pairings
}

// FindSimilarityPairings pairs deleted and inserted lines by content similarity.
// Uses a greedy algorithm: for each deleted line, find the most similar unmatched
// inserted line. Lines with similarity below threshold are left unpaired.
func FindSimilarityPairings(deletes, inserts []string, opts Options, threshold float64) []LinePairing {
	var pairings []LinePairing
	usedInserts := make([]bool, len(inserts))

	for i, del := range deletes {
		bestJ, bestSim := -1, threshold
		for j, ins := range inserts {
			if usedInserts[j] {
				continue
			}
			sim := ComputeTokenSimilarity(del, ins, opts)
			if sim > bestSim {
				bestJ, bestSim = j, sim
			}
		}
		if bestJ >= 0 {
			usedInserts[bestJ] = true
			pairings = append(pairings, LinePairing{
				DeleteIndex: i,
				InsertIndex: bestJ,
				Similarity:  bestSim,
			})
		}
	}
	return pairings
}

// LineDiffResult holds diff results for a single line in line-by-line mode.
type LineDiffResult struct {
	OldLineNum int    // line number in old file
	NewLineNum int    // line number in new file
	HasChanges bool   // true if this line contains changes
	Output     string // formatted output for this line
}

// WholeFileDiffResult holds the result of a whole-file diff operation.
type WholeFileDiffResult struct {
	Result     DiffResult     // the raw diff result
	Formatted  string         // formatted output
	HasChanges bool           // true if there are any differences
	Statistics DiffStatistics // statistics about the diff
}

// LineDiffOutput holds the results of a line-by-line diff operation.
type LineDiffOutput struct {
	Lines      []LineDiffResult // individual line results
	HasChanges bool             // true if there are any differences
	Statistics DiffStatistics   // aggregate statistics
}

// DiffWholeFiles performs a whole-file word-level diff and returns structured results.
// This is the main API for comparing two complete texts.
func DiffWholeFiles(text1, text2 string, opts Options, fmtOpts FormatOptions) WholeFileDiffResult {
	result := DiffStringsWithPositionsAndPreprocessing(text1, text2, opts)
	st := ComputeStatistics(text1, text2, result.Diffs, opts)
	formatted := FormatDiffResultAdvanced(result, fmtOpts)

	return WholeFileDiffResult{
		Result:     result,
		Formatted:  formatted,
		HasChanges: HasChanges(result.Diffs),
		Statistics: st,
	}
}

// DiffLineByLine compares files line by line with proper line-level diff tracking.
// This correctly tracks dual line numbers:
// - For equal lines: both old and new line numbers increment
// - For deleted lines: only old line number increments
// - For inserted lines: only new line number increments
//
// The algorithm parameter controls how deleted and inserted lines are paired:
// - "best": similarity-based matching (pairs lines with highest token overlap)
// - "normal" or "fast": positional matching (pairs lines by position)
func DiffLineByLine(text1, text2 string, opts Options, fmtOpts FormatOptions, algorithm string, threshold float64) LineDiffOutput {
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")

	// Create format options for per-line formatting (no line numbers here)
	lineFmtOpts := fmtOpts
	lineFmtOpts.ShowLineNumbers = false

	// First, do a line-level diff to find corresponding lines
	lineDiffs := DiffTokens(lines1, lines2)

	var results []LineDiffResult
	var anyChanges bool
	var totalStats DiffStatistics
	oldLineNum := 1
	newLineNum := 1

	i := 0
	for i < len(lineDiffs) {
		ld := lineDiffs[i]

		switch ld.Type {
		case Equal:
			results = append(results, LineDiffResult{
				OldLineNum: oldLineNum,
				NewLineNum: newLineNum,
				HasChanges: false,
				Output:     ld.Token,
			})
			oldLineNum++
			newLineNum++
			i++

		case Delete:
			// Collect all consecutive deletes
			var deletes []string
			for i < len(lineDiffs) && lineDiffs[i].Type == Delete {
				deletes = append(deletes, lineDiffs[i].Token)
				i++
			}

			// Collect all consecutive inserts that follow
			var inserts []string
			for i < len(lineDiffs) && lineDiffs[i].Type == Insert {
				inserts = append(inserts, lineDiffs[i].Token)
				i++
			}

			// Any deletes or inserts mean we have changes
			anyChanges = true

			// Get pairings based on selected algorithm
			var pairings []LinePairing
			switch algorithm {
			case "best":
				pairings = FindSimilarityPairings(deletes, inserts, opts, threshold)
			default:
				pairings = FindPositionalPairings(deletes, inserts)
			}

			// Build sets of which indices are paired
			pairedDeletes := make(map[int]int)
			pairedInserts := make(map[int]int)
			for _, p := range pairings {
				pairedDeletes[p.DeleteIndex] = p.InsertIndex
				pairedInserts[p.InsertIndex] = p.DeleteIndex
			}

			// Track which inserts have been output
			outputInserts := make([]bool, len(inserts))

			// Process each delete line
			for delIdx := 0; delIdx < len(deletes); delIdx++ {
				if insIdx, paired := pairedDeletes[delIdx]; paired {
					// Output any unpaired inserts before this paired insert
					for j := 0; j < insIdx; j++ {
						if !outputInserts[j] {
							if _, isPaired := pairedInserts[j]; !isPaired {
								outputInserts[j] = true

								insertDiffs := []Diff{{Type: Insert, Token: inserts[j]}}
								lineSt := ComputeStatistics("", inserts[j], insertDiffs, opts)
								totalStats.NewWords += lineSt.NewWords
								totalStats.InsertedWords += lineSt.InsertedWords

								output := FormatDiffsAdvanced(insertDiffs, lineFmtOpts)

								results = append(results, LineDiffResult{
									OldLineNum: oldLineNum,
									NewLineNum: newLineNum,
									HasChanges: true,
									Output:     output,
								})
								newLineNum++
							}
						}
					}

					// Output the paired line
					oldLine := deletes[delIdx]
					newLine := inserts[insIdx]
					outputInserts[insIdx] = true

					wordResult := DiffStringsWithPositionsAndPreprocessing(oldLine, newLine, opts)

					lineSt := ComputeStatistics(oldLine, newLine, wordResult.Diffs, opts)
					totalStats.OldWords += lineSt.OldWords
					totalStats.NewWords += lineSt.NewWords
					totalStats.DeletedWords += lineSt.DeletedWords
					totalStats.InsertedWords += lineSt.InsertedWords
					totalStats.CommonWords += lineSt.CommonWords

					output := FormatDiffResultAdvanced(wordResult, lineFmtOpts)

					results = append(results, LineDiffResult{
						OldLineNum: oldLineNum,
						NewLineNum: newLineNum,
						HasChanges: true,
						Output:     output,
					})
					oldLineNum++
					newLineNum++
				} else {
					// Unpaired delete
					deleteDiffs := []Diff{{Type: Delete, Token: deletes[delIdx]}}
					lineSt := ComputeStatistics(deletes[delIdx], "", deleteDiffs, opts)
					totalStats.OldWords += lineSt.OldWords
					totalStats.DeletedWords += lineSt.DeletedWords

					output := FormatDiffsAdvanced(deleteDiffs, lineFmtOpts)

					results = append(results, LineDiffResult{
						OldLineNum: oldLineNum,
						NewLineNum: newLineNum,
						HasChanges: true,
						Output:     output,
					})
					oldLineNum++
				}
			}

			// Output any remaining unpaired inserts
			for j := 0; j < len(inserts); j++ {
				if !outputInserts[j] {
					insertDiffs := []Diff{{Type: Insert, Token: inserts[j]}}
					lineSt := ComputeStatistics("", inserts[j], insertDiffs, opts)
					totalStats.NewWords += lineSt.NewWords
					totalStats.InsertedWords += lineSt.InsertedWords

					output := FormatDiffsAdvanced(insertDiffs, lineFmtOpts)

					results = append(results, LineDiffResult{
						OldLineNum: oldLineNum,
						NewLineNum: newLineNum,
						HasChanges: true,
						Output:     output,
					})
					newLineNum++
				}
			}

		case Insert:
			// Pure insertion
			anyChanges = true

			insertDiffs := []Diff{{Type: Insert, Token: ld.Token}}
			lineSt := ComputeStatistics("", ld.Token, insertDiffs, opts)
			totalStats.NewWords += lineSt.NewWords
			totalStats.InsertedWords += lineSt.InsertedWords

			output := FormatDiffsAdvanced(insertDiffs, lineFmtOpts)

			results = append(results, LineDiffResult{
				OldLineNum: oldLineNum,
				NewLineNum: newLineNum,
				HasChanges: true,
				Output:     output,
			})
			newLineNum++
			i++
		}
	}

	return LineDiffOutput{
		Lines:      results,
		HasChanges: anyChanges,
		Statistics: totalStats,
	}
}

// FilterWithContext returns only the lines that are changes or within contextLines
// of a change.
func FilterWithContext(lines []LineDiffResult, contextLines int) []LineDiffResult {
	if contextLines <= 0 {
		return lines
	}

	toPrint := make([]bool, len(lines))
	for i, r := range lines {
		if r.HasChanges {
			start := i - contextLines
			if start < 0 {
				start = 0
			}
			end := i + contextLines + 1
			if end > len(lines) {
				end = len(lines)
			}
			for j := start; j < end; j++ {
				toPrint[j] = true
			}
		}
	}

	var result []LineDiffResult
	for i, r := range lines {
		if toPrint[i] {
			result = append(result, r)
		}
	}
	return result
}
