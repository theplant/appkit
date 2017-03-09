package trace

import (
	"context"
	"net/http"

	"github.com/pborman/uuid"
)

type key int

const traceKey key = iota

////////////////////////////////////////////////////////////

// Opaque type for request ID.
type ID interface{}

func genTraceID() ID {
	return uuid.New()
}

func WithRequestTrace(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracedContext := context.WithValue(r.Context(), traceKey, genTraceID())
		h.ServeHTTP(w, r.WithContext(tracedContext))
	})
}

func RequestTrace(c context.Context) (ID, bool) {
	id, ok := c.Value(traceKey).(ID)
	return id, ok
}
