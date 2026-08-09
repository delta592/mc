package fswatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNodeAddGetDel(t *testing.T) {
	rootPath := filepath.Join(string(os.PathSeparator), "watch")
	root := newnode(rootPath)

	childPath := filepath.Join(rootPath, "a", "b")
	root.Add(childPath)

	got, err := root.Get(childPath)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != childPath {
		t.Fatalf("Get() name = %q, want %q", got.Name, childPath)
	}

	if err := root.Del(childPath); err != nil {
		t.Fatalf("Del() error = %v", err)
	}
	if _, err := root.Get(childPath); err == nil {
		t.Fatal("expected error after Del()")
	}
}

func TestNodeGetMissing(t *testing.T) {
	root := filepath.Join(string(os.PathSeparator), "watch")
	nd := newnode(root)
	if _, err := nd.Get(filepath.Join(root, "missing")); err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestNodeAddOutsideRoot(t *testing.T) {
	root := filepath.Join(string(os.PathSeparator), "watch")
	nd := newnode(root)
	if got := nd.Add("/other/path"); got.Name != "" {
		t.Fatalf("Add outside root should return empty node, got %+v", got)
	}
}

func TestErrNotExist(t *testing.T) {
	err := errnotexist("/missing")
	if err == nil {
		t.Fatal("errnotexist() should not be nil")
	}
}

func TestWatchpoint(t *testing.T) {
	wp := make(watchpoint)
	ch := make(chan EventInfo, 1)
	wp.Add(ch, Create)
	wp.Dispatch(testEventInfo{ev: Create, path: "/tmp/x"}, 0)
	select {
	case got := <-ch:
		if got.Event() != Create {
			t.Fatalf("got event %v", got.Event())
		}
	default:
		t.Fatal("expected dispatched event")
	}
	wp.Del(ch, Create)
}

type testEventInfo struct {
	ev   Event
	path string
}

func (t testEventInfo) Event() Event     { return t.ev }
func (t testEventInfo) Path() string     { return t.path }
func (t testEventInfo) Sys() interface{} { return nil }
