package log

import (
	"fmt"
	"os"

	slog "log"

	"time"

	"github.com/go-kit/kit/log"
)

/*
Human is for create a Logger that print human friendly log
*/
func Human() Logger {
	l := log.NewSyncWriter(os.Stdout)
	lg := Logger{
		log.LoggerFunc(func(values ...interface{}) (err error) {
			fmt.Fprint(l, PrettyFormat(values...))
			return
		}),
	}
	var timer log.Valuer = func() interface{} { return time.Now().Format("15:04:05.99") }
	lg = lg.With("ts", timer)

	slog.SetOutput(log.NewStdlibAdapter(lg))

	return lg
}

/*
PrettyFormat accepts log values and returns pretty output string
*/
func PrettyFormat(values ...interface{}) (r string) {
	var ts, msg, level, stacktrace, sql, sqlValues interface{}
	var shorts []interface{}
	var longs []interface{}
	var isSQL bool

	for i := 1; i < len(values); i += 2 {
		key := values[i-1]
		val := values[i]
		if key == "ts" {
			ts = val
			continue
		}
		if key == "msg" {
			msg = val
			continue
		}

		if key == "level" {
			level = val
			continue
		}

		if key == "stacktrace" {
			stacktrace = val
			continue
		}

		if key == "query" {
			sql = val
			isSQL = true
			continue
		}

		if isSQL && key == "values" {
			sqlValues = val
			continue
		}

		if len(fmt.Sprintf("%+v", val)) > 50 {
			longs = append(longs, fmt.Sprintf("\033[34m%+v\033[39m=%+v", key, val))
			continue
		}

		shorts = append(shorts, fmt.Sprintf("\033[34m%+v\033[39m=%+v", key, val))
	}

	var pvals = []interface{}{}
	if ts != nil {
		pvals = append(pvals, fmt.Sprintf("\033[36m%s\033[0m", ts))
	}

	if msg != nil {
		color := "39"
		level = fmt.Sprintf("%+v", level)
		switch level {
		case "crit":
			color = "35"
		case "error":
			color = "31"
		case "warn":
			color = "33"
		case "info":
			color = "32"
		case "debug":
			color = "90"
		}
		pvals = append(pvals, fmt.Sprintf("\033[%sm%s", color, msg))
	}

	pvals = append(pvals, shorts...)
	if len(longs) > 0 {
		pvals = append(pvals, "\n")
		for _, long := range longs {
			pvals = append(pvals, "          ", long, "\n")
		}
	}

	if sql != nil {
		pvals = append(pvals, fmt.Sprintf("\n            %s", sql), "\n")
		if sqlValues != nil {
			pvals = append(pvals, fmt.Sprintf("           \033[34m%s\033[0m=%s", "values", sqlValues), "\n")
		}
	}

	if stacktrace != nil {
		pvals = append(pvals, fmt.Sprintf("\n%s", stacktrace), "\n")
	}

	return fmt.Sprintln(pvals...)
}
