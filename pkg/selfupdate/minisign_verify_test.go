package selfupdate

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"aead.dev/minisign"
)

// signedFixture is a real minisign keypair plus a signature over payload, so
// tests can exercise the full load-and-verify path rather than bailing out on
// an unparseable key.
type signedFixture struct {
	publicKeyText string
	signature     []byte
	payload       []byte
}

func newSignedFixture(t *testing.T) signedFixture {
	t.Helper()
	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	pubText, err := pub.MarshalText()
	if err != nil {
		t.Fatalf("PublicKey.MarshalText() error = %v", err)
	}
	payload := []byte("mc binary contents")
	return signedFixture{
		publicKeyText: string(pubText),
		signature:     minisign.Sign(priv, payload),
		payload:       payload,
	}
}

func TestLoadFromURLAndVerify(t *testing.T) {
	f := newSignedFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(f.signature)
	}))
	defer srv.Close()

	v := NewVerifier()
	if err := v.LoadFromURL(srv.URL, f.publicKeyText, http.DefaultTransport); err != nil {
		t.Fatalf("LoadFromURL() error = %v", err)
	}
	if err := v.Verify(f.payload); err != nil {
		t.Fatalf("Verify() error = %v, want the signature to validate", err)
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	f := newSignedFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(f.signature)
	}))
	defer srv.Close()

	v := NewVerifier()
	if err := v.LoadFromURL(srv.URL, f.publicKeyText, http.DefaultTransport); err != nil {
		t.Fatalf("LoadFromURL() error = %v", err)
	}
	if err := v.Verify([]byte("tampered contents")); err == nil {
		t.Fatal("Verify() accepted a tampered payload, want an error")
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	signed := newSignedFixture(t)
	other := newSignedFixture(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(signed.signature)
	}))
	defer srv.Close()

	v := NewVerifier()
	// Load a valid signature but pair it with a different public key.
	if err := v.LoadFromURL(srv.URL, other.publicKeyText, http.DefaultTransport); err != nil {
		t.Fatalf("LoadFromURL() error = %v", err)
	}
	if err := v.Verify(signed.payload); err == nil {
		t.Fatal("Verify() accepted a signature from another key, want an error")
	}
}

func TestLoadFromFileAndVerify(t *testing.T) {
	f := newSignedFixture(t)
	sigPath := filepath.Join(t.TempDir(), "mc.minisig")
	if err := os.WriteFile(sigPath, f.signature, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	v := NewVerifier()
	if err := v.LoadFromFile(sigPath, f.publicKeyText); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if err := v.Verify(f.payload); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestLoadFromFileMalformedSignature(t *testing.T) {
	f := newSignedFixture(t)
	sigPath := filepath.Join(t.TempDir(), "bad.minisig")
	if err := os.WriteFile(sigPath, []byte("untrusted comment: x\nnot-base64\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	v := NewVerifier()
	if err := v.LoadFromFile(sigPath, f.publicKeyText); err == nil {
		t.Fatal("LoadFromFile() accepted a malformed signature, want an error")
	}
}

func TestLoadFromURLMalformedSignature(t *testing.T) {
	f := newSignedFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("definitely not a signature"))
	}))
	defer srv.Close()

	v := NewVerifier()
	if err := v.LoadFromURL(srv.URL, f.publicKeyText, http.DefaultTransport); err == nil {
		t.Fatal("LoadFromURL() accepted a malformed signature, want an error")
	}
}

func TestLoadFromURLBadStatusWithValidKey(t *testing.T) {
	f := newSignedFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	v := NewVerifier()
	err := v.LoadFromURL(srv.URL, f.publicKeyText, http.DefaultTransport)
	if err == nil {
		t.Fatal("LoadFromURL() succeeded on a 500, want an error")
	}
}

func TestLoadFromURLUnreachable(t *testing.T) {
	f := newSignedFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening any more

	v := NewVerifier()
	if err := v.LoadFromURL(url, f.publicKeyText, http.DefaultTransport); err == nil {
		t.Fatal("LoadFromURL() succeeded against a closed server, want an error")
	}
}

func TestLoadFromURLBadRequestURL(t *testing.T) {
	f := newSignedFixture(t)
	v := NewVerifier()
	// A control character in the URL makes http.NewRequest itself fail.
	if err := v.LoadFromURL("http://\x7f/bad", f.publicKeyText, http.DefaultTransport); err == nil {
		t.Fatal("LoadFromURL() accepted a malformed URL, want an error")
	}
}

// TestLoadFromURLLeavesVerifierUntouchedOnError checks a failed load does not
// clobber a previously loaded, good key/signature pair.
func TestLoadFromURLLeavesVerifierUntouchedOnError(t *testing.T) {
	f := newSignedFixture(t)
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(f.signature)
	}))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bad.Close()

	v := NewVerifier()
	if err := v.LoadFromURL(good.URL, f.publicKeyText, http.DefaultTransport); err != nil {
		t.Fatalf("LoadFromURL() error = %v", err)
	}
	if err := v.LoadFromURL(bad.URL, f.publicKeyText, http.DefaultTransport); err == nil {
		t.Fatal("second LoadFromURL() succeeded, want an error")
	}
	if err := v.Verify(f.payload); err != nil {
		t.Fatalf("Verify() error = %v, want the original signature still loaded", err)
	}
}

// TestVerifyWithoutLoadedSignature checks Verify on a zero Verifier reports an
// error rather than panicking.
func TestVerifyWithoutLoadedSignature(t *testing.T) {
	v := NewVerifier()
	if err := v.Verify([]byte("anything")); err == nil {
		t.Fatal("Verify() on an unloaded verifier succeeded, want an error")
	}
}
