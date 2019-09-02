package monitoring

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/tracing"
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
	BufferSize         int
	MaxBufferSize      int
	ServiceName        string
}

const (
	defaultBatchWriteInterval = time.Minute
	// https://docs.influxdata.com/influxdb/v1.7/concepts/glossary#batch
	// > InfluxData recommends batch sizes of 5,000-10,000 points, although different use cases may be better served by significantly smaller or larger batches.
	defaultBufferSize    = 5000
	defaultMaxBufferSize = 10000

	batchWriteIntervalParamName = "batch-write-interval"
	bufferSizeParamName         = "buffer-size"
	maxBufferSizeParamName      = "max-buffer-size"

	serviceNameParamName = "service-name"
)

func getBufferSize(values url.Values, key string, defaultValue int) (int, error) {
	size := values.Get(key)
	if size != "" {
		number, err := strconv.Atoi(size)
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

	bufferSize, err := getBufferSize(values, bufferSizeParamName, defaultBufferSize)
	if err != nil {
		return nil, err
	}

	maxBufferSize, err := getBufferSize(values, maxBufferSizeParamName, defaultMaxBufferSize)
	if err != nil {
		return nil, err
	}

	if bufferSize > maxBufferSize {
		return nil, errors.Errorf("%v can not be greater than %v", bufferSizeParamName, maxBufferSizeParamName)
	}

	return &influxMonitorCfg{
		Scheme:             u.Scheme,
		Host:               u.Host,
		Addr:               fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		Username:           username,
		Password:           password,
		Database:           database,
		BatchWriteInterval: batchWriteInterval,
		BufferSize:         bufferSize,
		MaxBufferSize:      maxBufferSize,
		ServiceName:        values.Get(serviceNameParamName),
	}, nil
}

// NewInfluxdbMonitor creates new monitoring influxdb
// client. config URL syntax is
// `https://<username>:<password>@<influxDB host>/<database>?batch-write-interval=timeDuration&buffer-size=number&max-buffer-size=number&service-name=name`
// batch-write-interval is optional, default is 60s,
// valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
//   exec batch write when we haven't sent data since batch-write-interval ago
// buffer-size is optional, default is 5000.
//   if buffered size reach buffer size then exec batch write.
// max-buffer-size is optional, default is 10000, it must > buffer-size,
//   if the batch write fails and buffered size reach max-buffer-size then clean up the buffer (mean the data is lost).
// service-name is optional
//   if set then all points will add tag service=service-name.
//
// The second return value is a function that will cause the batching
// goroutine to write buffered points, then terminate. This function
// will block until one attempt to flush the buffer completes (either
// success or failure).
//
// The third return value will be non-nil if monitorURL is invalid or
// not absolute.
//
// This function will not return error if InfluxDB is unavailable, but
// the returned Monitor will log errors if it cannot push metrics into
// InfluxDB.
func NewInfluxdbMonitor(config InfluxMonitorConfig, logger log.Logger) (Monitor, func(), error) {
	cfg, err := parseInfluxMonitorConfig(config)
	if err != nil {
		return nil, func() {}, err
	}

	httpConfig := influxdb.HTTPConfig{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	client, err := influxdb.NewHTTPClient(httpConfig)

	if err != nil {
		return nil, func() {}, errors.Wrapf(err, "couldn't initialize influxdb http client with http config %+v", httpConfig)
	}

	return NewInfluxdbMonitorWithClient(config, logger, client)
}

func NewInfluxdbMonitorWithClient(config InfluxMonitorConfig, logger log.Logger, client influxdb.Client) (Monitor, func(), error) {
	cfg, err := parseInfluxMonitorConfig(config)
	if err != nil {
		return nil, func() {}, err
	}

	logger = logger.With("context", "appkit/monitoring.influxdb")

	monitor := &influxdbMonitor{
		database: cfg.Database,
		client:   client,
		logger:   logger,

		pointChan:          make(chan *influxdb.Point),
		batchWriteInterval: cfg.BatchWriteInterval,
		bufferSize:         cfg.BufferSize,
		maxBufferSize:      cfg.MaxBufferSize,

		done: &sync.WaitGroup{},

		serviceName: cfg.ServiceName,
	}

	running := make(chan struct{})

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
			select {
			case <-t.C:
				// continue
			case <-running:
				_ = logger.Info().Log(
					"during", "influxdb.Client.Ping",
					"msg", "influxdb monitor closed, stopping influxdb pings",
				)
				return
			}

		}
	}()

	go monitor.batchWriteDaemon(running)

	_ = logger.Info().Log(
		"msg", fmt.Sprintf("influxdb instrumentation writing to %s://%s@%s/%s", cfg.Scheme, cfg.Username, cfg.Host, monitor.database),
		"batch-write-interval", cfg.BatchWriteInterval.String(),
		"buffer-size", cfg.BufferSize,
		"max-buffer-size", cfg.MaxBufferSize,
		"service-name", cfg.ServiceName,
	)

	return monitor, func() {
		_ = logger.Debug().Log(
			"msg", "closing influxdb monitor",
		)
		close(running)
		monitor.done.Wait()
	}, nil
}

