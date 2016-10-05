// Package db provides a common way to setup a `gorm.DB` object.
package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/theplant/appkit/log"
)

// Config is for configuration that you can embed in your app config.
type Config struct {
	Dialect string `default:"postgres"`
	Params  string `required:"true"`
	Debug   bool
}

// New creates a DB object.
func New(l log.Logger, config Config) (*gorm.DB, error) {
	var err error
	var db *gorm.DB

	l = l.With("context", "appkit/db.New")
	l.Debug().Log("msg", "opening database connection")

	db, err = gorm.Open(config.Dialect, config.Params)

	if err != nil {
		l.Error().Log(
			"during", "gorm.Open",
			"err", err,
			"msg", fmt.Sprintf("error configuring database: %v", err),
		)
		return db, err
	}

	db.SetLogger(log.GormLogger{l})

	if config.Debug {
		l.Debug().Log("msg", "opening gorm debug mode")
		db.LogMode(true)
	}

	l.Debug().Log("msg", "database good to go")
	return db, nil
}
