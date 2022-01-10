# Trace

This package provides interfaces for tracing functions and printing traces into logs.

## Usage

Import the package

```
import 'github.com/theplant/appkit/log/trace'
```

Inside a function, add these two lines to trace it:

```
func DoWork(ctx context.Context) err error {
	ctx, _ := trace.StartSpan(ctx, "<span.context>")
	defer func() { trace.EndSpan(ctx, err) }()
}
```

It will do these things:

Start a span when the function is executed
End the span when the process completed
Record the error which the function returns
Log span with the logger in context

You can add attributes to a span:

```
func DoWork(ctx context.Context) err error {
	// ctx, span := trace.StartSpan(ctx, "<span.context>")
	// or
	// span := trace.FromContext(ctx)

	span.AddAttributes(
		trace.Attribute("app.record_id", "id"),
	)
}
```

or you can use the helper `AppendKVs`:

```
func DoWork(ctx context.Context) err error {
	...

	trace.AppendKVs(ctx,
		"app.record_id", "id",
	)
}
```

And you can add inheritable attributes to a span. They will be inherited by child spans and printed into logs :

```
func DoWork(ctx context.Context) err error {
	// ctx, span := trace.StartSpan(ctx, "<span.context>")
	// or
	// span := trace.FromContext(ctx)

	span.AddInheritableAttributes(
		trace.Attribute("family.name", "..."),
	)
}
```

or you can use the helper `AppendInheritableKVs`:

```
func DoWork(ctx context.Context) err error {
	...

	trace.AppendInheritableKVs(ctx,
		"family.name", "...",
	)
}
```

## Guide(TODO)

Trace
- Entry points
	- `type`
	- `role`
- Calls out
	- `type`: `db`, `gcp.fcm`
	- `role`
Internal functions
	- `type`: `internal`
