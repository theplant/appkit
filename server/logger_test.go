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
	h = Compose(
		// Recovery should come before logReq to set the status code to 500
		Recovery,
		LogRequest,
	)(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, span := logtracing.StartSpan(r.Context(), "")
		if span.TraceID() != traceID {
			t.Errorf("traceID should be: %s, but got: %s", traceID, span.TraceID())
		}
	}))

	h.ServeHTTP(rw, req)
}
