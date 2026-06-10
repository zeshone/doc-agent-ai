package main

import (
	"bytes"
	"fmt"
)

// ---------------------------------------------------------------------------
// Reporter — output sink interface
// ---------------------------------------------------------------------------

// Reporter is the output sink for all user-visible install messages.
// The engine (installToPlatformWithReporter, installPlatforms*, executeInstall)
// calls Reporter methods instead of the package-level ok/warn/... helpers so
// that the TUI can collect structured messages without raw ANSI output
// corrupting its frame.
//
// The package-level helpers (ok, warn, info, dim, head, errOut) are preserved
// as-is for the interactive prompt flow; they are NOT called from the engine
// when a Reporter is provided.
type Reporter interface {
	// Ok reports a successful action (green checkmark in stdout mode).
	Ok(msg string)
	// Warn reports a non-fatal advisory (yellow warning in stdout mode).
	Warn(msg string)
	// ErrOut reports an error line (red cross in stdout mode).
	ErrOut(msg string)
	// Info reports an informational line (blue arrow in stdout mode).
	Info(msg string)
	// Dim reports a dimmed detail line (gray in stdout mode).
	Dim(msg string)
	// Head reports a section heading (bold in stdout mode).
	Head(msg string)
}

// ---------------------------------------------------------------------------
// stdoutReporter — default Reporter; byte-identical to the old helpers
// ---------------------------------------------------------------------------

// stdoutReporter is the default Reporter. Its output is identical to the
// former package-level ok/warn/info/dim/head/errOut helpers so that headless
// and interactive-without-TUI installs produce the same output as before.
type stdoutReporter struct{}

// defaultReporter is the shared stdout reporter used by the backward-compat wrappers.
var defaultReporter Reporter = &stdoutReporter{}

func (s *stdoutReporter) Ok(msg string) {
	fmt.Printf("  %s✔%s %s\n", ansiGreen, ansiReset, msg)
}

func (s *stdoutReporter) Warn(msg string) {
	fmt.Printf("  %s⚠%s  %s\n", ansiYellow, ansiReset, msg)
}

func (s *stdoutReporter) ErrOut(msg string) {
	fmt.Printf("  %s✖%s %s\n", ansiRed, ansiReset, msg)
}

func (s *stdoutReporter) Info(msg string) {
	fmt.Printf("  %s→%s %s\n", ansiBlue, ansiReset, msg)
}

func (s *stdoutReporter) Dim(msg string) {
	fmt.Printf("%s  %s%s\n", ansiGray, msg, ansiReset)
}

func (s *stdoutReporter) Head(msg string) {
	fmt.Printf("\n%s  %s%s\n", ansiBold, msg, ansiReset)
}

// ---------------------------------------------------------------------------
// bufferReporter — in-memory Reporter for tests and TUI result collection
// ---------------------------------------------------------------------------

// bufferReporter captures all Reporter calls into an in-memory buffer.
// It is used in tests to assert install output without writing to stdout,
// and in slice 2b the TUI will use a variant to collect structured results.
type bufferReporter struct {
	buf bytes.Buffer
}

// newBufferReporter returns a new bufferReporter ready for use.
func newBufferReporter() *bufferReporter {
	return &bufferReporter{}
}

func (b *bufferReporter) Ok(msg string) {
	fmt.Fprintf(&b.buf, "  ✔ %s\n", msg)
}

func (b *bufferReporter) Warn(msg string) {
	fmt.Fprintf(&b.buf, "  ⚠  %s\n", msg)
}

func (b *bufferReporter) ErrOut(msg string) {
	fmt.Fprintf(&b.buf, "  ✖ %s\n", msg)
}

func (b *bufferReporter) Info(msg string) {
	fmt.Fprintf(&b.buf, "  → %s\n", msg)
}

func (b *bufferReporter) Dim(msg string) {
	fmt.Fprintf(&b.buf, "  %s\n", msg)
}

func (b *bufferReporter) Head(msg string) {
	fmt.Fprintf(&b.buf, "\n  %s\n", msg)
}

// String returns the full buffered output.
func (b *bufferReporter) String() string {
	return b.buf.String()
}
