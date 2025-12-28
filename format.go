package tokendiff

import (
	"fmt"
	"strings"
)

// FormatOptions configures diff output formatting.
type FormatOptions struct {
	// StartDelete is the string to mark the beginning of deleted text.
	// Default: "[-"
	StartDelete string

	// StopDelete is the string to mark the end of deleted text.
	// Default: "-]"
	StopDelete string

	// StartInsert is the string to mark the beginning of inserted text.
	// Default: "{+"
	StartInsert string

	// StopInsert is the string to mark the end of inserted text.
	// Default: "+}"
	StopInsert string

	// NoDeleted, when true, suppresses deleted tokens from output.
	NoDeleted bool

	// NoInserted, when true, suppresses inserted tokens from output.
	NoInserted bool

	// NoCommon, when true, suppresses unchanged tokens from output.
	NoCommon bool

	// UseColor enables ANSI color output. When true, DeleteColor and InsertColor
	// are used instead of text markers.
	UseColor bool

	// DeleteColor is the ANSI escape sequence for deleted text color.
	// Example: "\033[31m" for red
	DeleteColor string

	// InsertColor is the ANSI escape sequence for inserted text color.
	// Example: "\033[32m" for green
	InsertColor string

	// ColorReset is the ANSI escape sequence to reset colors.
	// Default: "\033[0m"
	ColorReset string

	// ClearToEOL is the ANSI escape sequence to clear to end of line.
	// Default: "\033[K"
	ClearToEOL string

	// RepeatMarkers, when true, repeats markers at line boundaries for
	// multi-line changes.
	RepeatMarkers bool

	// AggregateChanges, when true, combines adjacent changes of the same type.
	AggregateChanges bool

	// LessMode uses overstrike underlining for deleted text (for less -r).
	LessMode bool

	// PrinterMode uses overstrike bold for inserted text (for printing).
	PrinterMode bool

	// MatchContext is the minimum number of matching words between changes.
	// Equal tokens sandwiched between changes with fewer than this many
	// matches are converted to Delete+Insert pairs. 0 disables this feature.
	MatchContext int

	// ShowLineNumbers enables dual line number display (old:new format).
	ShowLineNumbers bool

	// LineNumWidth is the minimum width for line numbers. 0 means auto-calculate.
	LineNumWidth int

	// HeuristicSpacing uses NeedsSpaceBefore/After heuristics for spacing
	// when PreserveWhitespace is false. When true, spaces are not tokens and
	// spacing is determined heuristically.
	HeuristicSpacing bool
}

// ANSI escape code constants
const (
	ANSIReset       = "\033[0m"
	ANSIClearEOL    = "\033[K"
	ANSIDeleteColor = "\033[0;31;1m" // bold red
	ANSIInsertColor = "\033[0;32;1m" // bold green
	ANSIBold        = "\033[1m"
)

// ForegroundColors maps color names to ANSI foreground escape codes.
var ForegroundColors = map[string]string{
	"black":         "\033[30m",
	"red":           "\033[31m",
	"green":         "\033[32m",
	"yellow":        "\033[33m",
	"blue":          "\033[34m",
	"magenta":       "\033[35m",
	"cyan":          "\033[36m",
	"white":         "\033[37m",
	"brightblack":   "\033[90m",
	"brightred":     "\033[91m",
	"brightgreen":   "\033[92m",
	"brightyellow":  "\033[93m",
	"brightblue":    "\033[94m",
	"brightmagenta": "\033[95m",
	"brightcyan":    "\033[96m",
	"brightwhite":   "\033[97m",
}

// BackgroundColors maps color names to ANSI background escape codes.
var BackgroundColors = map[string]string{
	"black":         "\033[40m",
	"red":           "\033[41m",
	"green":         "\033[42m",
	"yellow":        "\033[43m",
	"blue":          "\033[44m",
	"magenta":       "\033[45m",
	"cyan":          "\033[46m",
	"white":         "\033[47m",
	"brightblack":   "\033[100m",
	"brightred":     "\033[101m",
	"brightgreen":   "\033[102m",
	"brightyellow":  "\033[103m",
	"brightblue":    "\033[104m",
	"brightmagenta": "\033[105m",
	"brightcyan":    "\033[106m",
	"brightwhite":   "\033[107m",
}

