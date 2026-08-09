package colorjson

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestCaseInsensitiveFieldMatching covers all four branches of foldFunc:
// simpleLetterEqualFold, asciiEqualFold, equalFoldRight (names containing
// s/S/k/K) and bytes.EqualFold (non-ASCII names).
func TestCaseInsensitiveFieldMatching(t *testing.T) {
	type target struct {
		Simple int `json:"abc"`   // letters only, no specials
		ASCII  int `json:"ab_c1"` // contains non-letters
		Spec   int `json:"ask"`   // contains 's' and 'k'
		NonASC int `json:"café"`  // non-ASCII
	}
	tests := []struct {
		name  string
		in    string
		check func(target) bool
	}{
		{"exact", `{"abc":1}`, func(v target) bool { return v.Simple == 1 }},
		{"upper simple", `{"ABC":1}`, func(v target) bool { return v.Simple == 1 }},
		{"mixed simple", `{"AbC":1}`, func(v target) bool { return v.Simple == 1 }},
		{"upper ascii", `{"AB_C1":1}`, func(v target) bool { return v.ASCII == 1 }},
		{"mixed ascii", `{"Ab_C1":1}`, func(v target) bool { return v.ASCII == 1 }},
		{"upper special", `{"ASK":1}`, func(v target) bool { return v.Spec == 1 }},
		{"mixed special", `{"AsK":1}`, func(v target) bool { return v.Spec == 1 }},
		{"upper non-ascii", `{"CAFÉ":1}`, func(v target) bool { return v.NonASC == 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got target
			if err := Unmarshal([]byte(tc.in), &got); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", tc.in, err)
			}
			if !tc.check(got) {
				t.Fatalf("Unmarshal(%s) = %+v, field not populated", tc.in, got)
			}
		})
	}
}

// TestEqualFoldRightMatchesSuffix is a direct regression test for
// equalFoldRight, which previously ended in `return len(t) < 0` and so could
// never report a match. Field names containing s/S/k/K — Status, Size, Key —
// silently failed to decode when the input used a different case.
func TestEqualFoldRightMatchesSuffix(t *testing.T) {
	tests := []struct {
		s, tt string
		want  bool
	}{
		{"ask", "ASK", true},
		{"ask", "ask", true},
		{"ask", "AsK", true},
		{"status", "STATUS", true},
		{"key", "KEY", true},
		{"ask", "asks", false}, // t longer than s
		{"asks", "ask", false}, // t shorter than s
		{"ask", "axk", false},  // genuine mismatch
		{"ask", "", false},     // empty t
		{"s", "ſ", true},       // long s folds to s
		{"k", "K", true},       // kelvin sign folds to k
		{"a", "ſ", false},      // non-special ASCII vs non-ASCII
	}
	for _, tc := range tests {
		if got := equalFoldRight([]byte(tc.s), []byte(tc.tt)); got != tc.want {
			t.Errorf("equalFoldRight(%q, %q) = %v, want %v", tc.s, tc.tt, got, tc.want)
		}
	}
}

func TestFoldFuncSelection(t *testing.T) {
	tests := []struct {
		in    string
		s, tt string
		want  bool
	}{
		{"abc", "abc", "ABC", true},
		{"ab_1", "ab_1", "AB_1", true},
		{"ab_1", "ab_1", "ab_2", false},
		{"café", "café", "CAFÉ", true},
	}
	for _, tc := range tests {
		fn := foldFunc([]byte(tc.in))
		if got := fn([]byte(tc.s), []byte(tc.tt)); got != tc.want {
			t.Errorf("foldFunc(%q)(%q, %q) = %v, want %v", tc.in, tc.s, tc.tt, got, tc.want)
		}
	}
}

func TestAsciiEqualFoldRejectsLengthMismatch(t *testing.T) {
	if asciiEqualFold([]byte("ab_1"), []byte("ab_12")) {
		t.Fatal("asciiEqualFold matched values of different length")
	}
	if asciiEqualFold([]byte("ab_1"), []byte("ab-1")) {
		t.Fatal("asciiEqualFold matched differing non-letters")
	}
}

