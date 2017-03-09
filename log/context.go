package log

import (
	"context"
	"net/http"

	"github.com/theplant/appkit/contexts/trace"
)

type key int

const loggerKey key = iota

func WithLogger(logger Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			traceID, ok := trace.RequestTrace(ctx)
			l := logger // don't overwrite logger
			if ok {
				l = logger.With("req_id", traceID)
			}
			newCtx := context.WithValue(ctx, loggerKey, l)
			h.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func FromContext(c context.Context) (Logger, bool) {
	logger, ok := c.Value(loggerKey).(Logger)
	return logger, ok
}

func ForceContext(c context.Context) Logger {
	logger, ok := FromContext(c)
	if !ok {
		logger = Default()
	}
	return logger
}
