# tokendiff

A Go library and CLI for token-level diffing with delimiter support.

tokendiff uses a histogram diff algorithm that groups semantically related changes together, producing more readable output than traditional Myers-based approaches for complex structural changes.

## Motivation

Traditional diff tools operate at the line level. Word-based tools like `wdiff` improve on this but can produce suboptimal results when comparing code. For example, when comparing:

```
void someFunction(SomeType var)
void someFunction(SomeOtherType var)
```

`wdiff` reports that `someFunction(SomeType` changed to `someFunction(SomeOtherType` - grouping the function name with the parameter type.

tokendiff treats delimiter characters like `(` as separate tokens, correctly identifying that only `SomeType` changed to `SomeOtherType`.

## Algorithm

This library uses the **histogram diff algorithm** via [diffx](https://github.com/dacharyc/diffx). The histogram algorithm is a variant of the patience diff algorithm that performs well on real-world text by:

1. Finding unique tokens that appear exactly once in each input (strong anchors)
2. Using frequency analysis to avoid matching common tokens that would create confusing output
3. Recursively diffing the regions between anchors

This approach produces output that groups semantically related changes together, making diffs easier to read than traditional Myers-based algorithms when comparing files with significant structural changes.

## Installation

### Library

```bash
go get github.com/dacharyc/tokendiff
```

### CLI Tool

```bash
go install github.com/dacharyc/tokendiff/cmd/tokendiff@latest
```

## CLI Usage

```
tokendiff [options] file1 file2
tokendiff [options] -stdin file2
```

### Options

**Input/Output:**
| Flag | Description |
|------|-------------|
| `-d "..."` | Custom delimiter characters |
| `-P, --punctuation` | Use Unicode punctuation as delimiters |
| `-W, --white-space "..."` | Custom whitespace characters |
| `--line-mode` | Compare files line by line |
| `-C N` | Show N lines of context (implies --line-mode) |
| `-L N, --line-numbers N` | Show line numbers with width N (0 for auto) |
| `-stdin` | Read first input from stdin |
| `--diff-input` | Read unified diff from stdin and apply token-level diff |

**Output Formatting:**
| Flag | Description |
|------|-------------|
| `-w "..."` | String to mark start of deleted text (default: `[-`) |
| `-x "..."` | String to mark end of deleted text (default: `-]`) |
| `-y "..."` | String to mark start of inserted text (default: `{+`) |
| `-z "..."` | String to mark end of inserted text (default: `+}`) |
| `-c, --color SPEC` | Set colors (format: `del_fg[:bg],ins_fg[:bg]`, or `list`) |
| `--no-color` | Disable colored output |
| `-l, --less-mode` | Use overstrike for `less -r` viewing |
| `-p, --printer` | Use overstrike for printing |
| `-R, --repeat-markers` | Repeat markers at line boundaries |
| `-a, --aggregate-changes` | Combine adjacent insertions/deletions |

**Output Suppression:**
| Flag | Description |
|------|-------------|
| `-1` | Suppress deleted words |
| `-2` | Suppress inserted words |
| `-3` | Suppress common words |

**Comparison:**
| Flag | Description |
|------|-------------|
| `-i, --ignore-case` | Case-insensitive comparison |
| `-m N, --match-context N` | Minimum matching words between changes |

**Other:**
| Flag | Description |
|------|-------------|
| `-s, --statistics` | Print diff statistics |
| `--profile NAME` | Use settings from `~/.tokendiffrc.<NAME>` |
| `-v, --version` | Show version |
| `-h` | Show help |

The CLI respects the `NO_COLOR` environment variable.

### Configuration Files

tokendiff supports configuration files to set default options:

- `~/.tokendiffrc` - Default configuration (loaded automatically)
- `~/.config/tokendiff/config` - XDG-compliant location (fallback)
- `~/.tokendiffrc.<profile>` - Named profile (use with `--profile`)

**Config file format:**
```
# Comment
option-name
option-name=value
```

**Example `~/.tokendiffrc.html`:**
```
# HTML output profile
start-delete=<del>
stop-delete=</del>
start-insert=<ins>
stop-insert=</ins>
no-color
```

**Usage:**
```bash
tokendiff --profile=html old.txt new.txt
```

Command-line options override configuration file settings.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Files are identical |
| 1 | Files differ |
| 2 | Error occurred |

### Examples

```bash
# Compare two files
tokendiff old.txt new.txt

# Line-by-line with context
tokendiff --line-mode -C 3 old.go new.go

# Compare git versions
git show HEAD~1:file.go | tokendiff -stdin file.go

# Custom delimiters
tokendiff -d "(){}[]" file1.txt file2.txt

# Case-insensitive comparison with statistics
tokendiff -i -s old.txt new.txt

# HTML-style markers
tokendiff -w '<del>' -x '</del>' -y '<ins>' -z '</ins>' old.txt new.txt

# View in less with overstrike highlighting
tokendiff -l old.txt new.txt | less -r

# Apply token-level diff to a unified diff
git diff | tokendiff --diff-input
diff -u old.txt new.txt | tokendiff --diff-input
```

## Library Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/dacharyc/tokendiff"
)