func TestSimpleLetterEqualFold(t *testing.T) {
	if !simpleLetterEqualFold([]byte("abc"), []byte("ABC")) {
		t.Fatal("simpleLetterEqualFold(abc, ABC) = false")
	}
	if simpleLetterEqualFold([]byte("abc"), []byte("abcd")) {
		t.Fatal("simpleLetterEqualFold matched values of different length")
	}
	if simpleLetterEqualFold([]byte("abc"), []byte("abd")) {
		t.Fatal("simpleLetterEqualFold matched differing letters")
	}
}

// TestUnmarshalErrorContext checks addErrorContext: a type error inside a
// struct field reports the struct and field it came from.
func TestUnmarshalErrorContext(t *testing.T) {
	type inner struct {
		N int `json:"n"`
	}
	var v inner
	err := Unmarshal([]byte(`{"n":"not a number"}`), &v)
	var te *UnmarshalTypeError
	if !errors.As(err, &te) {
		t.Fatalf("Unmarshal() error = %T(%v), want *UnmarshalTypeError", err, err)
	}
	if te.Field != "n" {
		t.Fatalf("UnmarshalTypeError.Field = %q, want %q", te.Field, "n")
	}
	if te.Struct != "inner" {
		t.Fatalf("UnmarshalTypeError.Struct = %q, want %q", te.Struct, "inner")
	}
	if !strings.Contains(te.Error(), "inner") {
		t.Fatalf("UnmarshalTypeError.Error() = %q, want it to name the struct", te)
	}
}

// TestNumberValidation drives isValidNumber over the grammar in RFC 7159.
func TestNumberValidation(t *testing.T) {
	valid := []string{
		"0", "-0", "1", "-1", "42", "-42",
		"0.5", "-0.5", "1.25",
		"1e5", "1E5", "1e+5", "1e-5", "1E+5", "1E-5",
		"0e0", "1.5e10", "-1.5E-10", "123.456e789",
	}
	for _, s := range valid {
		var n Number
		if err := Unmarshal([]byte(s), &n); err != nil {
			t.Errorf("Unmarshal(%q) into Number error = %v", s, err)
		}
		if n.String() != s {
			t.Errorf("Number.String() = %q, want %q", n.String(), s)
		}
	}

	// Values that are not valid JSON numbers are rejected when assigned
	// directly to a Number field via the ,string tag.
	type quoted struct {
		N Number `json:"n,string"`
	}
	for _, s := range []string{"", "-", "+1", "01", "1.", ".5", "1e", "1e+", "0x1", "abc", "1 2"} {
		var q quoted
		if err := Unmarshal([]byte(`{"n":"`+s+`"}`), &q); err == nil {
			t.Errorf("Unmarshal(%q) into Number succeeded, want an error", s)
		}
	}
}

func TestNumberAccessors(t *testing.T) {
	var v struct {
		N Number `json:"n"`
	}
	if err := Unmarshal([]byte(`{"n":123}`), &v); err != nil {
		t.Fatal(err)
	}
	i, err := v.N.Int64()
	if err != nil || i != 123 {
		t.Fatalf("Number.Int64() = %d, %v", i, err)
	}
	f, err := v.N.Float64()
	if err != nil || f != 123 {
		t.Fatalf("Number.Float64() = %v, %v", f, err)
	}

	if err := Unmarshal([]byte(`{"n":1.5}`), &v); err != nil {
		t.Fatal(err)
	}
	if _, err := v.N.Int64(); err == nil {
		t.Fatal("Number.Int64() on 1.5 succeeded, want an error")
	}
}

