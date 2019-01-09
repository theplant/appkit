package monitoring

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
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
	CacheEvents        int
	MaxCacheEvents     int
}

const (
	defaultBatchWriteInterval = time.Minute
	// https://docs.influxdata.com/influxdb/v1.7/concepts/glossary#batch
	// > InfluxData recommends batch sizes of 5,000-10,000 points, although different use cases may be better served by significantly smaller or larger batches.
	defaultCacheEvents    = 5000
	defaultMaxCacheEvents = 10000

	batchWriteIntervalParamName = "batch-write-interval"
	maxCacheEventsParamName     = "max-cache-events"
	cacheEventsParamName        = "cache-events"
)

func getCacheEvents(values url.Values, key string, defaultValue int) (int, error) {
	events := values.Get(key)
	if events != "" {
		number, err := strconv.Atoi(events)
		if err != nil {
			return 0, errors.Wrapf(err, "influxdb config parameter %s format error", key)
		}
		if number < 0 {
			return 0, errors.Errorf("influxdb config parameter %s format error", key)
		}

		return number, nil
	}

	return defaultValue, nil
}

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
		interval := values.Get(batchWriteIntervalParamName)
		if interval != "" {
			duration, err := time.ParseDuration(interval)
			if err != nil {
				return nil, errors.Wrapf(err, `influxdb config parameter %s format error, valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`, batchWriteIntervalParamName)
			}
			batchWriteInterval = duration
		}
	}
	if batchWriteInterval <= 0 {
		batchWriteInterval = defaultBatchWriteInterval
	}

	cacheEvents, err := getCacheEvents(values, cacheEventsParamName, defaultCacheEvents)
	if err != nil {
		return nil, err
	}

	maxCacheEvents, err := getCacheEvents(values, maxCacheEventsParamName, defaultMaxCacheEvents)
	if err != nil {
		return nil, err
	}

	if cacheEvents > maxCacheEvents {
		return nil, errors.Errorf("%v can not be greater than %v", cacheEventsParamName, maxCacheEventsParamName)
	}

	return &influxMonitorCfg{
		Scheme:             u.Scheme,
		Host:               u.Host,
		Addr:               fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		Username:           username,
		Password:           password,
		Database:           database,
		BatchWriteInterval: batchWriteInterval,
		CacheEvents:        cacheEvents,
		MaxCacheEvents:     maxCacheEvents,
	}, nil
}

// NewInfluxdbMonitor creates new monitoring influxdb
// client. config URL syntax is
// `https://<username>:<password>@<influxDB host>/<database>?batch-write-interval=timeDuration&cache-events=number&max-cache-events=number`
// batch-write-interval is optional, default is 60s,
// valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
//   every batch-write-interval exec batch write.
// cache-events is optional, default is 5000.
//   if event number reach cache-events then exec batch write.
// max-cache-events is optional, default is 10000, its must > cache-events,
//   if the batch write fails and event number reach max-cache-events then clean up the cache (mean the data is lost).
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

	monitor := &influxdbMonitor{
		database: cfg.Database,
		client:   client,
		logger:   logger,

		pointChan:          make(chan *influxdb.Point),
		batchWriteInterval: cfg.BatchWriteInterval,
		cacheEvent:         newCacheEvent(cfg.CacheEvents, cfg.MaxCacheEvents),
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
				_ = logger.Warn().Log(
					"err", err,
					"during", "influxdb.Client.Ping",
					"msg", fmt.Sprintf("couldn't ping influxdb: %v", err),
				)
			}

			<-t.C
		}
	}()

	go monitor.batchWriteTicker()

	_ = logger.Info().Log(
		"msg", fmt.Sprintf("influxdb instrumentation writing to %s://%s@%s/%s", cfg.Scheme, cfg.Username, cfg.Host, monitor.database),
	)

	return monitor, nil
}

// InfluxdbMonitor implements monitor.Monitor interface, it wraps
// the influxdb client configuration.
type influxdbMonitor struct {
	client   influxdb.Client
	database string
	logger   log.Logger

	pointChan          chan *influxdb.Point
	batchWriteInterval time.Duration
	cacheEvent         *cacheEvent
}

type cacheEvent struct {
	cacheEvents        int
	maxCacheEvents     int
	currentCacheEvents int
}

func newCacheEvent(cacheEvents int, maxCacheEvents int) *cacheEvent {
	return &cacheEvent{
		cacheEvents:        cacheEvents,
		maxCacheEvents:     maxCacheEvents,
		currentCacheEvents: cacheEvents,
	}
}

func (ce *cacheEvent) CurrentCacheEvents() int {
	return ce.currentCacheEvents
}

func (ce *cacheEvent) IncreaseCacheEvents() {
	currEvents := ce.currentCacheEvents + ce.cacheEvents
	if currEvents > ce.maxCacheEvents {
		ce.currentCacheEvents = ce.maxCacheEvents
	} else {
		ce.currentCacheEvents = currEvents
	}
}

func (ce *cacheEvent) Reset() {
	// This if is rarely true, so add it.
	if ce.currentCacheEvents != ce.cacheEvents {
		ce.currentCacheEvents = ce.cacheEvents
	}
}

func (im influxdbMonitor) batchWriteTicker() {
	var points []*influxdb.Point
	t := time.NewTicker(im.batchWriteInterval)

	for {
		select {
		case <-t.C:
			im.batchWriteAndCheckErr(&points)

		case pt := <-im.pointChan:
			points = append(points, pt)

			if len(points) >= im.cacheEvent.CurrentCacheEvents() {
				im.batchWriteAndCheckErr(&points)
			}
		}
	}
}

func (im influxdbMonitor) batchWriteAndCheckErr(points *[]*influxdb.Point) {
	err := im.batchWrite(points)
	if err != nil {
		im.cacheEvent.IncreaseCacheEvents()

		if len(*points) >= im.cacheEvent.CurrentCacheEvents() {
			*points = nil
			_ = im.logger.Error().Log(
				"msg", "influxdb write failed and event number reach max-cache-events, cache events was cleaned up",
			)
		}
	} else {
		im.cacheEvent.Reset()
	}
}

// Return an error if and only if client write failed.
//
// *points will be set to nil if write successful.
func (im influxdbMonitor) batchWrite(points *[]*influxdb.Point) error {
	if points == nil || len(*points) == 0 {
		return nil
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
		return nil
	}

	bp.AddPoints(*points)

	err = im.client.Write(bp)
	if err != nil {
		_ = im.logger.Error().Log(
			"database", im.database,
			"err", err,
			"during", "influxdb.client.Write",
			"msg", fmt.Sprintf("influxdb client write points failed: %v", err),
		)
		return errors.Wrap(err, "influxdb client write points failed")
	}

	*points = nil
	return nil
}

// InsertRecord part of monitor.Monitor.
func (im influxdbMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, at time.Time) {
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

	im.pointChan <- pt
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
