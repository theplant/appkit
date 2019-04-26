package service

import (
	"io"
	"net/http"
	"strings"

	"github.com/goji/httpauth"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/jinzhu/configor"
	newrelic "github.com/newrelic/go-agent"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/theplant/appkit/errornotifier"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/monitoring"
	"github.com/theplant/appkit/server"
	"github.com/theplant/appkit/tracing"
)

func middleware(logger log.Logger) (server.Middleware, io.Closer, error) {

	tC, tracer, err := tracing.Tracer(logger)
	if err != nil {
		// FIXME returning nil io.Closer
		return nil, nil, errors.Wrap(err, "error configuring tracer")
	}

	errorNotifier, eC := MustGetErrorNotifier(logger)
	monitor, mC := MustGetMonitor(logger)
	appconf := MustGetAppConfig()

	return server.Compose(
			httpAuthMiddleware(logger),
			corsMiddleware(logger),
			NewRelicMiddleWare(logger, appconf.NewRelicAppName, appconf.NewRelicAPIKey),
			monitoring.WithMonitor(monitor),
			errornotifier.Recover(errorNotifier),
			tracer,
			server.DefaultMiddleware(logger),
		), FuncCloser{
			NoopCloser(mC),
			tC,
			eC,
		}, nil
}

func NewRelicMiddleWare(log log.Logger, NewRelicAppName string, NewRelicAPIKey string) func(http.Handler) http.Handler {
	config := newrelic.NewConfig(NewRelicAppName, NewRelicAPIKey)
	app, err := newrelic.NewApplication(config)
	if err != nil {
		log.Warn().Log(
			"msg", "error creating new relic agent",
			"err", err,
		)
		return func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r)
			})
		}
	}

	if app == nil {
		panic("Both of app and err are nil when new Application!")
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			txn := app.StartTransaction(r.URL.Path, w, r)
			defer txn.End()
			handler.ServeHTTP(w, r)
		})
	}
}

var (
	_errorNotifier       errornotifier.Notifier
	_errorNotifierCloser io.Closer
)

func MustGetErrorNotifier(l log.Logger) (errornotifier.Notifier, io.Closer) {
	if _errorNotifier != nil {
		return _errorNotifier, _errorNotifierCloser
	}

	airbrakeConfig := MustGetAirbrakeConfig()
	n, closer, err := errornotifier.NewAirbrakeNotifier(airbrakeConfig)
	if err != nil {
		l.Warn().Log(
			"msg", "error creating airbrake notifier",
			"err", err,
		)

		_errorNotifier = errornotifier.NewLogNotifier(l)
		_errorNotifierCloser = NoopCloser(func() {
			_ = l.Info().Log(
				"msg", "error notifier is closed",
			)
		})
	} else {
		l.Info().Log(
			"msg", "creating airbrake notifier sucessful",
			"project_id", airbrakeConfig.ProjectID,
			"env", airbrakeConfig.Environment,
		)
		_errorNotifier = n
		_errorNotifierCloser = closer
	}

	return _errorNotifier, _errorNotifierCloser
}

var _monitor monitoring.Monitor

var _closer = func() {}

func MustGetMonitor(l log.Logger) (monitoring.Monitor, func()) {
	if _monitor != nil {
		return _monitor, _closer
	}

	c := MustGetInfluxDBConfig()
	if c.URL == "" {
		_monitor = monitoring.NewLogMonitor(l)
		l.Warn().Log(
			"msg", "error creating influxdb monitor",
			"err", "blank influxdb url",
		)
		return _monitor, _closer
	}
	if m, closer, err := monitoring.NewInfluxdbMonitor(monitoring.InfluxMonitorConfig(c.URL), l); err != nil {
		l.Warn().Log(
			"msg", "error creating influxdb monitor",
			"err", err,
		)
		_monitor = monitoring.NewLogMonitor(l)
	} else {
		_monitor = m
		_closer = closer
	}

	return _monitor, _closer
}

// AppConfig defines the app's required configuration
type AppConfig struct {
	NewRelicAPIKey  string
	NewRelicAppName string
}

var _appConfig *AppConfig

func MustGetAppConfig() *AppConfig {
	if _appConfig != nil {
		return _appConfig
	}

	_appConfig = &AppConfig{}
	err := configor.New(nil).Load(_appConfig)
	if err != nil {
		panic(err)
	}

	return _appConfig
}

var _airbrakeConfig *errornotifier.AirbrakeConfig

func MustGetAirbrakeConfig() errornotifier.AirbrakeConfig {
	if _airbrakeConfig != nil {
		return *_airbrakeConfig
	}

	_airbrakeConfig = &errornotifier.AirbrakeConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "AIRBRAKE"}).Load(_airbrakeConfig)
	if err != nil {
		panic(err)
	}

	return *_airbrakeConfig
}

type InfluxDBConfig struct {
	URL string
}

var _influxDBConfig *InfluxDBConfig

func MustGetInfluxDBConfig() InfluxDBConfig {
	if _influxDBConfig != nil {
		return *_influxDBConfig
	}

	config := InfluxDBConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "INFLUXDB"}).Load(&config)
	if err != nil {
		panic(err)
	}

	_influxDBConfig = &config
	return config
}

// NoopCloser is an adapter from `func()` to io.Closer, that calls
// given function and returns nil
type NoopCloser func()

// Close is part of io.Closer
func (f NoopCloser) Close() error {
	f()
	return nil
}

// FuncCloser aggregates io.Closers into a single io.Closer that
// collects errors from each io.Closer function in the array when
// closed.
type FuncCloser []io.Closer

// Close is part of io.Closer
func (f FuncCloser) Close() error {
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

	err := configor.New(&configor.Config{ENVPrefix: "API"}).Load(&config)
	if err != nil {
		panic(err)
	}

	if config.RawAllowedOrigins == "" {
		logger.Warn().Log(
			"msg", "not enabling CORS middleware, CORS configuration is blank",
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
	Username string
	Password string
}

func httpAuthMiddleware(logger log.Logger) server.Middleware {
	config := httpAuthConfig{}

	err := configor.New(&configor.Config{ENVPrefix: "BASICAUTH"}).Load(&config)
	if err != nil {
		panic(err)
	}

	if config.Username == "" {
		logger.Warn().Log(
			"msg", "not enabling HTTP basic auth middleware, username is blank",
		)
		return server.IdMiddleware
	}

	logger.Info().Log(
		"msg", "enabling HTTP basic auth middleware",
		"username", config.Username,
	)
	return httpauth.SimpleBasicAuth(config.Username, config.Password)
}
