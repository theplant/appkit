# logtracing

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

	// or get the active span from the context

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

## Key-values

### Common

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
- `http.status`: thes response status, only exists when getting the response successfully

For the server requests:

- `span.type`: `http`
- `span.role`: `server`
- `http.path`: the request path
- `http.method`: the request method
- `http.user_agent`: the request user agent
- `http.client_ip`: the request client IP

### Queue

- `span.type`: `queue`
- `span.role`: `consumer` or `producer`

### Function

For the internal functions:
- `span.role`: `internal`
