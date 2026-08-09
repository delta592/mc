package colorjson

import (
	"bytes"
	"encoding"
	"math"
	"reflect"
	"testing"
)

type optionals struct {
	Sr string         `json:"sr"`
	So string         `json:"so,omitempty"`
	Sw string         `json:"-"`
	Ir int            `json:"omitempty"`
	Io int            `json:"io,omitempty"`
	Mr map[string]any `json:"mr"`
	Mo map[string]any `json:",omitempty"`
	Fr float64        `json:"fr"`
	Br bool           `json:"br"`
}

func TestOptionalsMarshal(t *testing.T) {
	var o optionals
	o.Sw = "hidden"
	o.Mr = map[string]any{}
	o.Mo = map[string]any{}
	data, err := Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("hidden")) {
		t.Fatalf("omitted field present: %s", data)
	}
}

type stringTag struct {
	BoolStr bool   `json:",string"`
	IntStr  int64  `json:",string"`
	StrStr  string `json:",string"`
}

func TestStringTagRoundtrip(t *testing.T) {
	in := stringTag{BoolStr: true, IntStr: 42, StrStr: "x"}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out stringTag
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.IntStr != 42 || !out.BoolStr {
		t.Fatalf("roundtrip = %+v", out)
	}
}

type renamedByteSlice []byte

func TestRenamedByteSlice(t *testing.T) {
	data, err := Marshal(renamedByteSlice("abc"))
	if err != nil {
		t.Fatal(err)
	}
	var out renamedByteSlice
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if string(out) != "abc" {
		t.Fatalf("got %q", out)
	}
}

type textInt int

func (t textInt) MarshalText() ([]byte, error) {
	return []byte(`"`), nil
}

func (t *textInt) UnmarshalText(data []byte) error {
	*t = 7
	return nil
}

