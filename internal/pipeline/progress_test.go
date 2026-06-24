package pipeline

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestProgressLogger_TTY(t *testing.T) {
	// Override isTerminal to simulate TTY output
	oldIsTerminal := isTerminal
	isTerminal = func(w io.Writer) (bool, uintptr) {
		return true, 0
	}
	defer func() { isTerminal = oldIsTerminal }()

	var buf bytes.Buffer
	logger := NewProgressLogger(&buf, "    ")

	logger.Start("Hadolint")
	if !strings.Contains(buf.String(), "    [IN_PROGRESS] Hadolint") {
		t.Errorf("expected output to contain IN_PROGRESS status with prefix, got: %q", buf.String())
	}

	buf.Reset()
	logger.Complete("Hadolint", "[PASS]", 120, "")
	output := buf.String()

	// Assert it uses \r\x1b[2K (ANSI clear) to replace the line
	if !strings.Contains(output, "\r\x1b[2K") {
		t.Errorf("expected output to contain clear line sequences, got: %q", output)
	}
	if !strings.Contains(output, "    [PASS] Hadolint (120ms)") {
		t.Errorf("expected output to contain final PASS message with prefix, got: %q", output)
	}
}

func TestProgressLogger_NonTTY(t *testing.T) {
	// Override isTerminal to simulate non-TTY output
	oldIsTerminal := isTerminal
	isTerminal = func(w io.Writer) (bool, uintptr) {
		return false, 0
	}
	defer func() { isTerminal = oldIsTerminal }()

	var buf bytes.Buffer
	logger := NewProgressLogger(&buf, "  ")

	logger.Start("Hadolint")
	if buf.Len() > 0 {
		t.Errorf("expected no output during Start on non-TTY, got: %q", buf.String())
	}

	logger.Complete("Hadolint", "[PASS]", 120, "")
	output := buf.String()

	if strings.Contains(output, "[IN_PROGRESS]") || strings.Contains(output, "\x1b") {
		t.Errorf("expected no TTY codes on non-TTY output, got: %q", output)
	}
	if !strings.Contains(output, "  [PASS] Hadolint (120ms)") {
		t.Errorf("expected output to contain final PASS message with prefix, got: %q", output)
	}
}
