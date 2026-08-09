package fswatch

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// newTestRecursiveTree builds a recursiveTree backed by a fake watcher and
// tears it down when the test ends.
func newTestRecursiveTree(t *testing.T) (*recursiveTree, *fakeWatcher) {
	t.Helper()
	w := newFakeWatcher()
	tr := newRecursiveTree(w, make(chan EventInfo, buffer))
	t.Cleanup(func() { _ = tr.Close() })
	return tr, w
}

// waitEvent reads one event from c, failing if none arrives promptly.
func waitEvent(t *testing.T, c chan EventInfo) EventInfo {
	t.Helper()
	select {
	case ei := <-c:
		return ei
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for an event")
		return nil
	}
}

// expectNoEvent asserts nothing is delivered on c within a short grace period.
func expectNoEvent(t *testing.T, c chan EventInfo) {
	t.Helper()
	select {
	case ei := <-c:
		t.Fatalf("unexpected event %v on %q", ei.Event(), ei.Path())
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRecursiveTreeWatchSingleNode(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if got := w.opsFor("Watch"); len(got) != 1 || got[0] != dir {
		t.Fatalf("watcher calls = %s, want a single Watch(%s)", w, dir)
	}
}

func TestRecursiveTreeWatchRecursiveNode(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	// The "..." suffix requests a recursive watch.
	if err := tr.Watch(dir+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if got := w.opsFor("RecursiveWatch"); len(got) != 1 || got[0] != dir {
		t.Fatalf("watcher calls = %s, want RecursiveWatch(%s)", w, dir)
	}
}

// TestRecursiveTreeWatchChildOfExistingParent covers case 1: the new path is
// already covered by a recursive parent watch, so the parent is rewatched with
// a widened event set instead of a new watch being created.
func TestRecursiveTreeWatchChildOfExistingParent(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")

	parent := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", parent, Create); err != nil {
		t.Fatalf("Watch(parent) error = %v", err)
	}
	w.reset()

	child := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, child, Remove); err != nil {
		t.Fatalf("Watch(child) error = %v", err)
	}

	// Widening the event set of a recursive parent uses RecursiveRewatch, not
	// a fresh watch on the child.
	if n := w.countOp("Watch"); n != 0 {
		t.Fatalf("watcher calls = %s, want no plain Watch on the child", w)
	}
	last, ok := w.last()
	if !ok || last.op != "RecursiveRewatch" || last.path != root {
		t.Fatalf("watcher calls = %s, want RecursiveRewatch(%s)", w, root)
	}
	if last.new&Remove == 0 {
		t.Fatalf("RecursiveRewatch new event set = %v, want it to include Remove", last.new)
	}
}

// TestRecursiveTreeWatchChildAlreadyCovered covers the diff == none branch:
// the parent's event set already covers the child, so the watcher is untouched.
func TestRecursiveTreeWatchChildAlreadyCovered(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")

	parent := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", parent, Create, Remove, Write, Rename); err != nil {
		t.Fatalf("Watch(parent) error = %v", err)
	}
	w.reset()

	if err := tr.Watch(sub, parent, Create); err != nil {
		t.Fatalf("Watch(child) error = %v", err)
	}
	if len(w.ops()) != 0 {
		t.Fatalf("watcher calls = %s, want none for an already-covered child", w)
	}
}

// TestRecursiveTreeWatchNewParentOverOneChild covers case 2 with a single
// existing child: the child's watch is relocated up to the new parent.
func TestRecursiveTreeWatchNewParentOverOneChild(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")

	child := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, child, Create); err != nil {
		t.Fatalf("Watch(child) error = %v", err)
	}
	w.reset()

	parent := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", parent, Create); err != nil {
		t.Fatalf("Watch(parent) error = %v", err)
	}

	last, ok := w.last()
	if !ok || last.op != "RecursiveRewatch" {
		t.Fatalf("watcher calls = %s, want RecursiveRewatch", w)
	}
	if last.path != sub || last.newpath != root {
		t.Fatalf("RecursiveRewatch(%s, %s), want (%s, %s)", last.path, last.newpath, sub, root)
	}
}

// TestRecursiveTreeWatchNewParentOverManyChildren covers case 2 with several
// children: the parent gets one recursive watch and every child is unwatched.
func TestRecursiveTreeWatchNewParentOverManyChildren(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	root := mkdirs(t, "a", "b")
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")

	c := make(chan EventInfo, buffer)
	if err := tr.Watch(a, c, Create); err != nil {
		t.Fatalf("Watch(a) error = %v", err)
	}
	if err := tr.Watch(b, c, Create); err != nil {
		t.Fatalf("Watch(b) error = %v", err)
	}
	w.reset()

	parent := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", parent, Create); err != nil {
		t.Fatalf("Watch(parent) error = %v", err)
	}

	if got := w.opsFor("RecursiveWatch"); len(got) != 1 || got[0] != root {
		t.Fatalf("watcher calls = %s, want RecursiveWatch(%s)", w, root)
	}
	unwatched := w.opsFor("Unwatch")
	if len(unwatched) != 2 || unwatched[0] != a || unwatched[1] != b {
		t.Fatalf("Unwatch calls = %v, want [%s %s]", unwatched, a, b)
	}
}

