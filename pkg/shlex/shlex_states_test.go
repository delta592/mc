package shlex

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// TestSplitStateMachine drives every transition of the lexer state machine:
// bare words, both quote styles, escapes inside and outside quotes, comments
// and the interactions between them.
func TestSplitStateMachine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"only spaces", "   \t\r\n ", nil},
		{"single word", "one", []string{"one"}},
		{"two words", "one two", []string{"one", "two"}},
		{"tabs and newlines separate", "one\ttwo\nthree\rfour", []string{"one", "two", "three", "four"}},
		{"leading and trailing space", "  one  ", []string{"one"}},

		// Double quotes: escaping is honoured inside.
		{"double quoted", `"one two"`, []string{"one two"}},
		{"double quoted empty", `""`, []string{""}},
		{"double quoted adjacent", `"one"two`, []string{"onetwo"}},
		{"double quote starts word", `a"b c"`, []string{"ab c"}},
		{"escape inside double quotes", `"a\"b"`, []string{`a"b`}},
		{"backslash inside double quotes", `"a\\b"`, []string{`a\b`}},
		{"single quote inside double quotes", `"it's"`, []string{"it's"}},
		{"comment char inside double quotes", `"a#b"`, []string{"a#b"}},

		// Single quotes: nothing is special inside.
		{"single quoted", `'one two'`, []string{"one two"}},
		{"single quoted empty", `''`, []string{""}},
		{"backslash inside single quotes", `'a\b'`, []string{`a\b`}},
		{"double quote inside single quotes", `'a"b'`, []string{`a"b`}},
		{"comment char inside single quotes", `'a#b'`, []string{"a#b"}},
		{"single quoted adjacent", `'one'two`, []string{"onetwo"}},

		// Escapes outside quotes.
		{"escaped space", `one\ two`, []string{"one two"}},
		{"escaped quote", `\"`, []string{`"`}},
		{"escaped backslash", `a\\b`, []string{`a\b`}},
		{"escape starts word", `\ x`, []string{" x"}},
		{"escaped comment char", `a\#b`, []string{"a#b"}},

		// Comments.
		{"comment only", "#comment", nil},
		{"comment after word", "one #comment", []string{"one"}},
		{"comment terminated by newline", "one #comment\ntwo", []string{"one", "two"}},
		{"hash inside word is literal", "one#two", []string{"one#two"}},

		// Mixed.
		{"mixed quoting", `a "b c" 'd e' f\ g`, []string{"a", "b c", "d e", "f g"}},
		{"quotes concatenated", `"a"'b'c`, []string{"abc"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Split(tc.in)
			if err != nil {
				t.Fatalf("Split(%q) error = %v", tc.in, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("Split(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("Split(%q) = %#v, want %#v", tc.in, got, tc.want)
				}
			}
		})
	}
}

// TestSplitUnterminatedInput covers the eofRuneClass branch of each state that
// treats a premature EOF as an error.
func TestSplitUnterminatedInput(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{"unterminated double quote", `"abc`, "closing quote"},
		{"unterminated single quote", `'abc`, "closing quote"},
		{"trailing escape", `abc\`, "escape character"},
		{"trailing escape in double quotes", `"abc\`, "escape character"},
		{"escape only", `\`, "escape character"},
		{"double quote only", `"`, "closing quote"},
		{"single quote only", `'`, "closing quote"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Split(tc.in)
			if err == nil {
				t.Fatalf("Split(%q) succeeded, want an error", tc.in)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Split(%q) error = %v, want it to mention %q", tc.in, err, tc.wantErr)
			}
		})
	}
}

// TestTokenizerEmitsTypedTokens checks the tokenizer reports comment tokens,
// which Split and Lexer both discard.
func TestTokenizerEmitsTypedTokens(t *testing.T) {
	tk := NewTokenizer(strings.NewReader("word #comment\nother"))

	want := []Token{
		{tokenType: WordToken, value: "word"},
		{tokenType: CommentToken, value: "comment"},
		{tokenType: WordToken, value: "other"},
	}
	for i, w := range want {
		got, err := tk.Next()
		if err != nil {
			t.Fatalf("Next() #%d error = %v", i, err)
		}
		if !got.Equal(&w) {
			t.Fatalf("Next() #%d = %+v, want %+v", i, *got, w)
		}
	}
	if _, err := tk.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("Next() at end = %v, want io.EOF", err)
	}
}

func TestTokenizerCommentAtEOF(t *testing.T) {
	tk := NewTokenizer(strings.NewReader("#trailing"))
	got, err := tk.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	want := Token{tokenType: CommentToken, value: "trailing"}
	if !got.Equal(&want) {
		t.Fatalf("Next() = %+v, want %+v", *got, want)
	}
}

func TestLexerSkipsComments(t *testing.T) {
	l := NewLexer(strings.NewReader("#lead\nword #trail\nlast"))
	var got []string
	for {
		w, err := l.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		got = append(got, w)
	}
	if len(got) != 2 || got[0] != "word" || got[1] != "last" {
		t.Fatalf("Lexer produced %#v, want [word last]", got)
	}
}

func TestLexerPropagatesError(t *testing.T) {
	l := NewLexer(strings.NewReader(`"unterminated`))
	if _, err := l.Next(); err == nil {
		t.Fatal("Next() succeeded on unterminated input, want an error")
	}
}

func TestTokenEqual(t *testing.T) {
	a := &Token{tokenType: WordToken, value: "x"}
	tests := []struct {
		name string
		b    *Token
		want bool
	}{
		{"identical", &Token{tokenType: WordToken, value: "x"}, true},
		{"different value", &Token{tokenType: WordToken, value: "y"}, false},
		{"different type", &Token{tokenType: CommentToken, value: "x"}, false},
		{"nil", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := a.Equal(tc.b); got != tc.want {
				t.Fatalf("Equal() = %v, want %v", got, tc.want)
			}
		})
	}

	var nilToken *Token
	if nilToken.Equal(a) {
		t.Fatal("(*Token)(nil).Equal() = true, want false")
	}
}

func TestClassifyRune(t *testing.T) {
	c := newDefaultClassifier()
	tests := []struct {
		in   rune
		want runeTokenClass
	}{
		{'a', unknownRuneClass},
		{' ', spaceRuneClass},
		{'\t', spaceRuneClass},
		{'\n', spaceRuneClass},
		{'\r', spaceRuneClass},
		{'"', escapingQuoteRuneClass},
		{'\'', nonEscapingQuoteRuneClass},
		{'\\', escapeRuneClass},
		{'#', commentRuneClass},
	}
	for _, tc := range tests {
		if got := c.ClassifyRune(tc.in); got != tc.want {
			t.Errorf("ClassifyRune(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestSplitUnicode checks multi-byte runes survive the byte-oriented reader.
func TestSplitUnicode(t *testing.T) {
	got, err := Split(`héllo "wörld → ok" 日本語`)
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	want := []string{"héllo", "wörld → ok", "日本語"}
	if len(got) != len(want) {
		t.Fatalf("Split() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Split() = %#v, want %#v", got, want)
		}
	}
}

// errReader fails partway so the tokenizer surfaces a read error rather than
// treating it as EOF.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestTokenizerReadError(t *testing.T) {
	want := errors.New("read failed")
	tk := NewTokenizer(errReader{err: want})
	if _, err := tk.Next(); err == nil {
		t.Fatal("Next() succeeded on a failing reader, want an error")
	}
}
