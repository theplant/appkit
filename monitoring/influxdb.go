package monitoring

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

// InfluxMonitorConfig type for configuration of Monitor that sinks to
// InfluxDB
type InfluxMonitorConfig string

type influxMonitorCfg struct {
	Scheme             string
	Host               string
	Addr               string
	Username           string
	Password           string
	Database           string
	BatchWriteInterval time.Duration
	MaxCacheEvents     int
}

const (
	defaultBatchWriteInterval = time.Minute
	defaultMaxCacheEvents     = 10000

	batchWriteSecondIntervalParamName = "batch-write-second-interval"
	maxCacheEventsParamName           = "max-cache-events"
)

func parseInfluxMonitorConfig(config InfluxMonitorConfig) (*influxMonitorCfg, error) {
	monitorURL := string(config)

	u, err := url.Parse(monitorURL)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse influxdb url %v", monitorURL)
	} else if !u.IsAbs() {
		return nil, errors.Errorf("influxdb monitoring url %v not absolute url", monitorURL)
	}

	username := ""
	password := ""

	if u.User != nil {
		username = u.User.Username()
		// Skips identify of "whether password is set" as password not a must
		password, _ = u.User.Password()
	}

	database := strings.TrimLeft(u.Path, "/")

	if strings.TrimSpace(database) == "" {
		return nil, errors.Errorf("influxdb monitoring url %v not database", monitorURL)
	}

	values := u.Query()

	var batchWriteInterval time.Duration
	{
		interval := values.Get(batchWriteSecondIntervalParamName)
		if interval != "" {
			second, err := strconv.Atoi(interval)
			if err != nil {
				return nil, errors.Wrapf(err, "influxdb config parameter %s format error", batchWriteSecondIntervalParamName)
			}

			batchWriteInterval = time.Duration(second) * time.Second
		}
	}
	if batchWriteInterval == 0 {
		batchWriteInterval = defaultBatchWriteInterval
	}

	var maxCacheEvents int
	{
		events := values.Get(maxCacheEventsParamName)
		if events != "" {
			number, err := strconv.Atoi(events)
			if err != nil {
				return nil, errors.Wrapf(err, "influxdb config parameter %s format error", maxCacheEventsParamName)
			}
			if number < 0 {
				return nil, errors.Errorf("influxdb config parameter %s format error", maxCacheEventsParamName)
			}

			maxCacheEvents = number
		}
	}
	if maxCacheEvents <= 0 {
		maxCacheEvents = defaultMaxCacheEvents
	}

	return &influxMonitorCfg{
		Scheme:             u.Scheme,
		Host:               u.Host,
		Addr:               fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		Username:           username,
		Password:           password,
		Database:           database,
		BatchWriteInterval: batchWriteInterval,
		MaxCacheEvents:     maxCacheEvents,
	}, nil
}

// NewInfluxdbMonitor creates new monitoring influxdb
// client. config URL syntax is
// `https://<username>:<password>@<influxDB host>/<database>?batch-write-second-interval=seconds&max-cache-events=number`
// batch-write-second-interval is optional, default is 60,
// max-cache-events is optional, default is 10000.
//
// Will returns a error if monitorURL is invalid or not absolute.
//
// Will not return error if InfluxDB is unavailable, but the returned
// Monitor will log errors if it cannot push metrics into InfluxDB
func NewInfluxdbMonitor(config InfluxMonitorConfig, logger log.Logger) (Monitor, error) {
	cfg, err := parseInfluxMonitorConfig(config)
	if err != nil {
		return nil, err
	}

	httpConfig := influxdb.HTTPConfig{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	client, err := influxdb.NewHTTPClient(httpConfig)

	if err != nil {
		return nil, errors.Wrapf(err, "couldn't initialize influxdb http client with http config %+v", httpConfig)
	}

	monitor := influxdbMonitor{
		database: cfg.Database,
		client:   client,
		logger:   logger,
	}

	logger = logger.With(
		"scheme", cfg.Scheme,
		"username", cfg.Username,
		"database", monitor.database,
		"host", cfg.Host,
	)

	// check connectivity to InfluxDB every 5 minutes
	go func() {
		t := time.NewTicker(5 * time.Minute)

		for {
			// Ignore duration, version
			_, _, err = client.Ping(5 * time.Second)
			if err != nil {
				logger.Warn().Log(
					"err", err,
					"during", "influxdb.Client.Ping",
					"msg", fmt.Sprintf("couldn't ping influxdb: %v", err),
				)
			}

			<-t.C
		}
	}()

	logger.Info().Log(
		"msg", fmt.Sprintf("influxdb instrumentation writing to %s://%s@%s/%s", cfg.Scheme, cfg.Username, cfg.Host, monitor.database),
	)

	return &monitor, nil
}

// InfluxdbMonitor implements monitor.Monitor interface, it wraps
// the influxdb client configuration.
type influxdbMonitor struct {
	client   influxdb.Client
	database string
	logger   log.Logger
}

// InsertRecord part of monitor.Monitor.
func (im influxdbMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, at time.Time) {
	if fields == nil {
		fields = map[string]interface{}{}
	}

	fields["value"] = value

	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database: im.database,
	})

	l := im.logger.With("database", im.database,
		"measurement", measurement,
		"value", value,
		"tags", tags)

	if err != nil {
		l.Error().Log(
			"err", err,
			"during", "influxdb.NewBatchPoints",
			"msg", fmt.Sprintf("Error initializing batch points for %s: %v", measurement, err),
		)
	}

	pt, err := influxdb.NewPoint(measurement, tags, fields, at)

	if err != nil {
		l.Error().Log(
			"err", err,
			"during", "influxdb.NewPoint",
			"msg", fmt.Sprintf("Error initializing a point for %s: %v", measurement, err),
		)
	}

	bp.AddPoint(pt)

	if err := im.client.Write(bp); err != nil {
		im.logger.Error().Log(
			"err", err,
			"during", "influxdb.Client.Write",
			"msg", fmt.Sprintf("Error inserting record into %s: %v", measurement, err),
		)
	}
}

func (im influxdbMonitor) Count(measurement string, value float64, tags map[string]string, fields map[string]interface{}) {
	im.InsertRecord(measurement, value, tags, fields, time.Now())
}

// CountError logs a value in measurement, with the given error's
// message stored in an `error` tag.
func (im influxdbMonitor) CountError(measurement string, value float64, err error) {
	data := map[string]string{"error": err.Error()}
	im.Count(measurement, value, data, nil)
}

// CountSimple logs a value in measurement (with no tags).
func (im influxdbMonitor) CountSimple(measurement string, value float64) {
	im.Count(measurement, value, nil, nil)
}
