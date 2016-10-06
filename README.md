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
