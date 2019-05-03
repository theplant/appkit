package service

import (
	"context"
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

func ListenAndServe(app func(context.Context, *http.ServeMux) error) {
	ctx, closer := serviceContext()
	defer closer.Close()

	logger := log.ForceContext(ctx)

	m, c, err := middleware(ctx)
	if err != nil {
		err = errors.Wrap(err, "error configuring service middleware")
		logger.WithError(err).Log()
		return
	}
	defer c.Close()

	mux := http.NewServeMux()

	if err := app(ctx, mux); err != nil {
		err = errors.Wrap(err, "error configuring service")
		logger.WithError(err).Log()
		return
	}

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