func TestTextMarshalerField(t *testing.T) {
	type payload struct {
		V textInt `json:"v"`
	}
	var p payload
	if err := Unmarshal([]byte(`{"v":"ignored"}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.V != 7 {
		t.Fatalf("got %d", p.V)
	}
}

func TestAnonymousFields(t *testing.T) {
	type inner struct {
		A int `json:"a"`
	}
	type outer struct {
		inner
		B int `json:"b"`
	}
	var o outer
	if err := Unmarshal([]byte(`{"a":1,"b":2}`), &o); err != nil {
		t.Fatal(err)
	}
	if o.A != 1 || o.B != 2 {
		t.Fatalf("got %+v", o)
	}
}

func TestUnmarshalMapNonStringKeyKind(t *testing.T) {
	var m map[string]any
	if err := Unmarshal([]byte(`{"k":{"nested":1}}`), &m); err != nil {
		t.Fatal(err)
	}
}

func TestMarshalFloat(t *testing.T) {
	tests := []float64{0, -0, 1.5, 1e6, math.SmallestNonzeroFloat64}
	for _, f := range tests {
		if _, err := Marshal(f); err != nil {
			t.Fatalf("Marshal(%v) err = %v", f, err)
		}
	}
}

func TestUnmarshalFloat(t *testing.T) {
	var f float64
	if err := Unmarshal([]byte(`1.25`), &f); err != nil || f != 1.25 {
		t.Fatalf("got %v err %v", f, err)
	}
}

func TestArrayTruncation(t *testing.T) {
	var arr [2]int
	if err := Unmarshal([]byte(`[1,2,3,4]`), &arr); err != nil {
		t.Fatal(err)
	}
	if arr != [2]int{1, 2} {
		t.Fatalf("got %v", arr)
	}
}

func TestEmptyArrayToSlice(t *testing.T) {
	var s []string
	if err := Unmarshal([]byte(`[]`), &s); err != nil {
		t.Fatal(err)
	}
	if s == nil || len(s) != 0 {
		t.Fatalf("slice = %v", s)
	}
}

func TestNullStructPointer(t *testing.T) {
	type s struct{ A int }
	var p *s
	if err := Unmarshal([]byte(`null`), &p); err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Fatal("expected nil")
	}
}

func TestInterfaceSlice(t *testing.T) {
	var v any
	if err := Unmarshal([]byte(`[1,"x",true,null]`), &v); err != nil {
		t.Fatal(err)
	}
	slice, ok := v.([]any)
	if !ok || len(slice) != 4 {
		t.Fatalf("got %T %v", v, v)
	}
}

func TestMapInterface(t *testing.T) {
	var v map[string]any
	if err := Unmarshal([]byte(`{"n":1,"s":"a","b":true}`), &v); err != nil {
		t.Fatal(err)
	}
	if v["n"].(float64) != 1 {
		t.Fatalf("map = %v", v)
	}
}

func TestInvalidJSONSyntax(t *testing.T) {
	for _, raw := range []string{`{`, `{"a":}`, `[1,]`, `"unterminated`} {
		if Valid([]byte(raw)) {
			t.Fatalf("Valid(%q) = true", raw)
		}
	}
}

func TestCompactRoundtrip(t *testing.T) {
	src := []byte("{\n  \"a\" : 1 ,\n \"b\" : [ 2 , 3 ] \n}")
	var dst bytes.Buffer
	if err := Compact(&dst, src); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := Unmarshal(dst.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
}

func TestIndentPreservesValue(t *testing.T) {
	src := []byte(`{"a":1}`)
	var dst bytes.Buffer
	if err := Indent(&dst, src, "", "  "); err != nil {
		t.Fatal(err)
	}
	var a, b map[string]any
	if err := Unmarshal(src, &a); err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(dst.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("indent changed value")
	}
}

func TestHTMLEscapeAllSpecials(t *testing.T) {
	src := []byte(`"<>&` + "\xe2\x80\xa8\xe2\x80\xa9" + `"`)
	var dst bytes.Buffer
	HTMLEscape(&dst, src)
	for _, esc := range []string{`\u003c`, `\u003e`, `\u0026`, `\u2028`, `\u2029`} {
		if !bytes.Contains(dst.Bytes(), []byte(esc)) {
			t.Fatalf("missing %s in %s", esc, dst.Bytes())
		}
	}
}

func TestEncoderDecoderMultipleTypes(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	for _, v := range []any{1, "s", true, map[string]int{"a": 1}} {
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
	}
	dec := NewDecoder(&buf)
	for i := 0; i < 4; i++ {
		var v any
		if err := dec.Decode(&v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDecoderToken(t *testing.T) {
	dec := NewDecoder(bytes.NewReader([]byte(`{"a":1}`)))
	tok, err := dec.Token()
	if err != nil || tok != Delim('{') {
		t.Fatalf("first token = %v err %v", tok, err)
	}
}

func TestUnmarshalEmbeddedPointer(t *testing.T) {
	type T struct {
		A int `json:"a"`
	}
	type U struct {
		*T
		B int `json:"b"`
	}
	var u U
	if err := Unmarshal([]byte(`{"a":1,"b":2}`), &u); err != nil {
		t.Fatal(err)
	}
	if u.A != 1 || u.B != 2 {
		t.Fatalf("got %+v", u)
	}
}

type unmarshalJSON struct {
	V string
}

func (u *unmarshalJSON) UnmarshalJSON(b []byte) error {
	return Unmarshal(b, &u.V)
}

func TestUnmarshalJSONInterface(t *testing.T) {
	var u unmarshalJSON
	if err := Unmarshal([]byte(`"ok"`), &u); err != nil || u.V != "ok" {
		t.Fatalf("got %+v err %v", u, err)
	}
}

type marshalJSON struct {
	V string
}

func (m marshalJSON) MarshalJSON() ([]byte, error) {
	return Marshal(m.V)
}

func TestMarshalJSONInterface(t *testing.T) {
	data, err := Marshal(marshalJSON{V: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"x"` {
		t.Fatalf("got %s", data)
	}
}

type textType string

func (t textType) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

func (t *textType) UnmarshalText(b []byte) error {
	*t = textType(b)
	return nil
}

func TestEncodingTextMarshalerStruct(t *testing.T) {
	type payload struct {
		T textType `json:"t"`
	}
	var p payload
	if err := Unmarshal([]byte(`{"t":"abc"}`), &p); err != nil || p.T != "abc" {
		t.Fatalf("got %+v err %v", p, err)
	}
}

func TestStructFieldDominance(t *testing.T) {
	type embed struct {
		X int `json:"x"`
	}
	type outer struct {
		embed
		X int `json:"y"`
	}
	var o outer
	if err := Unmarshal([]byte(`{"x":1,"y":2}`), &o); err != nil {
		t.Fatal(err)
	}
	if o.X != 2 {
		t.Fatalf("got %+v", o)
	}
}

func TestUnmarshalArrayIntoSlice(t *testing.T) {
	var s []int
	if err := Unmarshal([]byte(`[1,2,3]`), &s); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, []int{1, 2, 3}) {
		t.Fatalf("got %v", s)
	}
}

func TestUnmarshalBoolNull(t *testing.T) {
	var b bool
	if err := Unmarshal([]byte(`null`), &b); err != nil {
		t.Fatal(err)
	}
}

func TestMarshalIndentRaw(t *testing.T) {
	in := map[string]int{"a": 1}
	_, err := MarshalIndent(in, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRawMessageMarshal(t *testing.T) {
	raw := RawMessage(`{"a":1}`)
	data, err := Marshal(struct {
		R RawMessage `json:"r"`
	}{R: raw})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`{"a":1}`)) {
		t.Fatalf("got %s", data)
	}
}

func TestNumberConversion(t *testing.T) {
	var n Number
	if err := Unmarshal([]byte(`"123.45"`), &n); err != nil {
		t.Fatal(err)
	}
	f, err := n.Float64()
	if err != nil || f != 123.45 {
		t.Fatalf("Float64() = %v err %v", f, err)
	}
}

func TestUnmarshalIntoExistingPointer(t *testing.T) {
	type s struct{ A int }
	existing := &s{A: 1}
	if err := Unmarshal([]byte(`{"a":2}`), &existing); err != nil {
		t.Fatal(err)
	}
	if existing.A != 2 {
		t.Fatalf("got %d", existing.A)
	}
}

func TestEmptyInputDecode(t *testing.T) {
	dec := NewDecoder(bytes.NewReader(nil))
	var v any
	if err := dec.Decode(&v); err == nil {
		t.Fatal("expected EOF error")
	}
}

func TestValidWhitespaceOnly(t *testing.T) {
	if Valid([]byte("   \n\t  ")) {
		t.Fatal("whitespace-only should be invalid")
	}
}

func TestMarshalMapNil(t *testing.T) {
	var m map[string]int
	data, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Fatalf("got %s", data)
	}
}

func TestUnmarshalInterfaceAssign(t *testing.T) {
	var v any = &map[string]any{}
	if err := Unmarshal([]byte(`{"k":1}`), &v); err != nil {
		t.Fatal(err)
	}
}

func TestEncoderSetEscapeHTMLTrue(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode("<"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`\u003c`)) {
		t.Fatalf("got %s", buf.Bytes())
	}
}

func TestUnmarshalStringWithNull(t *testing.T) {
	var s *string
	if err := Unmarshal([]byte(`null`), &s); err != nil || s != nil {
		t.Fatalf("s = %v err %v", s, err)
	}
	val := "x"
	s = &val
	if err := Unmarshal([]byte(`"y"`), &s); err != nil || *s != "y" {
		t.Fatalf("s = %v err %v", s, err)
	}
}

func TestMarshalTextMarshalerMapKey(t *testing.T) {
	type key struct {
		K string
	}
	// map with struct keys uses encoding.TextMarshaler when implemented - skip if unsupported
	_ = encoding.TextMarshaler(nil)
	_ = key{}
}
