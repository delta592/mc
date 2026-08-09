package colorjson

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/minio/pkg/v3/console"
)

// withColor forces colorized output for the duration of fn. console.Colorize
// normally no-ops unless stdout is a terminal, which it never is under `go
// test`, so the encoder's color paths are only reachable by swapping it out.
// The hook is process-global, so tests using it must not run in parallel.
func withColor(t *testing.T, fn func()) {
	t.Helper()
	saved := console.Colorize
	console.Colorize = func(tag string, data interface{}) string {
		return "\x1b[32m" + fmt.Sprint(data) + "\x1b[0m"
	}
	t.Cleanup(func() { console.Colorize = saved })
	fn()
}

// TestColorizedOutputIsScannable is the property this fork exists for: values
// encoded with ANSI colors must still be accepted by the scanner, so Valid,
// Compact and Indent keep working on colorized output.
func TestColorizedOutputIsScannable(t *testing.T) {
	withColor(t, func() {
		v := map[string]any{
			"str":  "value",
			"num":  42,
			"bool": true,
			"null": nil,
			"list": []any{1, "two", false, nil},
			"obj":  map[string]any{"nested": 1.5},
		}
		b, err := Marshal(v)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if !bytes.Contains(b, []byte{0x1b}) {
			t.Fatalf("Marshal() produced no ANSI escapes with color on: %q", b)
		}
		if !Valid(b) {
			t.Fatalf("Valid() = false for colorized output %q", b)
		}

		var indented bytes.Buffer
		if err := Indent(&indented, b, "", "  "); err != nil {
			t.Fatalf("Indent() error = %v on colorized output %q", err, b)
		}
		if !strings.Contains(indented.String(), "\n") {
			t.Fatalf("Indent() did not indent: %q", indented.String())
		}

		var compacted bytes.Buffer
		if err := Compact(&compacted, indented.Bytes()); err != nil {
			t.Fatalf("Compact() error = %v", err)
		}
		if !bytes.Equal(compacted.Bytes(), b) {
			t.Fatalf("Indent+Compact round-trip = %q, want %q", compacted.Bytes(), b)
		}
	})
}

func TestColorizedEncoderOutputIsScannable(t *testing.T) {
	withColor(t, func() {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"a": []int{1, 2}, "b": "x"}); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		if !Valid(bytes.TrimSpace(buf.Bytes())) {
			t.Fatalf("Valid() = false for colorized encoder output %q", buf.String())
		}
	})
}

// TestScannerAcceptsColorEscapes covers the color-aware scanner states that the
// fork adds on top of encoding/json: an ESC before a value (stateBeginColorESC
// and stateBeginColorRest), a bare ESC inside a string, and a backslash-escaped
// color run inside a string (stateInStringColorRest).
func TestScannerAcceptsColorEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"colored number in array", "[\x1b[32m1\x1b[0m]"},
		{"colored string in array", "[\x1b[32m\"s\"\x1b[0m]"},
		{"multi-parameter color", "[\x1b[1;32m\"s\"\x1b[0m]"},
		{"colored object value", "{\"k\":\x1b[32m1\x1b[0m}"},
		{"colored literals", "[\x1b[31mtrue\x1b[0m,\x1b[31mfalse\x1b[0m,\x1b[1mnull\x1b[0m]"},
		{"several colored elements", "[\x1b[32m1\x1b[0m,\x1b[31mnull\x1b[0m]"},
		{"bare ESC inside string", "[\"a\x1b[32mb\"]"},
		{"escaped color inside string", "[\"a\\[32mb\"]"},
		{"nested colored containers", "{\"o\":{\"a\":[\x1b[32m1\x1b[0m]}}"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !Valid([]byte(tc.in)) {
				t.Fatalf("Valid(%q) = false, want true", tc.in)
			}
			var compacted bytes.Buffer
			if err := Compact(&compacted, []byte(tc.in)); err != nil {
				t.Fatalf("Compact(%q) error = %v", tc.in, err)
			}
			var indented bytes.Buffer
			if err := Indent(&indented, []byte(tc.in), "", "\t"); err != nil {
				t.Fatalf("Indent(%q) error = %v", tc.in, err)
			}
		})
	}
}

