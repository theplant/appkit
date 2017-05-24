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
			var msg, level, stacktrace interface{}
			var others []interface{}
			for i := 1; i < len(values); i += 2 {
				key := values[i-1]
				val := values[i]
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

				others = append(others, fmt.Sprintf("\033[34m%+v\033[0m=%+v", key, val))
			}

			now := time.Now().Format("15:04:05.9999")
			var pvals = []interface{}{fmt.Sprintf("\033[36m%s\033[0m", now)}

			if msg != nil {
				colour := "37"
				level = fmt.Sprintf("%+v", level)
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
				pvals = append(pvals, fmt.Sprintf("\033[%sm%s\033[0m", colour, msg))
			}

			pvals = append(pvals, others...)
			if stacktrace != nil {
				pvals = append(pvals, fmt.Sprintf("\n%s", stacktrace), "\n")
			}
			fmt.Fprintln(l, pvals...)
			return
		}),
	}
	slog.SetOutput(log.NewStdlibAdapter(lg))

	return lg
}
