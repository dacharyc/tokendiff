// Command tokendiff performs token-level diffs with delimiter support.
//
// Usage:
//
//	tokendiff file1 file2
//	tokendiff -d "(){}[]" file1 file2
//	echo "old text" | tokendiff -stdin "new text"
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dacharyc/tokendiff"
	flag "github.com/spf13/pflag"
)

// Version is set at build time via -ldflags
var Version = "dev"

// Default colors for diff output (reset + color + bold)
const (
	defaultDeleteColor = "\033[0;31;1m" // bold red
	defaultInsertColor = "\033[0;32;1m" // bold green
	defaultChangeColor = "\033[0;33;1m" // bold yellow (for line markers)
)

// Exit codes
const (
	exitIdentical = 0 // files are identical
	exitDiffer    = 1 // files differ
	exitError     = 2 // error occurred
)

// config holds configuration from profile files
type config struct {
	delimiters          string
	whitespace          string
	usePunctuation      bool
	noColor             bool
	colorSpec           string
	lineNumbers         int
	lineByLine          bool
	context             int
	startDelete         string
	stopDelete          string
	startInsert         string
	stopInsert          string
	repeatMarkers       bool
	lessMode            bool
	printerMode         bool
	noDeleted           bool
	noInserted          bool
	noCommon            bool
	statistics          bool
	ignoreCase          bool
	matchContext        int
	algorithm           string  // line pairing algorithm: "best", "normal", "fast"
	similarityThreshold float64 // minimum similarity for line pairing (0.0-1.0)
}

// cliFlags holds all parsed command-line flags
type cliFlags struct {
	delimiters     *string
	whitespace     *string
	usePunctuation *bool
	noColor        *bool
	colorSpec      *string
	lineNumbers    *int
	lineByLine     *bool
	context        *int
	stdinMode      *bool
	help           *bool
	version        *bool
	startDelete    *string
	stopDelete     *string
	startInsert    *string
	stopInsert     *string
	repeatMarkers  *bool
	lessMode       *bool
	printerMode    *bool
	noDeleted      *bool
	noInserted     *bool
	noCommon       *bool
	statistics     *bool
	ignoreCase     *bool
	matchContext   *int
	diffInput      *bool
	algorithm      *string
	threshold      *float64
}

// prescanProfile extracts --profile value before flag parsing
func prescanProfile() string {
	for i, arg := range os.Args[1:] {
		if arg == "--profile" && i+1 < len(os.Args)-1 {
			return os.Args[i+2]
		}
		if strings.HasPrefix(arg, "--profile=") {
			return strings.TrimPrefix(arg, "--profile=")
		}
	}
	return ""
}

