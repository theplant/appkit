package tracing

import (
	"context"
	"fmt"
	"io"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/server"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func trace(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := log.ForceContext(ctx).With(
			"context", "appkit/tracing.trace",
		)

		// Extract tracing propagarion info from HTTP request
		opts := []opentracing.StartSpanOption{}
		wireContext, err := opentracing.GlobalTracer().Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header))

		if err == opentracing.ErrSpanContextNotFound {
			l.Debug().Log(
				"msg", "no span to propagate, starting new trace",
				"span_context", wireContext,
			)
		} else if err != nil {
			l.Warn().Log(
				"msg", fmt.Sprintf("failed to extract tracing headers from request, will start new span: %v", err),
				"err", err,
			)
		} else {
			opts = append(opts, ext.RPCServerOption(wireContext))
		}

		Span(ctx, r.URL.Path, func(ctx context.Context, span opentracing.Span) error {
			l.Info().Log(
				"msg", "tracing span",
				"span_context", span.Context(),
			)

			ext.SpanKind.Set(span, ext.SpanKindRPCServerEnum)
			ext.HTTPMethod.Set(span, r.Method)
			ext.HTTPUrl.Set(span, r.URL.String())

			h.ServeHTTP(w, r.WithContext(ctx))
			s, _ := contexts.HTTPStatus(ctx)
			ext.HTTPStatusCode.Set(span, uint16(s))
			if s >= 500 {
				ext.Error.Set(span, true)
			}
			return nil
		}, opts...)
	})
}

type loggedError struct {
	err interface{}
}

func (l loggedError) Error() string {
	return l.err.(error).Error()
}

func Span(ctx context.Context, name string, f func(context.Context, opentracing.Span) error, opts ...opentracing.StartSpanOption) (e error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, name, opts...)
	defer func() {
		err := recover()
		// if err != nil, f panicked. And if f panicked, e has to be
		// nil.
		if err != nil {
			var ok bool
			// can't use := as we want to assign to the return value
			e, ok = err.(error)

			if !ok {
				e = fmt.Errorf("panic with non-error: %#v", err)
				err = e
			}
		}

		if e != nil {
			ext.Error.Set(span, true)
			if _, logged := e.(loggedError); !logged {
				span.LogKV("error", e)
			}
		}

		span.Finish()

		// re-panic if necessary
		if err != nil {
			err = loggedError{err}
			panic(err)
		}
	}()

	e = f(ctx, span)
	return
}

type nullCloser struct{}

func (nullCloser) Close() error { return nil }

func Tracer(logger log.Logger) (io.Closer, server.Middleware, error) {
	logger = logger.With(
		"context", "appkit/tracing.Tracer",
	)

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		logger.Info().Log(
			"msg", fmt.Sprintf("didn't configure tracer: %v", err),
			"err", err,
		)
		return nullCloser{}, server.IdMiddleware, nil
	} else if cfg.ServiceName == "" {
		logger.Info().Log(
			"msg", fmt.Sprintf("didn't configure tracer: no service name set"),
		)
		return nullCloser{}, server.IdMiddleware, nil
	}
	closer, err := cfg.InitGlobalTracer("") // Name will come from environment
	return closer, trace, err
}