// TestUnmarshalMapKeyKinds covers the map-key decoding switch: string, named
// string, signed, unsigned and TextUnmarshaler keys.
func TestUnmarshalMapKeyKinds(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		var m map[string]int
		if err := Unmarshal([]byte(`{"a":1}`), &m); err != nil || m["a"] != 1 {
			t.Fatalf("m = %v, err = %v", m, err)
		}
	})
	t.Run("named string", func(t *testing.T) {
		type key string
		var m map[key]int
		if err := Unmarshal([]byte(`{"a":1}`), &m); err != nil || m["a"] != 1 {
			t.Fatalf("m = %v, err = %v", m, err)
		}
	})
	t.Run("int", func(t *testing.T) {
		var m map[int]string
		if err := Unmarshal([]byte(`{"-3":"a"}`), &m); err != nil || m[-3] != "a" {
			t.Fatalf("m = %v, err = %v", m, err)
		}
	})
	t.Run("uint", func(t *testing.T) {
		var m map[uint16]string
		if err := Unmarshal([]byte(`{"7":"a"}`), &m); err != nil || m[7] != "a" {
			t.Fatalf("m = %v, err = %v", m, err)
		}
	})
	t.Run("text unmarshaler", func(t *testing.T) {
		var m map[textKey]int
		if err := Unmarshal([]byte(`{"a":1}`), &m); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if m[textKey{S: "a"}] != 1 {
			t.Fatalf("m = %v, want key {a}", m)
		}
	})
}

// textKey decodes its map key through encoding.TextUnmarshaler.
type textKey struct{ S string }

func (k textKey) MarshalText() ([]byte, error) { return []byte(k.S), nil }

func (k *textKey) UnmarshalText(b []byte) error {
	k.S = string(b)
	return nil
}

func TestUnmarshalMapKeyErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  any
	}{
		{"int key not a number", `{"abc":1}`, new(map[int]int)},
		{"int key overflow", `{"99999":1}`, new(map[int8]int)},
		{"uint key negative", `{"-1":1}`, new(map[uint]int)},
		{"uint key overflow", `{"99999":1}`, new(map[uint8]int)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Unmarshal([]byte(tc.in), tc.out)
			var te *UnmarshalTypeError
			if !errors.As(err, &te) {
				t.Fatalf("Unmarshal(%s) error = %T(%v), want *UnmarshalTypeError", tc.in, err, err)
			}
		})
	}
}

// TestUnmarshalStringTagMisuse covers valueQuoted: a ,string field must be fed
// a quoted scalar, not a composite or bare value.
func TestUnmarshalStringTagMisuse(t *testing.T) {
	type quoted struct {
		N int `json:"n,string"`
	}
	for _, in := range []string{`{"n":[1]}`, `{"n":{"a":1}}`, `{"n":1}`, `{"n":true}`} {
		var q quoted
		if err := Unmarshal([]byte(in), &q); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want an error", in)
		}
	}

	// null into a ,string field clears it without error.
	var q quoted
	q.N = 5
	if err := Unmarshal([]byte(`{"n":null}`), &q); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}

	// A properly quoted value decodes.
	if err := Unmarshal([]byte(`{"n":"7"}`), &q); err != nil {
		t.Fatalf("Unmarshal(quoted) error = %v", err)
	}
	if q.N != 7 {
		t.Fatalf("q.N = %d, want 7", q.N)
	}

	// A quoted value that is not a valid int is an error.
	if err := Unmarshal([]byte(`{"n":"abc"}`), &q); err == nil {
		t.Error("Unmarshal(\"abc\") into ,string int succeeded, want an error")
	}
}

// jsonUnmarshalerSlice implements Unmarshaler, so arrays route through the
// unmarshaler branch of decodeState.array.
type jsonUnmarshalerSlice struct{ raw string }

func (u *jsonUnmarshalerSlice) UnmarshalJSON(b []byte) error {
	u.raw = string(b)
	return nil
}

