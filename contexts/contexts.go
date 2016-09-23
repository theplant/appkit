package contexts

import (
	"context"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/theplant/appkit/log"
)

type key int

const (
	traceKey key = iota
	statusKey
	loggerKey
	gormKey
)

////////////////////////////////////////////////////////////

var n = 0

type TraceID interface{}

func genTraceId() TraceID {
	n += 1
	return n
}

func WithRequestTrace(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracedContext := context.WithValue(r.Context(), traceKey, genTraceId())
		h.ServeHTTP(w, r.WithContext(tracedContext))
	})
}

func RequestTrace(c context.Context) (TraceID, bool) {
	id, ok := c.Value(traceKey).(TraceID)
	return id, ok
}

////////////////////////////////////////////////////////////

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
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

	if ok {
		status = sw.status
	}

	return status, ok

}

////////////////////////////////////////////////////////////

func WithLogger(logger log.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// FIXME don't *mandate* having a request trace ID
			traceID, _ := RequestTrace(ctx)
			newCtx := context.WithValue(ctx, loggerKey, logger.With("req_id", traceID))
			h.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func Logger(c context.Context) (log.Logger, bool) {
	logger, ok := c.Value(loggerKey).(log.Logger)
	return logger, ok
}

////////////////////////////////////////////////////////////

func WithGorm(db *gorm.DB) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger, _ := Logger(ctx)

			newDB := db.New()
			newDB.SetLogger(log.GormLogger{logger.With("context", "gorm")})

			newCtx := context.WithValue(ctx, gormKey, newDB)

			h.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func Gorm(c context.Context) (*gorm.DB, bool) {
	db, ok := c.Value(gormKey).(*gorm.DB)
	return db, ok
}
