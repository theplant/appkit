// Package routes is a commen place to put all applicatioin routes.
// In order to easy setup routes for application and testing.
package server

import (
	golog "log"
	"net/http"

	"github.com/theplant/appkit/log"
)

func NewServer(logger log.Logger, addr string, handler http.Handler) *http.Server {
	server := http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: golog.New(log.LogWriter(logger.Error()), "", golog.Llongfile),
	}
	return &server
}
