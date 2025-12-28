package tokendiff

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     Options
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			opts:     DefaultOptions(),
			expected: []string{"hello", "world"},
		},
		{
			name:     "function signature - the key dwdiff use case",
			input:    "someFunction(SomeType var)",
			opts:     Options{Delimiters: "()"},
			expected: []string{"someFunction", "(", "SomeType", "var", ")"},
		},
		{
			name:     "multiple delimiters",
			input:    "foo(bar, baz)",
			opts:     Options{Delimiters: "(),"},
			expected: []string{"foo", "(", "bar", ",", "baz", ")"},
		},
		{
			name:     "code with operators",
			input:    "x = y + z",
			opts:     Options{Delimiters: "=+"},
			expected: []string{"x", "=", "y", "+", "z"},
		},
		{
			name:     "nested delimiters",
			input:    "func(a[0])",
			opts:     Options{Delimiters: "()[]"},
			expected: []string{"func", "(", "a", "[", "0", "]", ")"},
		},
		{
			name:     "custom delimiters",
			input:    "hello|world",
			opts:     Options{Delimiters: "|"},
			expected: []string{"hello", "|", "world"},
		},
		{
			name:     "preserve whitespace",
			input:    "a b",
			opts:     Options{Delimiters: DefaultDelimiters, PreserveWhitespace: true},
			expected: []string{"a", " ", "b"},
		},
		{
			name:     "empty string",
			input:    "",
			opts:     DefaultOptions(),
			expected: nil,
		},
		{
			name:     "only delimiters",
			input:    "()",
			opts:     Options{Delimiters: "()"},
			expected: []string{"(", ")"},
		},
		{
			name:     "go function",
			input:    "func main() {",
			opts:     Options{Delimiters: "(){}"},
			expected: []string{"func", "main", "(", ")", "{"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Tokenize(tt.input, tt.opts)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestUnicode verifies the library handles Unicode text correctly
func TestUnicode(t *testing.T) {
	t.Run("tokenize unicode words", func(t *testing.T) {
		// Japanese, Chinese, emoji
		input := "こんにちは 世界 🌍"
		tokens := Tokenize(input, DefaultOptions())
		expected := []string{"こんにちは", "世界", "🌍"}
		if !reflect.DeepEqual(tokens, expected) {
			t.Errorf("Tokenize(%q) = %v, want %v", input, tokens, expected)
		}
	})

	t.Run("diff unicode text", func(t *testing.T) {
		old := "Hello 世界"
		new := "Hello 世界 🌍"
		diffs := DiffStrings(old, new, DefaultOptions())

		var insertions []string
		for _, d := range diffs {
			if d.Type == Insert {
				insertions = append(insertions, d.Token)
			}
		}
		expected := []string{"🌍"}
		if !reflect.DeepEqual(insertions, expected) {
			t.Errorf("Insertions = %v, want %v", insertions, expected)
		}
	})

	t.Run("unicode delimiters", func(t *testing.T) {
		// Using custom Unicode delimiter
		input := "foo→bar→baz"
		opts := Options{Delimiters: "→"}
		tokens := Tokenize(input, opts)
		expected := []string{"foo", "→", "bar", "→", "baz"}
		if !reflect.DeepEqual(tokens, expected) {
			t.Errorf("Tokenize(%q) = %v, want %v", input, tokens, expected)
		}
	})

	t.Run("mixed script diff", func(t *testing.T) {
		old := "function データ処理(input)"
		new := "function データ変換(input)"
		diffs := DiffStrings(old, new, Options{Delimiters: "()"})

		var deletions, insertions []string
		for _, d := range diffs {
			if d.Type == Delete {
				deletions = append(deletions, d.Token)
			} else if d.Type == Insert {
				insertions = append(insertions, d.Token)
			}
		}

		if len(deletions) != 1 || deletions[0] != "データ処理" {
			t.Errorf("Deletions = %v, want [データ処理]", deletions)
		}
		if len(insertions) != 1 || insertions[0] != "データ変換" {
			t.Errorf("Insertions = %v, want [データ変換]", insertions)
		}
	})
}

// TestCustomWhitespace tests the Whitespace option
func TestCustomWhitespace(t *testing.T) {
	// Use minimal delimiters to avoid interference with whitespace tests
	noDelimiters := "()"

	tests := []struct {
		name       string
		input      string
		whitespace string
		delimiters string
		expected   []string
	}{
		{
			name:       "default whitespace",
			input:      "hello world",
			whitespace: "",
			delimiters: noDelimiters,
			expected:   []string{"hello", "world"},
		},
		{
			name:       "pipe as whitespace",
			input:      "hello|world",
			whitespace: "|",
			delimiters: noDelimiters,
			expected:   []string{"hello", "world"},
		},
		{
			name:       "multiple custom whitespace chars",
			input:      "a|b:c",
			whitespace: "|:",
			delimiters: noDelimiters,
			expected:   []string{"a", "b", "c"},
		},
		{
			name:       "colon as whitespace in path",
			input:      "/usr/bin:/usr/local/bin",
			whitespace: ":",
			delimiters: noDelimiters,
			expected:   []string{"/usr/bin", "/usr/local/bin"},
		},
		{
			name:       "tab only whitespace",
			input:      "hello world\there",
			whitespace: "\t",
			delimiters: noDelimiters,
			expected:   []string{"hello world", "here"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				Delimiters: tt.delimiters,
				Whitespace: tt.whitespace,
			}
			result := Tokenize(tt.input, opts)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenize(%q, whitespace=%q) = %v, want %v",
					tt.input, tt.whitespace, result, tt.expected)
			}
		})
	}
}

