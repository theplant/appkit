package monitoring

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

// InfluxMonitorConfig type for configuration of Monitor that sinks to
// InfluxDB
type InfluxMonitorConfig string

var configRegexp = regexp.MustCompile(`^(?P<scheme>https?):\/\/(?:(?P<username>.*?)(?::(?P<password>.*?)|)@)?(?P<host>.+?)\/(?P<database>.+?)$`)

func parseConfig(config string) (addr, username, password, database string, err error) {
	match := configRegexp.FindStringSubmatch(config)
	if match == nil {
		return "", "", "", "", errors.New("config format error")
	}

	var scheme string
	var host string
	for i, name := range configRegexp.SubexpNames() {
		switch name {
		case "scheme":
			scheme = match[i]
		case "username":
			username = match[i]
		case "password":
			password = match[i]
		case "host":
			host = match[i]
		case "database":
			database = match[i]
		}
	}

	return scheme + "://" + host, username, password, database, nil
}

// NewInfluxdbMonitor creates new monitoring influxdb
// client. config URL syntax is `https://<username>:<password>@<influxDB host>/<database>`
//
// Will returns a error if monitorURL is invalid or not absolute.
//
// Will not return error if InfluxDB is unavailable, but the returned
// Monitor will log errors if it cannot push metrics into InfluxDB
func NewInfluxdbMonitor(config InfluxMonitorConfig, logger log.Logger) (Monitor, error) {
	monitorURL := string(config)

	addr, username, password, database, err := parseConfig(monitorURL)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse influxdb url %v", monitorURL)
	}

	httpConfig := influxdb.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	}

	client, err := influxdb.NewHTTPClient(httpConfig)

	if err != nil {
		return nil, errors.Wrapf(err, "couldn't initialize influxdb http client with http config %+v", httpConfig)
	}

	if strings.TrimSpace(database) == "" {
		return nil, errors.Errorf("influxdb monitoring url %v not database", monitorURL)
	}

	monitor := influxdbMonitor{
		database: database,
		client:   client,
		logger:   logger,
	}

	logger = logger.With(
		"addr", addr,
		"username", username,
		"database", monitor.database,
	)

	// check connectivity to InfluxDB every 5 minutes
	go func() {
		t := time.NewTimer(5 * time.Minute)

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
		"msg", fmt.Sprintf("influxdb instrumentation writing to %s/%s on user %v", addr, monitor.database, username),
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
