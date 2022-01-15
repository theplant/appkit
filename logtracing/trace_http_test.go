package logtracing

import (
	"net/http"
	"testing"
)

func TestTraceHTTPRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://test/hello", nil)
	if err != nil {
		t.Fatalf("err should be nil")
	}

	resp, err := TraceHTTPRequest(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK"}, nil
	}, "testTraceHTTPRequest", req)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("response status code should be 200")
	}

	if err != nil {
		t.Fatalf("err should be nil")
	}
}
