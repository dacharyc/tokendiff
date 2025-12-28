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

// formatDeleteToken formats a Delete token with appropriate markers/colors.
func formatDeleteToken(token string, opts FormatOptions) string {
	if opts.NoDeleted || token == "\n" {
		if opts.NoDeleted {
			return ""
		}
		return "\n"
	}
	if opts.LessMode || opts.PrinterMode {
		return OverstrikeUnderline(token)
	}
	if opts.ShowLineNumbers && opts.UseColor {
		return token
	}
	if opts.UseColor {
		if opts.RepeatMarkers && strings.Contains(token, "\n") {
			token = strings.ReplaceAll(token, "\n", opts.ColorReset+"\n"+opts.DeleteColor)
		}
		return opts.DeleteColor + token + opts.ColorReset
	}
	if opts.RepeatMarkers && strings.Contains(token, "\n") {
		token = strings.ReplaceAll(token, "\n", opts.StopDelete+"\n"+opts.StartDelete)
	}
	return opts.StartDelete + token + opts.StopDelete
}

// formatInsertToken formats an Insert token with appropriate markers/colors.
func formatInsertToken(token string, opts FormatOptions) string {
	if opts.NoInserted || token == "\n" {
		if opts.NoInserted {
			return ""
		}
		return "\n"
	}
	if opts.LessMode || opts.PrinterMode {
		return OverstrikeBold(token)
	}
	if opts.ShowLineNumbers && opts.UseColor {
		return token
	}
	if opts.UseColor {
		if opts.RepeatMarkers && strings.Contains(token, "\n") {
			token = strings.ReplaceAll(token, "\n", opts.ClearToEOL+opts.ColorReset+"\n"+opts.InsertColor)
		}
		return opts.InsertColor + token + opts.ColorReset
	}
	if opts.RepeatMarkers && strings.Contains(token, "\n") {
		token = strings.ReplaceAll(token, "\n", opts.StopInsert+"\n"+opts.StartInsert)
	}
	return opts.StartInsert + token + opts.StopInsert
}

// formatNonEqualToken formats a non-Equal token with markers and colors.
func formatNonEqualToken(d Diff, opts FormatOptions) string {
	switch d.Type {
	case Delete:
		return formatDeleteToken(d.Token, opts)
	case Insert:
		return formatInsertToken(d.Token, opts)
	}
	return ""
}

// diffFormatter holds state for formatting a DiffResult with line numbers and colors.
type diffFormatter struct {
	opts               FormatOptions
	result             DiffResult
	lines              []string
	currentLine        strings.Builder
	colorState         Operation
	prevLineEndedColor bool
	oldLine            int
	newLine            int
	lastText1Pos       int
	lastText2Pos       int
	idx1               int
	idx2               int
}

// newDiffFormatter creates a new formatter for the given result and options.
func newDiffFormatter(result DiffResult, opts FormatOptions) *diffFormatter {
	return &diffFormatter{
		opts:       opts,
		result:     result,
		colorState: -1,
		oldLine:    1,
		newLine:    1,
	}
}

// linePrefix returns the line number prefix for the current position.
func (f *diffFormatter) linePrefix() string {
	if !f.opts.ShowLineNumbers {
		return ""
	}
	oldWidth := f.opts.LineNumWidth + 1
	newWidth := f.opts.LineNumWidth + 2
	return fmt.Sprintf("%*d:%-*d", oldWidth, f.oldLine, newWidth, f.newLine)
}

// writeContent writes content with line number tracking and color state management.
func (f *diffFormatter) writeContent(content string, diffType Operation) {
	if f.opts.ShowLineNumbers && f.opts.UseColor {
		if diffType == Equal && f.colorState != -1 {
			f.currentLine.WriteString(f.opts.ColorReset)
			f.colorState = -1
		} else if diffType != Equal && f.colorState != diffType {
			if f.colorState != -1 {
				f.currentLine.WriteString(f.opts.ColorReset)
			}
			f.colorState = diffType
			if diffType == Delete {
				f.currentLine.WriteString(f.opts.DeleteColor)
			} else {
				f.currentLine.WriteString(f.opts.InsertColor)
			}
		}
	}

	for _, r := range content {
		if r == '\n' {
			f.flushLine(diffType)
		} else {
			f.currentLine.WriteRune(r)
		}
	}
}

