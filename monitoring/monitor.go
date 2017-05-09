// Package monitoring is a monitor "provider" that provides
// a way for monitoring. It uses influxdb monitor by default.
package monitoring

import (
	"time"

	"github.com/theplant/appkit/log"
)

// Monitor defines an interface for inserting record.
type Monitor interface {
	InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, time time.Time)
	Count(measurement string, value float64, tags map[string]string, fields map[string]interface{})
	CountError(measurement string, value float64, err error)
	CountSimple(measurement string, value float64)
}

// NewLogMonitor creates Monitor that logs metrics to passed
// log.Logger
func NewLogMonitor(l log.Logger) Monitor {
	return logMonitor{l}
}

type logMonitor struct {
	logger log.Logger
}

func (l logMonitor) InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, time time.Time) {
	logger := withTags(l.logger, tags)
	logger = withFields(logger, fields)

	logger.Info().Log(
		"metric", measurement,
		"value", value,
		"time", time,
	)
}

func (l logMonitor) Count(measurement string, value float64, tags map[string]string, fields map[string]interface{}) {
	logger := withTags(l.logger, tags)
	logger = withFields(logger, fields)
	logger.Info().Log(
		"metric", measurement,
		"value", value,
	)
}
func (l logMonitor) CountError(measurement string, value float64, err error) {
	l.logger.Error().Log(
		"metric", measurement,
		"value", value,
		"err", err,
	)

}
func (l logMonitor) CountSimple(measurement string, value float64) {
	l.logger.Info().Log(
		"metric", measurement,
		"value", value,
	)
}

func withTags(logger log.Logger, tags map[string]string) log.Logger {
	t := []interface{}{}
	for k, v := range tags {
		t = append(t, k, v)
	}
	return logger.With(t...)
}

func withFields(logger log.Logger, tags map[string]interface{}) log.Logger {
	t := []interface{}{}
	for k, v := range tags {
		t = append(t, k, v)
	}
	return logger.With(t...)
}
