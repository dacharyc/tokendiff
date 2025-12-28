package tokendiff

import (
	"testing"
)

func TestDiscardConfusingTokens(t *testing.T) {
	tests := []struct {
		name              string
		tokens1           []string
		tokens2           []string
		wantFiltered1Len  int
		wantFiltered2Len  int
		wantDiscardTokens []string // tokens that should be discarded (appear too frequently)
	}{
		{
			name:              "empty inputs",
			tokens1:           []string{},
			tokens2:           []string{},
			wantFiltered1Len:  0,
			wantFiltered2Len:  0,
			wantDiscardTokens: nil,
		},
		{
			name:    "no high-frequency tokens - disjoint sets",
			tokens1: []string{"a", "b", "c"},
			tokens2: []string{"d", "e", "f"},
			// Per-file counting: no token appears in the other file at all
			// Tokens with 0 count in other file are kept (they can't match anyway)
			wantFiltered1Len:  3,
			wantFiltered2Len:  3,
			wantDiscardTokens: nil,
		},
		{
			name:    "high-frequency token with provisional rules",
			tokens1: []string{"a", "-", "b", "-", "c", "-", "d", "-", "e"},
			tokens2: []string{"-", "x", "-", "y", "-", "z", "-", "w", "-"},
			// total = 18, sqrt(18) ≈ 4.24, so threshold is 4
			// Per-file counting:
			//   - For tokens1: "-" appears 5 times in tokens2 (5 > 4 → provisional)
			//   - For tokens2: "-" appears 4 times in tokens1 (4 NOT > 4 → keep)
			// Provisional rules: each "-" in tokens1 is isolated between kept tokens
			// (a, b, c, d, e don't exist in tokens2 so they stay kept)
			// Each single-token run has 100% provisional ratio → converted back to keep
			// Result: all tokens are kept
			wantFiltered1Len:  9,
			wantFiltered2Len:  9,
			wantDiscardTokens: nil, // provisional rules restore them
		},
		{
			name:    "common words kept in diff scenario",
			tokens1: []string{"@@", "-117,6", "+117,34", "@@", "#", "Remove", "-", "essential", "for", "integration"},
			tokens2: []string{"@@", "-7,6", "+7,8", "@@", "##", "Features", "-", "Display", "HTML", "from", "files"},
			// Total = 21, sqrt(21) ≈ 4.6, threshold = 4
			// Per-file counting:
			//   - "@@" appears 2 times in each file (2 <= 4 → keep)
			//   - "-" appears 1 time in each file (1 <= 4 → keep)
			// No token exceeds threshold in the other file
			wantFiltered1Len:  10,
			wantFiltered2Len:  11,
			wantDiscardTokens: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered1, filtered2, map1, map2 := DiscardConfusingTokens(tt.tokens1, tt.tokens2)

			if len(filtered1) != tt.wantFiltered1Len {
				t.Errorf("DiscardConfusingTokens() filtered1 len = %d, want %d (got %v)",
					len(filtered1), tt.wantFiltered1Len, filtered1)
			}
			if len(filtered2) != tt.wantFiltered2Len {
				t.Errorf("DiscardConfusingTokens() filtered2 len = %d, want %d (got %v)",
					len(filtered2), tt.wantFiltered2Len, filtered2)
			}

			// Verify map1 correctly maps filtered indices to original
			if len(map1) != len(filtered1) {
				t.Errorf("map1 length = %d, want %d", len(map1), len(filtered1))
			}
			for i, origIdx := range map1 {
				if origIdx < 0 || origIdx >= len(tt.tokens1) {
					t.Errorf("map1[%d] = %d, out of range for tokens1", i, origIdx)
				} else if tt.tokens1[origIdx] != filtered1[i] {
					t.Errorf("map1[%d] = %d maps to %q, but filtered1[%d] = %q",
						i, origIdx, tt.tokens1[origIdx], i, filtered1[i])
				}
			}

			// Verify map2 correctly maps filtered indices to original
			if len(map2) != len(filtered2) {
				t.Errorf("map2 length = %d, want %d", len(map2), len(filtered2))
			}
			for i, origIdx := range map2 {
				if origIdx < 0 || origIdx >= len(tt.tokens2) {
					t.Errorf("map2[%d] = %d, out of range for tokens2", i, origIdx)
				} else if tt.tokens2[origIdx] != filtered2[i] {
					t.Errorf("map2[%d] = %d maps to %q, but filtered2[%d] = %q",
						i, origIdx, tt.tokens2[origIdx], i, filtered2[i])
				}
			}

			// Verify discarded tokens are not in filtered results
			for _, discardToken := range tt.wantDiscardTokens {
				for _, tok := range filtered1 {
					if tok == discardToken {
						t.Errorf("filtered1 contains discarded token %q", discardToken)
					}
				}
				for _, tok := range filtered2 {
					if tok == discardToken {
						t.Errorf("filtered2 contains discarded token %q", discardToken)
					}
				}
			}
		})
	}
}
