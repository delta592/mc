// Copyright (c) 2015-2022 MinIO, Inc.
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

package cmd

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidPERMS(t *testing.T) {
	perms := accessPerms("none")
	require.Equal(t, true, perms.isValidAccessPERM())
	require.Equal(t, "none", string(perms))
	perms = accessPerms("public")
	require.Equal(t, true, perms.isValidAccessPERM())
	require.Equal(t, "public", string(perms))
	perms = accessPerms("private")
	require.Equal(t, true, perms.isValidAccessPERM())
	require.Equal(t, "private", string(perms))
	perms = accessPerms("download")
	require.Equal(t, true, perms.isValidAccessPERM())
	require.Equal(t, "download", string(perms))
	perms = accessPerms("upload")
	require.Equal(t, true, perms.isValidAccessPERM())
	require.Equal(t, "upload", string(perms))
}

func TestInvalidPERMS(t *testing.T) {
	perms := accessPerms("invalid")
	require.Equal(t, false, perms.isValidAccessPERM())
}

func TestGetMcConfigDir(t *testing.T) {
	dir, err := getMcConfigDir()
	require.Nil(t, err)
	require.NotEqual(t, "", dir)
	require.Equal(t, dir, mustGetMcConfigDir())
}

func TestGetMcConfigPath(t *testing.T) {
	dir, err := getMcConfigPath()
	require.Nil(t, err)
	switch runtime.GOOS {
	case "linux", "freebsd", "darwin", "solaris":
		require.Equal(t, filepath.Join(mustGetMcConfigDir(), "config.json"), dir)
	case "windows":
		require.Equal(t, filepath.Join(mustGetMcConfigDir(), "config.json"), dir)
	default:
		t.Fatalf("unsupported platform")
	}
	require.Equal(t, dir, mustGetMcConfigPath())
}

func TestIsvalidAliasName(t *testing.T) {
	assert.Equal(t, true, isValidAlias("helloWorld0"))
	assert.Equal(t, true, isValidAlias("hello_World0"))
	assert.Equal(t, true, isValidAlias("h0SFD2k24Fdsa"))
	assert.Equal(t, true, isValidAlias("fdslka-4"))
	assert.Equal(t, true, isValidAlias("fdslka-"))
	assert.Equal(t, false, isValidAlias("helloWorld$"))
	assert.Equal(t, false, isValidAlias("h0SFD2k2#Fdsa"))
	assert.Equal(t, false, isValidAlias("0dslka-4"))
	assert.Equal(t, false, isValidAlias("-fdslka"))
}

func TestHumanizedTime(t *testing.T) {
	hTime := timeDurationToHumanizedDuration(time.Duration(10) * time.Second)
	require.Equal(t, int64(0), hTime.Minutes)
	require.Equal(t, int64(0), hTime.Hours)
	require.Equal(t, int64(0), hTime.Days)

	hTime = timeDurationToHumanizedDuration(time.Duration(10) * time.Minute)
	require.Equal(t, int64(0), hTime.Hours)
	require.Equal(t, int64(0), hTime.Days)

	hTime = timeDurationToHumanizedDuration(time.Duration(10) * time.Hour)
	require.Equal(t, int64(0), hTime.Days)

	hTime = timeDurationToHumanizedDuration(time.Duration(24) * time.Hour)
	require.NotEqual(t, int64(0), hTime.Days)
}
