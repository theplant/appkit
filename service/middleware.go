package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/goji/httpauth"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/jinzhu/configor"
	newrelic "github.com/newrelic/go-agent"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/theplant/appkit/credentials/aws"
	"github.com/theplant/appkit/errornotifier"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/monitoring"
	"github.com/theplant/appkit/server"
	"github.com/theplant/appkit/tracing"
)

func middleware(ctx context.Context) (server.Middleware, io.Closer, error) {
	logger := log.ForceContext(ctx)

	tC, tracer, err := tracing.Tracer(logger)
	if err != nil {
		logger.Warn().Log(
			"msg", errors.Wrap(err, "error configuring tracer"),
			"err", err,
		)

		// tracing returns a null closer if there's an error
		tC = noopCloser
		tracer = server.IdMiddleware
	}

	return server.Compose(
		withAWSSession(aws.ForceContext(ctx)),
		httpAuthMiddleware(logger),
		corsMiddleware(logger),
		newRelicMiddleware(logger),
		monitoring.WithMonitor(monitoring.ForceContext(ctx)),
		errornotifier.Recover(errornotifier.ForceContext(ctx)),
		tracer,
		server.DefaultMiddleware(logger),
	), tC, nil
}

////////////////////////////////////////////////////////////
// NEW RELIC

type newRelicConfig struct {
	APIKey  string
	AppName string
}

func newRelicMiddleware(log log.Logger) func(http.Handler) http.Handler {
	cfg := newRelicConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "NEWRELIC"}).Load(&cfg)
	if err != nil {
		panic(err)
	}

	if cfg.AppName == "" {
		cfg.AppName = os.Getenv("SERVICE_NAME")
	}

	config := newrelic.NewConfig(cfg.AppName, cfg.APIKey)
	app, err := newrelic.NewApplication(config)
	if err != nil {
		log.Warn().Log(
			"msg", errors.Wrap(err, "not enabling new relic middleware: error creating new relic agent"),
			"err", err,
		)

		return server.IdMiddleware
	}

	if app == nil {
		panic("both of app and err are nil when calling newrelic.NewApplication")
	}

	log.Info().Log(
		"msg", fmt.Sprintf("enabling new relic middleware, reporting as %s", cfg.AppName),
		"app_name", cfg.AppName,
	)

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			txn := app.StartTransaction(r.URL.Path, w, r)
			defer txn.End()
			handler.ServeHTTP(w, r)
		})
	}
}

////////////////////////////////////////////////////////////

// NoopCloser is an adapter from `func()` to io.Closer, that calls
// given function and returns nil
type noopCloserF func()

// Close is part of io.Closer
func (f noopCloserF) Close() error {
	f()
	return nil
}

// FuncCloser aggregates io.Closers into a single io.Closer that
// collects errors from each io.Closer function in the array when
// closed.
type funcCloser []io.Closer

// Close is part of io.Closer
func (f funcCloser) Close() error {
	var err error
	for _, c := range f {
		if e := c.Close(); e != nil {
			err = multierror.Append(err, e)
		}
	}

	return err
}

////////////////////////////////////////////////////////////
// CORS

type corsConfig struct {
	RawAllowedOrigins string
	AllowedOrigins    []string
	AllowCredentials  bool
}

func corsMiddleware(logger log.Logger) server.Middleware {
	config := corsConfig{}

	err := configor.New(&configor.Config{ENVPrefix: "CORS"}).Load(&config)
	if err != nil {
		panic(err)
	}

	if config.RawAllowedOrigins == "" {
		logger.Warn().Log(
			"msg", "not enabling CORS middleware: CORS configuration is blank",
			"raw_allowed_origins", config.RawAllowedOrigins,
			"allowed_credentials", config.AllowCredentials,
		)
		return server.IdMiddleware
	}

	config.AllowedOrigins = strings.Split(config.RawAllowedOrigins, ",")
	for i, allowedOrigin := range config.AllowedOrigins {
		config.AllowedOrigins[i] = strings.TrimSpace(allowedOrigin)
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowCredentials: config.AllowCredentials,
	})

	logger.Info().Log(
		"msg", "enabling CORS middleware",
		"allowed_origins", strings.Join(config.AllowedOrigins, ","),
		"allow_credentials", config.AllowCredentials,
	)

	return c.Handler
}

////////////////////////////////////////////////////////////
// HTTP Basic Auth

type httpAuthConfig struct {
	Username                 string
	Password                 string
	UserAgentWhitelistRegexp string
	PathWhitelistRegexp      string
}

func httpAuthMiddleware(logger log.Logger) server.Middleware {
	config := httpAuthConfig{}

	err := configor.New(&configor.Config{ENVPrefix: "BASICAUTH"}).Load(&config)
	if err != nil {
		panic(err)
	}

	if config.Username == "" {
		logger.Info().Log(
			"msg", "not enabling HTTP basic auth middleware: username is blank",
		)
		return server.IdMiddleware
	}

	httpAuthMiddleware := httpauth.SimpleBasicAuth(config.Username, config.Password)

	if config.UserAgentWhitelistRegexp == "" && config.PathWhitelistRegexp == "" {
		logger.Info().Log(
			"msg", "enabling HTTP basic auth middleware",
			"username", config.Username,
		)
		return httpAuthMiddleware
	}

	var userAgentRegexp, pathRegexp *regexp.Regexp

	if config.UserAgentWhitelistRegexp != "" {
		userAgentRegexp, err = regexp.Compile(config.UserAgentWhitelistRegexp)
		if err != nil {
			panic(errors.Wrap(err, fmt.Sprintf("error compiling http basic auth user-agent whitelist regexp %q", config.UserAgentWhitelistRegexp)))
		}
	}

	if config.PathWhitelistRegexp != "" {
		pathRegexp, err = regexp.Compile(config.PathWhitelistRegexp)
		if err != nil {
			panic(errors.Wrap(err, fmt.Sprintf("error compiling http basic auth path whitelist regexp %q", config.PathWhitelistRegexp)))
		}
	}

	logger.Info().Log(
		"msg", "enabling HTTP basic auth middleware with whitelists",
		"username", config.Username,
		"user_agent_whitelist", config.UserAgentWhitelistRegexp,
		"path_whitelist", config.PathWhitelistRegexp,
	)

	return func(h http.Handler) http.Handler {
		authedHandler := httpAuthMiddleware(h)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (userAgentRegexp != nil && userAgentRegexp.MatchString(r.Header.Get("User-Agent"))) ||
				(pathRegexp != nil && pathRegexp.MatchString(r.URL.Path)) {
				h.ServeHTTP(w, r)
				return
			}

			authedHandler.ServeHTTP(w, r)
		})
	}
}

////////////////////////////////////////////////////////////
// AWS Session in request context

func withAWSSession(s *session.Session) server.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r.WithContext(aws.Context(r.Context(), s)))
		})
	}
}
