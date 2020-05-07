package log

import (
	"fmt"
	"io"
	stdl "log"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/theplant/appkit/kerrs"
)

var isHuman bool
var isJSON bool

func init() {
	human := strings.ToLower(os.Getenv("APPKIT_LOG_HUMAN"))
	if len(human) > 0 && human != "false" && human != "0" {
		isHuman = true
	}

	isJSON = true
	json := strings.ToLower(os.Getenv("APPKIT_LOG_JSON"))
	if json == "false" || json == "0" {
		isJSON = false
	}
}

type Logger struct {
	log.Logger
}

/*
SetStdLogOutput redirect go standard log into this logger
*/
func SetStdLogOutput(logger Logger) {
	stdl.SetOutput(log.NewStdlibAdapter(logger))
}

func (l Logger) With(keysvals ...interface{}) Logger {
	l.Logger = log.With(l.Logger, keysvals...)
	return l
}

/*
WrapError wrap an original error to kerrs and add to the structured log
*/
func (l Logger) WrapError(err error) log.Logger {
	if err == nil {
		return l
	}
	return l.WithError(kerrs.Wrapv(err, ""))
}

/*
WithError add an err to structured log
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
	if isHuman {
		return Human()
	}

	l := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	if isJSON {
		l = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	}

	var timer log.Valuer = func() interface{} { return time.Now().Format(time.RFC3339Nano) }

	lg := Logger{
		Logger: l,
	}
	lg = lg.With("ts", timer, "caller", log.Caller(4))

	return lg
}

// NewNopLogger returns a logger that doesn't do anything. This just a wrapper of
// `go-kit/log.nopLogger`.
func NewNopLogger() Logger {
	return Logger{Logger: log.NewNopLogger()}
}

// NewTestLogger returns a logger that won't log timestamps or caller,
// to make output stable for use in tests.
func NewTestLogger() Logger {
	return Logger{
		Logger: log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout)),
	}
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
			dur, sql, rowsAffected := values[2].(time.Duration), values[3].(string), values[5].(int64)
			if isHuman {
				sql = humanSql(values)
			}
			sqlLog(log, dur, sql, rowsAffected)
			return
		} else if level == "log" {
			logLog(log, values[2:])
		}
	} else {
		l.Info().Log("msg", fmt.Sprintf("%+v", values))
	}
}

func sqlLog(l Logger, dur time.Duration, query string, rowsAffected int64) {
	logger := l.Debug()
	if dur > 100*time.Millisecond {
		logger = l.Warn()
	} else if dur > 50*time.Millisecond {
		logger = l.Info()
	}

	args := []interface{}{"query_us", int64(dur / time.Microsecond), "query", query, "rows_affected", rowsAffected}

	logger.Log(args...)
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
