package docagent

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	installpkg "github.com/zeshone/doc-agent-ai/internal/install"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiBlue   = "\x1b[34m"
	ansiGray   = "\x1b[90m"
)

type bufferReporter struct {
	buf bytes.Buffer
}

func newBufferReporter() *bufferReporter { return &bufferReporter{} }

func (b *bufferReporter) Ok(msg string)     { fmt.Fprintf(&b.buf, "  ✔ %s\n", msg) }
func (b *bufferReporter) Warn(msg string)   { fmt.Fprintf(&b.buf, "  ⚠  %s\n", msg) }
func (b *bufferReporter) ErrOut(msg string) { fmt.Fprintf(&b.buf, "  ✖ %s\n", msg) }
func (b *bufferReporter) Info(msg string)   { fmt.Fprintf(&b.buf, "  → %s\n", msg) }
func (b *bufferReporter) Dim(msg string)    { fmt.Fprintf(&b.buf, "  %s\n", msg) }
func (b *bufferReporter) Head(msg string)   { fmt.Fprintf(&b.buf, "\n  %s\n", msg) }

type stdoutReporter struct {
	w io.Writer
}

func newStdoutReporter() Reporter { return &stdoutReporter{w: os.Stdout} }

func (s *stdoutReporter) Ok(msg string)     { fmt.Fprintf(s.w, "  %s✔%s %s\n", ansiGreen, ansiReset, msg) }
func (s *stdoutReporter) Warn(msg string)   { fmt.Fprintf(s.w, "  %s⚠%s  %s\n", ansiYellow, ansiReset, msg) }
func (s *stdoutReporter) ErrOut(msg string) { fmt.Fprintf(s.w, "  %s✖%s %s\n", ansiRed, ansiReset, msg) }
func (s *stdoutReporter) Info(msg string)   { fmt.Fprintf(s.w, "  %s→%s %s\n", ansiBlue, ansiReset, msg) }
func (s *stdoutReporter) Dim(msg string)    { fmt.Fprintf(s.w, "%s  %s%s\n", ansiGray, msg, ansiReset) }
func (s *stdoutReporter) Head(msg string)   { fmt.Fprintf(s.w, "\n%s  %s%s\n", ansiBold, msg, ansiReset) }

func newPlatformForTest(t *testing.T, id string, homeDir string) Platform {
	t.Helper()
	plat := installpkg.NewPlatformForTest(id, homeDir)
	if plat == nil {
		t.Fatalf("unknown platform: %s", id)
	}
	return plat
}
