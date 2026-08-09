package fswatch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestNonrecursiveTree builds a nonrecursiveTree backed by a fake watcher.
func newTestNonrecursiveTree(t *testing.T) (*nonrecursiveTree, *fakeWatcher, chan EventInfo) {
	t.Helper()
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newNonrecursiveTree(w, events, nil)
	t.Cleanup(func() { _ = tr.Close() })
	return tr, w, events
}

func TestNonrecursiveTreeWatchSingleNode(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if got := w.opsFor("Watch"); len(got) != 1 || got[0] != dir {
		t.Fatalf("watcher calls = %s, want Watch(%s)", w, dir)
	}
}

// TestNonrecursiveTreeWatchRecursiveWatchesEveryDir checks that a recursive
// request is emulated by setting an individual watch on each existing
// directory in the subtree.
func TestNonrecursiveTreeWatchRecursiveWatchesEveryDir(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	root := mkdirs(t, "a", filepath.Join("a", "b"), "c")
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(root+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	want := []string{
		root,
		filepath.Join(root, "a"),
		filepath.Join(root, "a", "b"),
		filepath.Join(root, "c"),
	}
	got := w.opsFor("Watch")
	if len(got) != len(want) {
		t.Fatalf("Watch calls = %v, want one per directory %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Watch calls = %v, want %v", got, want)
		}
	}
}

func TestNonrecursiveTreeWatchExpandsEventSet(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch(Create) error = %v", err)
	}
	w.reset()
	if err := tr.Watch(dir, c, Remove); err != nil {
		t.Fatalf("Watch(Remove) error = %v", err)
	}

	last, ok := w.last()
	if !ok || last.op != "Rewatch" {
		t.Fatalf("watcher calls = %s, want Rewatch to widen the event set", w)
	}
	if last.new&(Create|Remove) != Create|Remove {
		t.Fatalf("Rewatch new event set = %v, want Create|Remove", last.new)
	}
}

func TestNonrecursiveTreeWatchSameEventSetIsNop(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()
	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if len(w.ops()) != 0 {
		t.Fatalf("watcher calls = %s, want none for an unchanged event set", w)
	}
}

func TestNonrecursiveTreeWatchEmptyEventSetIsNop(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if len(w.ops()) != 0 {
		t.Fatalf("watcher calls = %s, want none", w)
	}
}

func TestNonrecursiveTreeWatchNilChannelPanics(t *testing.T) {
	tr, _, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	defer func() {
		if recover() == nil {
			t.Fatal("Watch(nil channel) did not panic")
		}
	}()
	_ = tr.Watch(dir, nil, Create)
}

func TestNonrecursiveTreeWatchNonexistentPath(t *testing.T) {
	tr, _, _ := newTestNonrecursiveTree(t)
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(filepath.Join(mkdirs(t), "nope"), c, Create); err == nil {
		t.Fatal("Watch() on a nonexistent path succeeded, want an error")
	}
}

func TestNonrecursiveTreeWatchPropagatesWatcherError(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)
	w.fail("Watch", errWatcherBoom)

	if err := tr.Watch(dir, c, Create); !errors.Is(err, errWatcherBoom) {
		t.Fatalf("Watch() error = %v, want %v", err, errWatcherBoom)
	}

	// The watchpoint was rolled back, so a retry issues Watch again.
	w.fail("Watch", nil)
	w.reset()
	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() retry error = %v", err)
	}
	if w.countOp("Watch") != 1 {
		t.Fatalf("watcher calls = %s, want a fresh Watch after rollback", w)
	}
}

func TestNonrecursiveTreeStopUnwatches(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()

	tr.Stop(c)
	if got := w.opsFor("Unwatch"); len(got) != 1 || got[0] != dir {
		t.Fatalf("watcher calls = %s, want Unwatch(%s)", w, dir)
	}
}

func TestNonrecursiveTreeStopShrinksSharedWatch(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	first := make(chan EventInfo, buffer)
	second := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, first, Create); err != nil {
		t.Fatalf("Watch(first) error = %v", err)
	}
	if err := tr.Watch(dir, second, Remove); err != nil {
		t.Fatalf("Watch(second) error = %v", err)
	}
	w.reset()

	tr.Stop(second)
	last, ok := w.last()
	if !ok || last.op != "Rewatch" {
		t.Fatalf("watcher calls = %s, want Rewatch", w)
	}
	if last.new&Remove != 0 {
		t.Fatalf("Rewatch new event set = %v, want Remove dropped", last.new)
	}
}

