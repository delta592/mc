package colorjson

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMarshalUnmarshalBasicTypes(t *testing.T) {
	tests := []any{
		nil,
		true,
		false,
		float64(42),
		"hello",
		[]any{1, 2, 3},
		map[string]any{"a": 1, "b": "two"},
	}
	for _, tc := range tests {
		data, err := Marshal(tc)
		if err != nil {
			t.Fatalf("Marshal(%v) error = %v", tc, err)
		}
		if !Valid(data) {
			t.Fatalf("Valid(%q) = false", data)
		}
		var got any
		if err := Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(%q) error = %v", data, err)
		}
	}
}

type sampleStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count,omitempty"`
	Skip  string `json:"-"`
}

func TestMarshalStruct(t *testing.T) {
	in := sampleStruct{Name: "mc", Count: 0, Skip: "hidden"}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hidden") {
		t.Fatalf("Marshal() included skipped field: %s", data)
	}
	var out sampleStruct
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "mc" || out.Skip != "" {
		t.Fatalf("Unmarshal() = %+v", out)
	}
}

func TestUnmarshalErrors(t *testing.T) {
	var v map[string]any
	if err := Unmarshal([]byte("{"), &v); err == nil {
		t.Fatal("expected syntax error")
	}
	if err := Unmarshal([]byte("null"), nil); err == nil {
		t.Fatal("expected invalid unmarshal target error")
	}
}

func TestValid(t *testing.T) {
	if !Valid([]byte(`{"a":1}`)) {
		t.Fatal("Valid() returned false for valid JSON")
	}
	if Valid([]byte(`{"a":`)) {
		t.Fatal("Valid() returned true for invalid JSON")
	}
}

func TestCompactIndent(t *testing.T) {
	src := []byte(`{"a":1,"b":[2,3]}`)
	var compacted bytes.Buffer
	if err := Compact(&compacted, src); err != nil {
		t.Fatalf("Compact() error = %v", err)
	}
	var indented bytes.Buffer
	if err := Indent(&indented, compacted.Bytes(), "", "  "); err != nil {
		t.Fatalf("Indent() error = %v", err)
	}
	if !strings.Contains(indented.String(), "\n") {
		t.Fatalf("Indent() = %q", indented.String())
	}
}

func TestHTMLEscape(t *testing.T) {
	var dst bytes.Buffer
	HTMLEscape(&dst, []byte(`"<&>"`))
	got := dst.String()
	for _, esc := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if !strings.Contains(got, esc) {
			t.Fatalf("HTMLEscape() = %q, missing %q", got, esc)
		}
	}
}

func TestEncoderDecoder(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(map[string]int{"x": 1}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	dec := NewDecoder(&buf)
	var got map[string]int
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got["x"] != 1 {
		t.Fatalf("Decode() = %v", got)
	}
}

func TestMarshalUnsupportedType(t *testing.T) {
	ch := make(chan int)
	if _, err := Marshal(ch); err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestUnmarshalTypeMismatch(t *testing.T) {
	var n int
	err := Unmarshal([]byte(`"not-a-number"`), &n)
	var typeErr *UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("Unmarshal() error = %v, want *UnmarshalTypeError", err)
	}
}

func TestMarshalIndentNonTTY(t *testing.T) {
	in := map[string]string{"k": "v"}
	b, err := MarshalIndent(in, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"k"`)) {
		t.Fatalf("MarshalIndent() = %s", b)
	}
}

func TestDecoderDisallowUnknownFields(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"known":1,"unknown":2}`))
	dec.DisallowUnknownFields()
	var s struct {
		Known int `json:"known"`
	}
	if err := dec.Decode(&s); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestNumber(t *testing.T) {
	var n Number
	if err := Unmarshal([]byte(`123.45`), &n); err != nil {
		t.Fatal(err)
	}
	if n.String() != "123.45" {
		t.Fatalf("Number.String() = %q", n.String())
	}
}

func TestRawMessage(t *testing.T) {
	var raw RawMessage
	if err := Unmarshal([]byte(`{"a":1}`), &raw); err != nil {
		t.Fatal(err)
	}
	var m map[string]int
	if err := Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, map[string]int{"a": 1}) {
		t.Fatalf("RawMessage round-trip = %v", m)
	}
}

