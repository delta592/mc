package limiter

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockTransport struct {
	req *http.Request
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.req = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("response body")),
	}, nil
}

func TestNewZeroLimits(t *testing.T) {
	tr := &mockTransport{}
	got := New(0, 0, tr)
	if got != tr {
		t.Fatal("New(0, 0, transport) should return original transport")
	}
}

func TestRoundTripNilTransport(t *testing.T) {
	l := New(1024, 1024, nil).(*limiter)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := l.RoundTrip(req); err == nil {
		t.Fatal("expected error for nil transport")
	}
}

func TestRoundTripUploadDownload(t *testing.T) {
	tr := &mockTransport{}
	l := New(1<<20, 1<<20, tr).(*limiter)

	body := bytes.NewReader([]byte("request body"))
	req, err := http.NewRequest(http.MethodPut, "http://example.com", body)
	if err != nil {
		t.Fatal(err)
	}

	res, err := l.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer res.Body.Close()

	if _, err := io.ReadAll(res.Body); err != nil {
		t.Fatalf("ReadAll(response) error = %v", err)
	}
}

func TestRateReaderNilLimiter(t *testing.T) {
	l := &limiter{}
	r := l.limitReader(bytes.NewReader([]byte("abc")), nil)
	buf := make([]byte, 3)
	n, err := r.Read(buf)
	if err != nil || n != 3 {
		t.Fatalf("limitReader(nil) Read() = (%d, %v)", n, err)
	}
}