func TestNonrecursiveTreeStopRecursiveUnwatchesSubtree(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	root := mkdirs(t, "a", filepath.Join("a", "b"))
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(root+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()

	tr.Stop(c)
	got := w.opsFor("Unwatch")
	want := []string{root, filepath.Join(root, "a"), filepath.Join(root, "a", "b")}
	if len(got) != len(want) {
		t.Fatalf("Unwatch calls = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Unwatch calls = %v, want %v", got, want)
		}
	}
}

func TestNonrecursiveTreeStopUnknownChannelIsNop(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()

	tr.Stop(make(chan EventInfo, buffer))
	if len(w.ops()) != 0 {
		t.Fatalf("watcher calls = %s, want none", w)
	}
}

func TestNonrecursiveTreeCloseClosesWatcher(t *testing.T) {
	w := newFakeWatcher()
	tr := newNonrecursiveTree(w, make(chan EventInfo, buffer), nil)
	if err := tr.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if w.countOp("Close") != 1 {
		t.Fatalf("watcher calls = %s, want Close", w)
	}
}

func TestNonrecursiveTreeCloseReturnsWatcherError(t *testing.T) {
	w := newFakeWatcher()
	w.fail("Close", errWatcherBoom)
	tr := newNonrecursiveTree(w, make(chan EventInfo, buffer), nil)
	if err := tr.Close(); !errors.Is(err, errWatcherBoom) {
		t.Fatalf("Close() error = %v, want %v", err, errWatcherBoom)
	}
}

func TestNonrecursiveTreeDispatchToLeaf(t *testing.T) {
	tr, _, events := newTestNonrecursiveTree(t)
	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	events <- &testEvent{path: sub, ev: Create}
	got := waitEvent(t, c)
	if got.Path() != sub || got.Event() != Create {
		t.Fatalf("received %v on %q", got.Event(), got.Path())
	}
}

func TestNonrecursiveTreeDispatchFiltersEvents(t *testing.T) {
	tr, _, events := newTestNonrecursiveTree(t)
	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	events <- &testEvent{path: sub, ev: Write}
	expectNoEvent(t, c)
}

func TestNonrecursiveTreeDispatchToRecursiveParent(t *testing.T) {
	tr, _, events := newTestNonrecursiveTree(t)
	root := mkdirs(t, "a")
	deep := filepath.Join(root, "a", "file.txt")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	events <- &testEvent{path: deep, ev: Create}
	got := waitEvent(t, c)
	if got.Path() != deep {
		t.Fatalf("received %q, want %q", got.Path(), deep)
	}
}

// TestNonrecursiveTreeAutoWatchesNewDir covers the internal() goroutine: when a
// directory is created inside a recursive watchpoint, the tree must set a watch
// on it so events from the new subtree are seen.
func TestNonrecursiveTreeAutoWatchesNewDir(t *testing.T) {
	tr, w, events := newTestNonrecursiveTree(t)
	root := mkdirs(t)
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	created := filepath.Join(root, "fresh")
	if err := os.Mkdir(created, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	w.reset()

	// isDir true routes the event into the internal channel.
	events <- &testEvent{path: created, ev: Create, dir: true}

	if !eventuallyWatched(w, created) {
		t.Fatalf("watcher calls = %s, want Watch(%s) for the new directory", w, created)
	}
}

// TestNonrecursiveTreeAutoUnwatchesRemovedDir covers internal()'s Remove branch.
func TestNonrecursiveTreeAutoUnwatchesRemovedDir(t *testing.T) {
	tr, w, events := newTestNonrecursiveTree(t)
	root := mkdirs(t, "gone")
	gone := filepath.Join(root, "gone")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", c, Create, Remove); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()

	events <- &testEvent{path: gone, ev: Remove, dir: true}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, p := range w.opsFor("Unwatch") {
			if p == gone {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("watcher calls = %s, want Unwatch(%s)", w, gone)
}

// eventuallyWatched polls for a Watch call on path, since internal() runs
// asynchronously.
func eventuallyWatched(w *fakeWatcher, path string) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, p := range w.opsFor("Watch") {
			if p == path {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestNonrecursiveTreeWatchNestedRecursiveThenChild covers watchrec's Walk
// branch, taken when a recursive watchpoint already exists on the node.
func TestNonrecursiveTreeWatchNestedRecursiveThenChild(t *testing.T) {
	tr, w, _ := newTestNonrecursiveTree(t)
	root := mkdirs(t, "a")
	first := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", first, Create); err != nil {
		t.Fatalf("Watch(first) error = %v", err)
	}
	w.reset()

	second := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", second, Remove); err != nil {
		t.Fatalf("Watch(second) error = %v", err)
	}
	// Widening a recursive watchpoint rewatches the existing directories.
	if w.countOp("Rewatch") == 0 {
		t.Fatalf("watcher calls = %s, want Rewatch for the widened event set", w)
	}
}
