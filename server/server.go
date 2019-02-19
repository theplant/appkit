// Package routes is a commen place to put all applicatioin routes.
// In order to easy setup routes for application and testing.
package server

import (
	"context"
	"fmt"
	"io"
	golog "log"
	"net/http"
	"time"

	"github.com/theplant/appkit/log"
)

type Config struct {
	Addr string `default:":9800"`
}

func newServer(config Config, logger log.Logger, handler http.Handler) *http.Server {
	server := http.Server{
		Addr:     config.Addr,
		ErrorLog: golog.New(log.LogWriter(logger.Error()), "", golog.Llongfile),
		Handler:  handler,
	}

	return &server
}

// ListenAndServe will start a HTTP server on config.Addr, using
// handler to handle requests. This function will never return.
func ListenAndServe(config Config, logger log.Logger, handler http.Handler) {
	GoListenAndServe(config, logger, handler)
	// Ignore io.Closer, listen forever
	var ch chan struct{}
	<-ch
}

type serverCloser func() error

func (s serverCloser) Close() error {
	return s()
}

// GoListenAndServe will start a HTTP server, on a separate goroutine,
// on config.Addr, using handler to handle requests.
//
// Returns an io.Closer that can be used to terminate the HTTP
// server. The closer will block with the same semantics as
// net/http.Server.Shutdown
// (https://godoc.org/net/http#Server.Shutdown)
func GoListenAndServe(config Config, logger log.Logger, handler http.Handler) io.Closer {
	logger = logger.With("during", "server.ListenAndServe")
	s := newServer(config, logger, handler)

	go func() {
		logger.Info().Log(
			"addr", config.Addr,
			"msg", fmt.Sprintf("HTTP server listening on %s", config.Addr),
			"wait_us", sinceStart(),
		)

		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Log(
				"msg", fmt.Sprintf("error in ListenAndServe: %v", err),
				"serve_us", sinceStart(),
				"err", err,
			)
		}
	}()

	return serverCloser(func() error {
		logger.Info().Log(
			"msg", fmt.Sprintf("shutting down HTTP server on %v", config.Addr),
			"addr", config.Addr,
			"serve_us", sinceStart(),
		)
		return s.Shutdown(context.Background())
	})

}

var start = time.Now()

func sinceStart() int64 {
	return int64(time.Now().Sub(start) / time.Microsecond)
}
