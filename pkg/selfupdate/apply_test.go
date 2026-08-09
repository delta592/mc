package selfupdate

import (
	"bytes"
	"crypto"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumFor(t *testing.T) {
	sum, err := checksumFor(crypto.SHA256, []byte("payload"))
	if err != nil {
		t.Fatalf("checksumFor() error = %v", err)
	}
	if len(sum) != 32 {
		t.Fatalf("checksumFor() len = %d, want 32", len(sum))
	}
}

func TestChecksumForUnavailableHash(t *testing.T) {
	_, err := checksumFor(crypto.Hash(999), []byte("payload"))
	if err == nil {
		t.Fatal("expected error for unavailable hash")
	}
}

func TestOptionsGetMode(t *testing.T) {
	o := Options{}
	if o.getMode() != 0o755 {
		t.Fatalf("getMode() default = %o, want 0755", o.getMode())
	}
	o.TargetMode = 0o700
	if o.getMode() != 0o700 {
		t.Fatalf("getMode() custom = %o, want 0700", o.getMode())
	}
}

func TestOptionsGetPath(t *testing.T) {
	o := Options{TargetPath: "/tmp/custom-binary"}
	path, err := o.getPath()
	if err != nil || path != "/tmp/custom-binary" {
		t.Fatalf("getPath() = (%q, %v)", path, err)
	}

	o.TargetPath = ""
	path, err = o.getPath()
	if err != nil || path == "" {
		t.Fatalf("getPath() from executable = (%q, %v)", path, err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	payload := []byte("hello world")
	sum, err := checksumFor(crypto.SHA256, payload)
	if err != nil {
		t.Fatal(err)
	}

	o := Options{Checksum: sum}
	if err := o.verifyChecksum(payload); err != nil {
		t.Fatalf("verifyChecksum() match error = %v", err)
	}

	o.Checksum = []byte("bad")
	if err := o.verifyChecksum(payload); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestCheckPermissions(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mc-test-binary")
	if err := os.WriteFile(target, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := (&Options{TargetPath: target}).CheckPermissions(); err != nil {
		t.Fatalf("CheckPermissions() error = %v", err)
	}
}

func TestRollbackError(t *testing.T) {
	if RollbackError(nil) != nil {
		t.Fatal("RollbackError(nil) should be nil")
	}
	if RollbackError(errors.New("plain")) != nil {
		t.Fatal("RollbackError(plain) should be nil")
	}
	orig := errors.New("apply failed")
	rollback := errors.New("rollback failed")
	if got := RollbackError(&rollbackErr{error: orig, rollbackErr: rollback}); got != rollback {
		t.Fatalf("RollbackError() = %v, want %v", got, rollback)
	}
}

func TestPrepareAndCheckBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mc-test-binary")
	if err := os.WriteFile(target, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	payload := []byte("new-binary")
	sum, err := checksumFor(crypto.SHA256, payload)
	if err != nil {
		t.Fatal(err)
	}

	o := Options{
		TargetPath: target,
		Checksum:   sum,
	}
	if err := PrepareAndCheckBinary(bytes.NewReader(payload), o); err != nil {
		t.Fatalf("PrepareAndCheckBinary() error = %v", err)
	}

	newPath := filepath.Join(dir, ".mc-test-binary.new")
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("ReadFile(new binary) error = %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("new binary = %q, want %q", data, payload)
	}
}

func TestApplyAndCommitBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mc-test-binary")
	oldPayload := []byte("old-binary")
	newPayload := []byte("new-binary")
	if err := os.WriteFile(target, oldPayload, 0o755); err != nil {
		t.Fatal(err)
	}

	sum, err := checksumFor(crypto.SHA256, newPayload)
	if err != nil {
		t.Fatal(err)
	}

	o := Options{
		TargetPath: target,
		Checksum:   sum,
	}
	if err := Apply(bytes.NewReader(newPayload), o); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, newPayload) {
		t.Fatalf("committed binary = %q, want %q", data, newPayload)
	}
}

func TestApplyPatch(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mc-test-binary")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	o := Options{
		TargetPath: target,
		Patcher: patcherFunc(func(old io.Reader, applied io.Writer, patch io.Reader) error {
			patchData, _ := io.ReadAll(patch)
			_, err := applied.Write(patchData)
			return err
		}),
	}
	if err := PrepareAndCheckBinary(bytes.NewReader([]byte("patched")), o); err != nil {
		t.Fatalf("PrepareAndCheckBinary() with patch error = %v", err)
	}
}

type patcherFunc func(old io.Reader, applied io.Writer, patch io.Reader) error

func (f patcherFunc) Patch(old io.Reader, applied io.Writer, patch io.Reader) error {
	return f(old, applied, patch)
}

func TestNewVerifier(t *testing.T) {
	if NewVerifier() == nil {
		t.Fatal("NewVerifier() returned nil")
	}
}

func TestVerifierVerifyInvalidSignature(t *testing.T) {
	v := NewVerifier()
	if err := v.Verify([]byte("payload")); err == nil {
		t.Fatal("expected verification failure with empty verifier")
	}
}
