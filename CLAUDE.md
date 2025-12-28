# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -v -run TestDiffStrings ./...

# Run CLI tests only
go test -v ./cmd/tokendiff/...

# Build the CLI
go build ./cmd/tokendiff

# Install the CLI globally
go install ./cmd/tokendiff

# Run benchmarks
go test -bench=. ./...
```

## Architecture

tokendiff is a Go library and CLI for token-level diffing with delimiter support. It uses the histogram diff algorithm via [diffx](https://github.com/dacharyc/diffx).

### Core Pipeline

1. **Tokenization** (`tokenize.go`): Splits text into tokens using configurable delimiters and whitespace. Key functions:
   - `Tokenize()` - basic tokenization
   - `TokenizeWithPositions()` - tracks byte offsets for whitespace reconstruction

2. **Diffing** (`tokendiff.go`): Computes diffs using diffx's histogram algorithm. Core types:
   - `Operation` - Equal, Insert, Delete
   - `Diff` - single operation on a token
   - `Options` - configures delimiters, whitespace handling, case sensitivity

3. **Post-processing** (`postprocess.go`): Transforms raw diffs:
   - `AggregateDiffs()` - combines adjacent same-type operations
   - `ApplyMatchContext()` - converts isolated equals to delete+insert pairs
   - `ShiftBoundaries()` - improves diff readability

4. **Formatting** (`format.go`): Renders diffs as text with markers, colors, or overstrike. `FormatOptions` controls output style.

5. **Line-level diffing** (`linediff.go`): Pairs deleted/inserted lines for line-mode output using positional or similarity matching.

6. **Unified diff parsing** (`unified.go`): Parses `diff -u` / `git diff` output for `--diff-input` mode.

### CLI Structure

The CLI (`cmd/tokendiff/main.go`) wraps the library with:
- Flag parsing via spf13/pflag
- Configuration file support (`~/.tokendiffrc`, `~/.config/tokendiff/config`, profiles)
- Stdin input support
- Exit codes: 0 (identical), 1 (differ), 2 (error)

### Key Design Decisions

- Default delimiters are empty (whitespace-only splitting) to match dwdiff behavior
- Case-insensitive mode compares lowercase but preserves original case in output
- The histogram algorithm avoids spurious matches on common words like "the", "for", "in"
- `DiffResult` includes token positions to reconstruct original whitespace for Equal tokens
