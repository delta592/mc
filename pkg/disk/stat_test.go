package disk

import (
	"os"
	"strings"
	"testing"
)

func TestGetFileSystemAttrs(t *testing.T) {
	f, err := os.CreateTemp("", "mc-disk-test-*")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	defer f.Close()

	attrs, err := GetFileSystemAttrs(path)
	if err != nil {
		t.Fatalf("GetFileSystemAttrs() error = %v", err)
	}

	for _, key := range []string{"atime:", "/gid:", "/mode:", "/mtime:", "/uid:"} {
		if !strings.Contains(attrs, key) {
			t.Fatalf("GetFileSystemAttrs() = %q, missing %q", attrs, key)
		}
	}
}

func TestGetFileSystemAttrsMissingFile(t *testing.T) {
	if _, err := GetFileSystemAttrs("/no/such/file/for/mc-disk-test"); err == nil {
		t.Fatal("expected error for missing file")
	}
}
