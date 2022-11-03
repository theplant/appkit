package logtracing

import "context"

func TraceFunc(ctx context.Context, name string, f func(context.Context) error) (err error) {
	ctx, _ = StartSpan(ctx, name)
	defer func() { EndSpan(ctx, err) }()
	defer RecordPanic(ctx)

	return f(ctx)
}

func InternalFuncKVs() []interface{} {
	return []interface{}{
		"span.role", "internal",
	}
}
