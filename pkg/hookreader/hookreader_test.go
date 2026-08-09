// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package hookreader

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// customReader - implements custom progress reader.
type customReader struct {
	readBytes int
}

func (c *customReader) Read(b []byte) (n int, err error) {
	c.readBytes += len(b)
	return len(b), nil
}

// Tests hook reader implementation.
func TestHookReader(t *testing.T) {
	var buffer bytes.Buffer
	writer := &buffer
	_, err := writer.Write([]byte("Hello"))
	require.NoError(t, err)
	progress := &customReader{}
	reader := NewHook(&buffer, progress)
	b := make([]byte, 3)
	n, err := reader.Read(b)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, 3, progress.readBytes)
}

func TestNewHookNil(t *testing.T) {
	src := strings.NewReader("data")
	require.Equal(t, src, NewHook(src, nil))
}

func TestHookReaderSeek(t *testing.T) {
	src := strings.NewReader("0123456789")
	hook := strings.NewReader("0123456789")
	reader := NewHook(src, hook).(io.Seeker)
	n, err := reader.Seek(5, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, int64(5), n)
}

func TestHookReaderSeekFromHook(t *testing.T) {
	source := strings.NewReader("abc")
	hook := strings.NewReader("abc")
	reader := NewHook(source, hook).(io.Seeker)
	n, err := reader.Seek(1, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}

type seekableReader struct {
	data string
	pos  int64
}

func (s *seekableReader) Read(p []byte) (int, error) {
	if s.pos >= int64(len(s.data)) {
		return 0, io.EOF
	}
	n := copy(p, s.data[s.pos:])
	s.pos += int64(n)
	return n, nil
}

func (s *seekableReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.pos = offset
	case io.SeekCurrent:
		s.pos += offset
	case io.SeekEnd:
		s.pos = int64(len(s.data)) + offset
	}
	return s.pos, nil
}

func TestHookReaderSeekUsesHook(t *testing.T) {
	source := &readOnlyReader{data: "abc"}
	hook := &seekableReader{data: "abc"}
	reader := NewHook(source, hook).(io.Seeker)
	n, err := reader.Seek(2, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}

type readOnlyReader struct {
	data string
	pos  int
}

func (r *readOnlyReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestHookReaderSourceError(t *testing.T) {
	reader := NewHook(&errorReader{err: io.ErrUnexpectedEOF}, &customReader{})
	_, err := reader.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestHookReaderHookError(t *testing.T) {
	reader := NewHook(strings.NewReader("a"), &errorReader{err: io.ErrShortBuffer})
	_, err := reader.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.ErrShortBuffer)
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

func (e *errorReader) Seek(offset int64, _ int) (int64, error) {
	return offset, nil
}
