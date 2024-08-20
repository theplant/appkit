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
	Sampler      Sampler
	StartTime    time.Time
	TraceID      TraceID
	ParentSpanID SpanID
}

type StartOption func(*StartOptions)

func WithSampler(sampler Sampler) StartOption {
	return func(o *StartOptions) {
		o.Sampler = sampler
	}
}

func WithStartTime(t time.Time) StartOption {
	return func(o *StartOptions) {
		o.StartTime = t
	}
}

func WithTraceID(id TraceID) StartOption {
	return func(o *StartOptions) {
		o.TraceID = id
	}
}

func WithParentSpanID(id SpanID) StartOption {
	return func(o *StartOptions) {
		o.ParentSpanID = id
	}
}

func StartSpan(ctx context.Context, name string, o ...StartOption) (context.Context, *span) {
	var (
		opts        StartOptions
		cfg         = config.Load().(*Config)
		idGenerator = cfg.IDGenerator

		parent       = SpanFromContext(ctx)
		parentSpanID SpanID
		traceID      TraceID
		spanID       = idGenerator.NewSpanID()
		isSampled    bool
		startTime    time.Time
	)

	for _, op := range o {
		op(&opts)
	}

	if parent == nil {
		if opts.ParentSpanID.IsValid() {
			parentSpanID = opts.ParentSpanID
		}
		if opts.TraceID.IsValid() {
			traceID = opts.TraceID
		} else {
			traceID = idGenerator.NewTraceID()
		}
	} else {
		parentSpanID = parent.spanID
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

	if opts.StartTime.IsZero() {
		startTime = time.Now()
	} else {
		startTime = opts.StartTime
	}

	s := span{
		parentSpanID: parentSpanID,

		traceID:   traceID,
		spanID:    spanID,
		name:      name,
		isSampled: isSampled,

		startTime: startTime,
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
	ExportSpan(s)
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

	if s.parentSpanID.IsValid() {
		keyvals = append(keyvals, "span.parent_id", s.parentSpanID)
	}

	keyvals = append(keyvals, s.keyvals...)

	if s.panic != nil {
		keyvals = append(keyvals,
			"msg", fmt.Sprintf("%s (%v) -> panic: %+v (%T)", s.name, dur, s.panic, s.panic),
			"span.panic", fmt.Sprintf("%s", s.panic),
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
			"span.err", s.err.Error(),
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

func ExportSpan(s *span) {
	if s == nil || s.IsRecording() {
		return
	}

	exp, _ := exporters.Load().(exportersMap)
	if s.isSampled && len(exp) > 0 {
		sd := makeSpanData(s)

		for e := range exp {
			e.ExportSpan(sd)
		}
	}
}

func makeSpanData(s *span) *SpanData {
	return &SpanData{
		ParentSpanID: s.parentSpanID,

		TraceID:   s.traceID,
		SpanID:    s.spanID,
		Name:      s.name,
		IsSampled: s.isSampled,

		StartTime: s.startTime,
		EndTime:   s.endTime,

		Err:   s.err,
		Panic: s.panic,

		Keyvals: append(s.keyvals[:0:0], s.keyvals...),
	}
}
