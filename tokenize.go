package tokendiff

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenPos represents a token's position in the original text.
type TokenPos struct {
	Start int // byte offset of token start
	End   int // byte offset of token end (exclusive)
}

// TokenizeWithPositions splits text into tokens and tracks their positions.
// This allows reconstructing original spacing for Equal content in diffs.
func TokenizeWithPositions(text string, opts Options) ([]string, []TokenPos) {
	// Determine delimiter check function
	var isDelimiter func(r rune) bool

	if opts.UsePunctuation {
		isDelimiter = unicode.IsPunct
	} else {
		if opts.Delimiters == "" {
			opts.Delimiters = DefaultDelimiters
		}
		delimSet := make(map[rune]bool)
		for _, r := range opts.Delimiters {
			delimSet[r] = true
		}
		isDelimiter = func(r rune) bool {
			return delimSet[r]
		}
	}

	// Determine whitespace check function
	var isWS func(r rune) bool
	if opts.Whitespace == "" {
		isWS = isWhitespace
	} else {
		wsSet := make(map[rune]bool)
		for _, r := range opts.Whitespace {
			wsSet[r] = true
		}
		isWS = func(r rune) bool {
			return wsSet[r]
		}
	}

	var tokens []string
	var positions []TokenPos
	var currentWord strings.Builder
	var wordStart int = -1

	flushWord := func(pos int) {
		if currentWord.Len() > 0 {
			tokens = append(tokens, currentWord.String())
			positions = append(positions, TokenPos{Start: wordStart, End: pos})
			currentWord.Reset()
			wordStart = -1
		}
	}

	i := 0
	for _, r := range text {
		runeLen := utf8.RuneLen(r)
		switch {
		case isDelimiter(r):
			flushWord(i)
			tokens = append(tokens, string(r))
			positions = append(positions, TokenPos{Start: i, End: i + runeLen})

		case isWS(r):
			flushWord(i)
			if opts.PreserveWhitespace {
				tokens = append(tokens, string(r))
				positions = append(positions, TokenPos{Start: i, End: i + runeLen})
			}

		default:
			if wordStart == -1 {
				wordStart = i
			}
			currentWord.WriteRune(r)
		}
		i += runeLen
	}

	flushWord(i)
	return tokens, positions
}

// Tokenize splits text into tokens, treating delimiters as separate tokens.
// Whitespace separates words but is not included in output unless
// PreserveWhitespace is true.
func Tokenize(text string, opts Options) []string {
	// Determine delimiter check function
	var isDelimiter func(r rune) bool

	if opts.UsePunctuation {
		// Use Unicode punctuation category
		isDelimiter = unicode.IsPunct
	} else {
		// Use explicit delimiter set
		if opts.Delimiters == "" {
			opts.Delimiters = DefaultDelimiters
		}
		delimSet := make(map[rune]bool)
		for _, r := range opts.Delimiters {
			delimSet[r] = true
		}
		isDelimiter = func(r rune) bool {
			return delimSet[r]
		}
	}

	// Determine whitespace check function
	var isWS func(r rune) bool
	if opts.Whitespace == "" {
		isWS = isWhitespace
	} else {
		wsSet := make(map[rune]bool)
		for _, r := range opts.Whitespace {
			wsSet[r] = true
		}
		isWS = func(r rune) bool {
			return wsSet[r]
		}
	}

	var tokens []string
	var currentWord strings.Builder

	flushWord := func() {
		if currentWord.Len() > 0 {
			tokens = append(tokens, currentWord.String())
			currentWord.Reset()
		}
	}

	for _, r := range text {
		switch {
		case isDelimiter(r):
			// Delimiter: flush current word, add delimiter as its own token
			flushWord()
			tokens = append(tokens, string(r))

		case isWS(r):
			// Whitespace: flush current word
			flushWord()
			if opts.PreserveWhitespace {
				tokens = append(tokens, string(r))
			}

		default:
			// Regular character: add to current word
			currentWord.WriteRune(r)
		}
	}

	flushWord()
	return tokens
}

// isWhitespace returns true if r is a whitespace character.
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
