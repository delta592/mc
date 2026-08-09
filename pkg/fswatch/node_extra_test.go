package fswatch

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// abs builds an absolute path from the given segments, rooted at the platform
// path separator.
func abs(segments ...string) string {
	return filepath.Join(append([]string{string(os.PathSeparator)}, segments...)...)
}

func TestNodeAddNestedCreatesIntermediates(t *testing.T) {
	root := newnode(abs("watch"))
	deep := abs("watch", "a", "b", "c")
	if got := root.Add(deep); got.Name != deep {
		t.Fatalf("Add() = %q, want %q", got.Name, deep)
	}

	// Every intermediate node must be reachable.
	for _, p := range []string{
		abs("watch", "a"),
		abs("watch", "a", "b"),
		abs("watch", "a", "b", "c"),
	} {
		if _, err := root.Get(p); err != nil {
			t.Fatalf("Get(%q) error = %v", p, err)
		}
	}
}

func TestNodeAddIsIdempotent(t *testing.T) {
	root := newnode(abs("watch"))
	p := abs("watch", "a", "b")
	first := root.Add(p)
	first.Watch.Add(make(chan EventInfo, 1), Create)
	second := root.Add(p)
	if len(second.Watch) == 0 {
		t.Fatal("Add() replaced the existing node instead of returning it")
	}
}

func TestNodeGetErrors(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a", "b"))

	tests := []struct {
		name string
		path string
	}{
		{"outside root", abs("other", "a")},
		{"equal to root", abs("watch")},
		{"missing leaf", abs("watch", "a", "missing")},
		{"missing intermediate", abs("watch", "missing", "b")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := root.Get(tc.path); err == nil {
				t.Fatalf("Get(%q) succeeded, want an error", tc.path)
			}
		})
	}
}

func TestNodeDelPrunesEmptyParents(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a", "b", "c"))

	if err := root.Del(abs("watch", "a", "b", "c")); err != nil {
		t.Fatalf("Del() error = %v", err)
	}
	// The now-empty intermediates are pruned as well.
	if _, err := root.Get(abs("watch", "a", "b")); err == nil {
		t.Fatal("Get() found a node that should have been pruned")
	}
}

func TestNodeDelKeepsWatchedParents(t *testing.T) {
	root := newnode(abs("watch"))
	parent := root.Add(abs("watch", "a"))
	parent.Watch.Add(make(chan EventInfo, 1), Create)
	root.Add(abs("watch", "a", "b"))

	if err := root.Del(abs("watch", "a", "b")); err != nil {
		t.Fatalf("Del() error = %v", err)
	}
	if _, err := root.Get(abs("watch", "a")); err != nil {
		t.Fatalf("Get(parent) error = %v, want the watched parent kept", err)
	}
}

func TestNodeDelErrors(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a"))

	for _, p := range []string{
		abs("other", "a"),
		abs("watch", "missing"),
		abs("watch", "missing", "deep"),
	} {
		if err := root.Del(p); err == nil {
			t.Errorf("Del(%q) succeeded, want an error", p)
		}
	}
}

func TestNodeWalkVisitsSubtree(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a", "b"))
	root.Add(abs("watch", "c"))

	var seen []string
	if err := root.Walk(func(nd node) error {
		seen = append(seen, nd.Name)
		return nil
	}); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	sort.Strings(seen)
	want := []string{
		abs("watch"),
		abs("watch", "a"),
		abs("watch", "a", "b"),
		abs("watch", "c"),
	}
	if len(seen) != len(want) {
		t.Fatalf("Walk() visited %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("Walk() visited %v, want %v", seen, want)
		}
	}
}

