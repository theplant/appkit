package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/log"
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
}
