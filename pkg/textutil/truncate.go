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

	"github.com/mattn/go-runewidth"

	"github.com/delta592/mc/pkg/textutil/ansi"
)

type truncateWriter struct {
	width uint
	tail  string

	ansiWriter *ansi.Writer
	buf        bytes.Buffer
	inANSI     bool
}

func newTruncateWriter(width uint, tail string) *truncateWriter {
	w := &truncateWriter{
		width: width,
		tail:  tail,
	}
	w.ansiWriter = &ansi.Writer{
		Forward: &w.buf,
	}
	return w
}

// TruncateWithTail truncates s to the given printable width and appends tail.
func TruncateWithTail(s string, width uint, tail string) string {
	tw := runewidth.StringWidth(tail)
	if width < uint(tw) {
		return tail
	}

	w := newTruncateWriter(width-uint(tw), tail)
	_, _ = w.Write([]byte(s))
	return w.String()
}

func (w *truncateWriter) Write(b []byte) (int, error) {
	var curWidth uint

	for _, c := range string(b) {
		if c == ansi.Marker {
			w.inANSI = true
		} else if w.inANSI {
			if ansi.IsTerminator(c) {
				w.inANSI = false
			}
		} else {
			curWidth += uint(runewidth.RuneWidth(c))
		}

		if curWidth > w.width {
			n, err := w.buf.WriteString(w.tail)
			if w.ansiWriter.LastSequence() != "" {
				w.ansiWriter.ResetAnsi()
			}
			return n, err
		}

		_, err := w.ansiWriter.Write([]byte(string(c)))
		if err != nil {
			return 0, err
		}
	}

	return len(b), nil
}

func (w *truncateWriter) String() string {
	return w.buf.String()
}
