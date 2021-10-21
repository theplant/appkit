# Trace

This package provides interfaces for tracing functions and printing traces into logs.

## Usage

Inside a function, add these two lines to trace it:

```
func DoWork(ctx context.Context) err error {
	ctx, span := StartSpan(ctx, "<span.context>")
	defer func() { EndSpan(ctx, span, err) }()
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
	...
	span.AddAttributes(
		Attribute("app.record_id", "id"),
	)
}
```

And you can add inheritable attributes to a span. They will be inherited by children spans and printed into logs :

```
func DoWork(ctx context.Context) err error {
	...
	span.AddInheritableAttributes(
		Attribute("family.name", "..."),
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
