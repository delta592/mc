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

	"github.com/delta592/mc/pkg/probe"
	"github.com/stretchr/testify/require"
)

func TestGetMetricsV3Path(t *testing.T) {
	tests := []struct {
		subsys string
		bucket string
		want   string
	}{
		{"", "", metricsV3EndPointRoot},
		{"api", "", metricsV3EndPointRoot + "/api"},
		{"system", "", metricsV3EndPointRoot + "/system"},
		{"api", "mybucket", metricsV3EndPointRoot + "/bucket/api/mybucket"},
		{"replication", "logs", metricsV3EndPointRoot + "/bucket/replication/logs"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, getMetricsV3Path(tt.subsys, tt.bucket))
	}
}

func TestURLsEqual(t *testing.T) {
	src := &ClientContent{URL: ClientURL{Path: "s3://bucket/a"}}
	dst := &ClientContent{URL: ClientURL{Path: "s3://bucket/b"}}

	require.True(t, URLs{
		SourceContent: src,
		TargetContent: dst,
	}.Equal(URLs{
		SourceContent: src,
		TargetContent: dst,
	}))

	require.False(t, URLs{
		SourceContent: src,
		TargetContent: dst,
	}.Equal(URLs{
		SourceContent: &ClientContent{URL: ClientURL{Path: "s3://bucket/x"}},
		TargetContent: dst,
	}))

	require.False(t, URLs{
		SourceContent: src,
		TargetContent: nil,
	}.Equal(URLs{
		SourceContent: src,
		TargetContent: dst,
	}))
}

func TestURLsWithError(t *testing.T) {
	err := probe.NewError(nil)
	got := URLs{SourceAlias: "local"}.WithError(err)
	require.Equal(t, err, got.Error)
	require.Equal(t, "local", got.SourceAlias)
}