// InfluxdbMonitor implements monitor.Monitor interface, it wraps
// the influxdb client configuration.
type influxdbMonitor struct {
	client   influxdb.Client
	database string
	logger   log.Logger

	pointChan          chan *influxdb.Point
	batchWriteInterval time.Duration
	bufferSize         int
	maxBufferSize      int

	// We need a pointer here since:
	//
	// > A WaitGroup must not be copied after first use.
	//
	// https://godoc.org/sync#WaitGroup
	done *sync.WaitGroup

	serviceName string
}

func (im influxdbMonitor) batchWriteDaemon(running chan struct{}) {
	im.done.Add(1)
	defer func() {
		im.done.Done()

		if r := recover(); r != nil {
			_ = im.logger.Crit().Log(
				"during", "influxdb.influxdbMonitor.batchWriteDaemon",
				"msg", fmt.Sprintf("panic: %v", r),
				"recover", r,
			)
		}
	}()

	var points []*influxdb.Point
	nextWriteBufferSize := im.bufferSize
	after := time.After(im.batchWriteInterval)

	for {
		select {
		case <-after:
			im.batchWriteAndHandleErr(&points, &nextWriteBufferSize)

			after = time.After(im.batchWriteInterval)

		case pt := <-im.pointChan:
			points = append(points, pt)

			if len(points) >= nextWriteBufferSize {
				im.batchWriteAndHandleErr(&points, &nextWriteBufferSize)
			}

		case <-running:
			_ = im.logger.Debug().Log(
				"msg", "influxdb monitor buffer closed, flushing buffer",
				"point_count", len(points),
			)
			im.batchWriteAndHandleErr(&points, &nextWriteBufferSize)

			return
		}
	}

}

func increaseBufferSize(nextWriteBufferSize, bufferSize, maxBufferSize int) int {
	newSize := nextWriteBufferSize + bufferSize
	if newSize > maxBufferSize {
		return maxBufferSize
	} else {
		return newSize
	}
}

// *points will be set to nil if write successful.
func (im influxdbMonitor) batchWriteAndHandleErr(points *[]*influxdb.Point, nextWriteBufferSize *int) {
	if points == nil || len(*points) == 0 {
		return
	}

	var newPoints []*influxdb.Point
	pt, err := im.newRecord("influxdb-queue-length", len(*points), nil, nil, time.Now())
	if err != nil {
		newPoints = *points
	} else {
		newPoints = append(*points, pt)
	}

	err = im.batchWrite(newPoints)
	if err != nil {
		*nextWriteBufferSize = increaseBufferSize(*nextWriteBufferSize, im.bufferSize, im.maxBufferSize)

		if len(*points) >= im.maxBufferSize {
			*points = nil
			_ = im.logger.Error().Log(
				"msg", "influxdb write failed and buffered size reach max-buffer-size, buffer was cleaned up",
			)
		}
	} else {
		*points = nil
		*nextWriteBufferSize = im.bufferSize
	}
}

func (im influxdbMonitor) batchWrite(points []*influxdb.Point) error {
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
		return errors.Wrap(err, "influxdb.NewBatchPoints failed")
	}

	bp.AddPoints(points)

	return tracing.Span(context.Background(), "appkit/monitoring.influxdbMonitor", func(_ context.Context, span opentracing.Span) error {
		ext.SpanKind.Set(span, ext.SpanKindRPCClientEnum)
		ext.Component.Set(span, "influxdb-buffer")
		ext.PeerService.Set(span, "InfluxDB")
		ext.DBInstance.Set(span, im.database)
		span.LogKV("point-count", len(points))

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
		return err
	})
}

func (im influxdbMonitor) newRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, at time.Time) (*influxdb.Point, error) {
	if fields == nil {
		fields = map[string]interface{}{}
	}

	fields["value"] = value

	if im.serviceName != "" {
		if tags == nil {
			tags = map[string]string{}
		}

		tags["service"] = im.serviceName
	}

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
		return nil, errors.Wrap(err, "influxdb.NewPoint failed")
	}

	return pt, nil
}

// InsertRecord part of monitor.Monitor.
func (im influxdbMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, at time.Time) {
	pt, err := im.newRecord(measurement, value, tags, fields, at)
	if err != nil {
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
