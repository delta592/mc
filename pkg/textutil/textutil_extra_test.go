package textutil

import (
	"strings"
	"testing"
)

const (
	red   = "\x1b[31m"
	reset = "\x1b[0m"
)

// TestWordWrapPreservesANSI covers the ANSI-marker branches of writeString:
// escape sequences must not count toward the printable width, and must survive
// wrapping intact.
func TestWordWrapPreservesANSI(t *testing.T) {
	in := red + "hello" + reset + " " + red + "world" + reset
	got := WordWrap(in, 5)

	if !strings.Contains(got, red) || !strings.Contains(got, reset) {
		t.Fatalf("WordWrap() = %q, want the ANSI sequences preserved", got)
	}
	// Both words are exactly the limit, so they land on separate lines even
	// though the escape sequences add bytes.
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("WordWrap() = %q, want exactly one line break", got)
	}
}

func TestWordWrapANSIDoesNotCountTowardWidth(t *testing.T) {
	plain := WordWrap("abc def", 7)
	colored := WordWrap(red+"abc"+reset+" "+red+"def"+reset, 7)

	if strings.Count(plain, "\n") != strings.Count(colored, "\n") {
		t.Fatalf("colored wrap = %q (%d breaks), plain wrap = %q (%d breaks); ANSI must not affect width",
			colored, strings.Count(colored, "\n"), plain, strings.Count(plain, "\n"))
	}
}

// TestWordWrapKeepsNewlines covers the newline branch: by default WordWrap
// honors existing line breaks in the input.
func TestWordWrapKeepsNewlines(t *testing.T) {
	got := WordWrap("one\ntwo", 40)
	if got != "one\ntwo" {
		t.Fatalf("WordWrap() = %q, want the existing newline kept", got)
	}
}

func TestWordWrapNewlineAfterSpace(t *testing.T) {
	// A newline arriving while only whitespace is pending exercises the
	// space-flushing branch guarding the line break.
	got := WordWrap("one \ntwo", 40)
	if !strings.Contains(got, "\n") {
		t.Fatalf("WordWrap() = %q, want a line break", got)
	}
	if strings.Contains(got, "\n\n") {
		t.Fatalf("WordWrap() = %q, want no doubled line break", got)
	}
}

func TestWordWrapNewlineWhenLineIsFull(t *testing.T) {
	// The pending space would overflow the limit, so it is dropped rather than
	// written before the break.
	got := WordWrap("abcde \nx", 5)
	if !strings.Contains(got, "\n") {
		t.Fatalf("WordWrap() = %q, want a line break", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("WordWrap() = %q, want no trailing space before the break", got)
		}
	}
}

// TestWordWrapDiscardNewlines covers the !keepNewlines branch, which the public
// WordWrap never selects.
func TestWordWrapDiscardNewlines(t *testing.T) {
	w := &wordWrap{
		limit:        40,
		breakpoints:  defaultBreakpoints,
		newline:      defaultNewline,
		keepNewlines: false,
	}
	w.writeString("  one\ntwo  ")
	w.addWord()

	got := w.buf.String()
	if strings.Contains(got, "\n") {
		t.Fatalf("writeString() = %q, want newlines collapsed to spaces", got)
	}
	if got != "one two" {
		t.Fatalf("writeString() = %q, want %q", got, "one two")
	}
}

func TestWordWrapZeroLimitPassesThrough(t *testing.T) {
	in := "anything at all\nincluding newlines"
	if got := WordWrap(in, 0); got != in {
		t.Fatalf("WordWrap(_, 0) = %q, want the input unchanged", got)
	}
}

func TestWordWrapLongWordIsNotBroken(t *testing.T) {
	// A single word wider than the limit is emitted whole rather than split.
	got := WordWrap("supercalifragilistic", 5)
	if strings.Contains(got, "\n") {
		t.Fatalf("WordWrap() = %q, want an over-long word left intact", got)
	}
}

func TestWordWrapBreakpoint(t *testing.T) {
	got := WordWrap("well-known", 6)
	if !strings.Contains(got, "-") {
		t.Fatalf("WordWrap() = %q, want the hyphen retained", got)
	}
}

func TestInGroup(t *testing.T) {
	if !inGroup([]rune{'a', 'b'}, 'b') {
		t.Fatal("inGroup() = false for a member")
	}
	if inGroup([]rune{'a', 'b'}, 'c') {
		t.Fatal("inGroup() = true for a non-member")
	}
	if inGroup(nil, 'a') {
		t.Fatal("inGroup(nil, _) = true")
	}
}

// TestTruncateResetsANSI covers the branch that emits a reset when truncation
// cuts a string mid-styling, so the escape does not leak into later output.
func TestTruncateResetsANSI(t *testing.T) {
	got := TruncateWithTail(red+"hello world"+reset, 8, "...")
	if !strings.Contains(got, "...") {
		t.Fatalf("TruncateWithTail() = %q, want the tail appended", got)
	}
	// The reset is emitted after the tail so styling does not leak onward.
	if !strings.HasSuffix(got, reset) {
		t.Fatalf("TruncateWithTail() = %q, want it to end with a reset sequence", got)
	}
}

func TestTruncateWithTailBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width uint
		tail  string
		want  string
	}{
		{"shorter than width", "abc", 10, "...", "abc"},
		{"exactly width including tail", "abcde", 8, "...", "abcde"},
		{"tail eats into the budget", "abcde", 5, "...", "ab..."},
		{"width smaller than tail", "abcde", 2, "...", "..."},
		{"longer than width", "abcdefgh", 5, "...", "ab..."},
		{"empty tail", "abcdefgh", 3, "", "abc"},
		{"empty input", "", 5, "...", ""},
		{"zero width", "abc", 0, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TruncateWithTail(tc.in, tc.width, tc.tail); got != tc.want {
				t.Fatalf("TruncateWithTail(%q, %d, %q) = %q, want %q",
					tc.in, tc.width, tc.tail, got, tc.want)
			}
		})
	}
}

func TestTruncateWideRunes(t *testing.T) {
	// CJK runes are two cells wide, so only two fit in a width of 5 alongside
	// a one-cell tail.
	got := TruncateWithTail("日本語です", 5, ".")
	if got == "" {
		t.Fatal("TruncateWithTail() = empty")
	}
	if !strings.HasSuffix(got, ".") {
		t.Fatalf("TruncateWithTail() = %q, want the tail appended", got)
	}
}
