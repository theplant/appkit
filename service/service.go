package service

import (
	"context"
	"fmt"
	"io"
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

func ContextAndMiddleware() (context.Context, server.Middleware, io.Closer, error) {
	var funcClosers funcCloser
	ctx, ctxCloser := serviceContext()
	funcClosers = append(funcClosers, ctxCloser)

	logger := log.ForceContext(ctx)

	mw, mwCloser, err := middleware(ctx)
	if err != nil {
		funcClosers.Close()
		err = errors.Wrap(err, "error configuring service middleware")
		logger.WithError(err).Log()
		return nil, nil, nil, err
	}
	funcClosers = append(funcClosers, mwCloser)

	return ctx, mw, funcClosers, nil
}

func ListenAndServe(app func(context.Context, *http.ServeMux) error) {
	ctx, m, closer, err := ContextAndMiddleware()
	if err != nil {
		return
	}
	defer closer.Close()

	logger := log.ForceContext(ctx)

	mux := http.NewServeMux()

	if err := app(ctx, mux); err != nil {
		err = errors.Wrap(err, "error configuring service")
		logger.WithError(err).Log()
		return
	}

	cfg := server.Config{}
	cfg.Addr = os.Getenv("ADDR")
	if cfg.Addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "9800"
		}
		cfg.Addr = ":" + port
	}

	hc := server.GoListenAndServe(
		cfg,
		logger,
		m(mux),
	)
	// defers are LIFO, so the HTTP server will be closed *before* the
	// routes
	defer logErr(logger, hc.Close)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	sig := <-ch

	logger.Info().Log(
		"msg", fmt.Sprintf("received signal %v, exiting", sig),
		"signal", sig,
	)

}
