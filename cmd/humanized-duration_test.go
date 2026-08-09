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

func TestTimeDurationToHumanizedDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want humanizedDuration
	}{
		{500 * time.Millisecond, humanizedDuration{MilliSeconds: 500}},
		{30 * time.Second, humanizedDuration{Seconds: 30}},
		{90 * time.Second, humanizedDuration{Minutes: 1, Seconds: 30}},
		{3661 * time.Second, humanizedDuration{Hours: 1, Minutes: 1, Seconds: 1}},
		{25 * time.Hour, humanizedDuration{Days: 1, Hours: 1}},
	}
	for _, tt := range tests {
		got := timeDurationToHumanizedDuration(tt.in)
		require.Equal(t, tt.want, got)
	}
}

func TestHumanizedDurationString(t *testing.T) {
	tests := []struct {
		d    humanizedDuration
		want string
	}{
		{humanizedDuration{MilliSeconds: 250}, "250 milliseconds"},
		{humanizedDuration{Seconds: 5}, "5 seconds"},
		{humanizedDuration{Minutes: 2, Seconds: 3}, "2 minutes 3 seconds"},
		{humanizedDuration{Hours: 1, Minutes: 2, Seconds: 3}, "1 hours 2 minutes 3 seconds"},
		{humanizedDuration{Days: 3, Hours: 4, Minutes: 5, Seconds: 6}, "3 days 4 hours 5 minutes 6 seconds"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.d.String())
	}
}

func TestHumanizedDurationStringShort(t *testing.T) {
	tests := []struct {
		d    humanizedDuration
		want string
	}{
		{humanizedDuration{MilliSeconds: 250}, "250 milliseconds"},
		{humanizedDuration{Seconds: 5}, "5 seconds"},
		{humanizedDuration{Minutes: 2, Seconds: 3}, "2 minutes"},
		{humanizedDuration{Hours: 1, Minutes: 2, Seconds: 3}, "1 hours 2 minutes"},
		{humanizedDuration{Days: 1, Hours: 4, Minutes: 5, Seconds: 6}, "1 days, 4 hours"},
		{humanizedDuration{Days: 5, Hours: 4, Minutes: 5, Seconds: 6}, "5 days"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.d.StringShort())
	}
}
