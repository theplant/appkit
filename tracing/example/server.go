package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/theplant/appkit/errornotifier"
	"github.com/theplant/appkit/log"
	"github.com/theplant/appkit/monitoring"
	"github.com/theplant/appkit/server"
	"github.com/theplant/appkit/tracing"
)

func main() {

	sCfg := server.Config{Addr: ":9900"}
	log := log.Default()

	closer, tracer, err := tracing.Tracer(log)
	if err != nil {
		log.WithError(err).Log()
	} else {
		defer closer.Close()
	}

	monitor := monitoring.NewLogMonitor(log)
	notifier := errornotifier.NewLogNotifier(log)

	server.ListenAndServe(
		sCfg,
		log,
		server.Compose(
			monitoring.WithMonitor(monitor),
			errornotifier.Recover(notifier),
			tracer,
			server.DefaultMiddleware(log),
		)(http.HandlerFunc(errHandler)),
	)
}

func handler(w http.ResponseWriter, r *http.Request) {
	tracing.Span(r.Context(), "sub", func(ctx context.Context, span opentracing.Span) error {
		time.Sleep(300 * time.Millisecond)

		return tracing.Span(ctx, "subsub", func(ctx context.Context, span opentracing.Span) error {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(204)
			return nil
		})
	})
}

func errHandler(w http.ResponseWriter, r *http.Request) {
	tracing.Span(r.Context(), "errspan", func(ctx context.Context, span opentracing.Span) error {

		w.WriteHeader(500)
		return errors.New("upstream error")
	})
}

func panicHandler(w http.ResponseWriter, r *http.Request) {
	tracing.Span(r.Context(), "panicspan", func(ctx context.Context, span opentracing.Span) error {
		panic(errors.New("panic"))
	})
}

func panicNonErrorHandler(w http.ResponseWriter, r *http.Request) {
	tracing.Span(r.Context(), "panicnonerrorspan", func(ctx context.Context, span opentracing.Span) error {
		panic("non error")
	})
}
