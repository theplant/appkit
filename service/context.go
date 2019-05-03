package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/errornotifier"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/monitoring"
)

func serviceContext() (context.Context, io.Closer) {
	ctx := context.Background()

	logger, ctx := installLogger(ctx)

	_, mC, ctx := installMonitor(ctx, logger)

	_, nC, ctx := installErrorNotifier(ctx, logger)

	return ctx, FuncCloser{mC, nC}
}

func installLogger(ctx context.Context) (log.Logger, context.Context) {
	logger := log.Default()

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		logger.Warn().Log("msg", "creating service context, SERVICE_NAME not set")
	} else {
		logger = logger.With("svc", serviceName)
		logger.Info().Log(
			"msg", fmt.Sprintf("creating service context for %s", serviceName),
		)
	}

	return logger, log.Context(ctx, logger)
}

var noopCloser = NoopCloser(func() {})

////////////////////////////////////////////////////////////
// Metric Monitor

type InfluxDBConfig struct {
	URL string
}

func installMonitor(ctx context.Context, l log.Logger) (monitoring.Monitor, io.Closer, context.Context) {
	var monitor monitoring.Monitor
	var closer func()

	config := InfluxDBConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "INFLUXDB"}).Load(&config)
	if err != nil {
		goto err
	}

	if config.URL == "" {
		err = errors.New("blank influxdb url")
		goto err
	}

	monitor, closer, err = monitoring.NewInfluxdbMonitor(monitoring.InfluxMonitorConfig(config.URL), l)
	if err != nil {
		closer()
		goto err
	}

	return monitor, NoopCloser(closer), monitoring.Context(ctx, monitor)

err:
	l.Warn().Log(
		"msg", errors.Wrap(err, "error creating influxdb monitor"),
		"err", err,
	)
	return monitoring.NewLogMonitor(l), noopCloser, ctx
}

////////////////////////////////////////////////////////////
// Error Notifier

func installErrorNotifier(ctx context.Context, l log.Logger) (errornotifier.Notifier, io.Closer, context.Context) {
	airbrakeConfig := errornotifier.AirbrakeConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "AIRBRAKE"}).Load(&airbrakeConfig)
	if err != nil {
		panic(err)
	}

	n, closer, err := errornotifier.NewAirbrakeNotifier(airbrakeConfig)
	if err != nil {
		l.Warn().Log(
			"msg", errors.Wrap(err, "error creating airbrake notifier"),
			"err", err,
		)

		return errornotifier.NewLogNotifier(l), noopCloser, ctx
	}

	l.Info().Log(
		"msg", "creating airbrake notifier",
		"project_id", airbrakeConfig.ProjectID,
		"env", airbrakeConfig.Environment,
	)

	return n, closer, errornotifier.Context(ctx, n)
}
