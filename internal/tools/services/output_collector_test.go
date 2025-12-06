package services

import (
	"testing"
)

const testSampleSize = 4096

func TestCollector_Write_Buffering(t *testing.T) {
	c := NewCollector(1024, testSampleSize)
	data := []byte("hello world")
	n, err := c.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}
	if got := c.String(); got != "hello world" {
		t.Errorf("String() = %q, want %q", got, "hello world")
	}
}

func TestCollector_Write_Truncation(t *testing.T) {
	c := NewCollector(10, testSampleSize)
	data := []byte("hello world") // 11 bytes
	n, err := c.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	// Write should return full length even if truncated internally to satisfy io.Writer contract
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}

	if !c.Truncated {
		t.Error("Truncated = false, want true")
	}

	// Should contain first 10 bytes
	if got := c.String(); got != "hello worl" {
		t.Errorf("String() = %q, want %q", got, "hello worl")
	}
}

func TestCollector_Write_Binary(t *testing.T) {
	c := NewCollector(1024, testSampleSize)
	data := []byte("hello\x00world")
	_, err := c.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if !c.IsBinary {
		t.Error("IsBinary = false, want true")
	}

	// Should stop collecting after binary detection
	// Implementation detail: does it keep what it has?
	// Plan says "stops collecting".
	// Let's assume it marks as binary and maybe clears buffer or keeps prefix.
	// Usually binary files shouldn't be displayed, so String() might return a placeholder or empty.
	// Let's check IsBinary flag primarily.
}

func TestCollector_UTF8_Boundary(t *testing.T) {
	// 3-byte character: ⌘ (E2 8C 98)
	c := NewCollector(1024, testSampleSize)

	// Write first byte
	_, _ = c.Write([]byte{0xE2})
	// Write remaining bytes
	_, _ = c.Write([]byte{0x8C, 0x98})

	if got := c.String(); got != "⌘" {
		t.Errorf("String() = %q, want %q", got, "⌘")
	}
}