// defineFlags sets up all command-line flags with config defaults
func defineFlags(cfg config) cliFlags {
	_ = flag.String("profile", "", "use settings from ~/.tokendiffrc.<profile>")

	f := cliFlags{
		delimiters:     flag.StringP("delimiters", "d", cfg.delimiters, "delimiter characters"),
		whitespace:     flag.StringP("white-space", "W", cfg.whitespace, "whitespace characters"),
		usePunctuation: flag.BoolP("punctuation", "P", cfg.usePunctuation, "use punctuation characters as delimiters"),
		noColor:        flag.Bool("no-color", cfg.noColor, "disable colored output"),
		colorSpec:      flag.StringP("color", "c", cfg.colorSpec, "set colors for deleted/inserted text (format: del_fg[:del_bg],ins_fg[:ins_bg], or 'list')"),
		lineNumbers:    flag.IntP("line-numbers", "L", cfg.lineNumbers, "show line numbers with specified width (0 for auto-width)"),
		lineByLine:     flag.Bool("line-mode", cfg.lineByLine, "compare files line by line"),
		context:        flag.IntP("context", "C", cfg.context, "show N lines of context around changes (implies --line-mode)"),
		stdinMode:      flag.Bool("stdin", false, "read first input from stdin, second from argument"),
		help:           flag.BoolP("help", "h", false, "show help"),
		version:        flag.BoolP("version", "v", false, "show version"),
		startDelete:    flag.StringP("start-delete", "w", cfg.startDelete, "string to mark begin of deleted text"),
		stopDelete:     flag.StringP("stop-delete", "x", cfg.stopDelete, "string to mark end of deleted text"),
		startInsert:    flag.StringP("start-insert", "y", cfg.startInsert, "string to mark begin of inserted text"),
		stopInsert:     flag.StringP("stop-insert", "z", cfg.stopInsert, "string to mark end of inserted text"),
		repeatMarkers:  flag.BoolP("repeat-markers", "R", cfg.repeatMarkers, "repeat markers at line boundaries for multi-line changes"),
		lessMode:       flag.BoolP("less-mode", "l", cfg.lessMode, "use overstrike to highlight text for less -r"),
		printerMode:    flag.BoolP("printer", "p", cfg.printerMode, "use overstrike to highlight text for printing"),
		noDeleted:      flag.BoolP("no-deleted", "1", cfg.noDeleted, "suppress printing of deleted words"),
		noInserted:     flag.BoolP("no-inserted", "2", cfg.noInserted, "suppress printing of inserted words"),
		noCommon:       flag.BoolP("no-common", "3", cfg.noCommon, "suppress printing of common words"),
		statistics:     flag.BoolP("statistics", "s", cfg.statistics, "print statistics"),
		ignoreCase:     flag.BoolP("ignore-case", "i", cfg.ignoreCase, "ignore case when comparing"),
		matchContext:   flag.IntP("match-context", "m", cfg.matchContext, "minimum matching words between changes"),
		diffInput:      flag.Bool("diff-input", false, "read unified diff from stdin and apply word-level diff"),
		algorithm:      flag.StringP("algorithm", "A", cfg.algorithm, "line pairing algorithm: best (similarity), normal (positional), fast (positional)"),
		threshold:      flag.Float64("threshold", cfg.similarityThreshold, "minimum similarity for line pairing with -A best (0.0-1.0)"),
	}

	flag.Lookup("color").NoOptDefVal = "default"
	flag.Lookup("line-numbers").NoOptDefVal = "0"

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] file1 file2\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s [options] -stdin file2\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nWord-level diff with delimiter support.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s old.txt new.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --line-mode -C 3 old.go new.go\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  git show HEAD:file.go | %s -stdin file.go\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  git diff | %s --diff-input\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExit codes:\n")
		fmt.Fprintf(os.Stderr, "  0  files are identical\n")
		fmt.Fprintf(os.Stderr, "  1  files differ\n")
		fmt.Fprintf(os.Stderr, "  2  error occurred\n")
	}

	return f
}

// showColorList prints available colors and exits
func showColorList() {
	fmt.Println("Available colors:")
	colors := tokendiff.ColorNames()
	if len(colors) > 8 {
		fmt.Printf("  %s\n", strings.Join(colors[:8], ", "))
		fmt.Printf("  %s\n", strings.Join(colors[8:], ", "))
	} else {
		fmt.Printf("  %s\n", strings.Join(colors, ", "))
	}
	fmt.Println("\nUsage: -c delete_color[:delete_bg],insert_color[:insert_bg]")
	fmt.Println("Example: -c red,green")
	fmt.Println("Example: -c brightred:white,brightgreen:black")
	os.Exit(exitIdentical)
}

// parseColorSpec parses the color specification and returns delete/insert colors
func parseColors(colorSpec string) (deleteColor, insertColor string) {
	deleteColor = defaultDeleteColor
	insertColor = defaultInsertColor
	if colorSpec != "" && colorSpec != "default" {
		var err error
		deleteColor, insertColor, err = tokendiff.ParseColorSpec(colorSpec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exitError)
		}
	}
	return
}

// validateAlgorithm checks if the algorithm is valid
func validateAlgorithm(algorithm string) {
	switch algorithm {
	case "best", "normal", "fast":
		// valid
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid algorithm %q (use best, normal, or fast)\n", algorithm)
		os.Exit(exitError)
	}
}

// readInputTexts reads input from stdin or files
func readInputTexts(stdinMode bool) (text1, text2 string) {
	var err error
	if stdinMode {
		if flag.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "Error: -stdin mode requires one file argument")
			os.Exit(exitError)
		}
		text1, err = readStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(exitError)
		}
		text2, err = readFile(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", flag.Arg(0), err)
			os.Exit(exitError)
		}
	} else {
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "Error: requires two file arguments")
			flag.Usage()
			os.Exit(exitError)
		}
		text1, err = readFile(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", flag.Arg(0), err)
			os.Exit(exitError)
		}
		text2, err = readFile(flag.Arg(1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", flag.Arg(1), err)
			os.Exit(exitError)
		}
	}
	return
}

