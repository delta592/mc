package xattr

import (
	"errors"
	"os"
	"testing"
)

func TestErrorString(t *testing.T) {
	err := &Error{
		Op:   "xattr.Get",
		Path: "/tmp/file",
		Name: "user.test",
		Err:  errors.New("not found"),
	}
	got := err.Error()
	for _, want := range []string{"xattr.Get", "/tmp/file", "user.test", "not found"} {
		if got == "" || !contains(got, want) {
			t.Fatalf("Error() = %q, want substring %q", got, want)
		}
	}
}

func TestErrorUnwrap(t *testing.T) {
	base := errors.New("base")
	err := &Error{Err: base}
	if !errors.Is(err, base) {
		t.Fatal("Error.Unwrap() did not preserve base error")
	}
}

func TestSetGetRemoveList(t *testing.T) {
	if !XATTR_SUPPORTED {
		t.Skip("xattr not supported on this platform")
	}

	f, err := os.CreateTemp("", "mc-xattr-test-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	defer f.Close()

	const name = "user.mc.test"
	value := []byte("test-value")

	if err := Set(path, name, value); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := Get(path, name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != string(value) {
		t.Fatalf("Get() = %q, want %q", got, value)
	}

	list, err := List(path)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !containsString(list, name) {
		t.Fatalf("List() = %v, want %q", list, name)
	}

	if err := Remove(path, name); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := Get(path, name); err == nil {
		t.Fatal("expected error after Remove()")
	}
}

func TestLGetLSetLListLRemove(t *testing.T) {
	if !XATTR_SUPPORTED {
		t.Skip("xattr not supported on this platform")
	}

	f, err := os.CreateTemp("", "mc-xattr-ltest-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	defer f.Close()

	const name = "user.mc.ltest"
	if err := LSet(path, name, []byte("v")); err != nil {
		t.Fatalf("LSet() error = %v", err)
	}
	if _, err := LGet(path, name); err != nil {
		t.Fatalf("LGet() error = %v", err)
	}
	if _, err := LList(path); err != nil {
		t.Fatalf("LList() error = %v", err)
	}
	if err := LRemove(path, name); err != nil {
		t.Fatalf("LRemove() error = %v", err)
	}
}

func TestFGetFSetFListFRemove(t *testing.T) {
	if !XATTR_SUPPORTED {
		t.Skip("xattr not supported on this platform")
	}

	f, err := os.CreateTemp("", "mc-xattr-ftest-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)

	const name = "user.mc.ftest"
	if err := FSet(f, name, []byte("v")); err != nil {
		t.Fatalf("FSet() error = %v", err)
	}
	if _, err := FGet(f, name); err != nil {
		t.Fatalf("FGet() error = %v", err)
	}
	if _, err := FList(f); err != nil {
		t.Fatalf("FList() error = %v", err)
	}
	if err := FRemove(f, name); err != nil {
		t.Fatalf("FRemove() error = %v", err)
	}
	_ = path
}

func TestSetWithFlags(t *testing.T) {
	if !XATTR_SUPPORTED {
		t.Skip("xattr not supported on this platform")
	}

	f, err := os.CreateTemp("", "mc-xattr-flag-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	defer f.Close()

	const name = "user.mc.flag"
	if err := SetWithFlags(path, name, []byte("v"), 0); err != nil {
		t.Fatalf("SetWithFlags() error = %v", err)
	}
	if err := LSetWithFlags(path, name, []byte("v2"), 0); err != nil {
		t.Fatalf("LSetWithFlags() error = %v", err)
	}
	if err := FSetWithFlags(f, name, []byte("v3"), 0); err != nil {
		t.Fatalf("FSetWithFlags() error = %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexString(s, sub) >= 0)
}

func indexString(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func containsString(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
