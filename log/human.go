package log

import (
	"fmt"
	"os"
	"strings"

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
func PrettyFormat_(values ...interface{}) (r string) {
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

func ex(r map[string]interface{}, f string) (interface{}, bool) {
	v, ok := r[f]
	delete(r, f)
	//	if ok && len(v) > 0 {
	//		return strings.Join(v, "."), true
	//	}
	return v, ok

}

func mapVals(values ...interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	for i, key := range values {
		if i%2 == 1 {
			continue
		}

		m[key.(string)] = values[i+1]
	}
	return m
}

/*
PrettyFormat accepts log values and returns pretty output string
*/
func PrettyFormat(values ...interface{}) string {
	r := mapVals(values...)

	level, ok := ex(r, "level")
	if !ok {
		level = "<none>"
	}

	ts, ok := ex(r, "ts")
	var tsStr string
	if ok {
		tsStr = ts.(string)
	} else {
		tsStr = strings.Repeat(" ", len(time.StampMilli))
	}

	msgI, ok := ex(r, "msg")
	msg := "<no msg>"
	if ok {
		m, ok := msgI.(string)
		if ok && len(m) > 0 {
			msg = m
		}
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

	for k, v := range r {
		output = append(output, fmt.Sprintf("  %s=%v", k, v))
	}

	output = append(output, "", "")

	return strings.Join(output, "\n")
}