func main() {
	// Pre-scan for --profile flag before defining other flags
	profile := prescanProfile()

	// Load configuration from profile
	configPath, err := findConfigFile(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitError)
	}
	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", configPath, err)
		os.Exit(exitError)
	}

	// Define and parse flags
	f := defineFlags(cfg)
	flag.Parse()

	if *f.version {
		fmt.Printf("tokendiff version %s\n", Version)
		os.Exit(exitIdentical)
	}

	if *f.help {
		flag.Usage()
		os.Exit(exitIdentical)
	}

	// Handle -c list
	if *f.colorSpec == "list" {
		showColorList()
	}

	// Parse color spec and validate algorithm
	deleteColor, insertColor := parseColors(*f.colorSpec)
	validateAlgorithm(*f.algorithm)

	// Configure diff options
	opts := tokendiff.Options{
		Delimiters:         parseEscapeSequences(*f.delimiters),
		Whitespace:         parseEscapeSequences(*f.whitespace),
		UsePunctuation:     *f.usePunctuation,
		IgnoreCase:         *f.ignoreCase,
		PreserveWhitespace: false,
	}

	// Determine color output
	useColor := !*f.noColor && os.Getenv("NO_COLOR") == "" && (isTerminal(os.Stdout) || *f.colorSpec != "")
	if *f.lessMode || *f.printerMode {
		useColor = false
	}

	// Build format options using the core library's FormatOptions
	fmtOpts := tokendiff.FormatOptions{
		StartDelete:      *f.startDelete,
		StopDelete:       *f.stopDelete,
		StartInsert:      *f.startInsert,
		StopInsert:       *f.stopInsert,
		NoDeleted:        *f.noDeleted,
		NoInserted:       *f.noInserted,
		NoCommon:         *f.noCommon,
		UseColor:         useColor,
		DeleteColor:      deleteColor,
		InsertColor:      insertColor,
		ColorReset:       tokendiff.ANSIReset,
		ClearToEOL:       tokendiff.ANSIClearEOL,
		RepeatMarkers:    *f.repeatMarkers,
		LessMode:         *f.lessMode,
		PrinterMode:      *f.printerMode,
		MatchContext:     *f.matchContext,
		HeuristicSpacing: true,
	}

	// Handle --diff-input mode
	if *f.diffInput {
		if err := tokendiff.ProcessUnifiedDiff(os.Stdin, os.Stdout, opts, fmtOpts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exitError)
		}
		os.Exit(exitIdentical)
	}

	// Context implies line-by-line mode
	lineByLine := *f.lineByLine
	if *f.context > 0 {
		lineByLine = true
	}

	// Line numbers imply line-by-line mode for correct output
	if *f.lineNumbers >= 0 {
		lineByLine = true
	}

	// Get input texts
	text1, text2 := readInputTexts(*f.stdinMode)

	// Set line number display options
	fmtOpts.ShowLineNumbers = *f.lineNumbers >= 0
	if *f.lineNumbers == 0 {
		// Auto-calculate width based on file lengths
		maxLines := max(strings.Count(text1, "\n"), strings.Count(text2, "\n")) + 1
		fmtOpts.LineNumWidth = len(fmt.Sprintf("%d", maxLines))
		if fmtOpts.LineNumWidth < 3 {
			fmtOpts.LineNumWidth = 3
		}
	} else {
		fmtOpts.LineNumWidth = *f.lineNumbers
	}

	var st tokendiff.DiffStatistics
	if lineByLine {
		output := tokendiff.DiffLineByLine(text1, text2, opts, fmtOpts, *f.algorithm, *f.threshold)
		st = output.Statistics

		// Print with context or all lines
		if *f.context > 0 {
			printWithContext(output.Lines, *f.context, fmtOpts)
		} else {
			printLineResults(output.Lines, fmtOpts)
		}
	} else {
		result := tokendiff.DiffWholeFiles(text1, text2, opts, fmtOpts)
		st = result.Statistics
		fmt.Println(result.Formatted)
	}

	if *f.statistics {
		printStatistics(st)
	}

	// Exit with appropriate code based on whether differences were found
	if st.DeletedWords > 0 || st.InsertedWords > 0 {
		os.Exit(exitDiffer)
	}
	os.Exit(exitIdentical)
}

