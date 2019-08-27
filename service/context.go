package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/credentials"
	"github.com/theplant/appkit/credentials/aws"
	"github.com/theplant/appkit/credentials/influxdb"
	"github.com/theplant/appkit/credentials/vault"
	"github.com/theplant/appkit/errornotifier"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/monitoring"
)

func serviceContext() (context.Context, io.Closer) {
	ctx := context.Background()

	serviceName := os.Getenv("SERVICE_NAME")

	logger, ctx := installLogger(ctx, serviceName)

	cfg := credentialsConfig(serviceName)

	vault, ctx := installVault(ctx, logger, cfg.Authn)

	ctx = installAWSSession(ctx, logger, cfg.AWSPath, vault)

	_, mC, ctx := installMonitor(ctx, logger, serviceName, vault)

	_, nC, ctx := installErrorNotifier(ctx, logger)

	return ctx, funcCloser{noopCloserF(func() {
		logger.Debug().Log(
			"msg", fmt.Sprintf("shutting down service context for %v", serviceName),
		)

		if vault != nil {
			logger.Debug().Log(
				"msg", "revoking vault token",
			)

			vault.Auth().Token().RevokeSelf("")
		}
	}), mC, nC}
}

func installLogger(ctx context.Context, serviceName string) (log.Logger, context.Context) {
	logger := log.Default()

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

var noopCloser = noopCloserF(func() {})

////////////////////////////////////////////////////////////
// Metric Monitor

type influxDBConfig struct {
	URL string
}

func installMonitor(ctx context.Context, l log.Logger, serviceName string, vault *api.Client) (monitoring.Monitor, io.Closer, context.Context) {
	var monitor monitoring.Monitor
	var closer func()
	var c monitoring.InfluxMonitorConfig

	config := influxDBConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "INFLUXDB"}).Load(&config)
	if err != nil {
		err = errors.Wrap(err, "error fetching influxdb config")
		goto err
	}

	if config.URL == "" {
		err = errors.New("blank influxdb url")
		goto err
	}

	c, err = influxdb.InfluxMonitorConfig(l, serviceName, vault, config.URL)
	if err != nil {
		err = errors.Wrap(err, "error constructing influxdb url")
		goto err
	}

	monitor, closer, err = monitoring.NewInfluxdbMonitor(c, l)
	if err != nil {
		closer()
		goto err
	}

	return monitor, noopCloserF(closer), monitoring.Context(ctx, monitor)

err:
	l.Warn().Log(
		"msg", errors.Wrap(err, "falling back to log monitor: error creating influxdb monitor"),
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
			"msg", errors.Wrap(err, "falling back to log error notifier: error creating airbrake notifier"),
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

////////////////////////////////////////////////////////////
// Credentials: Vault, AWS

func credentialsConfig(serviceName string) credentials.Config {
	config := credentials.Config{}

	err := configor.New(&configor.Config{ENVPrefix: "VAULT"}).Load(&config)
	if err != nil {
		panic(err)
	}

	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		config = credentials.WithServiceName(config, serviceName)
	}

	return config
}

func installVault(ctx context.Context, l log.Logger, config vault.Config) (*api.Client, context.Context) {
	v, err := vault.NewVaultClient(l, config)
	if err != nil {
		panic(err)
	}

	return v, vault.Context(ctx, v)
}

func installAWSSession(ctx context.Context, l log.Logger, awsPath string, vault *api.Client) context.Context {

	awsSession, err := aws.NewSession(l, vault, awsPath)
	if err != nil {
		panic(err)
	}

	return aws.Context(ctx, awsSession)
}
