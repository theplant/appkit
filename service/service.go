package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/server"
)

func logErr(l log.Logger, f func() error) {
	if err := f(); err != nil {
		l.WithError(err).Log()
	}
}

// envDuration reads a time.Duration (e.g. "30s") from the named environment
// variable. It is opt-in: if the variable is unset it returns 0, leaving the
// corresponding http.Server timeout disabled so services that don't configure
// it keep the previous "no timeout" behaviour. If the variable is set but
// cannot be parsed, or is negative (a negative deadline would make every
// request time out immediately), it logs a warning and returns 0, rather than
// silently applying a wrong (and possibly connection-killing) value.
func envDuration(logger log.Logger, name string) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		logger.Warn().Log(
			"msg", fmt.Sprintf("ignoring invalid duration in %s: %q", name, v),
			"env", name,
			"value", v,
			"err", err,
		)
		return 0
	}
	if d < 0 {
		logger.Warn().Log(
			"msg", fmt.Sprintf("ignoring negative duration in %s: %q", name, v),
			"env", name,
			"value", v,
		)
		return 0
	}
	return d
}

// serverTimeouts reads the opt-in http.Server timeouts from SERVER_* env vars.
// Kept separate from ListenAndServe so the env-var-to-field wiring is unit
// testable (a mismapped field would otherwise ship silently).
func serverTimeouts(logger log.Logger) (readHeader, read, write, idle time.Duration) {
	return envDuration(logger, "SERVER_READ_HEADER_TIMEOUT"),
		envDuration(logger, "SERVER_READ_TIMEOUT"),
		envDuration(logger, "SERVER_WRITE_TIMEOUT"),
		envDuration(logger, "SERVER_IDLE_TIMEOUT")
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

	// http.Server timeouts, opt-in per service via env. Unset => 0 => disabled,
	// preserving existing behaviour for services that don't configure them.
	cfg.ReadHeaderTimeout, cfg.ReadTimeout, cfg.WriteTimeout, cfg.IdleTimeout = serverTimeouts(logger)

	// Log only the enabled (non-zero) timeouts.
	kvs := []interface{}{"msg", "configured http.Server timeouts"}
	if cfg.ReadHeaderTimeout > 0 {
		kvs = append(kvs, "read_header_timeout", cfg.ReadHeaderTimeout)
	}
	if cfg.ReadTimeout > 0 {
		kvs = append(kvs, "read_timeout", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout > 0 {
		kvs = append(kvs, "write_timeout", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout > 0 {
		kvs = append(kvs, "idle_timeout", cfg.IdleTimeout)
	}
	if len(kvs) > 2 {
		logger.Info().Log(kvs...)
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
