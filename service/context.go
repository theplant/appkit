package service

import (
	"context"
	"fmt"
	"io"
	neturl "net/url"
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

	ctx = installAWSConfig(ctx, logger, cfg.AWSPath, vault)

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

func installMonitor(ctx context.Context, l log.Logger, serviceName string, vault *vault.Client) (monitoring.Monitor, io.Closer, context.Context) {
	var (
		monitor monitoring.Monitor
		closer  func()
		url     *neturl.URL
		q       neturl.Values
	)

	closer = noopCloser

	config := influxDBConfig{}
	err := configor.New(&configor.Config{ENVPrefix: "INFLUXDB"}).Load(&config)
	if err != nil {
		err = errors.Wrap(err, "error fetching influxdb config")
		goto err
	}

	url, err = neturl.Parse(config.URL)
	if err != nil {
		err = errors.Wrap(err, "error parsing influxdb config url")
		goto err
	}

	// attach service name to url
	q = url.Query()
	if q.Get("service-name") == "" && serviceName != "" {
		q.Set("service-name", serviceName)
		url.RawQuery = q.Encode()
	}

	if url.Scheme == "vault" {
		if vault != nil {
			monitor, closer, err = influxdb.NewInfluxDBMonitor(l, vault, url)
			err = errors.Wrap(err, "error creating influxdb+vault monitor")
		} else {
			err = errors.Wrap(err, "nil vault client when configured for influxdb+vault monitor")
		}
	} else {
		monitor, closer, err = monitoring.NewInfluxdbMonitor(monitoring.InfluxMonitorConfig(config.URL), l)
		err = errors.Wrap(err, "error creating influxdb monitor")
	}
	if err != nil {
		goto err
	}

	return monitor, noopCloserF(closer), monitoring.Context(ctx, monitor)

err:
	l.Warn().Log(
		"msg", errors.Wrap(err, "falling back to log monitor"),
		"err", err,
	)
	closer()
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

func installVault(ctx context.Context, l log.Logger, config vault.Config) (*vault.Client, context.Context) {
	v, err := vault.NewVaultClient(l, config)
	if err != nil {
		panic(err)
	}

	return v, vault.Context(ctx, v)
}

func installAWSConfig(ctx context.Context, l log.Logger, awsPath string, vault *vault.Client) context.Context {

	var client *api.Client
	if vault != nil {
		client = vault.Client
	}

	awsCfg, err := aws.NewConfig(ctx, l, client, awsPath)
	if err != nil {
		panic(err)
	}

	return aws.Context(ctx, awsCfg)
}
