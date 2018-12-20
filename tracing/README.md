# Opentracing

This package supports distributed tracing of requests by linking
together all of the parts of work that go into fulfilling a request.

For example, with a HTML front-end talking to back-end HTTPS APIs, it
will link the original front-end request with any/all HTTP requests
made to the back-end. Also, it can link together deeper requests made
by the *back-end* to other APIs and services.

Currently this library requires the jaeger-lib dependency to be pinned
at `v1.5.0`.

# Middleware

To configure tracing of incoming HTTP requests:

1. Set `JAEGER_*` environment variables.

2. Initialise tracer:

   
   ```
   closer, tracer, err := tracing.Tracer(log)
   if err != nil {
   	log.WithError(err).Log()
   } else {
   	defer closer.Close()
   }
   ```

   The returned `io.Closer` is used to ensure that traces are sent to
   the server before the process exits.

   Initialising the tracer installs it as the Opentracing *global*
   tracer.

3. Use the returned `tracer` middleware with your HTTP handlers:

   ```
   http.ListenAndServe(tracer(handler))
   ```

   If you're using `server.Compose`, add the tracer to the bottom of
   your stack, just above `server.DefaultMiddleware`. This will make
   the tracing system available to other middleware such as
   `errornotifier` (see below).

# Trace Propagagtion

The middleware will automatically continue spans if the incoming HTTP
request has span proagation headers.

To pass tracing headers downstream when making HTTP requests use
`Inject`:

```
// Transmit the span's TraceContext as HTTP headers on our
// outbound request.
opentracing.GlobalTracer().Inject(
	span.Context(),
	opentracing.HTTPHeaders,
	opentracing.HTTPHeadersCarrier(httpReq.Header))
```

# Adding Detail to Traces

this package exports a `Span` function that is used to add more detail
to a trace. When you perform some kind of sub-operation, such as an
SQL query, or API call to another service, wrap the call in a call to
`Span`:

```
tracing.Span(ctx, "<operation name>", func(ctx context.Context, span opentracing.Span) error {
     httpReq := ...

    // Add info to the span
    ext.HTTPMethod.Set(span, httpReq.Method)
    ext.HTTPUrl.Set(span, httpReq.URL.String())
	ext.SpanKind.Set(span, ext.SpanKindRPCClientEnum)

    // Propagate trace id to back-end service
    opentracing.GlobalTracer().Inject(
	    span.Context(),
	    opentracing.HTTPHeaders,
	    opentracing.HTTPHeadersCarrier(httpReq.Header))


    // perform your operation, and update span with any details
    resp, err := doOperation(httpReq)

    if err != nil {
        // `tracing.Span` will automatically mark the span as an
        // error and log the error message, if we return an error.
        return err
    }
    
    if opFailed(resp) {
        // manually mark the span as an error if there was some kind
        // of logical problem
        ext.Error.Set(span, true)
		span.LogKV("error", "the operation failed!!")
    }
    return nil
})
```


# Background

This package uses [OpenTracing](https://opentracing.io) and is
configured to use a [Jaeger](https://www.jaegertracing.io) server by
default. The OpenTracing site has a good overview of the concepts.

