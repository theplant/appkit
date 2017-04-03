package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/contexts/trace"
	"github.com/theplant/appkit/log"
)

type key int

const monitorKey key = iota

// WithMonitor wraps the given http.Handler to:
// 1. instrument requests via a Monitor
// 2. install monitor in request context for use by later handlers
func WithMonitor(m Monitor) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			defer func() {
				interval := time.Now().Sub(start)
				go func() {
					tags := tagsForContext(r)
					m.InsertRecord("request", float64(interval/time.Millisecond), tags, start)

				}()
			}()

			h.ServeHTTP(w, r.WithContext(Context(r.Context(), m)))
		})
	}
}

// Context installs a given Monitor in the returned context
func Context(c context.Context, m Monitor) context.Context {
	return context.WithValue(c, monitorKey, m)
}

// ForceContext extracts a Monitor from a (possibly nil) context, or
// returns a NewLogMonitor using a log from the context or
// log.Default()
func ForceContext(ctx context.Context) Monitor {
	var logger log.Logger
	if ctx != nil {
		val := ctx.Value(monitorKey)
		if monitor, ok := val.(Monitor); ok {
			return monitor
		}

		logger = log.ForceContext(ctx)
	} else {
		logger = log.Default()
	}
	return NewLogMonitor(logger)
}

func tagsForContext(r *http.Request) map[string]string {
	path := scrubPath(r.URL.Path)
	tags := map[string]string{
		"path":           path,
		"request_method": r.Method,
	}

	ctx := r.Context()

	if reqID, ok := trace.RequestTrace(ctx); ok {
		tags["req_id"] = fmt.Sprintf("%v", reqID)
	}

	if status, ok := contexts.HTTPStatus(ctx); ok {
		tags["response_code"] = strconv.Itoa(status)
	} else {
		log.ForceContext(ctx).Warn().Log(
			"msg", fmt.Sprintf("cannot determine response code for %s %s (perhaps no WithHTTPStatus in context?)", r.Method, path),
			"path", path,
			"method", r.Method,
		)
	}

	return tags
}

var idScrubber = regexp.MustCompile("[0-9]+")

func scrubPath(path string) string {
	return idScrubber.ReplaceAllString(path, ":id")
}
