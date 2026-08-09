// Copyright (c) 2020-2021 Christian Muehlheim. All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

package textutil

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/delta592/mc/pkg/textutil/ansi"
)

var (
	defaultBreakpoints = []rune{'-'}
	defaultNewline     = []rune{'\n'}
)

type wordWrap struct {
	limit        int
	breakpoints  []rune
	newline      []rune
	keepNewlines bool

	buf   bytes.Buffer
	space bytes.Buffer
	word  ansi.Buffer

	lineLen int
	inANSI  bool
}

func newWordWrap(limit int) *wordWrap {
	return &wordWrap{
		limit:        limit,
		breakpoints:  defaultBreakpoints,
		newline:      defaultNewline,
		keepNewlines: true,
	}
}

func (w *wordWrap) addSpace() {
	w.lineLen += w.space.Len()
	_, _ = w.buf.Write(w.space.Bytes())
	w.space.Reset()
}

func (w *wordWrap) addWord() {
	if w.word.Len() > 0 {
		w.addSpace()
		w.lineLen += w.word.PrintableRuneWidth()
		_, _ = w.buf.Write(w.word.Bytes())
		w.word.Reset()
	}
}

func (w *wordWrap) addNewLine() {
	_, _ = w.buf.WriteRune('\n')
	w.lineLen = 0
	w.space.Reset()
}

func inGroup(a []rune, c rune) bool {
	for _, v := range a {
		if v == c {
			return true
		}
	}
	return false
}

func (w *wordWrap) writeString(s string) {
	if w.limit == 0 {
		_, _ = w.buf.WriteString(s)
		return
	}

	if !w.keepNewlines {
		s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	}

	for _, c := range s {
		if c == ansi.Marker {
			_, _ = w.word.WriteRune(c)
			w.inANSI = true
		} else if w.inANSI {
			_, _ = w.word.WriteRune(c)
			if (c >= 0x40 && c <= 0x5a) || (c >= 0x61 && c <= 0x7a) {
				w.inANSI = false
			}
		} else if inGroup(w.newline, c) {
			if w.word.Len() == 0 {
				if w.lineLen+w.space.Len() > w.limit {
					w.lineLen = 0
				} else {
					_, _ = w.buf.Write(w.space.Bytes())
				}
				w.space.Reset()
			}

			w.addWord()
			w.addNewLine()
		} else if unicode.IsSpace(c) {
			w.addWord()
			_, _ = w.space.WriteRune(c)
		} else if inGroup(w.breakpoints, c) {
			w.addSpace()
			w.addWord()
			_, _ = w.buf.WriteRune(c)
		} else {
			_, _ = w.word.WriteRune(c)

			if w.lineLen+w.space.Len()+w.word.PrintableRuneWidth() > w.limit &&
				w.word.PrintableRuneWidth() < w.limit {
				w.addNewLine()
			}
		}
	}
}

// WordWrap wraps s to the given printable width.
func WordWrap(s string, limit int) string {
	w := newWordWrap(limit)
	w.writeString(s)
	w.addWord()
	return w.buf.String()
}