func TestUnmarshalerReceivesRawArrayAndObject(t *testing.T) {
	var u jsonUnmarshalerSlice
	if err := Unmarshal([]byte(`[1,2,3]`), &u); err != nil {
		t.Fatal(err)
	}
	if u.raw != `[1,2,3]` {
		t.Fatalf("UnmarshalJSON got %q, want the raw array", u.raw)
	}

	u = jsonUnmarshalerSlice{}
	if err := Unmarshal([]byte(`{"a":[1]}`), &u); err != nil {
		t.Fatal(err)
	}
	if u.raw != `{"a":[1]}` {
		t.Fatalf("UnmarshalJSON got %q, want the raw object", u.raw)
	}
}

// TestTextUnmarshalerTypeErrors covers literalStore's TextUnmarshaler branch
// when the incoming literal is not a string.
func TestTextUnmarshalerTypeErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"number", `1`},
		{"bool", `true`},
		{"array", `[1]`},
		{"object", `{"a":1}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var k textKey
			err := Unmarshal([]byte(tc.in), &k)
			if err == nil {
				t.Fatalf("Unmarshal(%s) into a TextUnmarshaler succeeded, want an error", tc.in)
			}
		})
	}

	// null leaves the value untouched without error.
	k := textKey{S: "keep"}
	if err := Unmarshal([]byte(`null`), &k); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
}

// TestArrayDecoding covers decodeState.array's length handling: growth,
// truncation and zero-filling.
func TestArrayDecoding(t *testing.T) {
	t.Run("slice grows", func(t *testing.T) {
		var s []int
		if err := Unmarshal([]byte(`[1,2,3,4,5]`), &s); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(s, []int{1, 2, 3, 4, 5}) {
			t.Fatalf("s = %v", s)
		}
	})
	t.Run("slice shrinks", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5}
		if err := Unmarshal([]byte(`[9]`), &s); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(s, []int{9}) {
			t.Fatalf("s = %v, want [9]", s)
		}
	})
	t.Run("slice emptied", func(t *testing.T) {
		s := []int{1, 2}
		if err := Unmarshal([]byte(`[]`), &s); err != nil {
			t.Fatal(err)
		}
		if len(s) != 0 {
			t.Fatalf("s = %v, want empty", s)
		}
	})
	t.Run("array zero filled", func(t *testing.T) {
		a := [4]int{9, 9, 9, 9}
		if err := Unmarshal([]byte(`[1,2]`), &a); err != nil {
			t.Fatal(err)
		}
		if a != [4]int{1, 2, 0, 0} {
			t.Fatalf("a = %v, want [1 2 0 0]", a)
		}
	})
	t.Run("array truncated", func(t *testing.T) {
		var a [2]int
		if err := Unmarshal([]byte(`[1,2,3,4]`), &a); err != nil {
			t.Fatal(err)
		}
		if a != [2]int{1, 2} {
			t.Fatalf("a = %v, want [1 2]", a)
		}
	})
	t.Run("array into non-array", func(t *testing.T) {
		var m map[string]int
		if err := Unmarshal([]byte(`[1]`), &m); err == nil {
			t.Fatal("Unmarshal([1]) into a map succeeded, want an error")
		}
	})
}

// TestUnquoteEscapes covers the escape handling in unquoteBytes, including
// surrogate pairs and their failure modes.
func TestUnquoteEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"quote", `"\""`, `"`},
		{"backslash", `"\\"`, `\`},
		{"solidus", `"\/"`, `/`},
		{"backspace", `"\b"`, "\b"},
		{"formfeed", `"\f"`, "\f"},
		{"newline", `"\n"`, "\n"},
		{"carriage return", `"\r"`, "\r"},
		{"tab", `"\t"`, "\t"},
		{"basic unicode", `"A"`, "A"},
		{"unicode above ascii", `"é"`, "é"},
		{"surrogate pair", `"😀"`, "\U0001f600"},
		{"lone high surrogate", `"\ud83d"`, "�"},
		{"lone low surrogate", `"\ude00"`, "�"},
		{"high surrogate then ascii", `"\ud83dA"`, "�A"},
		{"mixed", `"a\tbAc"`, "a\tbAc"},
		{"empty", `""`, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s string
			if err := Unmarshal([]byte(tc.in), &s); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", tc.in, err)
			}
			if s != tc.want {
				t.Fatalf("Unmarshal(%s) = %q, want %q", tc.in, s, tc.want)
			}
		})
	}
}

func TestUnquoteInvalidEscapes(t *testing.T) {
	for _, in := range []string{
		`"\q"`, `"\u12"`, `"\uZZZZ"`, `"\`, `"abc`, `"\u"`,
	} {
		var s string
		if err := Unmarshal([]byte(in), &s); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want an error", in)
		}
	}
}

// TestByteSliceDecoding covers the base64 branch of literalStore.
func TestByteSliceDecoding(t *testing.T) {
	var b []byte
	if err := Unmarshal([]byte(`"YWJj"`), &b); err != nil {
		t.Fatal(err)
	}
	if string(b) != "abc" {
		t.Fatalf("b = %q, want abc", b)
	}

	if err := Unmarshal([]byte(`"not!base64"`), &b); err == nil {
		t.Fatal("Unmarshal of invalid base64 into []byte succeeded, want an error")
	}

	// A string cannot be decoded into a slice of a non-byte element type.
	var ints []int
	if err := Unmarshal([]byte(`"abc"`), &ints); err == nil {
		t.Fatal("Unmarshal of a string into []int succeeded, want an error")
	}
}

// TestLiteralTypeMismatches walks literalStore's saveError branches for each
// literal kind landing in an incompatible Go type.
func TestLiteralTypeMismatches(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  any
	}{
		{"bool into int", `true`, new(int)},
		{"bool into string", `true`, new(string)},
		{"string into int", `"x"`, new(int)},
		{"string into bool", `"x"`, new(bool)},
		{"string into float", `"x"`, new(float64)},
		{"number into bool", `1`, new(bool)},
		{"number into string", `1`, new(string)},
		{"int overflow", `99999`, new(int8)},
		{"uint overflow", `99999`, new(uint8)},
		{"negative into uint", `-1`, new(uint)},
		{"float overflow", `1e400`, new(float64)},
		{"object into int", `{}`, new(int)},
		{"array into int", `[]`, new(int)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := Unmarshal([]byte(tc.in), tc.out); err == nil {
				t.Fatalf("Unmarshal(%s) into %T succeeded, want an error", tc.in, tc.out)
			}
		})
	}
}

// TestNullIntoEveryKind checks that a JSON null leaves each destination kind
// at its zero value without error.
func TestNullIntoEveryKind(t *testing.T) {
	i := 5
	s := "x"
	b := true
	f := 1.5
	sl := []int{1}
	m := map[string]int{"a": 1}
	var iface any = "x"

	for _, out := range []any{&i, &s, &b, &f, &sl, &m, &iface} {
		if err := Unmarshal([]byte(`null`), out); err != nil {
			t.Fatalf("Unmarshal(null) into %T error = %v", out, err)
		}
	}
	// null is a no-op for scalars but clears interfaces.
	if i != 5 || s != "x" || !b || f != 1.5 {
		t.Fatalf("null modified a scalar: %d %q %v %v", i, s, b, f)
	}
	if iface != nil {
		t.Fatalf("null did not clear the interface: %v", iface)
	}
}

// TestUnmarshalIntoInterfaceKinds covers the generic interface decoding path.
func TestUnmarshalIntoInterfaceKinds(t *testing.T) {
	var v any
	if err := Unmarshal([]byte(`{"s":"x","n":1,"b":true,"z":null,"a":[1,"y"],"o":{"k":2}}`), &v); err != nil {
		t.Fatal(err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("v = %T, want map[string]any", v)
	}
	want := map[string]any{
		"s": "x",
		"n": float64(1),
		"b": true,
		"z": nil,
		"a": []any{float64(1), "y"},
		"o": map[string]any{"k": float64(2)},
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("v = %#v\nwant %#v", m, want)
	}
}

func TestUnmarshalUseNumberInterface(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"n":1.5,"a":[2]}`))
	dec.UseNumber()
	var v map[string]any
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	if _, ok := v["n"].(Number); !ok {
		t.Fatalf("v[n] = %T, want Number", v["n"])
	}
	arr := v["a"].([]any)
	if _, ok := arr[0].(Number); !ok {
		t.Fatalf("v[a][0] = %T, want Number", arr[0])
	}
}

