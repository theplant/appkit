package contexts

import (
	"context"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/theplant/appkit/log"
)

var n = 0

type TraceID interface{}

type traceKey int

var key traceKey = 0

func genTraceId() TraceID {
	n += 1
	return n
}

func TraceRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracedContext := context.WithValue(r.Context(), key, genTraceId())
		h.ServeHTTP(w, r.WithContext(tracedContext))
	})
}

func GetTraceID(c context.Context) (TraceID, bool) {
	id, ok := c.Value(key).(TraceID)
	return id, ok
}

var skey traceKey = 1

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func Status(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		sContext := context.WithValue(r.Context(), skey, sw)
		h.ServeHTTP(sw, r.WithContext(sContext))
	})
}

func ResponseStatus(c context.Context) (int, bool) {
	status := http.StatusOK // Default
	sw, ok := c.Value(skey).(*statusWriter)

	if ok {
		status = sw.status
	}

	return status, ok

}

var lkey traceKey = 2

func WithLogger(logger log.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			traceID, _ := GetTraceID(ctx)
			newCtx := context.WithValue(ctx, lkey, logger.With("req_id", traceID))
			h.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func Logger(c context.Context) (log.Logger, bool) {
	logger, ok := c.Value(lkey).(log.Logger)
	return logger, ok
}

var dbkey traceKey = 3

func WithGorm(db *gorm.DB) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			logger, _ := Logger(ctx)

			newDB := db.New()
			newDB.SetLogger(log.GormLogger{logger.With("context", "gorm")})

			newCtx := context.WithValue(ctx, dbkey, newDB)

			h.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func Gorm(c context.Context) (*gorm.DB, bool) {
	db, ok := c.Value(dbkey).(*gorm.DB)
	return db, ok
}