// TestScannerRejectsMalformedColorEscapes covers the error returns in the
// color states.
func TestScannerRejectsMalformedColorEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"ESC not followed by bracket", "[\x1bX1]"},
		{"bad char in color parameters", "[\x1b[3Xm1]"},
		{"unterminated in-string color", "[\"a\\[9\"]"},
		{"bad char after in-string color", "[\"a\\[9X\"]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if Valid([]byte(tc.in)) {
				t.Fatalf("Valid(%q) = true, want false", tc.in)
			}
			var buf bytes.Buffer
			err := Compact(&buf, []byte(tc.in))
			if err == nil {
				t.Fatalf("Compact(%q) succeeded, want an error", tc.in)
			}
			if !strings.Contains(err.Error(), "color escape code") {
				t.Fatalf("Compact(%q) error = %v, want a color escape code error", tc.in, err)
			}
		})
	}
}

// TestScannerSyntaxErrors walks the non-color error branches of the scanner.
func TestScannerSyntaxErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"bare identifier", `tru`},
		{"misspelled true", `trux`},
		{"misspelled false", `falx`},
		{"misspelled null", `nulx`},
		{"leading zero", `01`},
		{"bare minus", `-`},
		{"minus then letter", `-x`},
		{"trailing dot", `1.`},
		{"dot then letter", `1.x`},
		{"exponent without digits", `1e`},
		{"exponent sign without digits", `1e+`},
		{"exponent letter", `1e+x`},
		{"unclosed object", `{`},
		{"unclosed array", `[`},
		{"missing value", `{"a":}`},
		{"missing colon", `{"a" 1}`},
		{"unquoted key", `{a:1}`},
		{"trailing comma in object", `{"a":1,}`},
		{"trailing comma in array", `[1,]`},
		{"double comma", `[1,,2]`},
		{"control char in string", "\"a\x01b\""},
		{"bad string escape", `"a\qb"`},
		{"bad unicode escape", `"\u00zz"`},
		{"trailing garbage", `{} x`},
		{"only comma", `,`},
		{"close mismatch", `[1}`},
		{"empty", ``},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if Valid([]byte(tc.in)) {
				t.Fatalf("Valid(%q) = true, want false", tc.in)
			}
			var buf bytes.Buffer
			if err := Compact(&buf, []byte(tc.in)); err == nil {
				t.Fatalf("Compact(%q) succeeded, want an error", tc.in)
			}
			var ibuf bytes.Buffer
			if err := Indent(&ibuf, []byte(tc.in), "", " "); err == nil {
				t.Fatalf("Indent(%q) succeeded, want an error", tc.in)
			}
		})
	}
}

// TestScannerValidNumbers exercises the numeric state machine.
func TestScannerValidNumbers(t *testing.T) {
	for _, in := range []string{
		`0`, `-0`, `1`, `-1`, `123`, `0.5`, `-0.5`, `1.25`,
		`1e5`, `1E5`, `1e+5`, `1e-5`, `1.5e10`, `-1.5E-10`, `0e0`,
	} {
		if !Valid([]byte(in)) {
			t.Fatalf("Valid(%q) = false, want true", in)
		}
		var v any
		if err := Unmarshal([]byte(in), &v); err != nil {
			t.Fatalf("Unmarshal(%q) error = %v", in, err)
		}
	}
}

func TestCompactEscapesHTMLAndSeparators(t *testing.T) {
	var buf bytes.Buffer
	HTMLEscape(&buf, []byte(`{"a":"<b>&c"}`))
	got := buf.String()
	for _, want := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if !strings.Contains(got, want) {
			t.Fatalf("HTMLEscape() = %q, missing %q", got, want)
		}
	}
	if strings.ContainsAny(got, "<>&") {
		t.Fatalf("HTMLEscape() = %q, want no raw HTML metacharacters left", got)
	}

	// U+2028 LINE SEPARATOR and U+2029 PARAGRAPH SEPARATOR are rewritten too.
	var sep bytes.Buffer
	HTMLEscape(&sep, []byte("\"\u2028\u2029\""))
	for _, want := range []string{`\u2028`, `\u2029`} {
		if !strings.Contains(sep.String(), want) {
			t.Fatalf("HTMLEscape() = %q, missing %q", sep.String(), want)
		}
	}
}

