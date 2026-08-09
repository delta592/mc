package fswatch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

// call records a single invocation on fakeWatcher.
type call struct {
	op      string
	path    string
	newpath string
	old     Event
	new     Event
}

func (c call) String() string {
	switch c.op {
	case "Watch", "RecursiveWatch":
		return fmt.Sprintf("%s(%s, %v)", c.op, c.path, c.new)
	case "Unwatch", "RecursiveUnwatch", "Close":
		return fmt.Sprintf("%s(%s)", c.op, c.path)
	case "Rewatch":
		return fmt.Sprintf("%s(%s, %v, %v)", c.op, c.path, c.old, c.new)
	default:
		return fmt.Sprintf("%s(%s, %s, %v, %v)", c.op, c.path, c.newpath, c.old, c.new)
	}
}

// fakeWatcher implements both watcher and recursiveWatcher, recording every
// call so tests can assert on the exact sequence a tree produces. Individual
// operations can be made to fail via errs.
type fakeWatcher struct {
	mu    sync.Mutex
	calls []call
	errs  map[string]error // keyed by op name
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{errs: make(map[string]error)}
}

func (w *fakeWatcher) record(c call) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, c)
	return w.errs[c.op]
}

func (w *fakeWatcher) Watch(path string, e Event) error {
	return w.record(call{op: "Watch", path: path, new: e})
}

func (w *fakeWatcher) Unwatch(path string) error {
	return w.record(call{op: "Unwatch", path: path})
}

func (w *fakeWatcher) Rewatch(path string, old, updated Event) error {
	return w.record(call{op: "Rewatch", path: path, old: old, new: updated})
}

func (w *fakeWatcher) RecursiveWatch(path string, e Event) error {
	return w.record(call{op: "RecursiveWatch", path: path, new: e})
}

func (w *fakeWatcher) RecursiveUnwatch(path string) error {
	return w.record(call{op: "RecursiveUnwatch", path: path})
}

func (w *fakeWatcher) RecursiveRewatch(oldpath, newpath string, old, updated Event) error {
	return w.record(call{
		op: "RecursiveRewatch", path: oldpath, newpath: newpath, old: old, new: updated,
	})
}

func (w *fakeWatcher) Close() error {
	return w.record(call{op: "Close"})
}

// ops returns the recorded operation names in order.
func (w *fakeWatcher) ops() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, len(w.calls))
	for i, c := range w.calls {
		out[i] = c.op
	}
	return out
}

// opsFor returns the paths passed to a given operation, sorted.
func (w *fakeWatcher) opsFor(op string) []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []string
	for _, c := range w.calls {
		if c.op == op {
			out = append(out, c.path)
		}
	}
	sort.Strings(out)
	return out
}

func (w *fakeWatcher) countOp(op string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, c := range w.calls {
		if c.op == op {
			n++
		}
	}
	return n
}

func (w *fakeWatcher) last() (call, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.calls) == 0 {
		return call{}, false
	}
	return w.calls[len(w.calls)-1], true
}

func (w *fakeWatcher) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = nil
}

func (w *fakeWatcher) fail(op string, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.errs[op] = err
}

func (w *fakeWatcher) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, len(w.calls))
	for i, c := range w.calls {
		out[i] = c.String()
	}
	return fmt.Sprint(out)
}

var errWatcherBoom = errors.New("watcher boom")

// testEvent is a platform-independent EventInfo used to drive trees directly.
// The package's own event type differs per backend (kqueue, FSEvents,
// inotify...), so tests that only exercise tree logic use this instead.
type testEvent struct {
	path string
	ev   Event
	dir  bool
}

func (e *testEvent) Event() Event         { return e.ev }
func (e *testEvent) Path() string         { return e.path }
func (e *testEvent) Sys() interface{}     { return nil }
func (e *testEvent) isDir() (bool, error) { return e.dir, nil }

var (
	_ EventInfo = (*testEvent)(nil)
	_ isDirer   = (*testEvent)(nil)
)

// mkdirs creates dir plus each of the given relative subdirectories inside a
// per-test temporary directory, and returns the canonical root. Canonicalising
// matters on macOS, where the temp dir sits behind a /var -> /private/var
// symlink that cleanpath resolves.
func mkdirs(t *testing.T, subdirs ...string) string {
	t.Helper()
	root, err := canonical(t.TempDir())
	if err != nil {
		t.Fatalf("canonical() error = %v", err)
	}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", sub, err)
		}
	}
	return root
}