func TestMarshalerUnmarshaler(t *testing.T) {
	type boxed struct {
		V string
	}
	in := boxed{V: "x"}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out boxed
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.V != "x" {
		t.Fatalf("round-trip = %+v", out)
	}
}

func TestSyntaxErrorOffset(t *testing.T) {
	err := Unmarshal([]byte(`{"a":}`), new(map[string]any))
	var syn *SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("Unmarshal() error = %T(%v), want *SyntaxError", err, err)
	}
	if syn.Offset == 0 {
		t.Fatal("SyntaxError.Offset should be non-zero")
	}
}

func TestSliceAndArray(t *testing.T) {
	var slice []int
	if err := Unmarshal([]byte(`[]`), &slice); err != nil {
		t.Fatal(err)
	}
	if slice == nil || len(slice) != 0 {
		t.Fatalf("Unmarshal([]) = %v", slice)
	}

	var arr [2]int
	if err := Unmarshal([]byte(`[1,2,3]`), &arr); err != nil {
		t.Fatal(err)
	}
	if arr != [2]int{1, 2} {
		t.Fatalf("array unmarshal = %v", arr)
	}
}

func TestPointerUnmarshal(t *testing.T) {
	var p *int
	if err := Unmarshal([]byte(`null`), &p); err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Fatal("expected nil pointer")
	}
	if err := Unmarshal([]byte(`42`), &p); err != nil {
		t.Fatal(err)
	}
	if p == nil || *p != 42 {
		t.Fatalf("pointer unmarshal = %v", p)
	}
}

func TestMapStringKey(t *testing.T) {
	var m map[string]int
	if err := Unmarshal([]byte(`{"one":1,"two":2}`), &m); err != nil {
		t.Fatal(err)
	}
	if m["one"] != 1 || m["two"] != 2 {
		t.Fatalf("map unmarshal = %v", m)
	}
}

func TestEncoderSetEscapeHTML(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode("<tag>"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `\u003c`) {
		t.Fatalf("SetEscapeHTML(false) still escaped HTML: %s", buf.String())
	}
}

func TestEncoderSetIndent(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\n") {
		t.Fatalf("SetIndent() output = %s", buf.String())
	}
}

func TestDecoderUseNumber(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`123`))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(Number); !ok {
		t.Fatalf("UseNumber() type = %T", v)
	}
}

func TestMoreDecoderInput(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":1}{"b":2}`))
	var first, second map[string]int
	if err := dec.Decode(&first); err != nil {
		t.Fatal(err)
	}
	if !dec.More() {
		t.Fatal("More() should be true")
	}
	if err := dec.Decode(&second); err != nil {
		t.Fatal(err)
	}
	if first["a"] != 1 || second["b"] != 2 {
		t.Fatalf("decoded = %v %v", first, second)
	}
}

func TestInvalidUnmarshalError(t *testing.T) {
	err := Unmarshal([]byte(`{}`), struct{}{})
	var iue *InvalidUnmarshalError
	if !errors.As(err, &iue) {
		t.Fatalf("Unmarshal() error = %T(%v)", err, err)
	}
}