// TestWhitespaceAndDelimiters tests interaction between whitespace and delimiters
func TestWhitespaceAndDelimiters(t *testing.T) {
	// When a character is both whitespace and delimiter, whitespace takes precedence
	// Actually, delimiters are checked first in the code, so delimiter wins
	input := "a,b c"
	opts := Options{
		Delimiters: ",",
		Whitespace: ", ", // comma is both whitespace and delimiter
	}
	result := Tokenize(input, opts)
	// Comma is checked as delimiter first, so it becomes a token
	expected := []string{"a", ",", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Tokenize(%q) = %v, want %v", input, result, expected)
	}
}

// TestUsePunctuation tests the UsePunctuation option
func TestUsePunctuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic punctuation",
			input:    "Hello, world!",
			expected: []string{"Hello", ",", "world", "!"},
		},
		{
			name:     "code with punctuation",
			input:    "foo(bar)",
			expected: []string{"foo", "(", "bar", ")"},
		},
		{
			name:     "hyphenated word",
			input:    "well-known",
			expected: []string{"well", "-", "known"},
		},
		{
			name:     "apostrophe",
			input:    "don't",
			expected: []string{"don", "'", "t"},
		},
		{
			name:     "quotes",
			input:    `"hello"`,
			expected: []string{`"`, "hello", `"`},
		},
		{
			name:     "underscore is punctuation",
			input:    "foo_bar",
			expected: []string{"foo", "_", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{UsePunctuation: true}
			result := Tokenize(tt.input, opts)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenize(%q, UsePunctuation=true) = %v, want %v",
					tt.input, result, tt.expected)
			}
		})
	}
}

// TestUsePunctuationVsDefault compares punctuation mode vs default (empty) delimiters
func TestUsePunctuationVsDefault(t *testing.T) {
	// Default delimiters are empty (matching original dwdiff behavior)
	// Only whitespace splits tokens by default

	input := "user@example"
	defaultResult := Tokenize(input, DefaultOptions())
	punctResult := Tokenize(input, Options{UsePunctuation: true})

	// With empty default delimiters, "user@example" is one token (no whitespace)
	if len(defaultResult) != 1 {
		t.Errorf("Default delimiters (empty): expected 1 token for %q, got %v", input, defaultResult)
	}

	// @ is Unicode punctuation (Po category), so should be split with -P flag
	if len(punctResult) != 3 {
		t.Errorf("Punctuation mode: expected 3 tokens for %q, got %v", input, punctResult)
	}
}

// TestTokenizeWithPositions tests position tracking during tokenization
func TestTokenizeWithPositions(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		opts       Options
		wantTokens []string
	}{
		{
			name:       "simple words",
			input:      "hello world",
			opts:       DefaultOptions(),
			wantTokens: []string{"hello", "world"},
		},
		{
			name:       "with delimiters",
			input:      "foo(bar)",
			opts:       Options{Delimiters: "()"},
			wantTokens: []string{"foo", "(", "bar", ")"},
		},
		{
			name:       "preserve whitespace",
			input:      "a b",
			opts:       Options{PreserveWhitespace: true},
			wantTokens: []string{"a", " ", "b"},
		},
		{
			name:       "multiline",
			input:      "line1\nline2",
			opts:       DefaultOptions(),
			wantTokens: []string{"line1", "line2"},
		},
		{
			name:       "multiline with preserve whitespace",
			input:      "line1\nline2",
			opts:       Options{PreserveWhitespace: true},
			wantTokens: []string{"line1", "\n", "line2"},
		},
		{
			name:       "empty string",
			input:      "",
			opts:       DefaultOptions(),
			wantTokens: nil,
		},
		{
			name:       "only whitespace",
			input:      "   ",
			opts:       DefaultOptions(),
			wantTokens: nil,
		},
		{
			name:       "unicode text",
			input:      "こんにちは 世界",
			opts:       DefaultOptions(),
			wantTokens: []string{"こんにちは", "世界"},
		},
		{
			name:       "with punctuation mode",
			input:      "hello, world!",
			opts:       Options{UsePunctuation: true},
			wantTokens: []string{"hello", ",", "world", "!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, positions := TokenizeWithPositions(tt.input, tt.opts)

			// Check tokens
			if !reflect.DeepEqual(tokens, tt.wantTokens) {
				t.Errorf("TokenizeWithPositions tokens = %v, want %v", tokens, tt.wantTokens)
			}

			// Check positions are valid
			if len(positions) != len(tokens) {
				t.Errorf("positions count = %d, tokens count = %d", len(positions), len(tokens))
			}

			// Verify each position maps correctly back to the input
			for i, pos := range positions {
				if pos.Start < 0 || pos.End > len(tt.input) || pos.Start > pos.End {
					t.Errorf("Invalid position[%d]: Start=%d, End=%d, input length=%d",
						i, pos.Start, pos.End, len(tt.input))
					continue
				}
				extracted := tt.input[pos.Start:pos.End]
				if extracted != tokens[i] {
					t.Errorf("Position[%d] extracts %q, but token is %q", i, extracted, tokens[i])
				}
			}
		})
	}
}

// Benchmark tokenization
func BenchmarkTokenize(b *testing.B) {
	text := "func processData(input []byte, config *Config) (Result, error) {"
	opts := DefaultOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Tokenize(text, opts)
	}
}
