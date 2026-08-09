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

// Package limiter implements throughput upload and download limits via http.RoundTripper
package limiter

import (
	"context"
	"errors"
	"io"
	"net/http"

	"golang.org/x/time/rate"
)

type limiter struct {
	upload    *rate.Limiter
	download  *rate.Limiter
	transport http.RoundTripper // HTTP transport that needs to be intercepted
}

type rateReader struct {
	r   io.Reader
	lim *rate.Limiter
}

func (r rateReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n <= 0 {
		return n, err
	}
	if waitErr := r.lim.WaitN(context.Background(), n); waitErr != nil {
		return n, waitErr
	}
	return n, err
}

func (l limiter) limitReader(r io.Reader, lim *rate.Limiter) io.Reader {
	if lim == nil {
		return r
	}
	return rateReader{r: r, lim: lim}
}

// RoundTrip executes user provided request and response hooks for each HTTP call.
func (l limiter) RoundTrip(req *http.Request) (res *http.Response, err error) {
	if l.transport == nil {
		return nil, errors.New("Invalid Argument")
	}

	type readCloser struct {
		io.Reader
		io.Closer
	}

	if req.Body != nil {
		req.Body = &readCloser{
			Reader: l.limitReader(req.Body, l.upload),
			Closer: req.Body,
		}
	}

	res, err = l.transport.RoundTrip(req)
	if res != nil && res.Body != nil {
		res.Body = &readCloser{
			Reader: l.limitReader(res.Body, l.download),
			Closer: res.Body,
		}
	}

	return res, err
}

// New return a ratelimited transport
func New(uploadLimit, downloadLimit int64, transport http.RoundTripper) http.RoundTripper {
	if uploadLimit == 0 && downloadLimit == 0 {
		return transport
	}

	var (
		uploadLimiter   *rate.Limiter
		downloadLimiter *rate.Limiter
	)

	if uploadLimit > 0 {
		uploadLimiter = rate.NewLimiter(rate.Limit(uploadLimit), int(uploadLimit))
	}

	if downloadLimit > 0 {
		downloadLimiter = rate.NewLimiter(rate.Limit(downloadLimit), int(downloadLimit))
	}

	return &limiter{
		upload:    uploadLimiter,
		download:  downloadLimiter,
		transport: transport,
	}
}
