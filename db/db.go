// Package db provides a common way to setup a `gorm.DB` object.
package db

import (
	"log"

	"github.com/jinzhu/gorm"
)

// Config is for configuration that you can embed in your app config.
type Config struct {
	Dialect string `default:"postgres"`
	Params  string `required:"true"`
	Debug   bool
}

// Setup creates a DB object.
func Setup(config Config) (*gorm.DB, error) {
	var err error
	var db *gorm.DB

	log.Println("appkit/db: opening database connection")

	db, err = gorm.Open(config.Dialect, config.Params)

	if err != nil {
		return db, err
	}

	if config.Debug {
		log.Println("appkit/db: opening gorm debug mode")
		db.LogMode(true)
	}

	return db, nil
}