func TestCaseInsensitiveUnmarshal(t *testing.T) {
	var s struct {
		MyField string `json:"myfield"`
	}
	if err := Unmarshal([]byte(`{"MYFIELD":"ok"}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.MyField != "ok" {
		t.Fatalf("case-insensitive unmarshal = %q", s.MyField)
	}
}

func TestOmitEmpty(t *testing.T) {
	type payload struct {
		A string `json:"a,omitempty"`
		B int    `json:"b,omitempty"`
	}
	data, err := Marshal(payload{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Fatalf("Marshal omitempty = %s", data)
	}
}

func TestStringTagOption(t *testing.T) {
	type payload struct {
		N int64 `json:",string"`
	}
	data, err := Marshal(payload{N: 42})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"42"`) {
		t.Fatalf("Marshal string tag = %s", data)
	}
}

func TestByteSlice(t *testing.T) {
	in := []byte("abc")
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, in) {
		t.Fatalf("[]byte round-trip = %q", out)
	}
}

func TestNullValues(t *testing.T) {
	var s *sampleStruct
	if err := Unmarshal([]byte(`null`), &s); err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatal("expected nil struct pointer")
	}
}

func TestInvalidUTF8Replacement(t *testing.T) {
	var s string
	raw := []byte{'"', 0xff, '"'}
	if err := Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Fatal("expected replacement character in string")
	}
}

func TestHTMLEscapeLineSeparator(t *testing.T) {
	var dst bytes.Buffer
	HTMLEscape(&dst, []byte("\xe2\x80\xa8"))
	if !strings.Contains(dst.String(), `\u2028`) {
		t.Fatalf("HTMLEscape line sep = %q", dst.String())
	}
}

func TestEncoderErrorOnUnsupported(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(make(chan int)); err == nil {
		t.Fatal("expected unsupported type error from encoder")
	}
}

func TestDecoderBuffered(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":1}`))
	if dec.Buffered() == nil {
		t.Fatal("Buffered() should not be nil")
	}
	var m map[string]int
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
}

func TestSyntaxErrorMessage(t *testing.T) {
	err := Unmarshal([]byte(`{`), new(map[string]any))
	var syn *SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("error = %T(%v)", err, err)
	}
	if syn.Error() == "" {
		t.Fatal("SyntaxError.Error() should not be empty")
	}
}

func TestInvalidUnmarshalErrorMessage(t *testing.T) {
	err := Unmarshal([]byte(`{}`), 1)
	var iue *InvalidUnmarshalError
	if !errors.As(err, &iue) {
		t.Fatalf("error = %T(%v)", err, err)
	}
	if iue.Error() == "" {
		t.Fatal("InvalidUnmarshalError.Error() should not be empty")
	}
}

func TestUnmarshalTypeErrorMessage(t *testing.T) {
	var n int
	err := Unmarshal([]byte(`true`), &n)
	var te *UnmarshalTypeError
	if !errors.As(err, &te) {
		t.Fatalf("error = %T(%v)", err, err)
	}
	if te.Error() == "" {
		t.Fatal("UnmarshalTypeError.Error() should not be empty")
	}
}

func TestUnsupportedTypeErrorMessage(t *testing.T) {
	_, err := Marshal(make(chan int))
	var ute *UnsupportedTypeError
	if !errors.As(err, &ute) {
		t.Fatalf("error = %T(%v)", err, err)
	}
	if ute.Error() == "" {
		t.Fatal("UnsupportedTypeError.Error() should not be empty")
	}
}

func TestMarshalFloatSpecial(t *testing.T) {
	for _, v := range []float64{0, -0, 1.5, 1e10} {
		if _, err := Marshal(v); err != nil {
			t.Fatalf("Marshal(%v) error = %v", v, err)
		}
	}
}

func TestUnmarshalIntoExistingMap(t *testing.T) {
	m := map[string]int{"keep": 1}
	if err := Unmarshal([]byte(`{"add":2}`), &m); err != nil {
		t.Fatal(err)
	}
	if m["keep"] != 1 || m["add"] != 2 {
		t.Fatalf("map merge = %v", m)
	}
}

func TestUnmarshalIntoExistingSlice(t *testing.T) {
	s := []int{9}
	if err := Unmarshal([]byte(`[1,2]`), &s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, []int{1, 2}) {
		t.Fatalf("slice reset = %v", s)
	}
}
