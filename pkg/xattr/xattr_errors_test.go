package xattr

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// missingPath returns a path that does not exist, so every xattr syscall
// against it fails and the wrapping *Error is returned.
func missingPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "no-such-file")
}

// closedFile returns an *os.File whose descriptor has already been closed, so
// the F* variants fail at the syscall layer.
func closedFile(t *testing.T) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "closed")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return f
}

// assertXattrError checks err is a non-nil *Error carrying the expected op.
func assertXattrError(t *testing.T, op string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s succeeded, want an error", op)
	}
	var xerr *Error
	if !errors.As(err, &xerr) {
		t.Fatalf("%s error = %T(%v), want *xattr.Error", op, err, err)
	}
	if xerr.Op != op {
		t.Fatalf("Error.Op = %q, want %q", xerr.Op, op)
	}
	if xerr.Error() == "" {
		t.Fatal("Error.Error() is empty")
	}
	if xerr.Unwrap() == nil {
		t.Fatal("Error.Unwrap() = nil, want the underlying syscall error")
	}
}

func TestPathVariantsErrorOnMissingFile(t *testing.T) {
	path := missingPath(t)
	data := []byte("value")

	tests := []struct {
		name string
		op   string
		run  func() error
	}{
		{"Get", "xattr.get", func() error { _, err := Get(path, "user.test"); return err }},
		{"LGet", "xattr.get", func() error { _, err := LGet(path, "user.test"); return err }},
		{"Set", "xattr.Set", func() error { return Set(path, "user.test", data) }},
		{"LSet", "xattr.LSet", func() error { return LSet(path, "user.test", data) }},
		{"SetWithFlags", "xattr.SetWithFlags", func() error { return SetWithFlags(path, "user.test", data, 0) }},
		{"LSetWithFlags", "xattr.LSetWithFlags", func() error { return LSetWithFlags(path, "user.test", data, 0) }},
		{"Remove", "xattr.Remove", func() error { return Remove(path, "user.test") }},
		{"LRemove", "xattr.LRemove", func() error { return LRemove(path, "user.test") }},
		{"List", "xattr.list", func() error { _, err := List(path); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertXattrError(t, tc.op, tc.run())
		})
	}
}

func TestLListErrorOnMissingFile(t *testing.T) {
	_, err := LList(missingPath(t))
	assertXattrError(t, "xattr.list", err)
}

func TestFileVariantsErrorOnClosedFile(t *testing.T) {
	data := []byte("value")

	tests := []struct {
		name string
		op   string
		run  func(*os.File) error
	}{
		{"FGet", "xattr.get", func(f *os.File) error { _, err := FGet(f, "user.test"); return err }},
		{"FSet", "xattr.FSet", func(f *os.File) error { return FSet(f, "user.test", data) }},
		{"FSetWithFlags", "xattr.FSetWithFlags", func(f *os.File) error { return FSetWithFlags(f, "user.test", data, 0) }},
		{"FRemove", "xattr.FRemove", func(f *os.File) error { return FRemove(f, "user.test") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertXattrError(t, tc.op, tc.run(closedFile(t)))
		})
	}
}

func TestFListErrorOnClosedFile(t *testing.T) {
	_, err := FList(closedFile(t))
	assertXattrError(t, "xattr.list", err)
}

// TestGetMissingAttribute checks reading an attribute that was never set.
func TestGetMissingAttribute(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := Get(path, "user.definitely-not-set"); err == nil {
		t.Fatal("Get() of an unset attribute succeeded, want an error")
	}
}

// TestListReflectsSetAndRemove checks List picks up an attribute once set and
// drops it again after removal. It does not assert the list is empty to begin
// with: macOS stamps com.apple.provenance on newly created files.
func TestListReflectsSetAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bare")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	const name = "user.mc.listed"
	before, err := List(path)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if containsAttr(before, name) {
		t.Fatalf("List() = %v, want it to not already contain %q", before, name)
	}

	if err := Set(path, name, []byte("v")); err != nil {
		t.Skipf("filesystem does not support user attributes: %v", err)
	}
	during, err := List(path)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !containsAttr(during, name) {
		t.Fatalf("List() = %v, want it to contain %q", during, name)
	}

	if err := Remove(path, name); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	after, err := List(path)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if containsAttr(after, name) {
		t.Fatalf("List() = %v, want %q removed", after, name)
	}
}

func containsAttr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// TestGetLargeValue forces the grow-the-buffer retry loop in get().
func TestGetLargeValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Comfortably larger than the initial buffer so at least one retry happens.
	want := make([]byte, 8192)
	for i := range want {
		want[i] = byte('a' + i%26)
	}
	if err := Set(path, "user.mc.big", want); err != nil {
		t.Skipf("filesystem does not support this attribute size: %v", err)
	}

	got, err := Get(path, "user.mc.big")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("Get() returned %d bytes, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Get() differs at byte %d", i)
		}
	}
}

func TestBytePtrFromSlice(t *testing.T) {
	ptr, size := bytePtrFromSlice(nil)
	if ptr != nil || size != 0 {
		t.Fatalf("bytePtrFromSlice(nil) = %v, %d, want nil, 0", ptr, size)
	}

	data := []byte{1, 2, 3}
	ptr, size = bytePtrFromSlice(data)
	if ptr == nil || size != 3 {
		t.Fatalf("bytePtrFromSlice(%v) = %v, %d, want non-nil, 3", data, ptr, size)
	}
	if *ptr != 1 {
		t.Fatalf("bytePtrFromSlice() pointed at %d, want the first element", *ptr)
	}
}

func TestStringsFromByteSlice(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []string
	}{
		{"empty", nil, nil},
		{"single", []byte("one\x00"), []string{"one"}},
		{"multiple", []byte("one\x00two\x00"), []string{"one", "two"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stringsFromByteSlice(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("stringsFromByteSlice(%q) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("stringsFromByteSlice(%q) = %v, want %v", tc.in, got, tc.want)
				}
			}
		})
	}
}