// printLineResults prints all line diff results
func printLineResults(results []tokendiff.LineDiffResult, fmtOpts tokendiff.FormatOptions) {
	for _, r := range results {
		printLineDiffResult(r, fmtOpts)
	}
}

// printLineDiffResult prints a single line diff result with appropriate formatting
func printLineDiffResult(r tokendiff.LineDiffResult, fmtOpts tokendiff.FormatOptions) {
	if fmtOpts.ShowLineNumbers {
		width := fmtOpts.LineNumWidth
		oldWidth := width + 1
		newWidth := width + 2
		oldStr := fmt.Sprintf("%*d", oldWidth, r.OldLineNum)
		newStr := fmt.Sprintf("%-*d", newWidth, r.NewLineNum)
		if r.OldLineNum == 0 {
			oldStr = strings.Repeat(" ", oldWidth)
		}
		if r.NewLineNum == 0 {
			newStr = strings.Repeat(" ", newWidth)
		}
		fmt.Printf("%s:%s%s\n", oldStr, newStr, r.Output)
	} else {
		prefix := "  "
		if r.HasChanges {
			prefix = "| "
			if fmtOpts.UseColor {
				prefix = defaultChangeColor + "| " + tokendiff.ANSIReset
			}
		}
		lineNum := r.NewLineNum
		if lineNum == 0 {
			lineNum = r.OldLineNum
		}
		fmt.Printf("%s%4d: %s\n", prefix, lineNum, r.Output)
	}
}

// printWithContext prints only changed lines with surrounding context
func printWithContext(results []tokendiff.LineDiffResult, contextLines int, fmtOpts tokendiff.FormatOptions) {
	// Find ranges to print
	toPrint := make([]bool, len(results))
	for i, r := range results {
		if r.HasChanges {
			start := max(0, i-contextLines)
			end := min(len(results), i+contextLines+1)
			for j := start; j < end; j++ {
				toPrint[j] = true
			}
		}
	}

	// Print with separators for gaps
	lastPrinted := -1
	for i, r := range results {
		if !toPrint[i] {
			continue
		}

		if lastPrinted >= 0 && i > lastPrinted+1 {
			fmt.Println("---")
		}

		printLineDiffResult(r, fmtOpts)
		lastPrinted = i
	}
}

// printStatistics prints diff statistics to stderr
func printStatistics(st tokendiff.DiffStatistics) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "old: %d words  %d %d%% common  %d %d%% deleted\n",
		st.OldWords,
		st.CommonWords, percent(st.CommonWords, st.OldWords),
		st.DeletedWords, percent(st.DeletedWords, st.OldWords))
	fmt.Fprintf(os.Stderr, "new: %d words  %d %d%% common  %d %d%% inserted\n",
		st.NewWords,
		st.CommonWords, percent(st.CommonWords, st.NewWords),
		st.InsertedWords, percent(st.InsertedWords, st.NewWords))
}

// percent calculates percentage, handling division by zero
func percent(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}

// readFile reads an entire file into a string
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readStdin reads all of stdin into a string
func readStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var sb strings.Builder
	for {
		line, err := reader.ReadString('\n')
		sb.WriteString(line)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

// isTerminal returns true if the file is a terminal
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// parseEscapeSequences converts escape sequences in a string.
func parseEscapeSequences(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'x', 'X':
				if i+3 < len(s) {
					hexStr := s[i+2 : i+4]
					if val, err := strconv.ParseUint(hexStr, 16, 8); err == nil {
						result.WriteByte(byte(val))
						i += 4
						continue
					}
				}
				result.WriteByte(s[i])
				i++
			case 'n':
				result.WriteByte('\n')
				i += 2
			case 't':
				result.WriteByte('\t')
				i += 2
			case 'r':
				result.WriteByte('\r')
				i += 2
			case '\\':
				result.WriteByte('\\')
				i += 2
			case '!':
				result.WriteByte('!')
				i += 2
			default:
				result.WriteByte(s[i])
				i++
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// findConfigFile returns the path to the config file for the given profile.
// If a profile is specified but the file doesn't exist, it returns an error.
func findConfigFile(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil // No home dir, use defaults
	}

	if profile == "" {
		path := filepath.Join(home, ".tokendiffrc")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		path = filepath.Join(xdgConfig, "tokendiff", "config")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", nil // No default config found, use defaults
	}

	// Profile explicitly specified - file must exist
	path := filepath.Join(home, ".tokendiffrc."+profile)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("profile config file not found: %s", path)
	}
	return path, nil
}

