package httptracer

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockTransport struct {
	called bool
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	m.called = true
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
	}, nil
}

type mockTracer struct {
	reqErr error
	resErr error
	reqHit bool
	resHit bool
}

func (m *mockTracer) Request(_ *http.Request) error {
	m.reqHit = true
	return m.reqErr
}

func (m *mockTracer) Response(_ *http.Response) error {
	m.resHit = true
	return m.resErr
}

func TestGetNewTraceTransport(t *testing.T) {
	tr := &mockTransport{}
	trace := &mockTracer{}
	got := GetNewTraceTransport(trace, tr)
	if got.Trace != trace || got.Transport != tr {
		t.Fatal("GetNewTraceTransport() did not set fields")
	}
}

func TestRoundTripNilTransport(t *testing.T) {
	rt := RoundTripTrace{Trace: &mockTracer{}}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("expected error for nil transport")
	}
}

func TestRoundTripSuccess(t *testing.T) {
	tr := &mockTransport{}
	trace := &mockTracer{}
	rt := GetNewTraceTransport(trace, tr)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	res, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	defer res.Body.Close()

	if !tr.called || !trace.reqHit || !trace.resHit {
		t.Fatal("expected transport and tracer hooks to be called")
	}
}

func TestRoundTripTransportError(t *testing.T) {
	rt := RoundTripTrace{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("transport failed")
		}),
		Trace: &mockTracer{},
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestRoundTripRequestHookError(t *testing.T) {
	tr := &mockTransport{}
	trace := &mockTracer{reqErr: errors.New("request hook failed")}
	rt := GetNewTraceTransport(trace, tr)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("expected request hook error")
	}
}

func TestRoundTripResponseHookError(t *testing.T) {
	tr := &mockTransport{}
	trace := &mockTracer{resErr: errors.New("response hook failed")}
	rt := GetNewTraceTransport(trace, tr)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("expected response hook error")
	}
}

func TestRoundTripNilTrace(t *testing.T) {
	tr := &mockTransport{}
	rt := RoundTripTrace{Transport: tr}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() with nil trace error = %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
