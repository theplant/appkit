package contexts

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeWriter is a ResponseWriter that also supports the deadline and flush
// capabilities of a real *http.response, recording whether each was reached.
type fakeWriter struct {
	header          http.Header
	status          int
	readDeadlineSet bool
	flushed         bool
}

func (f *fakeWriter) Header() http.Header {
	if f.header == nil {
		f.header = http.Header{}
	}
	return f.header
}
func (f *fakeWriter) Write(b []byte) (int, error)     { return len(b), nil }
func (f *fakeWriter) WriteHeader(status int)          { f.status = status }
func (f *fakeWriter) Flush()                          { f.flushed = true }
func (f *fakeWriter) SetReadDeadline(time.Time) error { f.readDeadlineSet = true; return nil }

// Unwrap must return the same ResponseWriter that was wrapped, so the
// http.ResponseController can walk down to the underlying connection.
func TestStatusWriterUnwrap(t *testing.T) {
	inner := &fakeWriter{}
	sw := &statusWriter{ResponseWriter: inner}

	if got := sw.Unwrap(); got != inner {
		t.Fatalf("Unwrap: want wrapped writer %p, got %p", inner, got)
	}
}

// http.ResponseController.SetReadDeadline must reach the underlying writer
// through statusWriter's Unwrap. Before Unwrap existed this silently failed
// with ErrNotSupported, which is why per-request read deadlines didn't work.
func TestStatusWriterResponseControllerReachesDeadline(t *testing.T) {
	inner := &fakeWriter{}
	sw := &statusWriter{ResponseWriter: inner}

	rc := http.NewResponseController(sw)
	if err := rc.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline through statusWriter: %v", err)
	}
	if !inner.readDeadlineSet {
		t.Fatal("SetReadDeadline did not reach the underlying writer")
	}
}

// End-to-end through the real middleware: a handler wrapped by WithHTTPStatus
// can still set a per-request read deadline via ResponseController. This is the
// production entry point (WithHTTPStatus is in DefaultMiddleware).
func TestWithHTTPStatusResponseControllerReachesDeadline(t *testing.T) {
	inner := &fakeWriter{}

	var handlerErr error
	h := WithHTTPStatus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerErr = http.NewResponseController(w).SetReadDeadline(time.Now().Add(time.Second))
	}))

	h.ServeHTTP(inner, httptest.NewRequest("GET", "/", nil))

	if handlerErr != nil {
		t.Fatalf("SetReadDeadline through WithHTTPStatus: %v", handlerErr)
	}
	if !inner.readDeadlineSet {
		t.Fatal("SetReadDeadline did not reach the underlying writer")
	}
}

// Regression: adding Unwrap must not break the SSE Flush support. Code that
// type-asserts http.Flusher on the wrapped writer must still succeed, and
// Flush must reach the underlying writer.
func TestWithHTTPStatusExposesFlusher(t *testing.T) {
	inner := &fakeWriter{}

	var asserted bool
	h := WithHTTPStatus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		asserted = ok
		if ok {
			f.Flush()
		}
	}))

	h.ServeHTTP(inner, httptest.NewRequest("GET", "/", nil))

	if !asserted {
		t.Fatal("wrapped writer no longer satisfies http.Flusher")
	}
	if !inner.flushed {
		t.Fatal("Flush did not reach the underlying writer")
	}
}

// WithHTTPStatus records the status written by the handler, and HTTPStatus
// defaults to 200 when the handler never calls WriteHeader.
func TestWithHTTPStatusCapturesStatus(t *testing.T) {
	cases := []struct {
		desc  string
		write int // 0 => don't call WriteHeader
		want  int
	}{
		{desc: "explicit 404", write: http.StatusNotFound, want: http.StatusNotFound},
		{desc: "default 200 when unwritten", write: 0, want: http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var got int
			var ok bool
			h := WithHTTPStatus(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if c.write != 0 {
					w.WriteHeader(c.write)
				}
				got, ok = HTTPStatus(r.Context())
			}))

			h.ServeHTTP(&fakeWriter{}, httptest.NewRequest("GET", "/", nil))

			if !ok {
				t.Fatal("HTTPStatus reported no statusWriter in context")
			}
			if got != c.want {
				t.Errorf("status: want %d, got %d", c.want, got)
			}
		})
	}
}
