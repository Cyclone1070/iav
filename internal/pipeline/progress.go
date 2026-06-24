package pipeline

import (
	"fmt"
	"io"
	"os"

	"github.com/moby/term"
)

var isTerminal = func(w io.Writer) (bool, uintptr) {
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err == nil {
			return (stat.Mode() & os.ModeCharDevice) != 0, f.Fd()
		}
	}
	return false, 0
}

type ProgressLogger struct {
	out    io.Writer
	tty    bool
	fd     uintptr
	prefix string
}

func NewProgressLogger(out io.Writer, prefix string) *ProgressLogger {
	tty, fd := isTerminal(out)
	return &ProgressLogger{
		out:    out,
		tty:    tty,
		fd:     fd,
		prefix: prefix,
	}
}

func (p *ProgressLogger) Start(stageName string) {
	if !p.tty || p.out == nil {
		return
	}
	msg := fmt.Sprintf("%s[IN_PROGRESS] %s...", p.prefix, stageName)
	_, _ = fmt.Fprint(p.out, msg)
}

func (p *ProgressLogger) Complete(stageName string, status string, durationMs int64, errStr string) {
	if p.out == nil {
		return
	}

	if errStr != "" {
		errStr = " - " + errStr
	}
	finalMsg := fmt.Sprintf("%s%s %s (%dms)%s\n", p.prefix, status, stageName, durationMs, errStr)

	if !p.tty {
		_, _ = fmt.Fprint(p.out, finalMsg)
		return
	}

	msg := fmt.Sprintf("%s[IN_PROGRESS] %s...", p.prefix, stageName)
	width := 80
	if ws, err := term.GetWinsize(p.fd); err == nil && ws.Width > 0 {
		width = int(ws.Width)
	}

	rows := (len(msg) + width - 1) / width

	// 1. Move cursor to the first row of the wrapped message
	if rows > 1 {
		_, _ = fmt.Fprintf(p.out, "\x1b[%dA", rows-1)
	}

	// 2. Clear all rows from top to bottom
	for r := range rows {
		_, _ = fmt.Fprint(p.out, "\r\x1b[2K")
		if r < rows-1 {
			_, _ = fmt.Fprint(p.out, "\x1b[1B") // Move down
		}
	}

	// 3. Move cursor back to the first row to write the final message
	if rows > 1 {
		_, _ = fmt.Fprintf(p.out, "\x1b[%dA\r", rows-1)
	} else {
		_, _ = fmt.Fprint(p.out, "\r")
	}

	// 4. Print final message (ends in newline)
	_, _ = fmt.Fprint(p.out, finalMsg)
}
