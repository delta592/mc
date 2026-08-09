package selfupdate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromURLInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("signature"))
	}))
	defer srv.Close()

	v := NewVerifier()
	if err := v.LoadFromURL(srv.URL, "not-a-valid-key", http.DefaultTransport); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestLoadFromURLBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	v := NewVerifier()
	if err := v.LoadFromURL(srv.URL, "not-a-valid-key", http.DefaultTransport); err == nil {
		t.Fatal("expected error for bad status")
	}
}

func TestLoadFromURLInvalidSignature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-a-signature"))
	}))
	defer srv.Close()

	v := NewVerifier()
	err := v.LoadFromURL(srv.URL, "not-a-valid-key", http.DefaultTransport)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadFromFileInvalidKey(t *testing.T) {
	dir := t.TempDir()
	sigPath := filepath.Join(dir, "test.minisig")
	if err := os.WriteFile(sigPath, []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewVerifier()
	if err := v.LoadFromFile(sigPath, "not-a-key"); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestLoadFromFileMissing(t *testing.T) {
	v := NewVerifier()
	if err := v.LoadFromFile(filepath.Join(t.TempDir(), "missing.minisig"), "not-a-key"); err == nil {
		t.Fatal("expected error for missing signature file")
	}
}

func TestHideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hidden")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := hideFile(path); err != nil {
		t.Fatalf("hideFile() error = %v", err)
	}
}
