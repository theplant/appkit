# Trace

This package provides APIs for tracing functions and printing traces into logs.

## Usage

Import the package

```
import 'github.com/theplant/appkit/logtracing'
```

Inside a function, add these two lines to trace it:

```
func DoWork(ctx context.Context) err error {
	ctx, _ := logtracing.StartSpan(ctx, "<span.context>")
	defer func() { logtracing.EndSpan(ctx, err) }()
}
```

It will trace the function, record the error, and finally log the span with the logger in context.

You can append key-values to an active span:

```
func DoWork(ctx context.Context) err error {
	ctx, _ := logtracing.StartSpan(ctx, "<span.context>")
	logtracing.AppendSpanKVs(ctx,
		"service", "greeter",
	)

	// or get the active span from context

	span := logtracing.SpanFromContext(ctx)
	span.AppendKVs(
		"service", "greeter",
	)
}
```

If you want to append key-values to all spans, you can append the key-values to the context with `ContextWithKVs`:

```
func DoWork(ctx context.Context) err error {
	// all the spans within this context will contain `"key": "value"`
	ctx = logtracing.ContextWithKVs(ctx, "key", "value")
	ctx, _ = logtracing.StartSpan(ctx, "test")
}
```

## Guidence

### XMLRPC

### GRPC

### HTTP

### Function
