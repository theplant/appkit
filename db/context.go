package db

import (
	"context"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/theplant/appkit/log"
)

type key int

const gormKey key = iota

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
