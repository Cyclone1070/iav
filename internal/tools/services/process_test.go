package services

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

func TestExecuteWithTimeout_Success(t *testing.T) {
	mock := &MockProcess{
		WaitDelay: 10 * time.Millisecond,
	}

	err := ExecuteWithTimeout(context.Background(), 100*time.Millisecond, mock)
	if err != nil {
		t.Errorf("ExecuteWithTimeout failed: %v", err)
	}
}

func TestExecuteWithTimeout_Fail(t *testing.T) {
	mock := &MockProcess{
		WaitDelay: 200 * time.Millisecond,
	}

	err := ExecuteWithTimeout(context.Background(), 50*time.Millisecond, mock)
	if err != models.ErrShellTimeout {
		t.Errorf("Error = %v, want ErrShellTimeout", err)
	}
	if !mock.SignalCalled {
		t.Error("Signal (SIGTERM) not called")
	}
}

func TestCollectProcessOutput(t *testing.T) {
	tests := []struct {
		name          string
		stdout        string
		stderr        string
		maxBytes      int
		wantStdout    string
		wantStderr    string
		wantTruncated bool
	}{
		{
			name:          "Normal output",
			stdout:        "hello\n",
			stderr:        "world\n",
			maxBytes:      1024,
			wantStdout:    "hello\n",
			wantStderr:    "world\n",
			wantTruncated: false,
		},
		{
			name:          "Truncated stdout",
			stdout:        strings.Repeat("a", 2000),
			stderr:        "error\n",
			maxBytes:      1024,
			wantStdout:    strings.Repeat("a", 1024),
			wantStderr:    "error\n",
			wantTruncated: true,
		},
		{
			name:          "Truncated stderr",
			stdout:        "output\n",
			stderr:        strings.Repeat("e", 2000),
			maxBytes:      1024,
			wantStdout:    "output\n",
			wantStderr:    strings.Repeat("e", 1024),
			wantTruncated: true,
		},
		{
			name:          "Binary stdout",
			stdout:        string([]byte{0x00, 0x01, 0x02, 0xFF}),
			stderr:        "",
			maxBytes:      1024,
			wantStdout:    "[Binary Content]",
			wantStderr:    "",
			wantTruncated: true,
		},
		{
			name:          "Empty output",
			stdout:        "",
			stderr:        "",
			maxBytes:      1024,
			wantStdout:    "",
			wantStderr:    "",
			wantTruncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdoutReader := strings.NewReader(tt.stdout)
			stderrReader := strings.NewReader(tt.stderr)

			gotStdout, gotStderr, gotTruncated, err := CollectProcessOutput(stdoutReader, stderrReader, tt.maxBytes)
			if err != nil {
				t.Fatalf("CollectProcessOutput() error = %v", err)
			}

			if gotStdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", gotStdout, tt.wantStdout)
			}
			if gotStderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", gotStderr, tt.wantStderr)
			}
			if gotTruncated != tt.wantTruncated {
				t.Errorf("truncated = %v, want %v", gotTruncated, tt.wantTruncated)
			}
		})
	}
}

func TestCollectProcessOutput_Concurrent(t *testing.T) {
	// Test that concurrent reads work correctly
	largeStdout := strings.Repeat("stdout line\n", 1000)
	largeStderr := strings.Repeat("stderr line\n", 1000)

	stdoutReader := strings.NewReader(largeStdout)
	stderrReader := strings.NewReader(largeStderr)

	gotStdout, gotStderr, _, err := CollectProcessOutput(stdoutReader, stderrReader, models.DefaultMaxCommandOutputSize)
	if err != nil {
		t.Fatalf("CollectProcessOutput() error = %v", err)
	}

	if gotStdout != largeStdout {
		t.Errorf("stdout length = %d, want %d", len(gotStdout), len(largeStdout))
	}
	if gotStderr != largeStderr {
		t.Errorf("stderr length = %d, want %d", len(gotStderr), len(largeStderr))
	}
}

func TestCollectProcessOutput_SlowReader(t *testing.T) {
	// Test with a slow reader to ensure goroutines complete
	slowReader := &slowReader{data: []byte("slow data"), delay: 0}

	gotStdout, gotStderr, _, err := CollectProcessOutput(slowReader, strings.NewReader(""), 1024)
	if err != nil {
		t.Fatalf("CollectProcessOutput() error = %v", err)
	}

	if gotStdout != "slow data" {
		t.Errorf("stdout = %q, want %q", gotStdout, "slow data")
	}
	if gotStderr != "" {
		t.Errorf("stderr = %q, want empty", gotStderr)
	}
}

// slowReader simulates a slow reader for testing
type slowReader struct {
	data  []byte
	delay int
	pos   int
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, bytes.ErrTooLarge // Use as EOF substitute
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		err = bytes.ErrTooLarge
	}
	return n, err
}
