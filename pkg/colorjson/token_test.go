package colorjson

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

// tokenize drains dec.Token until EOF and returns everything it produced.
func tokenize(t *testing.T, dec *Decoder) []Token {
	t.Helper()
	var got []Token
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatalf("Token() error = %v (after %v)", err, got)
		}
		got = append(got, tok)
	}
}

func TestTokenNestedObjectAndArray(t *testing.T) {
	const in = `{"a":[1,"two",true,null],"b":{"c":3.5}}`
	got := tokenize(t, NewDecoder(strings.NewReader(in)))
	want := []Token{
		Delim('{'),
		"a", Delim('['), float64(1), "two", true, nil, Delim(']'),
		"b", Delim('{'), "c", 3.5, Delim('}'),
		Delim('}'),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Token() stream =\n  %#v\nwant\n  %#v", got, want)
	}
}

func TestTokenTopLevelScalars(t *testing.T) {
	got := tokenize(t, NewDecoder(strings.NewReader(` 1 "x" false null `)))
	want := []Token{float64(1), "x", false, nil}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Token() = %#v, want %#v", got, want)
	}
}

func TestTokenEmptyContainers(t *testing.T) {
	got := tokenize(t, NewDecoder(strings.NewReader(`[] {} [[]] [{}]`)))
	want := []Token{
		Delim('['), Delim(']'),
		Delim('{'), Delim('}'),
		Delim('['), Delim('['), Delim(']'), Delim(']'),
		Delim('['), Delim('{'), Delim('}'), Delim(']'),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Token() = %#v, want %#v", got, want)
	}
}

func TestDelimString(t *testing.T) {
	for _, d := range []Delim{'[', ']', '{', '}'} {
		if got, want := d.String(), string(rune(d)); got != want {
			t.Fatalf("Delim(%q).String() = %q, want %q", rune(d), got, want)
		}
	}
}

// TestTokenErrorStates drives tokenError through each tokenState branch so the
// context string in the resulting SyntaxError is exercised.
func TestTokenErrorStates(t *testing.T) {
	// Token elides commas and colons, so ntokens counts only the values and
	// delimiters it actually yields.
	tests := []struct {
		name       string
		in         string
		ntokens    int // tokens to read successfully before the error
		want       string
		wantOffset int64
	}{
		{"top level", `}`, 0, "looking for beginning of value", 0},
		{"array start", `[}`, 1, "looking for beginning of value", 1},
		{"after array element", `[1}`, 2, "after array element", 2},
		{"object key", `{"a":1,]`, 3, "looking for beginning of object key string", 7},
		{"after object key", `{"a"]`, 2, "after object key", 4},
		{"after key value pair", `{"a":1]`, 3, "after object key:value pair", 6},
		{"object value", `{"a":]`, 2, "looking for beginning of value", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec := NewDecoder(strings.NewReader(tc.in))
			for i := 0; i < tc.ntokens; i++ {
				if _, err := dec.Token(); err != nil {
					t.Fatalf("Token() #%d error = %v, want success", i, err)
				}
			}
			_, err := dec.Token()
			var syn *SyntaxError
			if !errors.As(err, &syn) {
				t.Fatalf("Token() error = %T(%v), want *SyntaxError", err, err)
			}
			if !strings.Contains(syn.Error(), tc.want) {
				t.Fatalf("Token() error = %q, want it to contain %q", syn, tc.want)
			}
			if syn.Offset != tc.wantOffset {
				t.Fatalf("SyntaxError.Offset = %d, want %d", syn.Offset, tc.wantOffset)
			}
		})
	}
}

// TestTokenMismatchedDelim checks Token's guarantee that [ ] { } stay matched.
func TestTokenMismatchedDelim(t *testing.T) {
	for _, in := range []string{`[}`, `{]`, `[1,2}`, `{"a":1]`} {
		dec := NewDecoder(strings.NewReader(in))
		var err error
		for err == nil {
			_, err = dec.Token()
		}
		if errors.Is(err, io.EOF) {
			t.Fatalf("Token(%q) reached EOF, want a mismatched-delimiter error", in)
		}
	}
}

// TestTokenDecodeInterleaved mixes Token and Decode, which is the path through
// tokenPrepareForDecode.
func TestTokenDecodeInterleaved(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`[{"n":1},{"n":2}]`))

	if tok, err := dec.Token(); err != nil || tok != Delim('[') {
		t.Fatalf("Token() = %v, %v, want '['", tok, err)
	}

	var seen []int
	for dec.More() {
		var v struct {
			N int `json:"n"`
		}
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		seen = append(seen, v.N)
	}
	if !reflect.DeepEqual(seen, []int{1, 2}) {
		t.Fatalf("decoded %v, want [1 2]", seen)
	}

	if tok, err := dec.Token(); err != nil || tok != Delim(']') {
		t.Fatalf("Token() = %v, %v, want ']'", tok, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		t.Fatalf("Token() error = %v, want io.EOF", err)
	}
}

// TestDecodeObjectValueViaToken walks into an object with Token and then
// decodes the value, going through the tokenObjectColon branch.
func TestDecodeObjectValueViaToken(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"key":{"inner":7}}`))
	if _, err := dec.Token(); err != nil { // {
		t.Fatal(err)
	}
	tok, err := dec.Token() // "key"
	if err != nil {
		t.Fatal(err)
	}
	if tok != "key" {
		t.Fatalf("Token() = %v, want \"key\"", tok)
	}
	var v map[string]int
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if v["inner"] != 7 {
		t.Fatalf("Decode() = %v, want inner=7", v)
	}
}

// TestDecodeNotAtBeginningOfValue triggers the "not at beginning of value"
// guard by calling Decode while the decoder expects an object key.
func TestDecodeNotAtBeginningOfValue(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":1}`))
	if _, err := dec.Token(); err != nil { // {
		t.Fatal(err)
	}
	var v int
	err := dec.Decode(&v)
	var syn *SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("Decode() error = %T(%v), want *SyntaxError", err, err)
	}
	if !strings.Contains(syn.Error(), "not at beginning of value") {
		t.Fatalf("Decode() error = %q", syn)
	}
}