func main() {
    old := "void someFunction(SomeType var)"
    new := "void someFunction(SomeOtherType var)"

    diffs := tokendiff.DiffStrings(old, new, tokendiff.DefaultOptions())
    fmt.Println(tokendiff.FormatDiff(diffs))
    // Output: void someFunction([-SomeType-]{+SomeOtherType+} var)
}
```

### Working with Tokens

```go
// Tokenize text with delimiter awareness
tokens := tokendiff.Tokenize("foo(bar, baz)", tokendiff.DefaultOptions())
// tokens = ["foo", "(", "bar", ",", "baz", ")"]

// Diff pre-tokenized content
diffs := tokendiff.DiffTokens(tokens1, tokens2)
```

### Custom Delimiters

```go
opts := tokendiff.Options{
    Delimiters: "|:-",  // Custom delimiter set
}
diffs := tokendiff.DiffStrings(text1, text2, opts)
```

### Preserving Whitespace

```go
opts := tokendiff.Options{
    Delimiters:         tokendiff.DefaultDelimiters,
    PreserveWhitespace: true,  // Include whitespace as tokens
}
```

## API

### Types

```go
type Operation int
const (
    Equal  Operation = iota  // Token unchanged
    Insert                   // Token was added
    Delete                   // Token was removed
)

type Diff struct {
    Type  Operation
    Token string
}

type Options struct {
    Delimiters         string  // Characters to treat as separate tokens
    Whitespace         string  // Characters to treat as whitespace
    UsePunctuation     bool    // Use Unicode punctuation as delimiters
    PreserveWhitespace bool    // Include whitespace as tokens
    IgnoreCase         bool    // Case-insensitive comparison
}

type FormatOptions struct {
    StartDelete string  // Marker for start of deleted text (default: "[-")
    StopDelete  string  // Marker for end of deleted text (default: "-]")
    StartInsert string  // Marker for start of inserted text (default: "{+")
    StopInsert  string  // Marker for end of inserted text (default: "+}")
    NoDeleted   bool    // Suppress deleted tokens
    NoInserted  bool    // Suppress inserted tokens
    NoCommon    bool    // Suppress unchanged tokens
}
```

### Functions

**Tokenizing and Diffing:**
- `Tokenize(text string, opts Options) []string` - Split text into tokens
- `DiffTokens(tokens1, tokens2 []string) []Diff` - Diff two token slices
- `DiffStrings(text1, text2 string, opts Options) []Diff` - Tokenize and diff two strings
- `DefaultOptions() Options` - Get default options

**Diff Transformations:**
- `AggregateDiffs(diffs []Diff) []Diff` - Combine adjacent same-type operations
- `ApplyMatchContext(diffs []Diff, minContext int) []Diff` - Require minimum matching words between changes

**Formatting:**
- `FormatDiff(diffs []Diff) string` - Format diff with default markers
- `FormatDiffWithOptions(diffs []Diff, opts FormatOptions) string` - Format with custom markers
- `DefaultFormatOptions() FormatOptions` - Get default format options
- `HasChanges(diffs []Diff) bool` - Check if diff contains any changes
- `NeedsSpaceBefore(token string) bool` - Check if space should precede token
- `NeedsSpaceAfter(token string) bool` - Check if space should follow token

**Unified Diff Parsing:**
- `ParseUnifiedDiff(input string) ([]UnifiedDiff, error)` - Parse unified diff format
- `ApplyWordDiff(hunk DiffHunk, opts Options) []Diff` - Apply token-level diff to a hunk

## Default Delimiters

```
(){}[]<>,.;:!?"'`@#$%^&*+-=/\|~
```

## Performance

Benchmarks on Apple M1:

```
BenchmarkTokenize      ~2.5 µs/op
BenchmarkDiffStrings   ~10.5 µs/op
```

## License

MIT