// TestDisallowUnknownFieldsNested checks the option applies at every depth.
func TestDisallowUnknownFieldsNested(t *testing.T) {
	type inner struct {
		Known int `json:"known"`
	}
	type outer struct {
		In inner `json:"in"`
	}
	dec := NewDecoder(strings.NewReader(`{"in":{"known":1,"other":2}}`))
	dec.DisallowUnknownFields()
	var v outer
	err := dec.Decode(&v)
	if err == nil {
		t.Fatal("Decode() succeeded, want an unknown field error")
	}
	if !strings.Contains(err.Error(), "other") {
		t.Fatalf("Decode() error = %v, want it to name the unknown field", err)
	}
}

// TestUnmarshalIntoNonPointerKinds covers InvalidUnmarshalError for both nil
// and non-pointer destinations.
func TestUnmarshalIntoNonPointerKinds(t *testing.T) {
	tests := []struct {
		name string
		out  any
	}{
		{"nil", nil},
		{"non-pointer struct", struct{}{}},
		{"non-pointer int", 1},
		{"typed nil pointer", (*int)(nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Unmarshal([]byte(`{}`), tc.out)
			var iue *InvalidUnmarshalError
			if !errors.As(err, &iue) {
				t.Fatalf("Unmarshal() error = %T(%v), want *InvalidUnmarshalError", err, err)
			}
			if iue.Error() == "" {
				t.Fatal("InvalidUnmarshalError.Error() is empty")
			}
		})
	}
}