// flushLine finishes the current line and advances line numbers.
func (f *diffFormatter) flushLine(diffType Operation) {
	thisLineEndedColored := false
	if f.opts.ShowLineNumbers && f.opts.UseColor && f.colorState != -1 {
		f.currentLine.WriteString(f.opts.ClearToEOL)
		thisLineEndedColored = true
	}

	prefix := f.linePrefix()
	if f.prevLineEndedColor {
		prefix = f.opts.ColorReset + prefix
	}
	f.lines = append(f.lines, prefix+f.currentLine.String())
	f.currentLine.Reset()

	f.prevLineEndedColor = thisLineEndedColored

	if f.colorState != -1 {
		if f.colorState == Delete {
			f.currentLine.WriteString(f.opts.DeleteColor)
		} else {
			f.currentLine.WriteString(f.opts.InsertColor)
		}
	}

	switch diffType {
	case Equal:
		f.oldLine++
		f.newLine++
	case Delete:
		f.oldLine++
	case Insert:
		f.newLine++
	}
}

// skipCommonNewlines handles NoCommon mode by counting newlines without output.
func (f *diffFormatter) skipCommonNewlines(diffs []Diff, runStart, runEnd int) {
	for j := runStart; j < runEnd; j++ {
		for _, r := range diffs[j].Token {
			if r == '\n' {
				if f.opts.ShowLineNumbers {
					f.lines = append(f.lines, f.linePrefix()+f.currentLine.String())
					f.currentLine.Reset()
				}
				f.oldLine++
				f.newLine++
			}
		}
	}
}

// writeEqualFromPositions writes Equal content using position information.
func (f *diffFormatter) writeEqualFromPositions(runTokenCount int) {
	startPos := f.result.Positions2[f.idx2].Start
	if f.idx2 == 0 && f.currentLine.Len() == 0 && startPos > 0 {
		startPos = 0
	}
	endPos := f.result.Positions2[f.idx2+runTokenCount-1].End

	if f.lastText2Pos > 0 && startPos > f.lastText2Pos {
		gap := f.result.Text2[f.lastText2Pos:startPos]
		f.writeContent(gap, Equal)
	}

	f.writeContent(f.result.Text2[startPos:endPos], Equal)
	f.lastText2Pos = endPos
}

// writeEqualTokensFallback writes Equal tokens with heuristic spacing.
func (f *diffFormatter) writeEqualTokensFallback(diffs []Diff, runStart, runEnd int) {
	for j := runStart; j < runEnd; j++ {
		if j > runStart && NeedsSpaceAfter(diffs[j-1].Token) && NeedsSpaceBefore(diffs[j].Token) {
			f.writeContent(" ", Equal)
		} else if j == runStart && f.currentLine.Len() > 0 && runStart > 0 {
			prev := diffs[runStart-1]
			if NeedsSpaceAfter(prev.Token) && NeedsSpaceBefore(diffs[j].Token) {
				f.writeContent(" ", Equal)
			}
		}
		f.writeContent(diffs[j].Token, Equal)
	}
}

// processEqualRun handles a run of consecutive Equal diffs.
func (f *diffFormatter) processEqualRun(diffs []Diff, runStart, runEnd int) {
	if f.opts.NoCommon {
		f.skipCommonNewlines(diffs, runStart, runEnd)
		return
	}

	runTokenCount := runEnd - runStart
	hasPositions := f.idx2 < len(f.result.Positions2) && f.idx2+runTokenCount-1 < len(f.result.Positions2)

	if hasPositions {
		f.writeEqualFromPositions(runTokenCount)
	} else {
		f.writeEqualTokensFallback(diffs, runStart, runEnd)
	}
}

// processDeleteGap handles the gap before a Delete run.
func (f *diffFormatter) processDeleteGap() {
	if f.idx1 >= len(f.result.Positions1) {
		return
	}
	delStart := f.result.Positions1[f.idx1].Start

	// Determine the starting position for gap extraction
	gapStart := f.lastText1Pos

	// Special case: if this is the first text1 content and there's leading whitespace,
	// start from position 0 to capture it (mirrors Equal's handling)
	if f.lastText1Pos <= 0 && f.currentLine.Len() == 0 && delStart > 0 {
		gapStart = 0
	}

	// Skip if no gap to process
	if delStart <= gapStart {
		return
	}

	gap := f.result.Text1[gapStart:delStart]
	for _, r := range gap {
		if r == '\n' {
			if f.opts.ShowLineNumbers {
				thisLineEndedColored := false
				if f.opts.UseColor && f.colorState != -1 {
					f.currentLine.WriteString(f.opts.ClearToEOL)
					thisLineEndedColored = true
				}
				prefix := f.linePrefix()
				if f.prevLineEndedColor {
					prefix = f.opts.ColorReset + prefix
				}
				f.lines = append(f.lines, prefix+f.currentLine.String())
				f.currentLine.Reset()
				f.prevLineEndedColor = thisLineEndedColored
				if f.colorState != -1 {
					if f.colorState == Delete {
						f.currentLine.WriteString(f.opts.DeleteColor)
					} else {
						f.currentLine.WriteString(f.opts.InsertColor)
					}
				}
			} else {
				f.currentLine.WriteRune('\n')
			}
			f.oldLine++
		} else {
			if f.opts.ShowLineNumbers && f.opts.UseColor && f.colorState != -1 {
				f.currentLine.WriteString(f.opts.ColorReset)
				f.colorState = -1
			}
			f.currentLine.WriteRune(r)
		}
	}
}