func TestIndentPreservesWhitespaceHandling(t *testing.T) {
	src := []byte("{\n\t\"a\" : [ 1 , 2 ] ,\n\"b\":{}\n}")

	// A non-whitespace prefix is emitted verbatim on each new line.
	var prefixed bytes.Buffer
	if err := Indent(&prefixed, src, ">", "--"); err != nil {
		t.Fatalf("Indent() error = %v", err)
	}
	if !strings.Contains(prefixed.String(), ">--") {
		t.Fatalf("Indent() = %q, want the prefix and indent applied", prefixed.String())
	}

	// With whitespace-only formatting, Compact must undo Indent exactly.
	var indented bytes.Buffer
	if err := Indent(&indented, src, "", "  "); err != nil {
		t.Fatalf("Indent() error = %v", err)
	}
	var compacted bytes.Buffer
	if err := Compact(&compacted, indented.Bytes()); err != nil {
		t.Fatalf("Compact() error = %v", err)
	}
	if got := compacted.String(); got != `{"a":[1,2],"b":{}}` {
		t.Fatalf("Compact(Indent(src)) = %q", got)
	}
}

// TestIndentEmptyContainers checks the "elide the newline for empty
// object/array" branch of Indent.
func TestIndentEmptyContainers(t *testing.T) {
	var buf bytes.Buffer
	if err := Indent(&buf, []byte(`{"a":{},"b":[],"c":[[]]}`), "", "  "); err != nil {
		t.Fatalf("Indent() error = %v", err)
	}
	if strings.Contains(buf.String(), "{\n\n") || strings.Contains(buf.String(), "[\n\n") {
		t.Fatalf("Indent() emitted a blank line for an empty container: %q", buf.String())
	}
	var out map[string]any
	if err := Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("Unmarshal(indented) error = %v", err)
	}
}

// TestCompactAndIndentLeaveDestinationIntactOnError verifies both helpers
// truncate their partial output when the source turns out to be invalid.
func TestCompactAndIndentLeaveDestinationIntactOnError(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("prefix")
	if err := Compact(&buf, []byte(`{"a":1,`)); err == nil {
		t.Fatal("Compact() succeeded on truncated input")
	}
	if buf.String() != "prefix" {
		t.Fatalf("Compact() left %q in dst, want the original %q", buf.String(), "prefix")
	}

	var ibuf bytes.Buffer
	ibuf.WriteString("prefix")
	if err := Indent(&ibuf, []byte(`{"a":1,`), "", "  "); err == nil {
		t.Fatal("Indent() succeeded on truncated input")
	}
	if ibuf.String() != "prefix" {
		t.Fatalf("Indent() left %q in dst, want the original %q", ibuf.String(), "prefix")
	}
}

// TestScannerErrorIsSticky checks that once the scanner errors it keeps
// returning scanError through stateError.
func TestScannerErrorIsSticky(t *testing.T) {
	var s scanner
	s.reset()
	if got := s.step(&s, 'x'); got != scanError {
		t.Fatalf("step('x') = %d, want scanError", got)
	}
	// Every subsequent byte must stay in the error state.
	for _, c := range []byte(`{"a":1}`) {
		if got := s.step(&s, c); got != scanError {
			t.Fatalf("step(%q) after error = %d, want scanError", c, got)
		}
	}
	if s.err == nil {
		t.Fatal("scanner.err = nil after an error")
	}
}

// TestScannerDeepNesting drives the parse-state stack well past its initial
// capacity so pushParseState's growth path is exercised. This fork predates the
// stdlib's maxNestingDepth cap, so deep input is accepted rather than rejected.
func TestScannerDeepNesting(t *testing.T) {
	const depth = 5000
	deep := strings.Repeat("[", depth) + strings.Repeat("]", depth)
	if !Valid([]byte(deep)) {
		t.Fatalf("Valid() = false for %d levels of nesting, want true", depth)
	}

	// Unbalanced nesting is still an error.
	if Valid([]byte(strings.Repeat("[", depth))) {
		t.Fatal("Valid() = true for unterminated nesting, want false")
	}
}