// ColorNames returns a list of all available color names.
func ColorNames() []string {
	return []string{
		"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
		"brightblack", "brightred", "brightgreen", "brightyellow",
		"brightblue", "brightmagenta", "brightcyan", "brightwhite",
	}
}

// ParseColor parses a color specification and returns the ANSI escape sequence.
// The spec can be:
//   - A single color name: "red" -> foreground red
//   - Foreground:background: "red:white" -> red text on white background
//   - Empty string returns empty string (no color)
//
// Returns an error if the color name is not recognized.
func ParseColor(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", nil
	}

	parts := strings.SplitN(spec, ":", 2)
	fgName := strings.ToLower(strings.TrimSpace(parts[0]))

	var result string

	if fgName != "" {
		fg, ok := ForegroundColors[fgName]
		if !ok {
			return "", fmt.Errorf("unknown color: %s", fgName)
		}
		result = fg
	}

	if len(parts) > 1 {
		bgName := strings.ToLower(strings.TrimSpace(parts[1]))
		if bgName != "" {
			bg, ok := BackgroundColors[bgName]
			if !ok {
				return "", fmt.Errorf("unknown background color: %s", bgName)
			}
			result += bg
		}
	}

	return result, nil
}

// ParseColorSpec parses a color specification for diff output.
// The format is: "delete_color,insert_color" where each color can be
// "fg" or "fg:bg" (e.g., "red,green" or "red:white,green:black").
//
// If only one color is specified, it's used for deletions and the default
// insert color (bold green) is used for insertions.
//
// Returns the ANSI escape sequences for delete and insert colors.
func ParseColorSpec(spec string) (deleteColor, insertColor string, err error) {
	parts := strings.SplitN(spec, ",", 2)

	deleteColor, err = ParseColor(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("delete color: %w", err)
	}

	if len(parts) > 1 {
		insertColor, err = ParseColor(parts[1])
		if err != nil {
			return "", "", fmt.Errorf("insert color: %w", err)
		}
	} else {
		insertColor = ANSIInsertColor
	}

	return deleteColor, insertColor, nil
}

// ColorCode builds an ANSI escape sequence from component parts.
// fg is the foreground color name (or empty for default).
// bg is the background color name (or empty for none).
// bold adds the bold attribute if true.
// Returns an error if any color name is not recognized.
func ColorCode(fg, bg string, bold bool) (string, error) {
	var result string

	if bold {
		result = ANSIBold
	}

	if fg != "" {
		fgCode, ok := ForegroundColors[strings.ToLower(fg)]
		if !ok {
			return "", fmt.Errorf("unknown foreground color: %s", fg)
		}
		result += fgCode
	}

	if bg != "" {
		bgCode, ok := BackgroundColors[strings.ToLower(bg)]
		if !ok {
			return "", fmt.Errorf("unknown background color: %s", bg)
		}
		result += bgCode
	}

	return result, nil
}

// DefaultFormatOptions returns FormatOptions with default settings.
func DefaultFormatOptions() FormatOptions {
	return FormatOptions{
		StartDelete:      "[-",
		StopDelete:       "-]",
		StartInsert:      "{+",
		StopInsert:       "+}",
		ColorReset:       ANSIReset,
		ClearToEOL:       ANSIClearEOL,
		DeleteColor:      ANSIDeleteColor,
		InsertColor:      ANSIInsertColor,
		AggregateChanges: true,
		HeuristicSpacing: true,
	}
}

// NeedsSpaceBefore returns true if a space should precede this token
// when formatting diff output. Used internally by FormatDiff.
func NeedsSpaceBefore(token string) bool {
	if len(token) == 0 {
		return false
	}
	// Don't add space before:
	// - closing delimiters: ) ] } >
	// - opening delimiters (in code, "foo(" not "foo (")
	// - punctuation: , . : ; ! ? / (slash for paths)
	// - other delimiters that typically attach to preceding text
	first := rune(token[0])
	return !strings.ContainsRune(")]}>,.:;!?\"'([{</\\", first)
}

// NeedsSpaceAfter returns true if a space should follow this token
// when formatting diff output. Used internally by FormatDiff.
func NeedsSpaceAfter(token string) bool {
	if len(token) == 0 {
		return false
	}
	// Don't add space after:
	// - opening delimiters: ( [ { <
	// - path separators and similar: / \ .
	// - tokens that typically attach to following text
	last := rune(token[len(token)-1])
	return !strings.ContainsRune("([{<\"'/\\.", last)
}

