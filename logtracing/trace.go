package logtracing

import (
	"context"
	"fmt"
	"time"

	"github.com/theplant/appkit/log"
)

type kvsContextKey struct{}

var activeKVsKeys = kvsContextKey{}

func KVsFromContext(ctx context.Context) []interface{} {
	kvs, _ := ctx.Value(activeKVsKeys).([]interface{})
	return kvs
}

func ContextWithKVs(ctx context.Context, keyvals ...interface{}) context.Context {
	if len(keyvals)%2 != 0 {
		log.ForceContext(ctx).Warn().Log("msg", fmt.Sprintf("missing key or value for span attributes: %q", keyvals))
	}

	exisiting := KVsFromContext(ctx)
	if exisiting != nil {
		copy := append([]interface{}{}, exisiting...)
		keyvals = append(copy, keyvals...)
	}

	ctx = context.WithValue(ctx, activeKVsKeys, keyvals)
	return ctx
}

type spanContextKey struct{}

var activeSpanKey = spanContextKey{}

func SpanFromContext(ctx context.Context) *span {
	s, _ := ctx.Value(activeSpanKey).(*span)
	return s
}

func contextWithSpan(parent context.Context, s *span) context.Context {
	return context.WithValue(parent, activeSpanKey, s)
}

type StartOptions struct {
	Sampler Sampler
}

type StartOption func(*StartOptions)

func WithSampler(sampler Sampler) StartOption {
	return func(o *StartOptions) {
		o.Sampler = sampler
	}
}

func StartSpan(ctx context.Context, name string, o ...StartOption) (context.Context, *span) {
	var (
		opts        StartOptions
		cfg         = config.Load().(*Config)
		idGenerator = cfg.IDGenerator

		parent    = SpanFromContext(ctx)
		traceID   TraceID
		spanID    = idGenerator.NewSpanID()
		isSampled bool
	)

	for _, op := range o {
		op(&opts)
	}

	if parent == nil {
		traceID = idGenerator.NewTraceID()
	} else {
		traceID = parent.traceID
		isSampled = parent.isSampled
	}

	sampler := cfg.DefaultSampler
	if parent == nil || opts.Sampler != nil {
		if opts.Sampler != nil {
			sampler = opts.Sampler
		}

		var parentMeta spanMeta
		if parent != nil {
			parentMeta = parent.meta()
		}

		isSampled = sampler(SamplingParameters{
			ParentMeta: parentMeta,
			TraceID:    traceID,
			SpanID:     spanID,
			Name:       name,
		})
	}

	s := span{
		parent: parent,

		traceID:   traceID,
		spanID:    spanID,
		name:      name,
		isSampled: isSampled,

		startTime: time.Now(),
	}

	ctxKVs := KVsFromContext(ctx)
	if ctxKVs != nil {
		s.AppendKVs(ctxKVs...)
	}

	return contextWithSpan(ctx, &s), &s
}

func AppendSpanKVs(ctx context.Context, keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		log.ForceContext(ctx).Warn().Log("msg", fmt.Sprintf("missing key or value for span attributes: %q", keyvals))
	}

	s := SpanFromContext(ctx)
	if s == nil {
		return
	}

	s.AppendKVs(keyvals...)
}

func EndSpan(ctx context.Context, err error) {
	s := SpanFromContext(ctx)
	if s == nil {
		return
	}

	s.RecordError(err)
	s.End()
	LogSpan(ctx, s)
}

// If this method is called while panicing, function will record the panic into the span, and the panic is continued.
// 1. The function should be called before `EndSpan(ctx context.Context, err error)` or `(*span).End()`.
// 2. The function call should be deferred.
func RecordPanic(ctx context.Context) {
	s := SpanFromContext(ctx)
	if s == nil {
		return
	}

	if !s.IsRecording() {
		return
	}

	if recovered := recover(); recovered != nil {
		defer panic(recovered)
		s.RecordPanic(recovered)
	}
}

func LogSpan(ctx context.Context, s *span) {
	var (
		l       = log.ForceContext(ctx)
		keyvals []interface{}
		dur     = s.Duration()
	)

	keyvals = append(keyvals,
		"ts", s.startTime.Format(time.RFC3339Nano),
		"trace.id", s.traceID,
		"span.id", s.spanID,
		"span.context", s.name,
		"span.dur_ms", dur.Milliseconds(),
	)

	if s.isSampled {
		keyvals = append(keyvals, "span.is_sampled", 1)
	}

	if s.parent != nil {
		keyvals = append(keyvals, "span.parent_id", s.parent.spanID)
	}

	keyvals = append(keyvals, s.keyvals...)

	if s.panic != nil {
		keyvals = append(keyvals,
			"msg", fmt.Sprintf("%s (%v) -> panic: %+v (%T)", s.name, dur, s.panic, s.panic),
			"span.panic", s.panic,
			"span.panic_type", errType(s.panic),
			"span.with_panic", 1,
			"span.with_err", 1,
		)
		l.Crit().Log(keyvals...)
		return
	}

	if s.err != nil {
		keyvals = append(keyvals,
			"msg", fmt.Sprintf("%s (%v) -> error: %+v (%T)", s.name, dur, s.err, s.err),
			"span.err", s.err,
			"span.err_type", errType(s.err),
			"span.with_err", 1,
		)
		l.Error().Log(keyvals...)
		return
	}

	keyvals = append(keyvals,
		"msg", fmt.Sprintf("%s (%v) -> success", s.name, dur),
	)
	l.Info().Log(keyvals...)
}

type causer interface {
	Cause() error
}

func errType(err interface{}) string {
	if c, ok := err.(causer); ok {
		return fmt.Sprintf("%T (%T)", c.Cause(), err)
	}
	return fmt.Sprintf("%T", err)
}
