package service

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/server"
)

func logErr(l log.Logger, f func() error) {
	if err := f(); err != nil {
		l.WithError(err).Log()
	}
}

func ListenAndServe(app func(*http.ServeMux) error) {
	logger := log.Default()

	mux := http.NewServeMux()

	if err := app(mux); err != nil {
		err = errors.Wrap(err, "error configuring service")
		logger.WithError(err).Log()
		return
	}

	m, c, err := middleware(logger)
	if err != nil {
		err = errors.Wrap(err, "error configuring service middleware")
		logger.WithError(err).Log()
		return
	}
	defer c.Close()

	hc := server.GoListenAndServe(
		server.Config{
			Addr: ":9800",
		},
		logger,
		m(mux),
	)
	// defers are LIFO, so the HTTP server will be closed *before* the
	// routes
	defer logErr(logger, hc.Close)

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	sig := <-ch

	logger.Info().Log(
		"msg", fmt.Sprintf("received signal %v, exiting", sig),
		"signal", sig,
	)

}
