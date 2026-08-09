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
	"time"

	"github.com/stretchr/testify/require"
)

func TestMcVersionToReleaseTime(t *testing.T) {
	got, err := mcVersionToReleaseTime("2017-09-29T19:16:56Z")
	require.Nil(t, err)
	require.Equal(t, time.Date(2017, 9, 29, 19, 16, 56, 0, time.UTC), got)

	_, err = mcVersionToReleaseTime("not-a-date")
	require.NotNil(t, err)
}

func TestReleaseTagToReleaseTime(t *testing.T) {
	got, err := releaseTagToReleaseTime("RELEASE.2016-10-07T01-16-39Z")
	require.Nil(t, err)
	require.Equal(t, time.Date(2016, 10, 7, 1, 16, 39, 0, time.UTC), got)

	_, err = releaseTagToReleaseTime("BAD.2016-10-07T01-16-39Z")
	require.NotNil(t, err)

	_, err = releaseTagToReleaseTime("RELEASE.bad")
	require.NotNil(t, err)
}

func TestParseReleaseData(t *testing.T) {
	sha, releaseTime, tag, err := parseReleaseData("fbe246edbd382902db9a4035df7dce8cb441357d mc.RELEASE.2016-10-07T01-16-39Z")
	require.Nil(t, err)
	require.Equal(t, "fbe246edbd382902db9a4035df7dce8cb441357d", sha)
	require.Equal(t, "RELEASE.2016-10-07T01-16-39Z", tag)
	require.Equal(t, time.Date(2016, 10, 7, 1, 16, 39, 0, time.UTC), releaseTime)

	_, _, _, err = parseReleaseData("only-one-field")
	require.NotNil(t, err)

	_, _, _, err = parseReleaseData("deadbeef notmc.RELEASE.2016-10-07T01-16-39Z")
	require.NotNil(t, err)
}

func TestGetDownloadURL(t *testing.T) {
	tag := "RELEASE.2016-10-07T01-16-39Z"
	require.Contains(t, getDownloadURL("", tag), "archive/mc."+tag)
	require.Equal(t, "https://custom.example/archive/mc."+tag, getDownloadURL("https://custom.example/archive/old", tag))
}
