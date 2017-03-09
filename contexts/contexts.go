package contexts

import (
	"context"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/theplant/appkit/log"
)

type key int

const (
	statusKey key = iota
	gormKey
)

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

////////////////////////////////////////////////////////////

func WithGorm(db *gorm.DB) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r.WithContext(GormContext(r.Context(), db)))
		})
	}
}

func GormContext(c context.Context, db *gorm.DB) context.Context {
	logger, ok := log.FromContext(c)

	newDB := db.New()
	if ok {
		newDB.SetLogger(log.GormLogger{logger.With("context", "gorm")})
	}

	return context.WithValue(c, gormKey, newDB)
}

func Gorm(c context.Context) (*gorm.DB, bool) {
	db, ok := c.Value(gormKey).(*gorm.DB)
	return db, ok
}

func MustGetGorm(c context.Context) *gorm.DB {
	db, ok := Gorm(c)

	if !ok {
		panic("can not find gorm in context")
	}

	return db
}