// FormatDiff returns a human-readable representation of the diff.
// Deleted tokens are wrapped in [-...-] and inserted tokens in {+...+}.
func FormatDiff(diffs []Diff) string {
	var sb strings.Builder

	for i, d := range diffs {
		// Add space between tokens, but not:
		// - before the first token
		// - before closing delimiters or punctuation
		// - after opening delimiters
		// - between adjacent delete/insert operations (they should be joined)
		if i > 0 {
			prev := diffs[i-1]
			// Don't add space between delete and insert (show them adjacent)
			adjacentChange := (prev.Type == Delete && d.Type == Insert) ||
				(prev.Type == Insert && d.Type == Delete)

			if !adjacentChange && NeedsSpaceBefore(d.Token) && NeedsSpaceAfter(prev.Token) {
				sb.WriteString(" ")
			}
		}

		switch d.Type {
		case Equal:
			sb.WriteString(d.Token)
		case Delete:
			sb.WriteString("[-")
			sb.WriteString(d.Token)
			sb.WriteString("-]")
		case Insert:
			sb.WriteString("{+")
			sb.WriteString(d.Token)
			sb.WriteString("+}")
		}
	}

	return sb.String()
}

// FormatDiffWithOptions returns a formatted representation of the diff
// using the specified formatting options.
func FormatDiffWithOptions(diffs []Diff, opts FormatOptions) string {
	// Apply defaults for empty markers
	if opts.StartDelete == "" && opts.StopDelete == "" {
		opts.StartDelete = "[-"
		opts.StopDelete = "-]"
	}
	if opts.StartInsert == "" && opts.StopInsert == "" {
		opts.StartInsert = "{+"
		opts.StopInsert = "+}"
	}

	// Helper to check if a diff would be suppressed
	isSuppressed := func(d Diff) bool {
		switch d.Type {
		case Equal:
			return opts.NoCommon
		case Delete:
			return opts.NoDeleted
		case Insert:
			return opts.NoInserted
		}
		return false
	}

	var sb strings.Builder
	var lastOutput *Diff // Track last actually output token for spacing

	for i := range diffs {
		d := diffs[i]
		if isSuppressed(d) {
			continue
		}

		// Add space between tokens where appropriate
		if lastOutput != nil {
			adjacentChange := (lastOutput.Type == Delete && d.Type == Insert) ||
				(lastOutput.Type == Insert && d.Type == Delete)
			if !adjacentChange && NeedsSpaceBefore(d.Token) && NeedsSpaceAfter(lastOutput.Token) {
				sb.WriteString(" ")
			}
		}

		switch d.Type {
		case Equal:
			sb.WriteString(d.Token)
		case Delete:
			sb.WriteString(opts.StartDelete)
			sb.WriteString(d.Token)
			sb.WriteString(opts.StopDelete)
		case Insert:
			sb.WriteString(opts.StartInsert)
			sb.WriteString(d.Token)
			sb.WriteString(opts.StopInsert)
		}

		lastOutput = &diffs[i]
	}

	return sb.String()
}

// OverstrikeUnderline returns text with overstrike underlining (_\bchar for each char).
// This is used for less -r mode to highlight deleted text.
func OverstrikeUnderline(text string) string {
	var sb strings.Builder
	for _, r := range text {
		sb.WriteRune('_')
		sb.WriteRune('\b')
		sb.WriteRune(r)
	}
	return sb.String()
}

// OverstrikeBold returns text with overstrike bold (char\bchar for each char).
// This is used for printer mode to highlight inserted text.
func OverstrikeBold(text string) string {
	var sb strings.Builder
	for _, r := range text {
		sb.WriteRune(r)
		sb.WriteRune('\b')
		sb.WriteRune(r)
	}
	return sb.String()
}