func TestRecursiveTreeWatchEmptyEventSetIsNop(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir, c); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	if len(w.ops()) != 0 {
		t.Fatalf("watcher calls = %s, want none for an empty event set", w)
	}
}

func TestRecursiveTreeWatchNilChannelPanics(t *testing.T) {
	tr, _ := newTestRecursiveTree(t)
	dir := mkdirs(t)
	defer func() {
		if recover() == nil {
			t.Fatal("Watch(nil channel) did not panic")
		}
	}()
	_ = tr.Watch(dir, nil, Create)
}

func TestRecursiveTreeWatchNonexistentPath(t *testing.T) {
	tr, _ := newTestRecursiveTree(t)
	c := make(chan EventInfo, buffer)
	missing := filepath.Join(mkdirs(t), "does", "not", "exist")
	if err := tr.Watch(missing, c, Create); err == nil {
		t.Fatal("Watch() on a nonexistent path succeeded, want an error")
	}
}

// TestRecursiveTreeWatchPropagatesWatcherError checks the tree rolls the
// watchpoint back when the underlying watcher rejects the watch.
func TestRecursiveTreeWatchPropagatesWatcherError(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)
	w.fail("Watch", errWatcherBoom)

	if err := tr.Watch(dir, c, Create); !errors.Is(err, errWatcherBoom) {
		t.Fatalf("Watch() error = %v, want %v", err, errWatcherBoom)
	}

	// The failed watchpoint was rolled back, so a retry issues a fresh Watch.
	w.fail("Watch", nil)
	w.reset()
	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() retry error = %v", err)
	}
	if got := w.opsFor("Watch"); len(got) != 1 {
		t.Fatalf("watcher calls = %s, want the retry to issue Watch again", w)
	}
}

func TestRecursiveTreeRecursiveWatchError(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)
	w.fail("RecursiveWatch", errWatcherBoom)

	if err := tr.Watch(dir+"...", c, Create); !errors.Is(err, errWatcherBoom) {
		t.Fatalf("Watch() error = %v, want %v", err, errWatcherBoom)
	}
}

func TestRecursiveTreeStopUnwatches(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
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

// TestRecursiveTreeStopLastChannelUsesPlainUnwatch pins a subtlety of Stop:
// watchDel runs before watchIsRecursive, so removing the last channel clears
// the recursive marker first and the teardown goes out as a plain Unwatch even
// for a watchpoint created recursively. Matching upstream notify.
func TestRecursiveTreeStopLastChannelUsesPlainUnwatch(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)

	if err := tr.Watch(dir+"...", c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()

	tr.Stop(c)
	if got := w.opsFor("Unwatch"); len(got) != 1 || got[0] != dir {
		t.Fatalf("watcher calls = %s, want Unwatch(%s)", w, dir)
	}
	if w.countOp("RecursiveUnwatch") != 0 {
		t.Fatalf("watcher calls = %s, want no RecursiveUnwatch", w)
	}
}

// TestRecursiveTreeStopChildDropsParentWatch documents a defect, not desired
// behaviour.
//
// With a recursive watch on root and a second channel watching root/sub, the
// child is stored as an "inactive" watchpoint on root. Stopping only the child
// makes watchDel fold the inactive diff over a `none` active diff, which
// collapses the new event set to zero, so Stop issues RecursiveUnwatch(root)
// and tears down the OS-level watch. The parent channel stays registered in the
// tree, so it looks live but can never receive another event.
//
// The assertions below pin the current behaviour so the breakage is visible; if
// a fix lands, this test should start failing and be replaced with one
// asserting the parent's watch survives (RecursiveRewatch, narrowing the set).
func TestRecursiveTreeStopChildDropsParentWatch(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")

	parent := make(chan EventInfo, buffer)
	if err := tr.Watch(root+"...", parent, Create); err != nil {
		t.Fatalf("Watch(parent) error = %v", err)
	}
	child := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, child, Remove); err != nil {
		t.Fatalf("Watch(child) error = %v", err)
	}
	w.reset()

	tr.Stop(child)

	if got := w.opsFor("RecursiveUnwatch"); len(got) != 1 || got[0] != root {
		t.Fatalf("watcher calls = %s, want RecursiveUnwatch(%s)", w, root)
	}

	// ...yet the parent channel is still registered on the root node.
	nd, err := tr.root.Get(root)
	if err != nil {
		t.Fatalf("root.Get(%s) error = %v", root, err)
	}
	if nd.Watch[parent] == 0 {
		t.Fatal("parent channel was unregistered; the defect may be fixed — see the comment above")
	}
}

// TestRecursiveTreeStopShrinksSharedWatch covers Stop's rewatch branch: one of
// two channels goes away, so the watch is narrowed rather than removed.
func TestRecursiveTreeStopShrinksSharedWatch(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
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
		t.Fatalf("watcher calls = %s, want Rewatch to narrow the event set", w)
	}
	if last.new&Remove != 0 {
		t.Fatalf("Rewatch new event set = %v, want Remove dropped", last.new)
	}
}