func TestNodeWalkSkipsSubtree(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a", "b"))

	var seen []string
	err := root.Walk(func(nd node) error {
		seen = append(seen, nd.Name)
		if nd.Name == abs("watch", "a") {
			return errSkip
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	for _, name := range seen {
		if name == abs("watch", "a", "b") {
			t.Fatal("Walk() descended past errSkip")
		}
	}
}

func TestNodeWalkPropagatesError(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a"))
	boom := errors.New("boom")
	if err := root.Walk(func(node) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("Walk() error = %v, want %v", err, boom)
	}
}

// TestNodeWalkSkipsInactiveChild checks Walk ignores the empty-named node used
// to store inactive watchpoints.
func TestNodeWalkSkipsInactiveChild(t *testing.T) {
	root := newnode(abs("watch"))
	child := root.Add(abs("watch", "a"))
	watchAddInactive(child, make(chan EventInfo, 1), Create)

	if err := root.Walk(func(nd node) error {
		if nd.Name == "" {
			t.Fatal("Walk() visited the inactive-watchpoint node")
		}
		return nil
	}); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
}

func TestNodeWalkPathVisitsAncestors(t *testing.T) {
	root := newnode(abs("watch"))
	target := abs("watch", "a", "b")
	root.Add(target)

	var seen []string
	var base string
	err := root.WalkPath(target, func(nd node, isbase bool) error {
		seen = append(seen, nd.Name)
		if isbase {
			base = nd.Name
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkPath() error = %v", err)
	}
	if base != target {
		t.Fatalf("WalkPath() base = %q, want %q", base, target)
	}
	want := []string{abs("watch"), abs("watch", "a"), target}
	if len(seen) != len(want) {
		t.Fatalf("WalkPath() visited %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("WalkPath() visited %v, want %v", seen, want)
		}
	}
}

func TestNodeWalkPathErrors(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a"))

	if err := root.WalkPath(abs("other", "a"), func(node, bool) error { return nil }); err == nil {
		t.Fatal("WalkPath() outside root succeeded, want an error")
	}
	if err := root.WalkPath(abs("watch", "missing"), func(node, bool) error { return nil }); err == nil {
		t.Fatal("WalkPath() to a missing node succeeded, want an error")
	}
	if err := root.WalkPath(abs("watch", "missing", "deep"), func(node, bool) error { return nil }); err == nil {
		t.Fatal("WalkPath() through a missing intermediate succeeded, want an error")
	}
}

func TestNodeWalkPathSkipStopsEarly(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a", "b"))

	calls := 0
	err := root.WalkPath(abs("watch", "a", "b"), func(node, bool) error {
		calls++
		return errSkip
	})
	if err != nil {
		t.Fatalf("WalkPath() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("WalkPath() made %d calls, want 1 before errSkip stopped it", calls)
	}
}

func TestNodeWalkPathPropagatesError(t *testing.T) {
	root := newnode(abs("watch"))
	root.Add(abs("watch", "a", "b"))
	boom := errors.New("boom")

	if err := root.WalkPath(abs("watch", "a", "b"), func(node, bool) error {
		return boom
	}); !errors.Is(err, boom) {
		t.Fatalf("WalkPath() error = %v, want %v", err, boom)
	}
}

// TestNodeAddDirWalksFilesystem covers AddDir, which mirrors the on-disk
// directory layout into the node tree.
func TestNodeAddDirWalksFilesystem(t *testing.T) {
	dir := mkdirs(t, "a", filepath.Join("a", "b"), "c")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	root := newnode(dir)
	var seen []string
	if err := root.AddDir(func(nd node) error {
		seen = append(seen, nd.Name)
		return nil
	}); err != nil {
		t.Fatalf("AddDir() error = %v", err)
	}

	sort.Strings(seen)
	want := []string{dir, filepath.Join(dir, "a"), filepath.Join(dir, "a", "b"), filepath.Join(dir, "c")}
	if len(seen) != len(want) {
		t.Fatalf("AddDir() visited %v, want %v (files must be skipped)", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("AddDir() visited %v, want %v", seen, want)
		}
	}
}

func TestNodeAddDirSkip(t *testing.T) {
	dir := mkdirs(t, "a", filepath.Join("a", "b"))
	root := newnode(dir)

	var seen []string
	if err := root.AddDir(func(nd node) error {
		seen = append(seen, nd.Name)
		if nd.Name == filepath.Join(dir, "a") {
			return errSkip
		}
		return nil
	}); err != nil {
		t.Fatalf("AddDir() error = %v", err)
	}
	for _, name := range seen {
		if name == filepath.Join(dir, "a", "b") {
			t.Fatal("AddDir() descended past errSkip")
		}
	}
}

func TestNodeAddDirPropagatesError(t *testing.T) {
	dir := mkdirs(t, "a")
	root := newnode(dir)
	boom := errors.New("boom")

	err := root.AddDir(func(node) error { return boom })
	if err == nil {
		t.Fatal("AddDir() succeeded, want an error")
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("AddDir() error = %T(%v), want *os.PathError", err, err)
	}
}

func TestNodeAddDirMissingDirectory(t *testing.T) {
	root := newnode(filepath.Join(mkdirs(t), "missing"))
	if err := root.AddDir(func(node) error { return nil }); err == nil {
		t.Fatal("AddDir() on a missing directory succeeded, want an error")
	}
}

func TestRootAddGetDelWalk(t *testing.T) {
	r := root{nd: newnode("")}
	p := abs("watch", "a")
	if got := r.Add(p); got.Name != p {
		t.Fatalf("Add() = %q, want %q", got.Name, p)
	}
	if got, err := r.Get(p); err != nil || got.Name != p {
		t.Fatalf("Get() = %q, %v", got.Name, err)
	}

	var seen []string
	if err := r.Walk(p, func(nd node) error {
		seen = append(seen, nd.Name)
		return nil
	}); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(seen) != 1 || seen[0] != p {
		t.Fatalf("Walk() visited %v, want [%s]", seen, p)
	}

	var walked []string
	if err := r.WalkPath(p, func(nd node, _ bool) error {
		walked = append(walked, nd.Name)
		return nil
	}); err != nil {
		t.Fatalf("WalkPath() error = %v", err)
	}
	if len(walked) == 0 {
		t.Fatal("WalkPath() visited nothing")
	}

	if err := r.Del(p); err != nil {
		t.Fatalf("Del() error = %v", err)
	}
	if _, err := r.Get(p); err == nil {
		t.Fatal("Get() succeeded after Del()")
	}
}

func TestRootGetMissing(t *testing.T) {
	r := root{nd: newnode("")}
	if _, err := r.Get(abs("watch", "missing")); err == nil {
		t.Fatal("Get() succeeded for a missing path, want an error")
	}
}

func TestRootWalkMissing(t *testing.T) {
	r := root{nd: newnode("")}
	if err := r.Walk(abs("watch", "missing"), func(node) error { return nil }); err == nil {
		t.Fatal("Walk() succeeded for a missing path, want an error")
	}
}

func TestRootAddDir(t *testing.T) {
	dir := mkdirs(t, "a")
	r := root{nd: newnode("")}
	var seen []string
	if err := r.AddDir(dir, func(nd node) error {
		seen = append(seen, nd.Name)
		return nil
	}); err != nil {
		t.Fatalf("AddDir() error = %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("AddDir() visited %v, want the root and one child", seen)
	}
}
