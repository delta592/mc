/*
Copyright 2012 Google Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shlex

import (
	"strings"
	"testing"
)

var testString = "one two \"three four\" \"five \\\"six\\\"\" seven#eight # nine # ten\n eleven 'twelve\\' thirteen=13 fourteen/14"

func TestClassifier(t *testing.T) {
	classifier := newDefaultClassifier()
	tests := map[rune]runeTokenClass{
		' ':  spaceRuneClass,
		'"':  escapingQuoteRuneClass,
		'\'': nonEscapingQuoteRuneClass,
		'#':  commentRuneClass,
	}
	for runeChar, want := range tests {
		got := classifier.ClassifyRune(runeChar)
		if got != want {
			t.Errorf("ClassifyRune(%v) -> %v. Want: %v", runeChar, got, want)
		}
	}
}

func TestSplit(t *testing.T) {
	expectedStrings := []string{"one", "two", "three four", "five \"six\"", "seven#eight", "eleven", "twelve\\", "thirteen=13", "fourteen/14"}
	got, err := Split(testString)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(expectedStrings) {
		t.Fatalf("Split() -> %v. Want: %v", got, expectedStrings)
	}
	for i, want := range expectedStrings {
		if got[i] != want {
			t.Errorf("Split()[%v] of %q -> %q. Want: %q", i, testString, got[i], want)
		}
	}
}

func TestTokenizer(t *testing.T) {
	testInput := strings.NewReader(testString)
	expectedTokens := []*Token{
		{WordToken, "one"},
		{WordToken, "two"},
		{WordToken, "three four"},
		{WordToken, "five \"six\""},
		{WordToken, "seven#eight"},
		{CommentToken, " nine # ten"},
		{WordToken, "eleven"},
		{WordToken, "twelve\\"},
		{WordToken, "thirteen=13"},
		{WordToken, "fourteen/14"},
	}

	tokenizer := NewTokenizer(testInput)
	for i, want := range expectedTokens {
		got, err := tokenizer.Next()
		if err != nil {
			t.Error(err)
		}
		if !got.Equal(want) {
			t.Errorf("Tokenizer.Next()[%v] of %q -> %v. Want: %v", i, testString, got, want)
		}
	}
}

func TestLexer(t *testing.T) {
	lexer := NewLexer(strings.NewReader("one two"))
	first, err := lexer.Next()
	if err != nil || first != "one" {
		t.Fatalf("Lexer.Next() first = (%q, %v)", first, err)
	}
	second, err := lexer.Next()
	if err != nil || second != "two" {
		t.Fatalf("Lexer.Next() second = (%q, %v)", second, err)
	}
}

func TestSplitEmpty(t *testing.T) {
	got, err := Split("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("Split(\"\") = %v, want empty", got)
	}
}

func TestTokenEqualNil(t *testing.T) {
	a := &Token{tokenType: WordToken, value: "x"}
	if a.Equal(nil) || (*Token)(nil).Equal(a) {
		t.Fatal("Equal() with nil should be false")
	}
	if !a.Equal(&Token{tokenType: WordToken, value: "x"}) {
		t.Fatal("Equal() should match same token")
	}
}

func TestLexerComments(t *testing.T) {
	lexer := NewLexer(strings.NewReader("hello # comment\nworld"))
	first, err := lexer.Next()
	if err != nil || first != "hello" {
		t.Fatalf("first token = (%q, %v)", first, err)
	}
	second, err := lexer.Next()
	if err != nil || second != "world" {
		t.Fatalf("second token = (%q, %v)", second, err)
	}
}

func TestSplitSingleQuotes(t *testing.T) {
	got, err := Split(`'quoted value' plain`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "quoted value" {
		t.Fatalf("Split() = %v", got)
	}
}