func TestRecursiveTreeStopUnknownChannelIsNop(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.reset()

	tr.Stop(make(chan EventInfo, buffer))
	if len(w.ops()) != 0 {
		t.Fatalf("watcher calls = %s, want none for an unknown channel", w)
	}
}

func TestRecursiveTreeStopToleratesWatcherError(t *testing.T) {
	tr, w := newTestRecursiveTree(t)
	dir := mkdirs(t)
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(dir, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	w.fail("Unwatch", errWatcherBoom)

	// Stop swallows watcher errors; it must not panic.
	tr.Stop(c)
}

func TestRecursiveTreeCloseClosesWatcher(t *testing.T) {
	w := newFakeWatcher()
	tr := newRecursiveTree(w, make(chan EventInfo, buffer))
	if err := tr.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if w.countOp("Close") != 1 {
		t.Fatalf("watcher calls = %s, want Close", w)
	}
}

func TestRecursiveTreeCloseReturnsWatcherError(t *testing.T) {
	w := newFakeWatcher()
	w.fail("Close", errWatcherBoom)
	tr := newRecursiveTree(w, make(chan EventInfo, buffer))
	if err := tr.Close(); !errors.Is(err, errWatcherBoom) {
		t.Fatalf("Close() error = %v, want %v", err, errWatcherBoom)
	}
}

// TestRecursiveTreeDispatchToLeaf checks an event on a watched path reaches the
// subscriber.
func TestRecursiveTreeDispatchToLeaf(t *testing.T) {
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newRecursiveTree(w, events)
	t.Cleanup(func() { _ = tr.Close() })

	root := mkdirs(t, "sub")
	sub := filepath.Join(root, "sub")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	events <- &testEvent{path: sub, ev: Create, dir: true}
	got := waitEvent(t, c)
	if got.Path() != sub || got.Event() != Create {
		t.Fatalf("received %v on %q, want Create on %q", got.Event(), got.Path(), sub)
	}
}

// TestRecursiveTreeDispatchToRecursiveParent checks an event deep inside a
// recursive watchpoint is delivered to the parent's subscriber.
func TestRecursiveTreeDispatchToRecursiveParent(t *testing.T) {
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newRecursiveTree(w, events)
	t.Cleanup(func() { _ = tr.Close() })

	root := mkdirs(t, filepath.Join("a", "b"))
	deep := filepath.Join(root, "a", "b", "file.txt")
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

// TestRecursiveTreeDispatchFiltersEvents checks an event outside the
// subscriber's event set is not delivered.
func TestRecursiveTreeDispatchFiltersEvents(t *testing.T) {
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newRecursiveTree(w, events)
	t.Cleanup(func() { _ = tr.Close() })

	dir := mkdirs(t, "sub")
	sub := filepath.Join(dir, "sub")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	events <- &testEvent{path: sub, ev: Remove}
	expectNoEvent(t, c)
}

// TestRecursiveTreeDispatchUnknownPath exercises the "did not reach leaf" path
// in dispatch.
func TestRecursiveTreeDispatchUnknownPath(t *testing.T) {
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newRecursiveTree(w, events)
	t.Cleanup(func() { _ = tr.Close() })

	dir := mkdirs(t, "sub")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(filepath.Join(dir, "sub"), c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	events <- &testEvent{path: filepath.Join(dir, "elsewhere", "file"), ev: Create}
	expectNoEvent(t, c)
}

// TestRecursiveTreeStopThenNoDispatch verifies Stop detaches the channel.
func TestRecursiveTreeStopThenNoDispatch(t *testing.T) {
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newRecursiveTree(w, events)
	t.Cleanup(func() { _ = tr.Close() })

	dir := mkdirs(t, "sub")
	sub := filepath.Join(dir, "sub")
	c := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, c, Create); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	tr.Stop(c)

	events <- &testEvent{path: sub, ev: Create}
	expectNoEvent(t, c)
}

// TestRecursiveTreeMultipleSubscribers checks one event fans out to every
// channel registered on the path.
func TestRecursiveTreeMultipleSubscribers(t *testing.T) {
	w := newFakeWatcher()
	events := make(chan EventInfo, buffer)
	tr := newRecursiveTree(w, events)
	t.Cleanup(func() { _ = tr.Close() })

	dir := mkdirs(t, "sub")
	sub := filepath.Join(dir, "sub")
	first := make(chan EventInfo, buffer)
	second := make(chan EventInfo, buffer)
	if err := tr.Watch(sub, first, Create); err != nil {
		t.Fatalf("Watch(first) error = %v", err)
	}
	if err := tr.Watch(sub, second, Create); err != nil {
		t.Fatalf("Watch(second) error = %v", err)
	}

	events <- &testEvent{path: sub, ev: Create}
	if got := waitEvent(t, first); got.Event() != Create {
		t.Fatalf("first got %v", got.Event())
	}
	if got := waitEvent(t, second); got.Event() != Create {
		t.Fatalf("second got %v", got.Event())
	}
}