// processDeleteRun handles a run of consecutive Delete diffs.
func (f *diffFormatter) processDeleteRun(diffs []Diff, runStart, runEnd int) {
	f.processDeleteGap()

	// Extract original text from text1
	runLen := runEnd - runStart
	if f.idx1 < len(f.result.Positions1) && f.idx1+runLen-1 < len(f.result.Positions1) {
		startPos := f.result.Positions1[f.idx1].Start
		endPos := f.result.Positions1[f.idx1+runLen-1].End
		original := f.result.Text1[startPos:endPos]
		formatted := formatNonEqualToken(Diff{Type: Delete, Token: original}, f.opts)
		f.writeContent(formatted, Delete)
		f.lastText1Pos = endPos
	} else {
		for j := runStart; j < runEnd; j++ {
			if j > runStart && NeedsSpaceAfter(diffs[j-1].Token) && NeedsSpaceBefore(diffs[j].Token) {
				f.writeContent(" ", Delete)
			}
			formatted := formatNonEqualToken(diffs[j], f.opts)
			f.writeContent(formatted, Delete)
		}
	}
}

// processInsertGap handles the gap before an Insert run.
func (f *diffFormatter) processInsertGap() {
	if f.idx2 >= len(f.result.Positions2) {
		return
	}
	insStart := f.result.Positions2[f.idx2].Start

	// Determine the starting position for gap extraction
	gapStart := f.lastText2Pos

	// Special case: if this is the first text2 content and there's leading whitespace,
	// start from position 0 to capture it (mirrors Equal's handling in writeEqualFromPositions)
	if f.lastText2Pos <= 0 && f.currentLine.Len() == 0 && insStart > 0 {
		gapStart = 0
	}

	// Skip if no gap to process
	if insStart <= gapStart {
		return
	}

	gap := f.result.Text2[gapStart:insStart]
	for _, r := range gap {
		if r == '\n' {
			if f.opts.ShowLineNumbers {
				thisLineEndedColored := false
				if f.opts.UseColor && f.colorState != -1 {
					f.currentLine.WriteString(f.opts.ClearToEOL)
					thisLineEndedColored = true
				}
				prefix := f.linePrefix()
				if f.prevLineEndedColor {
					prefix = f.opts.ColorReset + prefix
				}
				f.lines = append(f.lines, prefix+f.currentLine.String())
				f.currentLine.Reset()
				f.prevLineEndedColor = thisLineEndedColored
				if f.colorState != -1 {
					if f.colorState == Delete {
						f.currentLine.WriteString(f.opts.DeleteColor)
					} else {
						f.currentLine.WriteString(f.opts.InsertColor)
					}
				}
			} else {
				f.currentLine.WriteRune('\n')
			}
			f.newLine++
		} else {
			if f.opts.ShowLineNumbers && f.opts.UseColor && f.colorState != -1 {
				f.currentLine.WriteString(f.opts.ColorReset)
				f.colorState = -1
			}
			f.currentLine.WriteRune(r)
		}
	}
}

// processInsertRun handles a run of consecutive Insert diffs.
func (f *diffFormatter) processInsertRun(diffs []Diff, runStart, runEnd int) {
	f.processInsertGap()

	// Extract original text from text2
	runLen := runEnd - runStart
	if f.idx2 < len(f.result.Positions2) && f.idx2+runLen-1 < len(f.result.Positions2) {
		startPos := f.result.Positions2[f.idx2].Start
		endPos := f.result.Positions2[f.idx2+runLen-1].End
		original := f.result.Text2[startPos:endPos]
		formatted := formatNonEqualToken(Diff{Type: Insert, Token: original}, f.opts)
		f.writeContent(formatted, Insert)
		f.lastText2Pos = endPos
	} else {
		for j := runStart; j < runEnd; j++ {
			if j > runStart && NeedsSpaceAfter(diffs[j-1].Token) && NeedsSpaceBefore(diffs[j].Token) {
				f.writeContent(" ", Insert)
			}
			formatted := formatNonEqualToken(diffs[j], f.opts)
			f.writeContent(formatted, Insert)
		}
	}
}

