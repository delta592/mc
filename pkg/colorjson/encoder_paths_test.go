package colorjson

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// valMarshaler implements Marshaler on its value receiver.
type valMarshaler struct{ N int }

func (v valMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"n":` + strconv.Itoa(v.N) + `}`), nil
}

// ptrMarshaler implements Marshaler only on its pointer receiver, which routes
// encoding through addrMarshalerEncoder for addressable values.
type ptrMarshaler struct{ N int }

func (p *ptrMarshaler) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	return []byte(`{"p":` + strconv.Itoa(p.N) + `}`), nil
}

// valText implements TextMarshaler on its value receiver.
type valText struct{ S string }

func (v valText) MarshalText() ([]byte, error) { return []byte(v.S), nil }

// ptrText implements TextMarshaler only on its pointer receiver.
type ptrText struct{ S string }

func (p *ptrText) MarshalText() ([]byte, error) { return []byte(p.S), nil }

func TestMarshalerValueAndPointerReceivers(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"value receiver", valMarshaler{N: 1}, `{"n":1}`},
		{"pointer to value receiver", &valMarshaler{N: 2}, `{"n":2}`},
		{"pointer receiver", &ptrMarshaler{N: 3}, `{"p":3}`},
		{"non-addressable value", ptrMarshaler{N: 9}, `{"N":9}`},
		{"addressable field", &struct{ M ptrMarshaler }{ptrMarshaler{N: 4}}, `{"M":{"p":4}}`},
		{"slice of pointer receiver", []ptrMarshaler{{N: 5}}, `[{"p":5}]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMarshalerNilPointerEncodesNull(t *testing.T) {
	var p *ptrMarshaler
	got, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "null" {
		t.Fatalf("Marshal(nil *ptrMarshaler) = %s, want null", got)
	}

	got, err = Marshal(struct{ M *ptrMarshaler }{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"M":null}` {
		t.Fatalf("Marshal() = %s, want {\"M\":null}", got)
	}
}

// erroringMarshaler always fails, exercising the MarshalerError path.
type erroringMarshaler struct{}

var errMarshalBoom = errors.New("marshal boom")

func (erroringMarshaler) MarshalJSON() ([]byte, error) { return nil, errMarshalBoom }

func TestMarshalerErrorReportsCause(t *testing.T) {
	_, err := Marshal(erroringMarshaler{})
	if err == nil {
		t.Fatal("Marshal() succeeded, want an error")
	}
	var me *MarshalerError
	if !errors.As(err, &me) {
		t.Fatalf("Marshal() error = %T(%v), want *MarshalerError", err, err)
	}
	if me.Err != errMarshalBoom {
		t.Fatalf("MarshalerError.Err = %v, want %v", me.Err, errMarshalBoom)
	}
	if !strings.Contains(me.Error(), errMarshalBoom.Error()) {
		t.Fatalf("MarshalerError.Error() = %q, want it to mention the cause", me)
	}
	// This fork predates the stdlib's MarshalerError.Unwrap, so errors.Is
	// cannot see through to the cause; callers must reach for .Err instead.
	if errors.Is(err, errMarshalBoom) {
		t.Fatal("MarshalerError now unwraps; simplify callers that reach for .Err")
	}
}

// invalidJSONMarshaler returns bytes that are not valid JSON.
type invalidJSONMarshaler struct{}

func (invalidJSONMarshaler) MarshalJSON() ([]byte, error) { return []byte(`{not json}`), nil }

func TestMarshalerInvalidJSONRejected(t *testing.T) {
	if _, err := Marshal(invalidJSONMarshaler{}); err == nil {
		t.Fatal("Marshal() accepted invalid JSON from MarshalJSON, want an error")
	}
	// Also through the addressable path.
	if _, err := Marshal(struct{ M invalidJSONMarshaler }{}); err == nil {
		t.Fatal("Marshal() accepted invalid JSON from a field's MarshalJSON")
	}
}

// erroringText always fails MarshalText.
type erroringText struct{}

var errTextBoom = errors.New("text boom")

func (erroringText) MarshalText() ([]byte, error) { return nil, errTextBoom }

func TestTextMarshalerError(t *testing.T) {
	_, err := Marshal(erroringText{})
	var me *MarshalerError
	if !errors.As(err, &me) {
		t.Fatalf("Marshal() error = %T(%v), want *MarshalerError", err, err)
	}
	if me.Err != errTextBoom {
		t.Fatalf("MarshalerError.Err = %v, want %v", me.Err, errTextBoom)
	}
}

func TestTextMarshalerValueAndPointerReceivers(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"value receiver", valText{S: "a"}, `"a"`},
		{"pointer to value receiver", &valText{S: "b"}, `"b"`},
		{"pointer receiver", &ptrText{S: "c"}, `"c"`},
		{"addressable field", &struct{ T ptrText }{ptrText{S: "d"}}, `{"T":"d"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestTextMarshalerNilPointerEncodesNull(t *testing.T) {
	var p *ptrText
	got, err := Marshal(struct{ T *ptrText }{T: p})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"T":null}` {
		t.Fatalf("Marshal() = %s, want {\"T\":null}", got)
	}
}

// TestTextMarshalerEscaping drives stringBytes, which is only reached through
// TextMarshaler output, across each of its escaping branches.
func TestTextMarshalerEscaping(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"quote and backslash", `a"b\c`, `"a\"b\\c"`},
		{"newline", "a\nb", `"a\nb"`},
		{"carriage return", "a\rb", `"a\rb"`},
		{"tab", "a\tb", `"a\tb"`},
		{"control byte", "a\x01b", `"a\u0001b"`},
		{"nul byte", "a\x00b", `"a\u0000b"`},
		{"escape byte", "a\x1bb", `"a\u001bb"`},
		{"html metacharacters", "<a>&b", `"\u003ca\u003e\u0026b"`},
		{"line separator", "a\u2028b", `"a\u2028b"`},
		{"paragraph separator", "a\u2029b", `"a\u2029b"`},
		{"multibyte kept intact", "h\u00e9llo\u2192", "\"h\u00e9llo\u2192\""},
		{"plain", "plain", `"plain"`},
		{"empty", "", `""`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Marshal(valText{S: tc.in})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal(%q) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestTextMarshalerInvalidUTF8(t *testing.T) {
	got, err := Marshal(valText{S: string([]byte{'a', 0xff, 'b'})})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `"a\ufffdb"` {
		t.Fatalf("Marshal() = %s, want the replacement character escape", got)
	}
}

// TestStringEscapingPreservesANSI checks the fork's divergence from
// encoding/json: a raw ESC in a Go string is written through unescaped so
// embedded color codes survive, unlike other control bytes.
func TestStringEscapingPreservesANSI(t *testing.T) {
	got, err := Marshal("a\x1b[32mb")
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !bytes.Contains(got, []byte{0x1b}) {
		t.Fatalf("Marshal() = %q, want the raw ESC preserved", got)
	}
	if bytes.Contains(got, []byte(`\u001b`)) {
		t.Fatalf("Marshal() = %q, want ESC left unescaped", got)
	}
}

func TestStringEscapingOtherControlBytes(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"a\x00b", `"a\u0000b"`},
		{"a\x01b", `"a\u0001b"`},
		{"a\x1fb", `"a\u001fb"`},
		{"a\nb", `"a\nb"`},
		{"a\rb", `"a\rb"`},
		{"a\tb", `"a\tb"`},
		{`a"b`, `"a\"b"`},
		{`a\b`, `"a\\b"`},
		{"a\u2028b", `"a\u2028b"`},
		{"a\u2029b", `"a\u2029b"`},
	}
	for _, tc := range tests {
		got, err := Marshal(tc.in)
		if err != nil {
			t.Fatalf("Marshal(%q) error = %v", tc.in, err)
		}
		if string(got) != tc.want {
			t.Fatalf("Marshal(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestMarshalStringNoEscapeHTML(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]string{"k": "<v>&"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "<v>&") {
		t.Fatalf("Encode() = %q, want raw HTML metacharacters", buf.String())
	}
}

// TestUintEncoding covers uintEncoder for every unsigned kind, plain and
// ,string-quoted.
func TestUintEncoding(t *testing.T) {
	type uints struct {
		U    uint    `json:"u"`
		U8   uint8   `json:"u8"`
		U16  uint16  `json:"u16"`
		U32  uint32  `json:"u32"`
		U64  uint64  `json:"u64"`
		UPtr uintptr `json:"uptr"`
	}
	in := uints{U: 1, U8: 255, U16: 65535, U32: 4294967295, U64: 18446744073709551615, UPtr: 7}
	got, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"u":1,"u8":255,"u16":65535,"u32":4294967295,"u64":18446744073709551615,"uptr":7}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}

	var out uints
	if err := Unmarshal(got, &out); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if out != in {
		t.Fatalf("round-trip = %+v, want %+v", out, in)
	}
}

func TestUintStringTag(t *testing.T) {
	type quoted struct {
		U uint64 `json:"u,string"`
		I int64  `json:"i,string"`
		B bool   `json:"b,string"`
	}
	got, err := Marshal(quoted{U: 9, I: -9, B: true})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"u":"9","i":"-9","b":"true"}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}

	var out quoted
	if err := Unmarshal(got, &out); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if out.U != 9 || out.I != -9 || !out.B {
		t.Fatalf("round-trip = %+v", out)
	}
}

// TestIsEmptyValueAllKinds pins omitempty behaviour for every kind
// isEmptyValue inspects.
func TestIsEmptyValueAllKinds(t *testing.T) {
	type all struct {
		Arr   [0]int         `json:"arr,omitempty"`
		Map   map[string]int `json:"map,omitempty"`
		Slice []int          `json:"slice,omitempty"`
		Str   string         `json:"str,omitempty"`
		Bool  bool           `json:"bool,omitempty"`
		Int   int            `json:"int,omitempty"`
		Uint  uint           `json:"uint,omitempty"`
		Float float64        `json:"float,omitempty"`
		Iface any            `json:"iface,omitempty"`
		Ptr   *int           `json:"ptr,omitempty"`
		// Structs are never considered empty.
		Struct struct{} `json:"struct,omitempty"`
	}
	got, err := Marshal(all{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"struct":{}}` {
		t.Fatalf("Marshal(zero) = %s, want only the struct field", got)
	}

	n := 1
	full := all{
		Map: map[string]int{"a": 1}, Slice: []int{1}, Str: "s", Bool: true,
		Int: 1, Uint: 1, Float: 1, Iface: "i", Ptr: &n,
	}
	got, err = Marshal(full)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, key := range []string{"map", "slice", "str", "bool", "int", "uint", "float", "iface", "ptr"} {
		if !strings.Contains(string(got), `"`+key+`"`) {
			t.Fatalf("Marshal(full) = %s, missing %q", got, key)
		}
	}
	if strings.Contains(string(got), `"arr"`) {
		t.Fatalf("Marshal(full) = %s, zero-length array should be omitted", got)
	}
}

// TestEmbeddedFieldConflictSkipped covers dominantField's ambiguity case: two
// fields promoted to the same name at the same depth cancel each other out.
func TestEmbeddedFieldConflictSkipped(t *testing.T) {
	type A struct {
		Dup string
	}
	type B struct {
		Dup string
	}
	type conflict struct {
		A
		B
	}
	got, err := Marshal(conflict{A: A{Dup: "a"}, B: B{Dup: "b"}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("Marshal() = %s, want {} for an ambiguous promoted field", got)
	}
}

// TestEmbeddedShallowFieldWins checks the non-ambiguous side of the same rule.
func TestEmbeddedShallowFieldWins(t *testing.T) {
	type inner struct {
		Name string `json:"name"`
	}
	type outer struct {
		inner
		Name string `json:"name"`
	}
	got, err := Marshal(outer{inner: inner{Name: "deep"}, Name: "shallow"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"name":"shallow"}` {
		t.Fatalf("Marshal() = %s, want the shallower field to win", got)
	}
}

// TestTaggedFieldBeatsUntagged checks that a JSON tag promotes a field over an
// untagged one at the same depth.
func TestTaggedFieldBeatsUntagged(t *testing.T) {
	type A struct {
		Dup string
	}
	type B struct {
		Dup string `json:"Dup"`
	}
	type conflict struct {
		A
		B
	}
	got, err := Marshal(conflict{A: A{Dup: "untagged"}, B: B{Dup: "tagged"}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"Dup":"tagged"}` {
		t.Fatalf("Marshal() = %s, want the tagged field to win", got)
	}
}

func TestEmbeddedPointerStruct(t *testing.T) {
	type inner struct {
		X int `json:"x"`
	}
	type outer struct {
		*inner
		Y int `json:"y"`
	}
	got, err := Marshal(outer{inner: &inner{X: 1}, Y: 2})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"x":1,"y":2}` {
		t.Fatalf("Marshal() = %s", got)
	}

	// A nil embedded pointer is skipped rather than panicking.
	if _, err := Marshal(outer{Y: 3}); err == nil {
		t.Log("nil embedded pointer encoded without error")
	}
}

// TestUnexportedEmbeddedNonStructIgnored covers typeFields' filter for
// unexported embedded fields of non-struct type.
func TestUnexportedEmbeddedNonStructIgnored(t *testing.T) {
	type myInt int
	type s struct {
		myInt
		X int `json:"x"`
	}
	got, err := Marshal(s{myInt: 5, X: 1})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"x":1}` {
		t.Fatalf("Marshal() = %s, want the unexported embedded field ignored", got)
	}
}

// TestMapKeyTypes covers resolve() for the map key kinds it supports.
func TestMapKeyTypes(t *testing.T) {
	type keyed string
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string", map[string]int{"b": 2, "a": 1}, `{"a":1,"b":2}`},
		{"named string", map[keyed]int{"a": 1}, `{"a":1}`},
		{"int", map[int]string{2: "b", 1: "a"}, `{"1":"a","2":"b"}`},
		{"int8", map[int8]int{-1: 1}, `{"-1":1}`},
		{"int64", map[int64]int{9: 1}, `{"9":1}`},
		{"uint", map[uint]int{3: 1}, `{"3":1}`},
		{"uint8", map[uint8]int{255: 1}, `{"255":1}`},
		{"uint64", map[uint64]int{4: 1}, `{"4":1}`},
		{"uintptr", map[uintptr]int{5: 1}, `{"5":1}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMapKeyTextMarshaler(t *testing.T) {
	got, err := Marshal(map[valText]int{{S: "b"}: 2, {S: "a"}: 1})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"a":1,"b":2}` {
		t.Fatalf("Marshal() = %s, want keys sorted by their text form", got)
	}
}

func TestMapKeyTextMarshalerError(t *testing.T) {
	_, err := Marshal(map[erroringText]int{{}: 1})
	var me *MarshalerError
	if !errors.As(err, &me) {
		t.Fatalf("Marshal() error = %T(%v), want *MarshalerError", err, err)
	}
	if me.Err != errTextBoom {
		t.Fatalf("MarshalerError.Err = %v, want %v", me.Err, errTextBoom)
	}
}

func TestMapNilAndEmpty(t *testing.T) {
	var nilMap map[string]int
	got, err := Marshal(nilMap)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "null" {
		t.Fatalf("Marshal(nil map) = %s, want null", got)
	}

	got, err = Marshal(map[string]int{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("Marshal(empty map) = %s, want {}", got)
	}
}

func TestSliceAndArrayEncoding(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil slice", []int(nil), "null"},
		{"empty slice", []int{}, "[]"},
		{"slice", []int{1, 2}, "[1,2]"},
		{"array", [2]int{1, 2}, "[1,2]"},
		{"empty array", [0]int{}, "[]"},
		{"nested", [][]int{{1}, {}}, "[[1],[]]"},
		{"nil byte slice", []byte(nil), "null"},
		{"byte slice", []byte("ab"), `"YWI="`},
		{"array of byte", [2]byte{1, 2}, "[1,2]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestUnsupportedTypes covers unsupportedTypeEncoder and the float
// NaN/Inf rejection.
func TestUnsupportedTypes(t *testing.T) {
	tests := []struct {
		name string
		in   any
	}{
		{"channel", make(chan int)},
		{"func", func() {}},
		{"complex", complex(1, 2)},
		{"map with unsupported value", map[string]chan int{"a": nil}},
		{"slice of func", []func(){nil}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Marshal(tc.in)
			var ute *UnsupportedTypeError
			if !errors.As(err, &ute) {
				t.Fatalf("Marshal() error = %T(%v), want *UnsupportedTypeError", err, err)
			}
			if ute.Error() == "" {
				t.Fatal("UnsupportedTypeError.Error() is empty")
			}
		})
	}
}

func TestUnsupportedFloatValues(t *testing.T) {
	for _, in := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		_, err := Marshal(in)
		var uve *UnsupportedValueError
		if !errors.As(err, &uve) {
			t.Fatalf("Marshal(%v) error = %T(%v), want *UnsupportedValueError", in, err, err)
		}
		if uve.Error() == "" {
			t.Fatal("UnsupportedValueError.Error() is empty")
		}
	}
}

// TestCyclicPointerDepth guards against the encoder mishandling shared
// pointers (not a cycle, but the same pointer reached twice).
func TestSharedPointerEncodedTwice(t *testing.T) {
	n := 7
	got, err := Marshal(struct{ A, B *int }{A: &n, B: &n})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"A":7,"B":7}` {
		t.Fatalf("Marshal() = %s", got)
	}
}

// TestFloatFormatting checks the shortest-representation logic, including the
// exponent rewriting branch for very large and very small magnitudes.
func TestFloatFormatting(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{1.5, "1.5"},
		{1e20, "100000000000000000000"},
		{1e21, "1e+21"},
		{1e-7, "1e-7"},
		{-1e21, "-1e+21"},
		{3.14159, "3.14159"},
	}
	for _, tc := range tests {
		got, err := Marshal(tc.in)
		if err != nil {
			t.Fatalf("Marshal(%v) error = %v", tc.in, err)
		}
		if string(got) != tc.want {
			t.Fatalf("Marshal(%v) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestFloat32Formatting(t *testing.T) {
	got, err := Marshal(float32(1.5))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "1.5" {
		t.Fatalf("Marshal(float32) = %s", got)
	}
}

// TestInterfaceEncoding covers interfaceEncoder including its nil branch.
func TestInterfaceEncoding(t *testing.T) {
	var nilIface any
	got, err := Marshal([]any{nilIface, 1, "s", true, []any{}, map[string]any{}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `[null,1,"s",true,[],{}]` {
		t.Fatalf("Marshal() = %s", got)
	}
}

func TestPointerChainEncoding(t *testing.T) {
	n := 1
	p := &n
	pp := &p
	got, err := Marshal(pp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "1" {
		t.Fatalf("Marshal(**int) = %s, want 1", got)
	}

	var nilPP **int
	got, err = Marshal(nilPP)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != "null" {
		t.Fatalf("Marshal(nil **int) = %s, want null", got)
	}
}

// TestInvalidTagIgnored checks isValidTag: a tag with illegal characters is
// ignored and the Go field name is used instead.
func TestInvalidTagIgnored(t *testing.T) {
	type s struct {
		A string `json:"a b"`   // spaces are permitted by isValidTag
		B string `json:"c\\d"`  // backslash is reserved, so this tag is ignored
		C string `json:"ok_-$"` // other punctuation is permitted
	}
	got, err := Marshal(s{A: "1", B: "2", C: "3"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"a b":"1","B":"2","ok_-$":"3"}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

func TestAnonymousStructPointerNil(t *testing.T) {
	type inner struct {
		X int `json:"x"`
	}
	got, err := Marshal(struct {
		P *inner `json:"p"`
	}{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"p":null}` {
		t.Fatalf("Marshal() = %s", got)
	}
}

// TestEncodeStatePoolReuse runs many marshals to exercise the encodeState pool
// and confirm no state leaks between calls.
func TestEncodeStatePoolReuse(t *testing.T) {
	for i := 0; i < 200; i++ {
		got, err := Marshal(map[string]int{"i": i})
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		want := `{"i":` + strconv.Itoa(i) + `}`
		if string(got) != want {
			t.Fatalf("Marshal() = %s, want %s", got, want)
		}
	}
}

func TestMarshalIndentProducesValidJSON(t *testing.T) {
	in := map[string]any{"a": []int{1, 2}, "b": map[string]int{"c": 3}}
	b, err := MarshalIndent(in, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	var out map[string]any
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal(MarshalIndent(...)) error = %v", err)
	}
	if !reflect.DeepEqual(out["b"], map[string]any{"c": float64(3)}) {
		t.Fatalf("round-trip = %#v", out)
	}
}
