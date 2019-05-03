package errornotifier

import (
	"context"
	"fmt"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ctxtrace "github.com/theplant/appkit/contexts/trace"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/tracing"
)

type key int

const ctxKey key = iota

// Recover wraps an http.Handler to report all `panic`s to Airbrake.
func Recover(n Notifier) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c := Context(req.Context(), n)
			err := NotifyOnPanic(n, req, func() {
				h.ServeHTTP(w, req.WithContext(c))
			})
			if err != nil {
				panic(err)
			}
		})
	}
}

// ForceContext extracts a notifier from the request context, falling
// back to a LogNotifier using the context's logger.
func ForceContext(c context.Context) Notifier {
	if c != nil {
		notifier, ok := c.Value(ctxKey).(Notifier)
		if ok {
			return notifier
		}
	}

	return NewLogNotifier(log.ForceContext(c))
}

// Context installs a given Error Notifier in the returned context
func Context(c context.Context, n Notifier) context.Context {
	return context.WithValue(c, ctxKey, n)
}

// NotifyOnPanic will notify Airbrake if function f panics, and will
// return the error that caused the panic (if any)
//
// This is for wrapping Goroutines to prevent panics from bringing
// down the whole application.
func NotifyOnPanic(n Notifier, req *http.Request, f func()) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		if e, ok := r.(error); !ok {
			err = fmt.Errorf("%v", r)
		} else {
			err = e
		}

		var ctx context.Context
		if req != nil {
			ctx = req.Context()
		} else {
			ctx = context.Background()
		}

		_ = tracing.Span(ctx, "appkit/errornotifier.NotifyOnPanic", func(ctx context.Context, span opentracing.Span) error {
			ext.SpanKind.Set(span, ext.SpanKindRPCClientEnum)

			notifyCtx := map[string]interface{}{}
			if ctxtraceID, ok := ctxtrace.RequestTrace(ctx); ok {
				notifyCtx["req_id"] = ctxtraceID
			}
			notifyCtx["span_context"] = fmt.Sprintf("%v", span.Context())

			n.Notify(err, req, notifyCtx)

			return nil
		})
		return
	}()

	f()
	return
}
