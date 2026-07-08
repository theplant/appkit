package contexts

import (
	"context"
	"net/http"
)

type key int

const statusKey key = iota

////////////////////////////////////////////////////////////

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

// Flush implements http.Flusher interface to support SSE streaming.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap exposes the wrapped ResponseWriter so that http.ResponseController
// can reach the underlying net.Conn (e.g. for SetReadDeadline /
// SetWriteDeadline). Without this, per-request deadline control silently fails
// once a handler is wrapped by WithHTTPStatus.
func (s *statusWriter) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func WithHTTPStatus(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		sContext := context.WithValue(r.Context(), statusKey, sw)
		h.ServeHTTP(sw, r.WithContext(sContext))
	})
}

func HTTPStatus(c context.Context) (int, bool) {
	status := http.StatusOK // Default
	sw, ok := c.Value(statusKey).(*statusWriter)

	if ok && sw.status != 0 {
		status = sw.status
	}

	return status, ok

}
