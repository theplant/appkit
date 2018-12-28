package monitoring

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
)

// InfluxMonitorConfig type for configuration of Monitor that sinks to
// InfluxDB
type InfluxMonitorConfig string

type influxMonitorCfg struct {
	Addr               string
	Username           string
	Password           string
	Database           string
	BatchWriteInterval time.Duration
}

const (
	defaultBatchWriteInterval = time.Minute
)

var (
	configRegexp = regexp.MustCompile(`^(?P<scheme>https?):\/\/(?:(?P<username>.*?)(?::(?P<password>.*?)|)@)?(?P<host>.+?)\/(?P<database>.+?)(?:\?(?P<query>.*?))?$`)
)

func parseConfig(config string) (*influxMonitorCfg, error) {
	match := configRegexp.FindStringSubmatch(config)
	if match == nil {
		return nil, errors.New("influxdb config format error")
	}

	var scheme string
	var username string
	var password string
	var host string
	var database string
	var query string
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
		case "query":
			query = match[i]
		}
	}

	var batchWriteInterval time.Duration
	if query != "" {
		values, err := url.ParseQuery(query)
		if err != nil {
			return nil, errors.Wrap(err, "influxdb config query format error")
		}

		batchWriteSecondInterval := values.Get("batch-write-second-interval")
		if batchWriteSecondInterval != "" {
			second, err := strconv.Atoi(batchWriteSecondInterval)
			if err != nil {
				return nil, errors.Wrap(err, "influxdb config parameter batch-write-second-interval format error")
			}

			batchWriteInterval = time.Duration(second) * time.Second
		}

	}
	if batchWriteInterval == 0 {
		batchWriteInterval = defaultBatchWriteInterval
	}

	return &influxMonitorCfg{
		Addr:               scheme + "://" + host,
		Username:           username,
		Password:           password,
		Database:           database,
		BatchWriteInterval: batchWriteInterval,
	}, nil
}

// NewInfluxdbMonitor creates new monitoring influxdb
// client. config URL syntax is `https://<username>:<password>@<influxDB host>/<database>?batch-write-second-interval=seconds`
// batch-write-second-interval is optional, default is 60.
//
// Will returns a error if monitorURL is invalid or not absolute.
//
// Will not return error if InfluxDB is unavailable, but the returned
// Monitor will log errors if it cannot push metrics into InfluxDB
func NewInfluxdbMonitor(config InfluxMonitorConfig, logger log.Logger) (Monitor, error) {
	monitorURL := string(config)

	cfg, err := parseConfig(monitorURL)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse influxdb url %v", monitorURL)
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

	if strings.TrimSpace(cfg.Database) == "" {
		return nil, errors.Errorf("influxdb monitoring url %v not database", monitorURL)
	}

	monitor := &influxdbMonitor{
		database: cfg.Database,
		client:   client,
		logger:   logger,

		batchWriteInterval: cfg.BatchWriteInterval,
	}

	logger = logger.With(
		"addr", cfg.Addr,
		"username", cfg.Username,
		"database", monitor.database,
		"batch-write-second-interval", int(cfg.BatchWriteInterval/time.Second),
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

	go monitor.batchWriteTicker()

	logger.Info().Log(
		"msg", fmt.Sprintf("influxdb instrumentation writing to %s/%s on user %v", cfg.Addr, monitor.database, cfg.Username),
	)

	return monitor, nil
}

// InfluxdbMonitor implements monitor.Monitor interface, it wraps
// the influxdb client configuration.
type influxdbMonitor struct {
	client   influxdb.Client
	database string
	logger   log.Logger

	batchWriteInterval time.Duration
	cachePoints        []*influxdb.Point
	cachePointsMutex   sync.Mutex
}

func (im *influxdbMonitor) batchWriteTicker() {
	t := time.NewTicker(im.batchWriteInterval)

	for {
		<-t.C

		im.batchWrite()
	}
}

func (im *influxdbMonitor) batchWrite() {
	if len(im.cachePoints) == 0 {
		return
	}

	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database: im.database,
	})
	if err != nil {
		_ = im.logger.Error().Log(
			"database", im.database,
			"err", err,
			"during", "influxdb.NewBatchPoints",
			"msg", fmt.Sprintf("NewBatchPoints failed: %v", err),
		)
		return
	}

	im.cachePointsMutex.Lock()
	defer im.cachePointsMutex.Unlock()

	bp.AddPoints(im.cachePoints)

	err = im.client.Write(bp)
	if err != nil {
		_ = im.logger.Error().Log(
			"database", im.database,
			"err", err,
			"during", "influxdb.client.Write",
			"msg", fmt.Sprintf("influxdb client write cache points failed: %v", err),
		)
		return
	}

	im.cachePoints = nil
}

// InsertRecord part of monitor.Monitor.
func (im *influxdbMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, at time.Time) {
	if fields == nil {
		fields = map[string]interface{}{}
	}

	fields["value"] = value

	pt, err := influxdb.NewPoint(measurement, tags, fields, at)

	if err != nil {
		_ = im.logger.Error().Log(
			"database", im.database,
			"measurement", measurement,
			"value", value,
			"tags", tags,
			"err", err,
			"during", "influxdb.NewPoint",
			"msg", fmt.Sprintf("Error initializing a point for %s: %v", measurement, err),
		)
		return
	}

	im.cachePointsMutex.Lock()
	defer im.cachePointsMutex.Unlock()

	im.cachePoints = append(im.cachePoints, pt)
}

func (im *influxdbMonitor) Count(measurement string, value float64, tags map[string]string, fields map[string]interface{}) {
	im.InsertRecord(measurement, value, tags, fields, time.Now())
}

// CountError logs a value in measurement, with the given error's
// message stored in an `error` tag.
func (im *influxdbMonitor) CountError(measurement string, value float64, err error) {
	data := map[string]string{"error": err.Error()}
	im.Count(measurement, value, data, nil)
}

// CountSimple logs a value in measurement (with no tags).
func (im *influxdbMonitor) CountSimple(measurement string, value float64) {
	im.Count(measurement, value, nil, nil)
}
