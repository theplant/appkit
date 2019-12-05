package log

import (
	"database/sql/driver"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"unicode"

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

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

var sqlRegexp = regexp.MustCompile(`\?`)
var numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)

// copied from gorm
func humanSql(values []interface{}) (sql string) {
	var formattedValues []string

	for _, value := range values[4].([]interface{}) {
		indirectValue := reflect.Indirect(reflect.ValueOf(value))
		if indirectValue.IsValid() {
			value = indirectValue.Interface()
			if t, ok := value.(time.Time); ok {
				formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
			} else if b, ok := value.([]byte); ok {
				if str := string(b); isPrintable(str) {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
				} else {
					formattedValues = append(formattedValues, "'<binary>'")
				}
			} else if r, ok := value.(driver.Valuer); ok {
				if value, err := r.Value(); err == nil && value != nil {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			} else {
				switch value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
					formattedValues = append(formattedValues, fmt.Sprintf("%v", value))
				default:
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				}
			}
		} else {
			formattedValues = append(formattedValues, "NULL")
		}
	}

	// differentiate between $n placeholders or else treat like ?
	if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
		sql = values[3].(string)
		for index, value := range formattedValues {
			placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
			sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
		}
	} else {
		formattedValuesLength := len(formattedValues)
		for index, value := range sqlRegexp.Split(values[3].(string), -1) {
			sql += value
			if index < formattedValuesLength {
				sql += formattedValues[index]
			}
		}
	}
	return
}
