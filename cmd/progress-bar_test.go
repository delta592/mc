// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixateBarCaption(t *testing.T) {
	require.Equal(t, "hello", fixateBarCaption("hello", 5))
	require.Equal(t, "hello", fixateBarCaption("hello", 5))
	require.True(t, strings.HasPrefix(fixateBarCaption("hello world foo bar", 8), "..."))
	require.Equal(t, 8, len(fixateBarCaption("hi", 8)))
}

func TestGetFixedWidth(t *testing.T) {
	require.Equal(t, 50, getFixedWidth(100, 50))
	require.Equal(t, 0, getFixedWidth(100, 0))
	require.Equal(t, 100, getFixedWidth(100, 100))
}
