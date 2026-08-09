package fswatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMinMax(t *testing.T) {
	if min(3, 5) != 3 || min(5, 3) != 3 {
		t.Fatal("min() failed")
	}
	if max(3, 5) != 5 || max(5, 3) != 5 {
		t.Fatal("max() failed")
	}
}

func TestNonil(t *testing.T) {
	if nonil(nil, nil) != nil {
		t.Fatal("nonil(nil, nil) should be nil")
	}
	err := os.ErrNotExist
	if nonil(nil, err) != err {
		t.Fatal("nonil(nil, err) should return err")
	}
}

func TestJoinEvents(t *testing.T) {
	if joinevents(nil) != All {
		t.Fatal("joinevents(nil) should return All")
	}
	got := joinevents([]Event{Create, Write})
	if got != Create|Write {
		t.Fatalf("joinevents() = %v, want %v", got, Create|Write)
	}
}

func TestSplitBase(t *testing.T) {
	dir, name := split(filepath.Join("a", "b", "c"))
	if name != "c" {
		t.Fatalf("split() name = %q, want c", name)
	}
	if dir == "" {
		t.Fatal("split() dir should not be empty")
	}
	if got := base(filepath.Join("a", "b", "c")); got != "c" {
		t.Fatalf("base() = %q, want c", got)
	}
}

func TestIndexRel(t *testing.T) {
	root := filepath.Join(string(os.PathSeparator), "root")
	name := filepath.Join(root, "child", "file")
	if indexrel(root, name) <= 0 {
		t.Fatalf("indexrel() = %d, want > 0", indexrel(root, name))
	}
	if indexrel(root, root) != -1 {
		t.Fatal("indexrel(root, root) should be -1")
	}
}

func TestIndexSep(t *testing.T) {
	if indexSep("abc") != -1 {
		t.Fatal("indexSep(abc) should be -1")
	}
	p := filepath.Join("a", "b")
	if indexSep(p) != 1 {
		t.Fatalf("indexSep(%q) = %d, want 1", p, indexSep(p))
	}
	if lastIndexSep(p) != 1 {
		t.Fatalf("lastIndexSep(%q) = %d, want 1", p, lastIndexSep(p))
	}
}

func TestEventString(t *testing.T) {
	s := Create.String()
	if s == "" {
		t.Fatal("Create.String() should not be empty")
	}
}

func TestCleanpath(t *testing.T) {
	dir := t.TempDir()
	realpath, isrec, err := cleanpath(dir + "...")
	if err != nil {
		t.Fatalf("cleanpath() error = %v", err)
	}
	if !isrec {
		t.Fatal("cleanpath(...suffix) should set isrec")
	}
	if realpath == "" {
		t.Fatal("cleanpath() returned empty path")
	}
}

func TestCanonical(t *testing.T) {
	dir := t.TempDir()
	got, err := canonical(dir)
	if err != nil {
		t.Fatalf("canonical() error = %v", err)
	}
	if got == "" {
		t.Fatal("canonical() returned empty path")
	}
}

func TestMustPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("must(err) should panic")
		}
	}()
	must(os.ErrInvalid)
}
