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

package ansi

import (
	"bytes"
	"io"
	"unicode/utf8"
)

type Writer struct {
	Forward io.Writer

	ansi       bool
	ansiseq    bytes.Buffer
	lastseq    bytes.Buffer
	seqchanged bool
	runeBuf    []byte
}

// Write is used to write content to the ANSI buffer.
func (w *Writer) Write(b []byte) (int, error) {
	for _, c := range string(b) {
		if c == Marker {
			w.ansi = true
			w.seqchanged = true
			_, _ = w.ansiseq.WriteRune(c)
		} else if w.ansi {
			_, _ = w.ansiseq.WriteRune(c)
			if IsTerminator(c) {
				w.ansi = false

				if bytes.HasSuffix(w.ansiseq.Bytes(), []byte("[0m")) {
					w.lastseq.Reset()
					w.seqchanged = false
				} else if c == 'm' {
					_, _ = w.lastseq.Write(w.ansiseq.Bytes())
				}

				_, _ = w.ansiseq.WriteTo(w.Forward)
			}
		} else {
			_, err := w.writeRune(c)
			if err != nil {
				return 0, err
			}
		}
	}

	return len(b), nil
}

func (w *Writer) writeRune(r rune) (int, error) {
	if w.runeBuf == nil {
		w.runeBuf = make([]byte, utf8.UTFMax)
	}
	n := utf8.EncodeRune(w.runeBuf, r)
	return w.Forward.Write(w.runeBuf[:n])
}

func (w *Writer) LastSequence() string {
	return w.lastseq.String()
}

func (w *Writer) ResetAnsi() {
	if !w.seqchanged {
		return
	}
	_, _ = w.Forward.Write([]byte("\x1b[0m"))
}