// TestUnmarshalAllocatesThroughPointers checks indirect() allocating each level
// of a pointer chain.
func TestUnmarshalAllocatesThroughPointers(t *testing.T) {
	type inner struct {
		X int `json:"x"`
	}
	var p ***inner
	if err := Unmarshal([]byte(`{"x":3}`), &p); err != nil {
		t.Fatal(err)
	}
	if p == nil || *p == nil || **p == nil || (***p).X != 3 {
		t.Fatalf("pointer chain not allocated: %v", p)
	}
}

func TestUnmarshalIgnoresUnexportedFields(t *testing.T) {
	type s struct {
		Exported   int `json:"exported"`
		unexported int
	}
	var v s
	if err := Unmarshal([]byte(`{"exported":1,"unexported":2}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.Exported != 1 {
		t.Fatalf("Exported = %d, want 1", v.Exported)
	}
	if v.unexported != 0 {
		t.Fatalf("unexported = %d, want it left alone", v.unexported)
	}
}

// TestUnmarshalTrailingData rejects content after a complete top-level value.
func TestUnmarshalTrailingData(t *testing.T) {
	for _, in := range []string{`{} {}`, `1 2`, `[] x`, `null null`} {
		var v any
		if err := Unmarshal([]byte(in), &v); err == nil {
			t.Errorf("Unmarshal(%q) succeeded, want an error", in)
		}
	}
}

// TestSyntaxErrorOffsets pins the reported offset for a few malformed inputs.
func TestSyntaxErrorOffsets(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{`{"a":}`, 6},
		{`[1,]`, 4},
		{`{,}`, 2},
	}
	for _, tc := range tests {
		var v any
		err := Unmarshal([]byte(tc.in), &v)
		var syn *SyntaxError
		if !errors.As(err, &syn) {
			t.Fatalf("Unmarshal(%q) error = %T(%v), want *SyntaxError", tc.in, err, err)
		}
		if syn.Offset != tc.want {
			t.Errorf("Unmarshal(%q) offset = %d, want %d", tc.in, syn.Offset, tc.want)
		}
	}
}