// finalize completes the formatting and returns the result string.
func (f *diffFormatter) finalize() string {
	// Reset color at end if still active
	if f.opts.UseColor && f.colorState != -1 {
		f.currentLine.WriteString(f.opts.ColorReset)
	}

	// Output final line
	if f.opts.ShowLineNumbers {
		if f.currentLine.Len() > 0 || len(f.lines) == 0 {
			f.lines = append(f.lines, f.linePrefix()+f.currentLine.String())
		}
		return strings.Join(f.lines, "\n")
	}

	if len(f.lines) > 0 {
		f.lines = append(f.lines, f.currentLine.String())
		return strings.Join(f.lines, "\n")
	}
	return f.currentLine.String()
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

// formatToken formats a single diff token with markers and colors.
func formatToken(d Diff, opts FormatOptions) string {
	switch d.Type {
	case Equal:
		if opts.NoCommon {
			return ""
		}
		return d.Token
	case Delete, Insert:
		return formatNonEqualToken(d, opts)
	}
	return ""
}

// needsHeuristicSpace determines if a space should be inserted between tokens.
func needsHeuristicSpace(prevToken string, prevType Operation, d Diff) bool {
	if prevToken == "" {
		return false
	}
	if prevType == d.Type {
		// Same type: only add space for Delete/Insert (not Equal)
		if d.Type != Equal {
			return NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(d.Token)
		}
		return false
	}
	// Type transition: add space if both tokens support it
	return NeedsSpaceAfter(prevToken) && NeedsSpaceBefore(d.Token)
}

// formatDiffsSimple formats diffs without line numbers.
func formatDiffsSimple(diffs []Diff, opts FormatOptions) string {
	var sb strings.Builder
	var prevToken string
	var prevType Operation = -1

	for _, d := range diffs {
		if opts.HeuristicSpacing && needsHeuristicSpace(prevToken, prevType, d) {
			sb.WriteString(" ")
		}
		sb.WriteString(formatToken(d, opts))
		prevToken = d.Token
		prevType = d.Type
	}
	return sb.String()
}

// formatDiffsWithLineNumbers formats diffs with line number tracking.
func formatDiffsWithLineNumbers(diffs []Diff, opts FormatOptions) string {
	var lines []string
	var currentLine strings.Builder
	oldLine := 1
	newLine := 1
	width := opts.LineNumWidth

	linePrefix := func() string {
		oldWidth := width + 1
		newWidth := width + 2
		return fmt.Sprintf("%*d:%-*d", oldWidth, oldLine, newWidth, newLine)
	}

	currentLine.WriteString(linePrefix())

	var prevToken string
	var prevType Operation = -1

	for _, d := range diffs {
		if opts.HeuristicSpacing && needsHeuristicSpace(prevToken, prevType, d) {
			currentLine.WriteString(" ")
		}
		formatted := formatToken(d, opts)

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

	if opts.ShowLineNumbers {
		return formatDiffsWithLineNumbers(diffs, opts)
	}
	return formatDiffsSimple(diffs, opts)
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

	// Create formatter and process diffs
	f := newDiffFormatter(result, opts)

	i := 0
	for i < len(diffs) {
		d := diffs[i]

		switch d.Type {
		case Equal:
			// Find the run of consecutive Equal tokens
			runStart := i
			for i < len(diffs) && diffs[i].Type == Equal {
				i++
			}
			f.processEqualRun(diffs, runStart, i)
			f.idx1 += i - runStart
			f.idx2 += i - runStart

		case Delete:
			// Find consecutive Delete tokens
			runStart := i
			for i < len(diffs) && diffs[i].Type == Delete {
				i++
			}
			f.processDeleteRun(diffs, runStart, i)
			f.idx1 += i - runStart

		case Insert:
			// Find consecutive Insert tokens
			runStart := i
			for i < len(diffs) && diffs[i].Type == Insert {
				i++
			}
			f.processInsertRun(diffs, runStart, i)
			f.idx2 += i - runStart
		}
	}

	return f.finalize()
}
