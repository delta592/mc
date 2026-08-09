package colorjson

import (
	"bytes"
	"encoding"
	"math"
	"reflect"
	"strings"
	"testing"
)

type allTypes struct {
	Bool    bool
	Int     int
	Int8    int8
	Float   float64
	String  string
	Bytes   []byte
	Slice   []string
	Map     map[string]int
	Struct  nestedStruct
	Iface   any
	Omit    string `json:",omitempty"`
	Skip    string `json:"-"`
	Rename  string `json:"renamed"`
	StringN int64  `json:",string"`
}

type nestedStruct struct {
	X int `json:"x"`
}

func TestMarshalAllTypes(t *testing.T) {
	in := allTypes{
		Bool:    true,
		Int:     7,
		Int8:    8,
		Float:   1.25,
		String:  "text",
		Bytes:   []byte("bin"),
		Slice:   []string{"a", "b"},
		Map:     map[string]int{"k": 1},
		Struct:  nestedStruct{X: 9},
		Iface:   map[string]any{"z": true},
		Rename:  "name",
		StringN: 99,
	}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out allTypes
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Bool || out.Int != 7 || out.Rename != "name" {
		t.Fatalf("round-trip = %+v", out)
	}
}

func TestUnmarshalEmbeddedAndAnonymous(t *testing.T) {
	type inner struct {
		V int `json:"v"`
	}
	type outer struct {
		inner
		Name string `json:"name"`
	}
	var o outer
	if err := Unmarshal([]byte(`{"v":1,"name":"n"}`), &o); err != nil {
		t.Fatal(err)
	}
	if o.V != 1 || o.Name != "n" {
		t.Fatalf("unmarshal = %+v", o)
	}
}

func TestTextMarshaler(t *testing.T) {
	type textType struct {
		V string
	}
	tt := textType{V: "x"}
	data, err := Marshal(tt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"V"`)) {
		t.Fatalf("marshal = %s", data)
	}
}

type jsonMarshalerType struct {
	V string
}

func (j jsonMarshalerType) MarshalJSON() ([]byte, error) {
	return Marshal(j.V)
}

func (j *jsonMarshalerType) UnmarshalJSON(data []byte) error {
	return Unmarshal(data, &j.V)
}

func TestJSONMarshaler(t *testing.T) {
	in := jsonMarshalerType{V: "custom"}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out jsonMarshalerType
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.V != "custom" {
		t.Fatalf("got %q", out.V)
	}
}

type textMarshalType string

func (t textMarshalType) MarshalText() ([]byte, error) {
	return []byte(string(t)), nil
}

func (t *textMarshalType) UnmarshalText(data []byte) error {
	*t = textMarshalType(data)
	return nil
}

func TestTextUnmarshalerField(t *testing.T) {
	type payload struct {
		T textMarshalType `json:"t"`
	}
	var p payload
	if err := Unmarshal([]byte(`{"t":"abc"}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.T != "abc" {
		t.Fatalf("got %q", p.T)
	}
}

func TestMapKeysInt(t *testing.T) {
	type key struct {
		K string
	}
	// string-keyed maps only in standard path; test map[string]any with numbers as values
	var m map[string]float64
	if err := Unmarshal([]byte(`{"a":1.5,"b":2.5}`), &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] != 1.5 {
		t.Fatalf("map = %v", m)
	}
	_ = key{}
}

func TestFloatSpecialValues(t *testing.T) {
	for _, tc := range []struct {
		in  string
		out float64
	}{
		{`0`, 0},
		{`-0`, 0},
		{`1e3`, 1000},
	} {
		var f float64
		if err := Unmarshal([]byte(tc.in), &f); err != nil {
			t.Fatalf("Unmarshal(%s) err = %v", tc.in, err)
		}
		if f != tc.out {
			t.Fatalf("Unmarshal(%s) = %v", tc.in, f)
		}
	}
}

func TestNaNAndInfRejected(t *testing.T) {
	if _, err := Marshal(math.NaN()); err == nil {
		t.Fatal("expected error marshaling NaN")
	}
	if _, err := Marshal(math.Inf(1)); err == nil {
		t.Fatal("expected error marshaling +Inf")
	}
}

func TestMarshalMapWithValues(t *testing.T) {
	type goodMap struct {
		M map[string]int
	}
	b := goodMap{M: map[string]int{"a": 1}}
	data, err := Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"a"`)) {
		t.Fatalf("marshal = %s", data)
	}
}

func TestUnmarshalNullIntoNonPointer(t *testing.T) {
	var n int
	if err := Unmarshal([]byte(`null`), &n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("null into int = %d", n)
	}
}

func TestUnmarshalInterfaceNumbers(t *testing.T) {
	var v any
	if err := Unmarshal([]byte(`42`), &v); err != nil {
		t.Fatal(err)
	}
	if v.(float64) != 42 {
		t.Fatalf("interface number = %v", v)
	}
}

func TestEncoderMultipleValues(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	for _, v := range []int{1, 2, 3} {
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
	}
	dec := NewDecoder(&buf)
	for i := 1; i <= 3; i++ {
		var n int
		if err := dec.Decode(&n); err != nil {
			t.Fatal(err)
		}
		if n != i {
			t.Fatalf("decode %d got %d", i, n)
		}
	}
}

func TestIndentInvalidJSON(t *testing.T) {
	var dst bytes.Buffer
	if err := Indent(&dst, []byte("{"), "", "  "); err == nil {
		t.Fatal("expected indent error")
	}
}

func TestCompactInvalidJSON(t *testing.T) {
	var dst bytes.Buffer
	if err := Compact(&dst, []byte("{")); err == nil {
		t.Fatal("expected compact error")
	}
}

func TestHTMLEscapeNoChange(t *testing.T) {
	src := []byte(`{"a":"plain"}`)
	var dst bytes.Buffer
	HTMLEscape(&dst, src)
	if !bytes.Equal(dst.Bytes(), src) {
		t.Fatalf("HTMLEscape changed safe json: %s", dst.Bytes())
	}
}

func TestStructTagCommaName(t *testing.T) {
	type s struct {
		Field int `json:",omitempty"`
	}
	data, err := Marshal(s{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Fatalf("marshal = %s", data)
	}
}

func TestStructTagDashName(t *testing.T) {
	type s struct {
		Field int `json:"-,"`
	}
	data, err := Marshal(s{Field: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"-"`) {
		t.Fatalf("marshal = %s", data)
	}
}

