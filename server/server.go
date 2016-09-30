// Package routes is a commen place to put all applicatioin routes.
// In order to easy setup routes for application and testing.
package server

import (
	"fmt"
	golog "log"
	"net/http"

	"github.com/theplant/appkit/log"
)

type Config struct {
	Addr string `default:":9800"`
}

func newServer(config Config, logger log.Logger) *http.Server {
	server := http.Server{
		Addr:     config.Addr,
		ErrorLog: golog.New(log.LogWriter(logger.Error()), "", golog.Llongfile),
	}
	return &server
}

func ListenAndServe(config Config, logger log.Logger, handler http.Handler) {
	logger = logger.With("during", "server.ListenAndServe")
	s := newServer(config, logger)
	s.Handler = handler

	logger.Info().Log(
		"addr", config.Addr,
		"msg", fmt.Sprintf("HTTP server listening on %s", config.Addr),
	)
	if err := s.ListenAndServe(); err != nil {
		logger.Error().Log(
			"msg", fmt.Sprintf("Error in ListenAndServe: %v", err),
			"err", err,
		)
	}
}
