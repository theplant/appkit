# Appkit

Packages to do common stuff.

# Log

Provide a default logfmt-style logger (using go-kit/log/levels) for your app.

```go
logger := log.Default()

err := someAction()
logger.Error().Log(
    "msg", "Something happened",
    "during", "someAction",
    "err", err
) // => level=error msg="Something happened", during=someAction err=<error as string>
```

You can pre-set common fields:

```go
l := logger.With("context", "README")

logger.Info("msg", "the message") // => level=info context=README msg="the message"
```


Supported levels:

* `Debug`
* `Info`
* `Warn`
* `Error`
* `Crit`

## Field recommendations

* `msg`: human-readable version of the log, eg. `"fmt.Sprintf("error loading config: %v", err)`, `"registering gorm callbacks"`.
* `context`: what the system was doing when the info was logged, eg `oauthprovider.endpoint`, `dbmigration`.
* `err`: the error being logged
* `during`: the function that returned an error or induced the log, eg `influxdb.Client.Write`

Use others that are relevant to your app or domain.

## No operation logger for testing

Provide a logger that doesn't do anything. This is quite useful for testing.

```go
logger := log.NewNopLogger()
```

## Set flag `APPKIT_LOG_HUMAN` for better dev experience

`export APPKIT_LOG_HUMAN=true` Will make the logger outputs to a format that is easily to read for developers.


# Server

## HTTP Listener

Helper for starting a HTTP server configured with a `log.Logger`. Provides `Config` and `ListenAndServe`.

## Middleware

* `ETag`: will md5 your response body and include the hash in the `ETag` HTTP header. If client provided the same ETag in `If-None-Match` HTTP header, will return `304 Not Modified` and discard the response.

  Does nothing (ie, passthrough) with non-`GET` requests and non-`200 OK` responses.

* `LogRequest`: logs incoming HTTP requests with `log`. Will log start and end of request. Uses logger from request `context.Context`, so any fields set with `log.Logger.With` will be included in the log.

* `Recovery`: recovers `panic` in HTTP handlers, sends `500 Internal Server Error` to the client, and re-`panic`s the recovered error.

* `DefaultMiddleware`: Default middleware stack: request -> record HTTP status -> trace -> log -> recover.

* `Compose`: helper to chain middleware together.

# DB

Helper for opening a `gorm.DB` connection configured with a `log.Logger`. Provides `Config` and `New`.

# [Monitoring](monitoring/README.md)

A basic interface for monitoring request times and other arbitrary data, and recording data into InfluxDB.

# [Error Notification](errornotifier/README.md)

Interface for pushing panics and arbitrary errors into error logging systems. Provides implementation for Airbrake monitoring.

# [Sessions](sessions/README.md)

This package is a wrapper of [gorilla/sessions](https://www.github.com/gorilla/sessions) to fix the potential [memory leaking problem](https://qortex.com/theplant#groups/560b63da8d93e34b8500da28/entry/58a297e98d93e316d10328f3).

# Contexts

Context wrappers and http.Handler middleware to setup and use various `context.Context`s.

* `RequestTrace`: generate unique id for each HTTP request. Useful for tracing everything that happens due to a single request. Used by `Logger`

* `Logger`: makes a given logger available via `context.Context`. Integrated with `RequestTrace` to add the request trace ID to anything logged via the context, if request is being traced. Used by `server.LogRequest` middleware.

* `HTTPStatus`: records the HTTP status code for the request. Used by `LogRequest` to record the final response HTTP status code.

* `Gorm`: make a `gorm.DB` available via `context.Context`.


## Naming Style

For an "ABC" context:

* `ABC` will extract the "value" from a `context.Context`. Generally returns the value and a `bool` to indicate whether the context actually had any value.
* `WithABC` is `http.Handler` middleware that will enable "ABC" in a HTTP handler.
* `ABCContext` will wrap a `context.Context` and provide a new context that can be passed to `ABC`.
* `MustGetABC` is a wrapper around `ABC` that will `panic` when `ABC` would return false. Useful when you *need* the context value and the only way you'd handle a missing value would be to `panic`.

# [Encrypted Box](encryptedbox/README.md)

Secret Box provides a simple interface for encryption of data for storage at rest.

It is implemented as a simple wrapper around `golang.org/x/crypto/nacl/secretbox` that takes care of handling the nonce.

# [Tracing](tracing/README.md)

Tracing supports distributed tracing of requests by linking together
all of the parts of work that go into fulfilling a request.

For example, with a HTML front-end talking to back-end HTTPS APIs, it
will link the original front-end request with any/all HTTP requests
made to the back-end. Also, it can link together deeper requests made
by the *back-end* to other APIs and services.

For now, It's implemented with [OpenTracing](https://opentracing.io)
and expects to talk to a [Jaeger](https://www.jaegertracing.io)
back-end.

# Credentials

Package to provide single interface for acquiring credentials for apps
running on different platforms. Credentials that can be sourced:

* AWS
* InfluxDB

Places to source credentials from:

* [Vault](https://www.vaultproject.io)
* Local environment

# [Service](service/README.md)

Package to provide a common harness for running HTTP apps, that
configures middleware that we use nearly all the time, and provides a
standard way to configure the different parts of the service.
