package textutil

import (
	"strings"
	"testing"
)

func TestTruncateWithTail(t *testing.T) {
	tests := []struct {
		s     string
		width uint
		tail  string
		want  string
	}{
		{"hello world", 8, "...", "hello..."},
		{"short", 10, "...", "short"},
		{"hello", 2, "...", "..."},
		{"\x1b[31mred\x1b[0m text", 6, "..", "\x1b[31mred\x1b[0m .."},
	}
	for _, tt := range tests {
		got := TruncateWithTail(tt.s, tt.width, tt.tail)
		if got != tt.want {
			t.Errorf("TruncateWithTail(%q, %d, %q) = %q, want %q", tt.s, tt.width, tt.tail, got, tt.want)
		}
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		s     string
		limit int
		want  string
	}{
		{"hello world", 0, "hello world"},
		{"hello world", 20, "hello world"},
		{"one two three", 5, "one\ntwo\nthree"},
		{"line-one\nline-two", 80, "line-one\nline-two"},
	}
	for _, tt := range tests {
		got := WordWrap(tt.s, tt.limit)
		if got != tt.want {
			t.Errorf("WordWrap(%q, %d) = %q, want %q", tt.s, tt.limit, got, tt.want)
		}
	}
}

func TestWordWrapHyphenBreak(t *testing.T) {
	got := WordWrap("long-word-here", 5)
	if !strings.Contains(got, "-") {
		t.Fatalf("WordWrap() = %q, expected hyphen breakpoint", got)
	}
}
