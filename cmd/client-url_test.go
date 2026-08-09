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
	"testing"

	"github.com/stretchr/testify/require"
)

// TestURL - tests url parsing and fields.
func TestURL(t *testing.T) {
	urlStr := "foo?.go"
	url := newClientURL(urlStr)
	require.Equal(t, "foo?.go", url.Path)

	urlStr = "https://s3.amazonaws.com/mybucket/foo?.go"
	url = newClientURL(urlStr)
	require.Equal(t, "https", url.Scheme)
	require.Equal(t, "s3.amazonaws.com", url.Host)
	require.Equal(t, "/mybucket/foo?.go", url.Path)
}

// TestURLJoinPath - tests joining two different urls.
func TestURLJoinPath(t *testing.T) {
	// Join two URLs
	url1 := "http://s3.mycompany.io/dev"
	url2 := "http://s3.aws.amazon.com/mybucket/bin/zgrep"
	url := urlJoinPath(url1, url2)
	require.Equal(t, "http://s3.mycompany.io/dev/mybucket/bin/zgrep", url)

	// Join URL and a path
	url1 = "http://s3.mycompany.io/dev"
	url2 = "mybucket/bin/zgrep"
	url = urlJoinPath(url1, url2)
	require.Equal(t, "http://s3.mycompany.io/dev/mybucket/bin/zgrep", url)

	// Check if it strips URL2's tailing `/`
	url1 = "http://s3.mycompany.io/dev"
	url2 = "mybucket/bin/"
	url = urlJoinPath(url1, url2)
	require.Equal(t, "http://s3.mycompany.io/dev/mybucket/bin/", url)
}

func Test_isURLPrefix(t *testing.T) {
	type args struct {
		src  string
		dest string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{"s3/test", "s3/test/test"}, true},
		{"test2", args{"s3/test/", "s3/test/test"}, true},
		{"test3", args{"s3/test/test", "s3/test/"}, true},
		{"test4", args{"s3/test/test", "s3/test/test.123"}, false},
		{"test5", args{"s3/test/", "s3/test/test/test/test"}, true},
		{"test6", args{"s3/test/*", "s3/test/test/"}, true},
		{"test7", args{"s3/test/*", "s3/test1/test/"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isURLPrefix(tt.args.src, tt.args.dest); got != tt.want {
				t.Errorf("isURLPrefix() = %v, want %v", got, tt.want)
			}
			if got := isURLPrefix(tt.args.dest, tt.args.src); got != tt.want {
				t.Errorf("isURLPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
