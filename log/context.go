package log

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
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

// Context installs a given Logger in the returned context
func Context(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext extracts a Logger from a (possibly nil) context.
func FromContext(c context.Context) (Logger, bool) {
	if c != nil {
		logger, ok := c.Value(loggerKey).(Logger)
		return logger, ok
	}
	return Logger{}, false
}

func Start(ctx context.Context) Logger {
	return StartWith(ctx, time.Now())
}

func StartWith(ctx context.Context, start time.Time) Logger {
	l := ForceContext(ctx)
	return l.With("duration", log.Valuer(func() interface{} {
		return fmt.Sprintf("%.3fms", float64(time.Since(start))/float64(time.Millisecond))
	}))
}

// ForceContext extracts a Logger from a (possibly nil) context, or
// returns a log.Default().
func ForceContext(c context.Context) Logger {
	logger, ok := FromContext(c)
	if !ok {
		logger = Default()
	}
	return logger
}