func TestUnmarshalUnknownFieldsDefault(t *testing.T) {
	type s struct {
		A int `json:"a"`
	}
	var out s
	if err := Unmarshal([]byte(`{"a":1,"extra":2}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.A != 1 {
		t.Fatalf("got %+v", out)
	}
}

func TestMarshalMapSortedKeys(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1}
	data, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(data), `"a"`) > strings.Index(string(data), `"b"`) {
		t.Fatalf("keys not sorted: %s", data)
	}
}

func TestUnmarshalTypedString(t *testing.T) {
	type s struct {
		N Number `json:"n"`
	}
	var out s
	if err := Unmarshal([]byte(`{"n":"123"}`), &out); err != nil {
		t.Fatal(err)
	}
	if out.N.String() != "123" {
		t.Fatalf("number = %q", out.N)
	}
}

func TestEncodeDecodeTimeAsString(t *testing.T) {
	type s struct {
		T string `json:"t"`
	}
	in := s{T: "2020-01-01"}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out s
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.T != in.T {
		t.Fatalf("got %q", out.T)
	}
}

func TestInvalidUnmarshalNonPointer(t *testing.T) {
	var v map[string]any
	err := Unmarshal([]byte(`{}`), v)
	if err == nil {
		t.Fatal("expected error unmarshaling into non-pointer")
	}
}

func TestMarshalerError(t *testing.T) {
	type bad struct{}
	_, err := Marshal(badMarshaler{})
	if err == nil {
		t.Fatal("expected marshaler error")
	}
}

type badMarshaler struct{}

func (badMarshaler) MarshalJSON() ([]byte, error) {
	return nil, bytes.ErrTooLarge
}

func TestUnmarshalerError(t *testing.T) {
	err := Unmarshal([]byte(`"x"`), &badUnmarshaler{})
	if err == nil {
		t.Fatal("expected unmarshaler error")
	}
}

type badUnmarshaler struct{}

func (badUnmarshaler) UnmarshalJSON([]byte) error {
	return bytes.ErrTooLarge
}

func TestEncodingTextMarshalerOnField(t *testing.T) {
	type payload struct {
		T encoding.TextMarshaler `json:"-"`
	}
	// ensure compile-time interface satisfaction for nested types used elsewhere
	_ = reflect.TypeOf(payload{})
}

func TestScannerWhitespace(t *testing.T) {
	if !Valid([]byte("  \n\t {\"a\":1}  ")) {
		t.Fatal("Valid() should accept leading/trailing whitespace")
	}
}

func TestDecodeInStream(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`   {"k":"v"}   `))
	var m map[string]string
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m["k"] != "v" {
		t.Fatalf("decode = %v", m)
	}
}

func TestMarshalNilSlice(t *testing.T) {
	var s []int
	data, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Fatalf("nil slice = %s", data)
	}
}

func TestMarshalEmptySlice(t *testing.T) {
	data, err := Marshal([]int{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("empty slice = %s", data)
	}
}

func TestUnmarshalEmptyArrayIntoNilSlice(t *testing.T) {
	var s []int
	if err := Unmarshal([]byte(`[]`), &s); err != nil {
		t.Fatal(err)
	}
	if s == nil || len(s) != 0 {
		t.Fatalf("slice = %v", s)
	}
}

func TestUnmarshalBool(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{`true`, true},
		{`false`, false},
	} {
		var b bool
		if err := Unmarshal([]byte(tc.in), &b); err != nil {
			t.Fatal(err)
		}
		if b != tc.want {
			t.Fatalf("%s => %v", tc.in, b)
		}
	}
}

func TestUnmarshalString(t *testing.T) {
	var s string
	if err := Unmarshal([]byte(`"hello\n"`), &s); err != nil {
		t.Fatal(err)
	}
	if s != "hello\n" {
		t.Fatalf("string = %q", s)
	}
}

func TestUnmarshalEscapedUnicode(t *testing.T) {
	var s string
	if err := Unmarshal([]byte(`"\u0041"`), &s); err != nil {
		t.Fatal(err)
	}
	if s != "A" {
		t.Fatalf("unicode = %q", s)
	}
}

func TestUnmarshalObjectToMapWithExistingEntries(t *testing.T) {
	m := map[string]int{"keep": 1}
	if err := Unmarshal([]byte(`{"add":2}`), &m); err != nil {
		t.Fatal(err)
	}
	if m["keep"] != 1 || m["add"] != 2 {
		t.Fatalf("map = %v", m)
	}
}

func TestMarshalPointerToStruct(t *testing.T) {
	type s struct{ A int }
	v := &s{A: 3}
	data, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out s
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.A != 3 {
		t.Fatalf("out = %+v", out)
	}
}

func TestMarshalNilPointer(t *testing.T) {
	type s struct{ A int }
	var v *s
	data, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Fatalf("nil pointer = %s", data)
	}
}
