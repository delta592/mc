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
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test list files in a folder.
func TestList(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	// Create multiple files.
	objectPath := filepath.Join(root, "object1")
	fsClient, err := fsNew(objectPath)
	require.Nil(t, err)

	data := "hello"

	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	},
	)
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	objectPath = filepath.Join(root, "object2")
	fsClient, err = fsNew(objectPath)
	require.Nil(t, err)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	fsClient, err = fsNew(root)
	require.Nil(t, err)

	// Verify previously create files and list them.
	var contents []*ClientContent
	for content := range fsClient.List(globalContext, ListOptions{ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	require.Nil(t, err)
	require.Equal(t, 1, len(contents))
	require.Equal(t, true, contents[0].Type.IsDir())

	// Create another file.
	objectPath = filepath.Join(root, "test1/newObject1")
	fsClient, err = fsNew(objectPath)
	require.Nil(t, err)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	fsClient, err = fsNew(root)
	require.Nil(t, err)

	contents = nil
	// List non recursive to list only top level files.
	for content := range fsClient.List(globalContext, ListOptions{ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	require.Nil(t, err)
	require.Equal(t, 1, len(contents))
	require.Equal(t, true, contents[0].Type.IsDir())

	fsClient, err = fsNew(root)
	require.Nil(t, err)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(globalContext, ListOptions{Recursive: true, ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	require.Nil(t, err)
	require.Equal(t, 3, len(contents))

	var regularFiles int
	var regularDirs int
	// Test number of expected files and directories.
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
		if content.Type.IsDir() {
			regularDirs++
			continue
		}
	}
	require.Equal(t, 0, regularDirs)
	require.Equal(t, 3, regularFiles)

	// Create an ignored file and list to verify if its ignored.
	objectPath = filepath.Join(root, "test1/.DS_Store")
	fsClient, err = fsNew(objectPath)
	require.Nil(t, err)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	fsClient, err = fsNew(root)
	require.Nil(t, err)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(globalContext, ListOptions{Recursive: true, ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	require.Nil(t, err)
	switch runtime.GOOS {
	case "darwin":
		require.Equal(t, 3, len(contents))
	default:
		require.Equal(t, 4, len(contents))
	}

	regularFiles = 0
	// Test number of expected files.
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
	}
	switch runtime.GOOS {
	case "darwin":
		require.Equal(t, 3, regularFiles)
	default:
		require.Equal(t, 4, regularFiles)
	}
}

// Test put bucket aka 'mkdir()' operation.
func TestPutBucket(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fsNew(bucketPath)
	require.Nil(t, err)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	require.Nil(t, err)
}

// Test stat bucket aka 'stat()' operation.
func TestStatBucket(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsClient, err := fsNew(bucketPath)
	require.Nil(t, err)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	require.Nil(t, err)
	_, err = fsClient.Stat(context.Background(), StatOptions{})
	require.Nil(t, err)
}

// Test bucket acl fails for directories.
func TestBucketACLFails(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fsNew(bucketPath)
	require.Nil(t, err)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	require.Nil(t, err)

	// On windows setting permissions is not supported.
	if runtime.GOOS != "windows" {
		err = fsClient.SetAccess(context.Background(), "readonly", false)
		require.Nil(t, err)

		_, _, err = fsClient.GetAccess(context.Background())
		require.Nil(t, err)
	}
}

// Test creating a file.
func TestPut(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	require.Nil(t, err)

	data := "hello"
	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	},
	)

	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)
}

// Test read a file.
func TestGet(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	require.Nil(t, err)

	data := "hello"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	reader, _, err = fsClient.Get(context.Background(), GetOptions{})
	require.Nil(t, err)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	require.NoError(t, e)
	require.Equal(t, []byte(data), results.Bytes())
}

// Test get range in a file.
func TestGetRange(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	require.Nil(t, err)

	data := "hello world"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	reader, _, err = fsClient.Get(context.Background(), GetOptions{})
	require.Nil(t, err)
	var results bytes.Buffer
	buf := make([]byte, 5)
	m, e := reader.(io.ReaderAt).ReadAt(buf, 0)
	require.NoError(t, e)
	require.Equal(t, 5, m)
	_, e = results.Write(buf)
	require.NoError(t, e)
	require.Equal(t, []byte("hello"), results.Bytes())
}

// Test stat file.
func TestStatObject(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	require.Nil(t, err)

	data := "hello"
	dataLen := len(data)
	reader := bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(dataLen), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	},
	)
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)

	content, err := fsClient.Stat(context.Background(), StatOptions{})
	require.Nil(t, err)
	require.Equal(t, int64(dataLen), content.Size)
}

// Test copy.
func TestCopy(t *testing.T) {
	root, e := os.MkdirTemp(os.TempDir(), "fs-")
	require.NoError(t, e)
	defer os.RemoveAll(root)
	sourcePath := filepath.Join(root, "source")
	targetPath := filepath.Join(root, "target")
	fsClientTarget, err := fsNew(targetPath)
	require.Nil(t, err)
	fsClientSource, err := fsNew(sourcePath)
	require.Nil(t, err)

	data := "hello world"
	reader := bytes.NewReader([]byte(data))
	n, err := fsClientSource.Put(context.Background(), reader, int64(len(data)), nil, PutOptions{
		metadata: map[string]string{
			"Content-Type": "application/octet-stream",
		},
	})
	require.Nil(t, err)
	require.Equal(t, int64(len(data)), n)
	err = fsClientTarget.Copy(context.Background(), sourcePath, CopyOptions{size: int64(len(data))}, nil)
	require.Nil(t, err)
}
