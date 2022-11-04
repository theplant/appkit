****# logtracing

This package is a toolkit for log-based tracing. It provides several APIs for adding traces for the application and logging them in a standard format.

## Usage

Import the package

```
import 'github.com/theplant/appkit/logtracing'
```

Inside a function, you can use these three APIs to track it:
- `StartSpan(context.Context, string)` to start a span
- `EndSpan(context.Context, error)` to end the span, also log it in an agreed format
- `RecordPanic(context.Context)` to record the panic into the span

```
func DoWork(ctx context.Context) err error {
	ctx, _ := logtracing.StartSpan(ctx, "<span.context>")
	defer func() { logtracing.EndSpan(ctx, err) }()
	defer RecordPanic(ctx)
}
```

It will create a new span, record the error, and log the span with the logger in context.

You can append key-values to an active span with `AppendSpanKvs`:

```
func DoWork(ctx context.Context) err error {
	ctx, _ := logtracing.StartSpan(ctx, "<span.context>")
	defer func() { logtracing.EndSpan(ctx, err) }()
	defer RecordPanic(ctx)

	logtracing.AppendSpanKVs(ctx,
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
	defer func() { logtracing.EndSpan(ctx, err) }()
	defer RecordPanic(ctx)
}
```

For flexibility, the package provides the following APIs to manipulate the span directly:

- `(*span).AppendKVs(...interface{})`
- `(*span).RecordError(err error)`
- `(*span).RecordPanic(panic interface{}})`
- `(*span).End()`
- `LogSpan(ctx context, s *span)`

## Key-values

### Common

With `logtracing.StartSpan` and `logtracing.EndSpan`, these are automatically added to the span:

- `ts`
- `msg`
- `trace.id`
- `span.id`
- `span.context`
- `span.dur_ms`
- `span.parent_id`

If the span records an error, these key-values will be added:

- `span.err`
- `span.err_type`
- `span.with_err`

### Reference

- `span.type`: This usually describes "how" the span is sent/received, what kind of underlying API or network transport is used, eg  `http`, `sql`, `grpc`, `dns`, `aws.<service>`
- `span.role`: This describes what role this span plays in an interaction with another service, eg for HTTP a `client` makes a request to a `server`. Or in a `queue`, a `producer` adds work to the queue, that is consumed by a `consumer`.

Data that is for specific types should use the type as a prefix when adding supplementary keys-values, For example:

- With `span.type=http`, the HTTP method would be logged as `http.method`, the response status would be logged as `http.status=200`.

- AWS S3 API calls (`span.type=aws.s3`), use `s3` or `aws.s3` as a prefix, an object used in the API call could be logged as `s3.object=<object key>` (or `aws.s3.object=<object key>`)

### XMLRPC

- `span.type`: `xmlrpc`
- `span.role`: `client`
- `xmlrpc.method`: the service method
- `http.method`: `post`
- `http.url`: the server URL
- `xmlrpc.fault_string`: only exists when getting a fault error

### GRPC

- `span.type`: `grpc`
- `span.role`: `client` or `server`
- `grpc.service`: gRPC service name
- `grpc.method`: gRPC method name
- `grpc.code`: gRPC response status code

The package provides server and client interceptors to log these key-values automatically.

### HTTP

For the client requests:

- `span.type`: `http`
- `span.role`: `client`
- `http.url`: the request full URL
- `http.method`: the request method
- `http.status`: the response status, only exists when getting the response successfully

For the server requests:

- `span.type`: `http`
- `span.role`: `server`
- `http.path`: the request path
- `http.method`: the request method
- `http.user_agent`: the request user agent
- `http.client_ip`: the request client IP
- `http.status`: the response status

### Queue

- `span.type`: `queue`
- `span.role`: `consumer` or `producer`

### Function

For internal functions:
- `span.role`: `internal`

## How to migrate from `util/trace.go`

1. Use `logtracing.TraceFunc` to replace `util.Lt`
2. use `logtracing.AppendSpanKVs` to replace `util.AppendKVs`
3. use `logtracing.TraceHTTPRequest` to replace `util.LtRequest`
4. use `logtracing.HTTPTransport` to replace `util.LtTransport`
5. use `logtracing.XMLRPCClientKVs` to replace `util.XMLRpcKVs`
