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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgKVSEmpty(t *testing.T) {
	var kvs argKVS
	require.True(t, kvs.Empty())

	kvs = argKVS{{Key: "k", Value: "v"}}
	require.False(t, kvs.Empty())
}

func TestArgKVSSetGetLookup(t *testing.T) {
	kvs := argKVS{{Key: "a", Value: "1"}}
	kvs.Set("b", "2")
	require.Equal(t, "1", kvs.Get("a"))
	require.Equal(t, "2", kvs.Get("b"))
	require.Equal(t, "", kvs.Get("missing"))

	v, ok := kvs.Lookup("a")
	require.True(t, ok)
	require.Equal(t, "1", v)

	_, ok = kvs.Lookup("missing")
	require.False(t, ok)

	kvs.Set("a", "updated")
	require.Equal(t, "updated", kvs.Get("a"))
}