// TestTokenPrepareForDecodeErrors covers the malformed-separator paths in
// tokenPrepareForDecode: a missing comma between array elements and a missing
// colon after an object key.
func TestTokenPrepareForDecodeErrors(t *testing.T) {
	t.Run("expected comma", func(t *testing.T) {
		dec := NewDecoder(strings.NewReader(`[1 2]`))
		if _, err := dec.Token(); err != nil { // [
			t.Fatal(err)
		}
		var n int
		if err := dec.Decode(&n); err != nil { // 1
			t.Fatal(err)
		}
		err := dec.Decode(&n)
		if err == nil || !strings.Contains(err.Error(), "expected comma after array element") {
			t.Fatalf("Decode() error = %v, want 'expected comma after array element'", err)
		}
	})

	t.Run("expected colon", func(t *testing.T) {
		dec := NewDecoder(strings.NewReader(`{"a" 1}`))
		if _, err := dec.Token(); err != nil { // {
			t.Fatal(err)
		}
		if _, err := dec.Token(); err != nil { // "a"
			t.Fatal(err)
		}
		var n int
		err := dec.Decode(&n)
		if err == nil || !strings.Contains(err.Error(), "expected colon after object key") {
			t.Fatalf("Decode() error = %v, want 'expected colon after object key'", err)
		}
	})
}

func TestTokenTruncatedInput(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":`))
	if _, err := dec.Token(); err != nil { // {
		t.Fatal(err)
	}
	if _, err := dec.Token(); err != nil { // "a"
		t.Fatal(err)
	}
	if _, err := dec.Token(); err == nil {
		t.Fatal("Token() on truncated input succeeded, want an error")
	}
}

// TestDecoderErrorIsSticky verifies a failed Decode latches dec.err.
func TestDecoderErrorIsSticky(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{invalid}`))
	var v map[string]any
	first := dec.Decode(&v)
	if first == nil {
		t.Fatal("Decode() succeeded on invalid input")
	}
	if second := dec.Decode(&v); second == nil {
		t.Fatal("second Decode() succeeded, want the latched error")
	}
}

func TestDecoderUnexpectedEOF(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":1`))
	var v map[string]int
	err := dec.Decode(&v)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Decode() error = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecoderEOFOnEmptyInput(t *testing.T) {
	dec := NewDecoder(strings.NewReader(""))
	var v any
	if err := dec.Decode(&v); !errors.Is(err, io.EOF) {
		t.Fatalf("Decode() error = %v, want io.EOF", err)
	}
}

// errReader fails after handing out its payload, so the decoder observes a
// non-EOF read error mid-stream.
type errReader struct {
	data []byte
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestDecoderReadError(t *testing.T) {
	want := errors.New("boom")
	dec := NewDecoder(&errReader{data: []byte(`{"a":`), err: want})
	var v map[string]int
	if err := dec.Decode(&v); !errors.Is(err, want) {
		t.Fatalf("Decode() error = %v, want %v", err, want)
	}
}

// failWriter rejects every write so Encoder.Encode latches a write error.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestEncoderWriteError(t *testing.T) {
	want := errors.New("write failed")
	enc := NewEncoder(failWriter{err: want})
	if err := enc.Encode(map[string]int{"a": 1}); !errors.Is(err, want) {
		t.Fatalf("Encode() error = %v, want %v", err, want)
	}
	// The error is sticky.
	if err := enc.Encode(1); !errors.Is(err, want) {
		t.Fatalf("second Encode() error = %v, want %v", err, want)
	}
}

func TestDecoderBufferedRemainder(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":1}trailing`))
	var v map[string]int
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	rest, err := io.ReadAll(dec.Buffered())
	if err != nil {
		t.Fatal(err)
	}
	if string(rest) != "trailing" {
		t.Fatalf("Buffered() = %q, want %q", rest, "trailing")
	}
}

func TestRawMessageNilMarshal(t *testing.T) {
	var m RawMessage
	b, err := m.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "null" {
		t.Fatalf("RawMessage(nil).MarshalJSON() = %s, want null", b)
	}
}

func TestRawMessageNilUnmarshalJSON(t *testing.T) {
	var m *RawMessage
	if err := m.UnmarshalJSON([]byte(`1`)); err == nil {
		t.Fatal("UnmarshalJSON on nil *RawMessage succeeded, want an error")
	}
}

// TestDecoderLargeInputRefill pushes enough data through the decoder to force
// several refill/grow cycles on the internal buffer.
func TestDecoderLargeInputRefill(t *testing.T) {
	var sb strings.Builder
	sb.WriteByte('[')
	const n = 4000
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"name":"a-fairly-long-string-value-to-force-buffer-growth"}`)
	}
	sb.WriteByte(']')

	dec := NewDecoder(strings.NewReader(sb.String()))
	if _, err := dec.Token(); err != nil {
		t.Fatal(err)
	}
	count := 0
	for dec.More() {
		var v map[string]string
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("Decode() #%d error = %v", count, err)
		}
		count++
	}
	if count != n {
		t.Fatalf("decoded %d elements, want %d", count, n)
	}
}
