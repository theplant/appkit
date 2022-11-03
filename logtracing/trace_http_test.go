package logtracing

import (
	"errors"
	"net/http"
	"testing"
)

func TestTraceHTTPRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://test/hello", nil)
	if err != nil {
		t.Fatalf("err should be nil")
	}

	var s *span
	resp, err := TraceHTTPRequest(func(r *http.Request) (*http.Response, error) {
		s = SpanFromContext(r.Context())
		if s == nil {
			t.Fatalf("span should not be nil")
		}

		if s.name != "testTraceHTTPRequest.call(/hello)" {
			t.Fatalf("span context should be testTraceHTTPRequest, actual: %s", s.name)
		}

		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK"}, nil
	}, "testTraceHTTPRequest", req)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("response status code should be 200")
	}

	if err != nil {
		t.Fatalf("err should be nil")
	}

	panicErr := errors.New("I'm the danger")

	defer func() {
		recovered := recover()
		if recovered != panicErr {
			t.Fatalf("should receive panic")
		}

		if s.panic != panicErr {
			t.Fatalf("panic should be recorded in span")
		}
	}()

	func() {
		resp, err = TraceHTTPRequest(func(r *http.Request) (*http.Response, error) {
			s = SpanFromContext(r.Context())
			panic(panicErr)
		}, "panicedRequest", req)
	}()
}