// FormatDiffsAdvanced formats diffs with comprehensive options including colors,
// line numbers, overstrike modes, and marker repetition.
// This is a more feature-rich alternative to FormatDiffWithOptions.
func FormatDiffsAdvanced(diffs []Diff, opts FormatOptions) string {
	// Set defaults for color reset/clear if not provided
	if opts.ColorReset == "" {
		opts.ColorReset = ANSIReset
	}
	if opts.ClearToEOL == "" {
		opts.ClearToEOL = ANSIClearEOL
	}

	// Apply match context first (before aggregation)
	if opts.MatchContext > 0 {
		diffs = ApplyMatchContext(diffs, opts.MatchContext)
	}

	// Apply aggregation if requested
	if opts.AggregateChanges {
		diffs = AggregateDiffs(diffs)
	}

	// Helper to format a single token with markers and colors
	formatToken := func(d Diff) string {
		switch d.Type {
		case Equal:
			if opts.NoCommon {
				return ""
			}
			return d.Token
		case Delete:
			if opts.NoDeleted {
				return ""
			}
			token := d.Token
			// Pure newline tokens don't get markers - they just affect line tracking
			if token == "\n" {
				return "\n"
			}
			// Overstrike modes: underline deleted text, no markers
			if opts.LessMode || opts.PrinterMode {
				return OverstrikeUnderline(token)
			}
			// With colors: no text markers (like original dwdiff)
			if opts.UseColor {
				// Handle repeat markers for multi-line changes
				if opts.RepeatMarkers && strings.Contains(token, "\n") {
					token = strings.ReplaceAll(token, "\n",
						opts.ColorReset+"\n"+opts.DeleteColor)
				}
				return opts.DeleteColor + token + opts.ColorReset
			}
			// Without colors: use text markers
			if opts.RepeatMarkers && strings.Contains(token, "\n") {
				token = strings.ReplaceAll(token, "\n",
					opts.StopDelete+"\n"+opts.StartDelete)
			}
			return opts.StartDelete + token + opts.StopDelete
		case Insert:
			if opts.NoInserted {
				return ""
			}
			token := d.Token
			// Pure newline tokens don't get markers - they just affect line tracking
			if token == "\n" {
				return "\n"
			}
			// Overstrike modes: bold inserted text, no markers
			if opts.LessMode || opts.PrinterMode {
				return OverstrikeBold(token)
			}
			// With colors: no text markers (like original dwdiff)
			if opts.UseColor {
				// Handle repeat markers for multi-line changes
				if opts.RepeatMarkers && strings.Contains(token, "\n") {
					token = strings.ReplaceAll(token, "\n",
						opts.ColorReset+"\n"+opts.InsertColor)
				}
				return opts.InsertColor + token + opts.ColorReset
			}
			// Without colors: use text markers
			if opts.RepeatMarkers && strings.Contains(token, "\n") {
				token = strings.ReplaceAll(token, "\n",
					opts.StopInsert+"\n"+opts.StartInsert)
			}
			return opts.StartInsert + token + opts.StopInsert
		}
		return ""
	}

	// Simple path: no line numbers
	if !opts.ShowLineNumbers {
		var sb strings.Builder
		var prevToken string
		var prevType Operation = -1
		for _, d := range diffs {
			if opts.HeuristicSpacing && prevToken != "" {
				needSpace := false
				if prevType == d.Type {
					// Same type: only add space for Delete/Insert (not Equal)
					if d.Type != Equal {
						needSpace = NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(d.Token)
					}
				} else {
					// Type transition: add space if both tokens support it
					needSpace = NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(d.Token)
				}
				if needSpace {
					sb.WriteString(" ")
				}
			}
			sb.WriteString(formatToken(d))
			prevToken = d.Token
			prevType = d.Type
		}
		return sb.String()
	}

	// Line numbers path: track old/new line positions
	var lines []string
	var currentLine strings.Builder
	oldLine := 1
	newLine := 1
	width := opts.LineNumWidth

	linePrefix := func() string {
		// Old line: width + 1 (right-aligned with one leading space for largest number)
		// New line: width + 2 (left-aligned with trailing space plus breathing room)
		oldWidth := width + 1
		newWidth := width + 2
		return fmt.Sprintf("%*d:%-*d", oldWidth, oldLine, newWidth, newLine)
	}

	currentLine.WriteString(linePrefix())

	var prevToken string
	var prevType Operation = -1
	for _, d := range diffs {
		if opts.HeuristicSpacing && prevToken != "" {
			needSpace := false
			if prevType == d.Type {
				// Same type: only add space for Delete/Insert (not Equal)
				if d.Type != Equal {
					needSpace = NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(d.Token)
				}
			} else {
				// Type transition: add space if both tokens support it
				needSpace = NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(d.Token)
			}
			if needSpace {
				currentLine.WriteString(" ")
			}
		}
		formatted := formatToken(d)

		// Handle newlines within the formatted token
		if strings.Contains(formatted, "\n") {
			parts := strings.Split(formatted, "\n")
			for j, part := range parts {
				currentLine.WriteString(part)
				if j < len(parts)-1 {
					lines = append(lines, currentLine.String())
					currentLine.Reset()

					switch d.Type {
					case Equal:
						oldLine++
						newLine++
					case Delete:
						oldLine++
					case Insert:
						newLine++
					}

					currentLine.WriteString(linePrefix())
				}
			}
		} else {
			currentLine.WriteString(formatted)
		}
		prevToken = d.Token
		prevType = d.Type
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

// FormatDiffResultAdvanced formats a DiffResult preserving original spacing for Equal content.
// This uses position information to extract original text for Equal runs instead of
// reconstructing from tokens, which loses whitespace information.
// When opts.ShowLineNumbers is true, it tracks and displays line numbers based on
// SOURCE positions in the original texts.
func FormatDiffResultAdvanced(result DiffResult, opts FormatOptions) string {
	diffs := result.Diffs

	// Set defaults for color reset/clear if not provided
	if opts.ColorReset == "" {
		opts.ColorReset = ANSIReset
	}
	if opts.ClearToEOL == "" {
		opts.ClearToEOL = ANSIClearEOL
	}

	// When showing line numbers with colors, we need repeat-markers behavior
	// to properly color each line of multi-line changes.
	if opts.ShowLineNumbers && opts.UseColor {
		opts.RepeatMarkers = true
	}

	// Apply match context first (before aggregation)
	if opts.MatchContext > 0 {
		diffs = ApplyMatchContext(diffs, opts.MatchContext)
	}

	// Track current position in each token list
	var idx1, idx2 int

	// Track the last position in text2 that we've output
	lastText2Pos := 0

	// Line number tracking
	oldLine := 1
	newLine := 1
	width := opts.LineNumWidth

	// Helper to generate line number prefix
	linePrefix := func() string {
		if !opts.ShowLineNumbers {
			return ""
		}
		oldWidth := width + 1
		newWidth := width + 2
		return fmt.Sprintf("%*d:%-*d", oldWidth, oldLine, newWidth, newLine)
	}

	// Helper to format non-Equal tokens with markers and colors
	// When UseColor is true, we use colors only (no text markers) - matching dwdiff behavior
	// When ShowLineNumbers && UseColor, writeContent handles colors, so return plain token
	formatNonEqualToken := func(d Diff) string {
		switch d.Type {
		case Delete:
			if opts.NoDeleted {
				return ""
			}
			token := d.Token
			if token == "\n" {
				return "\n"
			}
			if opts.LessMode || opts.PrinterMode {
				return OverstrikeUnderline(token)
			}
			// When using colors with line numbers, writeContent handles the coloring
			// so just return the plain token
			if opts.ShowLineNumbers && opts.UseColor {
				return token
			}
			// When using colors without line numbers, add color codes directly
			if opts.UseColor {
				if opts.RepeatMarkers && strings.Contains(token, "\n") {
					token = strings.ReplaceAll(token, "\n",
						opts.ColorReset+"\n"+opts.DeleteColor)
				}
				return opts.DeleteColor + token + opts.ColorReset
			}
			// No colors - use text markers
			if opts.RepeatMarkers && strings.Contains(token, "\n") {
				token = strings.ReplaceAll(token, "\n",
					opts.StopDelete+"\n"+opts.StartDelete)
			}
			return opts.StartDelete + token + opts.StopDelete
		case Insert:
			if opts.NoInserted {
				return ""
			}
			token := d.Token
			if token == "\n" {
				return "\n"
			}
			if opts.LessMode || opts.PrinterMode {
				return OverstrikeBold(token)
			}
			// When using colors with line numbers, writeContent handles the coloring
			// so just return the plain token
			if opts.ShowLineNumbers && opts.UseColor {
				return token
			}
			// When using colors without line numbers, add color codes directly
			if opts.UseColor {
				if opts.RepeatMarkers && strings.Contains(token, "\n") {
					token = strings.ReplaceAll(token, "\n",
						opts.ClearToEOL+opts.ColorReset+"\n"+opts.InsertColor)
				}
				return opts.InsertColor + token + opts.ColorReset
			}
			// No colors - use text markers
			if opts.RepeatMarkers && strings.Contains(token, "\n") {
				token = strings.ReplaceAll(token, "\n",
					opts.StopInsert+"\n"+opts.StartInsert)
			}
			return opts.StartInsert + token + opts.StopInsert
		}
		return ""
	}

	// Helper to write content with line number tracking
	var lines []string
	var currentLine strings.Builder
	var colorState Operation = -1
	var prevLineEndedColored bool

	writeContent := func(content string, diffType Operation) {
		if opts.ShowLineNumbers && opts.UseColor {
			if diffType == Equal && colorState != -1 {
				currentLine.WriteString(opts.ColorReset)
				colorState = -1
			} else if diffType != Equal && colorState != diffType {
				if colorState != -1 {
					currentLine.WriteString(opts.ColorReset)
				}
				colorState = diffType
				if diffType == Delete {
					currentLine.WriteString(opts.DeleteColor)
				} else {
					currentLine.WriteString(opts.InsertColor)
				}
			}
		}

		for _, r := range content {
			if r == '\n' {
				thisLineEndedColored := false
				if opts.ShowLineNumbers && opts.UseColor && colorState != -1 {
					currentLine.WriteString(opts.ClearToEOL)
					thisLineEndedColored = true
				}

				prefix := linePrefix()
				if prevLineEndedColored {
					prefix = opts.ColorReset + prefix
				}
				lines = append(lines, prefix+currentLine.String())
				currentLine.Reset()

				prevLineEndedColored = thisLineEndedColored

				if colorState != -1 {
					if colorState == Delete {
						currentLine.WriteString(opts.DeleteColor)
					} else {
						currentLine.WriteString(opts.InsertColor)
					}
				}

				switch diffType {
				case Equal:
					oldLine++
					newLine++
				case Delete:
					oldLine++
				case Insert:
					newLine++
				}
			} else {
				currentLine.WriteRune(r)
			}
		}
	}

	// Process diffs
	i := 0
	for i < len(diffs) {
		d := diffs[i]

		switch d.Type {
		case Equal:
			if opts.NoCommon {
				for _, r := range d.Token {
					if r == '\n' {
						if opts.ShowLineNumbers {
							lines = append(lines, linePrefix()+currentLine.String())
							currentLine.Reset()
						}
						oldLine++
						newLine++
					}
				}
				idx1++
				idx2++
				i++
				continue
			}

			// Find the run of consecutive Equal tokens
			runStart := i
			for i < len(diffs) && diffs[i].Type == Equal {
				i++
			}
			runEnd := i
			runTokenCount := runEnd - runStart

			// Extract original text from text2 using positions
			if idx2 < len(result.Positions2) && idx2+runTokenCount-1 < len(result.Positions2) {
				startPos := result.Positions2[idx2].Start
				if idx2 == 0 && currentLine.Len() == 0 && startPos > 0 {
					startPos = 0
				}
				endPos := result.Positions2[idx2+runTokenCount-1].End

				if lastText2Pos > 0 && startPos > lastText2Pos {
					gap := result.Text2[lastText2Pos:startPos]
					writeContent(gap, Equal)
				}

				original := result.Text2[startPos:endPos]
				writeContent(original, Equal)
				lastText2Pos = endPos
			} else {
				for j := runStart; j < runEnd; j++ {
					if j > runStart {
						if NeedsSpaceAfter(diffs[j-1].Token) && NeedsSpaceBefore(diffs[j].Token) {
							writeContent(" ", Equal)
						}
					} else if currentLine.Len() > 0 && runStart > 0 {
						prev := diffs[runStart-1]
						if NeedsSpaceAfter(prev.Token) && NeedsSpaceBefore(diffs[j].Token) {
							writeContent(" ", Equal)
						}
					}
					writeContent(diffs[j].Token, Equal)
				}
			}
			idx1 += runTokenCount
			idx2 += runTokenCount

		case Delete:
			// Find consecutive Delete tokens
			runStart := i
			for i < len(diffs) && diffs[i].Type == Delete {
				i++
			}
			runEnd := i

			if currentLine.Len() > 0 && runStart > 0 {
				prev := diffs[runStart-1]
				adjacentChange := prev.Type == Insert
				if !adjacentChange {
					needSpace := false
					if idx1 < len(result.Positions1) && idx1 > 0 {
						prevEnd := result.Positions1[idx1-1].End
						currStart := result.Positions1[idx1].Start
						if currStart > prevEnd {
							gap := result.Text1[prevEnd:currStart]
							needSpace = strings.ContainsAny(gap, " \t\n\r")
						}
					} else {
						needSpace = NeedsSpaceAfter(prev.Token) && NeedsSpaceBefore(diffs[runStart].Token)
					}
					if needSpace {
						writeContent(" ", Equal)
					}
				}
			}

			// Extract original text from text1
			if idx1 < len(result.Positions1) && idx1+runEnd-runStart-1 < len(result.Positions1) {
				startPos := result.Positions1[idx1].Start
				endPos := result.Positions1[idx1+runEnd-runStart-1].End
				original := result.Text1[startPos:endPos]
				formatted := formatNonEqualToken(Diff{Type: Delete, Token: original})
				writeContent(formatted, Delete)
			} else {
				for j := runStart; j < runEnd; j++ {
					if j > runStart && NeedsSpaceAfter(diffs[j-1].Token) && NeedsSpaceBefore(diffs[j].Token) {
						writeContent(" ", Delete)
					}
					formatted := formatNonEqualToken(diffs[j])
					writeContent(formatted, Delete)
				}
			}
			idx1 += runEnd - runStart

		case Insert:
			// Find consecutive Insert tokens
			runStart := i
			for i < len(diffs) && diffs[i].Type == Insert {
				i++
			}
			runEnd := i

			// Include gap before Insert
			if idx2 < len(result.Positions2) {
				insStart := result.Positions2[idx2].Start
				if lastText2Pos > 0 && insStart > lastText2Pos {
					gap := result.Text2[lastText2Pos:insStart]
					for _, r := range gap {
						if r == '\n' {
							if opts.ShowLineNumbers {
								thisLineEndedColored := false
								if opts.UseColor && colorState != -1 {
									currentLine.WriteString(opts.ClearToEOL)
									thisLineEndedColored = true
								}
								prefix := linePrefix()
								if prevLineEndedColored {
									prefix = opts.ColorReset + prefix
								}
								lines = append(lines, prefix+currentLine.String())
								currentLine.Reset()
								prevLineEndedColored = thisLineEndedColored
								if colorState != -1 {
									if colorState == Delete {
										currentLine.WriteString(opts.DeleteColor)
									} else {
										currentLine.WriteString(opts.InsertColor)
									}
								}
							} else {
								currentLine.WriteRune('\n')
							}
							newLine++
						} else {
							if opts.ShowLineNumbers && opts.UseColor && colorState != -1 {
								currentLine.WriteString(opts.ColorReset)
								colorState = -1
							}
							currentLine.WriteRune(r)
						}
					}
				}
			}

			// Extract original text from text2
			if idx2 < len(result.Positions2) && idx2+runEnd-runStart-1 < len(result.Positions2) {
				startPos := result.Positions2[idx2].Start
				endPos := result.Positions2[idx2+runEnd-runStart-1].End
				original := result.Text2[startPos:endPos]
				formatted := formatNonEqualToken(Diff{Type: Insert, Token: original})
				writeContent(formatted, Insert)
				lastText2Pos = endPos
			} else {
				for j := runStart; j < runEnd; j++ {
					if j > runStart && NeedsSpaceAfter(diffs[j-1].Token) && NeedsSpaceBefore(diffs[j].Token) {
						writeContent(" ", Insert)
					}
					formatted := formatNonEqualToken(diffs[j])
					writeContent(formatted, Insert)
				}
			}
			idx2 += runEnd - runStart
		}
	}

	// Reset color at end if still active
	if opts.UseColor && colorState != -1 {
		currentLine.WriteString(opts.ColorReset)
	}

	// Output final line
	if opts.ShowLineNumbers {
		if currentLine.Len() > 0 || len(lines) == 0 {
			lines = append(lines, linePrefix()+currentLine.String())
		}
		return strings.Join(lines, "\n")
	}

	if len(lines) > 0 {
		lines = append(lines, currentLine.String())
		return strings.Join(lines, "\n")
	}
	return currentLine.String()
}
