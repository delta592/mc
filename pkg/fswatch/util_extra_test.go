package fswatch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanpathResolvesSymlink(t *testing.T) {
	root := mkdirs(t, "real")
	real := filepath.Join(root, "real")
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got, isrec, err := cleanpath(link)
	if err != nil {
		t.Fatalf("cleanpath() error = %v", err)
	}
	if isrec {
		t.Fatal("cleanpath() set isrec without a ... suffix")
	}
	if got != real {
		t.Fatalf("cleanpath(%q) = %q, want %q", link, got, real)
	}
}

func TestCleanpathRecursiveSuffix(t *testing.T) {
	dir := mkdirs(t)
	got, isrec, err := cleanpath(dir + "...")
	if err != nil {
		t.Fatalf("cleanpath() error = %v", err)
	}
	if !isrec {
		t.Fatal("cleanpath() did not set isrec for a ... suffix")
	}
	if got != dir {
		t.Fatalf("cleanpath() = %q, want %q", got, dir)
	}
}

func TestCleanpathMissingPath(t *testing.T) {
	missing := filepath.Join(mkdirs(t), "nope", "deeper")
	if _, _, err := cleanpath(missing); err == nil {
		t.Fatal("cleanpath() on a missing path succeeded, want an error")
	}
}

func TestCanonicalRelativeSymlink(t *testing.T) {
	root := mkdirs(t, "real")
	real := filepath.Join(root, "real")
	link := filepath.Join(root, "rel")
	// A relative symlink target exercises the non-absolute branch of canonical.
	if err := os.Symlink("real", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got, err := canonical(link)
	if err != nil {
		t.Fatalf("canonical() error = %v", err)
	}
	if got != real {
		t.Fatalf("canonical(%q) = %q, want %q", link, got, real)
	}
}

func TestCanonicalChainedSymlinks(t *testing.T) {
	root := mkdirs(t, "real")
	real := filepath.Join(root, "real")
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	if err := os.Symlink(real, first); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(first, second); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got, err := canonical(second)
	if err != nil {
		t.Fatalf("canonical() error = %v", err)
	}
	if got != real {
		t.Fatalf("canonical(%q) = %q, want %q", second, got, real)
	}
}

// TestCanonicalCircularSymlink checks the iteration limit that guards against
// symlink loops.
func TestCanonicalCircularSymlink(t *testing.T) {
	root := mkdirs(t)
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := canonical(a)
	if err == nil {
		t.Fatal("canonical() on a symlink loop succeeded, want an error")
	}
	// Either the iteration guard trips or the OS reports ELOOP; both are fine,
	// what matters is that it terminates with an error.
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("canonical() error = %T(%v), want *os.PathError", err, err)
	}
}

func TestCanonicalMissingPath(t *testing.T) {
	if _, err := canonical(filepath.Join(mkdirs(t), "missing")); err == nil {
		t.Fatal("canonical() on a missing path succeeded, want an error")
	}
}

func TestSplitAndBase(t *testing.T) {
	tests := []struct {
		in        string
		dir, base string
	}{
		{abs("a", "b", "c"), abs("a", "b"), "c"},
		{abs("a"), "", "a"},
		{"plain", "", "plain"},
		{"", "", ""},
	}
	for _, tc := range tests {
		dir, b := split(tc.in)
		if dir != tc.dir || b != tc.base {
			t.Errorf("split(%q) = (%q, %q), want (%q, %q)", tc.in, dir, b, tc.dir, tc.base)
		}
		if got := base(tc.in); got != tc.base {
			t.Errorf("base(%q) = %q, want %q", tc.in, got, tc.base)
		}
	}
}

func TestIndexrelBoundaries(t *testing.T) {
	root := abs("root")
	tests := []struct {
		name string
		root string
		path string
		want int
	}{
		{"direct child", root, abs("root", "a"), len(root) + 1},
		{"same path", root, root, -1},
		{"shorter", root, abs("ro"), -1},
		{"sibling with shared prefix", root, abs("rootx"), -1},
		{"unrelated", root, abs("other"), -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := indexrel(tc.root, tc.path); got != tc.want {
				t.Fatalf("indexrel(%q, %q) = %d, want %d", tc.root, tc.path, got, tc.want)
			}
		})
	}
}

func TestIndexSepBoundaries(t *testing.T) {
	sep := string(os.PathSeparator)
	tests := []struct {
		in            string
		first, lastIx int
	}{
		{"abc", -1, -1},
		{"a" + sep + "b", 1, 1},
		{"a" + sep + "b" + sep + "c", 1, 3},
		{sep, 0, 0},
		{"", -1, -1},
	}
	for _, tc := range tests {
		if got := indexSep(tc.in); got != tc.first {
			t.Errorf("indexSep(%q) = %d, want %d", tc.in, got, tc.first)
		}
		if got := lastIndexSep(tc.in); got != tc.lastIx {
			t.Errorf("lastIndexSep(%q) = %d, want %d", tc.in, got, tc.lastIx)
		}
	}
}

func TestNonilPicksFirstError(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	if got := nonil(nil, first, second); !errors.Is(got, first) {
		t.Fatalf("nonil() = %v, want %v", got, first)
	}
	if got := nonil(nil, nil, nil); got != nil {
		t.Fatalf("nonil() = %v, want nil", got)
	}
	if got := nonil(); got != nil {
		t.Fatalf("nonil() = %v, want nil", got)
	}
}

func TestMustAcceptsNil(t *testing.T) {
	// must(nil) must not panic.
	must(nil)
}

func TestJoinEventsCombinations(t *testing.T) {
	tests := []struct {
		in   []Event
		want Event
	}{
		{nil, All},
		{[]Event{}, All},
		{[]Event{Create}, Create},
		{[]Event{Create, Remove}, Create | Remove},
		{[]Event{Create, Create}, Create},
		{[]Event{Create, Remove, Write, Rename}, All},
	}
	for _, tc := range tests {
		if got := joinevents(tc.in); got != tc.want {
			t.Errorf("joinevents(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestDebugHelpersAreCallable(t *testing.T) {
	// With NOTIFY_DEBUG unset these are no-ops, but they must stay safe to call
	// from any code path.
	dbgprint("test", 1)
	dbgprintf("test %d", 1)
	if got := dbgcallstack(4); got != nil && len(got) > 4 {
		t.Fatalf("dbgcallstack(4) returned %d frames, want at most 4", len(got))
	}
}

func TestEventStringNamesKnownEvents(t *testing.T) {
	got := (Create | Remove | Write | Rename).String()
	for _, want := range []string{"Create", "Remove", "Write", "Rename"} {
		if !strings.Contains(got, want) {
			t.Errorf("Event.String() = %q, missing %q", got, want)
		}
	}
}
