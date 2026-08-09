// Copyright (c) 2015-2024 MinIO, Inc.
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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIBytesShort(t *testing.T) {
	require.Equal(t, "1.0K", ibytesShort(1024))
	require.Equal(t, "1.0M", ibytesShort(1024*1024))
	require.Equal(t, "0B", ibytesShort(0))
}

func TestRoundDur(t *testing.T) {
	require.Equal(t, 2*time.Second, roundDur(1500*time.Millisecond+500*time.Millisecond))
	require.Equal(t, 1500*time.Millisecond, roundDur(1500*time.Millisecond+123*time.Microsecond))
	require.Equal(t, 512*time.Microsecond, roundDur(512*time.Microsecond))
}