// loadConfig reads a config file and returns the configuration.
func loadConfig(path string) (config, error) {
	cfg := defaultConfig()

	if path == "" {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var key, value string
		if idx := strings.Index(line, "="); idx >= 0 {
			key = strings.TrimSpace(line[:idx])
			value = strings.TrimSpace(line[idx+1:])
		} else {
			key = line
			value = "true"
		}

		if err := applyConfigOption(&cfg, key, value); err != nil {
			return cfg, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}

	return cfg, scanner.Err()
}

// defaultConfig returns a config with default values
func defaultConfig() config {
	return config{
		delimiters:          tokendiff.DefaultDelimiters,
		whitespace:          tokendiff.DefaultWhitespace,
		lineNumbers:         -1,
		startDelete:         "[-",
		stopDelete:          "-]",
		startInsert:         "{+",
		stopInsert:          "+}",
		algorithm:           "best",
		similarityThreshold: 0.1,
	}
}

// applyStringOption handles string config options
func applyStringOption(cfg *config, key, value string) bool {
	switch key {
	case "delimiters", "d":
		cfg.delimiters = value
	case "white-space", "W":
		cfg.whitespace = value
	case "color", "c":
		cfg.colorSpec = value
	case "start-delete", "w":
		cfg.startDelete = value
	case "stop-delete", "x":
		cfg.stopDelete = value
	case "start-insert", "y":
		cfg.startInsert = value
	case "stop-insert", "z":
		cfg.stopInsert = value
	default:
		return false
	}
	return true
}

// applyBoolOption handles boolean config options
func applyBoolOption(cfg *config, key, value string) bool {
	switch key {
	case "punctuation", "P":
		cfg.usePunctuation = parseBool(value)
	case "no-color":
		cfg.noColor = parseBool(value)
	case "line-mode":
		cfg.lineByLine = parseBool(value)
	case "repeat-markers", "R":
		cfg.repeatMarkers = parseBool(value)
	case "less-mode", "l":
		cfg.lessMode = parseBool(value)
	case "printer", "p":
		cfg.printerMode = parseBool(value)
	case "no-deleted", "1":
		cfg.noDeleted = parseBool(value)
	case "no-inserted", "2":
		cfg.noInserted = parseBool(value)
	case "no-common", "3":
		cfg.noCommon = parseBool(value)
	case "statistics", "s":
		cfg.statistics = parseBool(value)
	case "ignore-case", "i":
		cfg.ignoreCase = parseBool(value)
	default:
		return false
	}
	return true
}

// applyIntOption handles integer config options
func applyIntOption(cfg *config, key, value string) bool {
	switch key {
	case "line-numbers", "L":
		cfg.lineNumbers = parseInt(value, -1)
	case "context", "C":
		cfg.context = parseInt(value, 0)
	case "match-context", "m":
		cfg.matchContext = parseInt(value, 0)
	default:
		return false
	}
	return true
}

// applyConfigOption sets a config field based on key and value
func applyConfigOption(cfg *config, key, value string) error {
	if applyStringOption(cfg, key, value) {
		return nil
	}
	if applyBoolOption(cfg, key, value) {
		return nil
	}
	if applyIntOption(cfg, key, value) {
		return nil
	}

	// Special cases with validation
	switch key {
	case "algorithm", "A":
		switch value {
		case "best", "normal", "fast":
			cfg.algorithm = value
		default:
			return fmt.Errorf("invalid algorithm: %s (use best, normal, or fast)", value)
		}
	case "threshold":
		t := parseFloat(value, -1)
		if t < 0 || t > 1 {
			return fmt.Errorf("threshold must be a number between 0.0 and 1.0")
		}
		cfg.similarityThreshold = t
	default:
		return fmt.Errorf("unknown option: %s", key)
	}
	return nil
}

// parseBool parses a boolean value from a string
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1" || s == ""
}

// parseInt parses an integer value from a string
func parseInt(s string, defaultVal int) int {
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	if err != nil {
		return defaultVal
	}
	return val
}

// parseFloat parses a float value from a string
func parseFloat(s string, defaultVal float64) float64 {
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return defaultVal
	}
	return val
}
