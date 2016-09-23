package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	l "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
)

type Logger struct {
	*levels.Levels
}

func (logger Logger) With(keysvals ...interface{}) Logger {
	levels := logger.Levels.With(keysvals...)
	return Logger{&levels}
}

/*
func (logger Logger) Debug() l.Logger {
	return l.NewNopLogger()
}

func (logger Logger) Info() l.Logger {
	return l.NewNopLogger()
}
*/
func Default() Logger {
	//	logger := l.NewLogfmtLogger(l.NewSyncWriter(os.Stdout))
	//	logger = l.NewContext(logger).With("ts", l.DefaultTimestampUTC, "caller", l.DefaultCaller)

	logger := PrettyFmt(l.NewSyncWriter(os.Stdout))
	var timer l.Valuer = func() interface{} { return time.Now().Format(time.StampMilli) }
	logger = l.NewContext(logger).With("ts", timer, "caller", l.DefaultCaller)

	l := levels.New(logger)
	return Logger{&l}

}

type prettyFmt struct {
	w io.Writer
}

func PrettyFmt(w io.Writer) l.Logger {
	return &prettyFmt{w}
}

func (l prettyFmt) Log(keysvals ...interface{}) error {
	data := map[interface{}]interface{}{}

	for i, key := range keysvals {
		if i%2 == 0 {
			data[key] = keysvals[i+1]
		}
	}

	level, ok := data["level"]
	delete(data, "level")
	if !ok {
		level = "<none>"
	}

	ts, ok := data["ts"]
	delete(data, "ts")
	var tsStr string
	if ok {
		tsStr = ts.(string)
	} else {
		tsStr = strings.Repeat(" ", len(time.StampMilli))
	}

	msg, ok := data["msg"]
	delete(data, "msg")
	if msg == nil || len(msg.(string)) == 0 {
		msg = "<no msg>"
	}

	colour := "37"
	switch level {
	case "crit":
		colour = "35"
	case "error":
		colour = "31"
	case "warn":
		colour = "33"
	case "info":
		colour = "32"
	case "debug":
		colour = "30"
	}

	output := []string{fmt.Sprintf("\033[%s;1m%s: %s\033[0m (%s)", colour, tsStr, msg, level)}

	for k, v := range data {
		output = append(output, fmt.Sprintf("  %s=%v", k, v))
	}

	output = append(output, "", "")

	if _, err := l.w.Write([]byte(strings.Join(output, "\n"))); err != nil {
		return err
	}
	return nil
}

type logWriter struct {
	l.Logger
}

func (l logWriter) Write(p []byte) (int, error) {
	err := l.Log("msg", string(p))
	return len(p), err
}

func LogWriter(logger l.Logger) io.Writer {
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
