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
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrepareUpdateMessage(t *testing.T) {
	require.Equal(t, "", prepareUpdateMessage("", time.Hour))
	require.Equal(t, "", prepareUpdateMessage("https://example.com/mc", 0))
	require.Equal(t, "", prepareUpdateMessage("https://example.com/mc", -time.Hour))

	msg := prepareUpdateMessage("https://example.com/mc", 24*time.Hour)
	require.Contains(t, msg, "https://example.com/mc")
	require.Contains(t, msg, "older version")
}

func TestColorizeUpdateMessage(t *testing.T) {
	msg := colorizeUpdateMessage("https://example.com/mc", "2 days ago")
	require.Contains(t, msg, "https://example.com/mc")
	require.Contains(t, msg, "2 days ago")
	require.True(t, strings.HasPrefix(strings.TrimSpace(msg), "\n") || strings.Contains(msg, "older version") || strings.Contains(msg, "+"))
}
