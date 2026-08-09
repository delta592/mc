package ansi

import (
	"bytes"
	"testing"
)

func TestIsTerminator(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'m', true},
		{'H', true},
		{'z', true},
		{'0', false},
		{' ', false},
		{'\x1b', false},
	}
	for _, tt := range tests {
		if got := IsTerminator(tt.r); got != tt.want {
			t.Errorf("IsTerminator(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestPrintableRuneWidth(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"hello", 5},
		{"\x1b[31mred\x1b[0m", 3},
		{"", 0},
		{"日本", 4},
	}
	for _, tt := range tests {
		if got := PrintableRuneWidth(tt.s); got != tt.want {
			t.Errorf("PrintableRuneWidth(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestBufferPrintableRuneWidth(t *testing.T) {
	var buf Buffer
	_, _ = buf.WriteString("\x1b[1mbold\x1b[0m")
	if got := buf.PrintableRuneWidth(); got != 4 {
		t.Fatalf("Buffer.PrintableRuneWidth() = %d, want 4", got)
	}
}

func TestWriter(t *testing.T) {
	var out bytes.Buffer
	w := &Writer{Forward: &out}

	_, err := w.Write([]byte("\x1b[31mhello\x1b[0m"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if out.String() != "\x1b[31mhello\x1b[0m" {
		t.Fatalf("Write() output = %q", out.String())
	}

	_, _ = w.Write([]byte("\x1b[32mworld\x1b[0m"))
	seq := w.LastSequence()
	if seq != "" && seq != "\x1b[32m" {
		t.Fatalf("LastSequence() = %q", seq)
	}

	w.ResetAnsi()
	if !bytes.HasSuffix(out.Bytes(), []byte("\x1b[0m")) {
		t.Fatalf("ResetAnsi() did not append reset sequence, got %q", out.String())
	}
}

func TestWriterResetAnsiNoChange(t *testing.T) {
	var out bytes.Buffer
	w := &Writer{Forward: &out}
	before := out.Len()
	w.ResetAnsi()
	if out.Len() != before {
		t.Fatalf("ResetAnsi() with seqchanged=false wrote output")
	}
}
