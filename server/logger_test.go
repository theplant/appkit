package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/logtracing"
)

func TestLogRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/test?name=w", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	rw := httptest.NewRecorder()
	h := Compose(
		// Recovery should come before logReq to set the status code to 500
		Recovery,
		LogRequest,
		log.WithLogger(log.Default()),
		contexts.WithHTTPStatus,
	)(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		panic("test")
	}))

	h.ServeHTTP(rw, req)

	if rw.Result().StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rw.Result().StatusCode)
	}

	r16 := make([]byte, 16)
	r8 := make([]byte, 8)
	rand.Read(r16)
	rand.Read(r8)
	traceID := hex.EncodeToString(r16)
	spanID := hex.EncodeToString(r8)
	trace := fmt.Sprintf("%s-%s-%s-%s", "00", traceID, spanID, "00")
	req, err = http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	req.Header.Add(traceHeaderKey, trace)
	req.Header.Add("X-Forwarded-For", "192.168.1.1")
	req.Header.Add("X-Forwarded-For", "192.168.2.1")
	req.RemoteAddr = "192.168.3.1:12345"
	h = Compose(
		// Recovery should come before logReq to set the status code to 500
		Recovery,
		LogRequest,
	)(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, span := logtracing.StartSpan(r.Context(), "")
		if span.TraceID().String() != traceID {
			t.Errorf("traceID should be: %s, but got: %s", traceID, span.TraceID())
		}
	}))

	h.ServeHTTP(rw, req)
}

func TestLogRequestPanicIsVisibleToTailSampler(t *testing.T) {
	var sawPanic bool
	logtracing.ApplyConfig(logtracing.Config{
		TailSampler: func(s *logtracing.SpanData) bool {
			if s.Panic != nil {
				sawPanic = true
			}
			return true
		},
	})
	// ApplyConfig cannot unset TailSampler; leave a keep-all sampler behind.
	t.Cleanup(func() {
		logtracing.ApplyConfig(logtracing.Config{TailSampler: func(*logtracing.SpanData) bool { return true }})
	})

	req, err := http.NewRequest("GET", "http://example.com/panic", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	h := Compose(
		LogRequest,
		log.WithLogger(log.NewNopLogger()),
		contexts.WithHTTPStatus,
	)(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		panic("test")
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !sawPanic {
		t.Fatal("the panic must be recorded before span.End so a tail sampler can keep panicking requests")
	}
}
