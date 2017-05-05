package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/theplant/appkit/kerrs"
)

type Logger struct {
	log.Logger
}

func (l Logger) With(keysvals ...interface{}) Logger {
	l.Logger = log.With(l.Logger, keysvals...)
	return l
}

/*
WithError can log kerrs type of err to structured log
*/
func (l Logger) WithError(err error) log.Logger {
	keysvals, msg, stacktrace := kerrs.Extract(err)
	keysvals = append(keysvals, "msg", msg)
	if len(stacktrace) > 0 {
		keysvals = append(keysvals, "stacktrace", stacktrace)
	}
	l.Logger = level.Error(log.With(l.Logger, keysvals...))
	return l
}

func (l Logger) Debug() log.Logger {
	l.Logger = level.Debug(l.Logger)
	return l
}

func (l Logger) Info() log.Logger {
	l.Logger = level.Info(l.Logger)
	return l
}

func (l Logger) Crit() log.Logger {
	l.Logger = level.Error(l.Logger)
	return l
}

func (l Logger) Error() log.Logger {
	l.Logger = level.Error(l.Logger)
	return l
}

func (l Logger) Warn() log.Logger {
	l.Logger = level.Warn(l.Logger)
	return l
}

func Default() Logger {
	var timer log.Valuer = func() interface{} { return time.Now().Format(time.RFC3339Nano) }

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

	lg := Logger{
		Logger: l,
	}
	lg = lg.With("ts", timer, "caller", log.DefaultCaller)

	return lg
}

// NewNopLogger returns a logger that doesn't do anything. This just a wrapper of
// `go-kit/log.nopLogger`.
func NewNopLogger() Logger {
	return Logger{Logger: log.NewNopLogger()}
}

type logWriter struct {
	log.Logger
}

func (l logWriter) Write(p []byte) (int, error) {
	err := l.Log("msg", string(p))
	return len(p), err
}

func LogWriter(logger log.Logger) io.Writer {
	return &logWriter{logger}
}

type GormLogger struct {
	Logger
}

func (l GormLogger) Print(values ...interface{}) {
	if len(values) > 1 {
		level, source := values[0], values[1]
		log := l.With("type", level, "source", source)
		if level == "sql" {
			dur, sql, values := values[2].(time.Duration), values[3].(string), values[4].([]interface{})
			sqlLog(log, dur, sql, values)
			return
		} else if level == "log" {
			logLog(log, values[2:])
		}
	} else {
		l.Info().Log("msg", fmt.Sprintf("%+v", values))
	}
}

func sqlLog(l Logger, dur time.Duration, query string, values []interface{}) {
	logger := l.Debug()
	if dur > 100*time.Millisecond {
		logger = l.Warn()
	} else if dur > 50*time.Millisecond {
		logger = l.Info()
	}

	logger.Log("query_us", int64(dur/time.Microsecond), "query", query, "values", fmt.Sprintf("%+v", values))
}

func logLog(l Logger, values ...interface{}) {
	msg := ""
	if len(values) == 1 {
		if err, ok := values[0].(error); ok {
			l.Error().Log("msg", err)
		} else {
			msg = fmt.Sprintf("%+v", values[0])
		}
	} else {
		msg = fmt.Sprintf("%+v", values)
	}
	l.Info().Log("msg", msg)
}
