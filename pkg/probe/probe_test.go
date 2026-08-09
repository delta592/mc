// Copyright (c) 2015-2021 MinIO, Inc.
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

package probe_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/delta592/mc/pkg/probe"
)

func testDummy0() *probe.Error {
	_, e := os.Stat("this-file-cannot-exit")
	return probe.NewError(e)
}

func testDummy1() *probe.Error {
	return testDummy0().Trace("DummyTag1")
}

func testDummy2() *probe.Error {
	return testDummy1().Trace("DummyTag2")
}

func TestProbe(t *testing.T) {
	probe.Init() // Set project's root source path.
	probe.SetAppInfo("Commit-ID", "7390cc957239")
	es := testDummy2().Trace("TopOfStack")
	// Uncomment the following Println to visually test probe call trace.
	// fmt.Println("Expecting a simulated error here.", es)
	require.NotNil(t, es)

	newES := es.Trace()
	require.NotNil(t, newES)
}

func TestWrappedError(t *testing.T) {
	_, e := os.Stat("this-file-cannot-exit")
	es := probe.NewError(e) // *probe.Error
	e = probe.WrapError(es) // *probe.WrappedError
	_, ok := probe.UnwrapError(e)
	require.Equal(t, true, ok)
	require.Contains(t, e.Error(), "this-file-cannot-exit")
}

func TestNewErrorNil(t *testing.T) {
	require.Nil(t, probe.NewError(nil))
}

func TestErrorMethods(t *testing.T) {
	probe.Init()
	probe.SetAppInfo("Version", "test")

	base := errors.New("base error")
	err := probe.NewError(base).Trace("tag1").Trace("tag2")
	require.NotNil(t, err)
	require.Equal(t, base, err.ToGoError())
	require.Contains(t, err.String(), "base error")
	require.Contains(t, err.String(), "tag1")

	untraced := err.Untrace()
	require.NotNil(t, untraced)
	require.Nil(t, (*probe.Error)(nil).Trace())
	require.Nil(t, (*probe.Error)(nil).Untrace())
	require.Nil(t, (*probe.Error)(nil).ToGoError())
	require.Equal(t, "<nil>", (*probe.Error)(nil).String())
}

func TestGetSysInfo(t *testing.T) {
	info := probe.GetSysInfo()
	require.NotEmpty(t, info["host.os"])
	require.NotEmpty(t, info["host.arch"])
	require.NotEmpty(t, info["mem.used"])
}

func TestUnwrapErrorDefault(t *testing.T) {
	_, ok := probe.UnwrapError(errors.New("plain"))
	require.False(t, ok)
}
