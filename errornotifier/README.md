# Error Notification

This package provides an interface for pushing panics and arbitrary
errors into error logging systems:

```
type Notifier interface {
	Notify(interface{}, *http.Request) error
}
```

The second parameter `*http.Request` is optional, and will allow the
notifier to provide added context/metadata from the request in a
notification.

There are 4 implementations of the interface:

* Push to [Airbrake](https://airbrake.io)
* Print to appkit logger
* Buffering notifier (for testing)
* Failing notifier (for testing)

# Implementations/Configuration

## Airbrake Notifier Usage

The package provides `AirbrakeConfig`:

```
type AirbrakeConfig struct {
	ProjectID   int64
	Token       string
	Environment string `default:"dev"`
}
```

Include a field of type `errornotifier.AirbrakeConfig` in your app
config:

```
type AppConfig struct {
	// ...

	Airbrake errnotifier.AirbrakeConfig
}
```

And set appropriate values for this struct as part of your application
configuration.

Create an Airbrake error notifier:

```
if notifier, err := errornotifier.NewAirbrakeNotifier(appConfig.Airbrake); err != nil {
	// handle the error
}
```

Then use the returned `Notifier` as described below.

## Logging Notifier Usage

Use `NewLogNotifier` to construct your logging notifier:

```
notifier := errornotifier.NewLogNotifier(logger)
```

Then use the returned `Notifier` as described below.

## Buffering Notifier Usage

Create an `errornotifier/utils.BufferNotifier`

```
notifier := utils.BufferNotifier{}
```

Any calls to `errornotifier.Notifier.Notify` will be stored in
`notifier.Notices`. Use it as a `Notifier` as described below.

## Failing Notifier Usage

Inside a Go test, create a `errornotifier/utils.TestNotifier` with the
received `*testing.T`:

```
func TestStuff(t *testing.T) {
    notifier := utils.TestNotifier{t}
}
```

Any calls to `notifier.Notify` will trigger a test failure via
`testing.T.Fatal`. Use it as a `Notifier` as described below.

# Usage

Send error notifications via `errornotifier.Notifier.Notify`:

```
func handler(w http.ResponseWriter, r *http.Request) {
	var notifier errornotifier.Notifier = // ...

	err := doWork()
	if err != nil {
		notifier.Notify(err, r)
	}
}
```

## Middleware/Contextual usage

The package provides a middleware that sends any `panic`s in
downstream HTTP handlers to the notifier. Wrap your existing handler
using the middleware:

```
var notifier errornotifier.Notifier = // ...

notifierMiddleware := errornotifier.Recover(notifier)

return notifierMiddleware(handler)
```

This middleware will also make the notifier available to the HTTP
handler via the request's context, using `errornotifier.ForceContext`:

```
func handler(w http.ResponseWriter, r *http.Request) {
	notifier := errornotifier.ForceContext(r.Context())

    notifier.Notify(..., r)
}
```

This will always return a valid notifier:

1. Return notifier from request context, if any.
2. If there is no notifier in the context, return a logging notifier
   using `log.ForceContext`. This means that the logging notifier will
   use the requests logger, if present. Otherwise falling back to
   `log.Default()`.

## NotifyOnPanic

`errornotifier.NotifyOnPanic` is used to make using goroutines in HTTP
handlers a bit safer. It:

1. Executes the passed function.
2. Recovers a `panic` if necessary.
3. Notifies the passed notifier if a `panic` occurred.
4. Returns `nil` on no panic, or the `error` from `recover`.

Goroutines by default are not `recover`ed, so for example this will
*crash* a HTTP server (meaning *program exit*, not just aborting the
handler for a specific request), as the `panic` reaches to the top
level of the goroutine's stack without being handled.

```
func handler(w http.ResponseWriter, r *http.Request) {
    go func() {
        panic(errors.New("error"))
    }
}
```

to avoid this, and spawn goroutines without fear that an unhandled
`panic` brings down the whole webserver, wrap the goroutine:

```
func handler(w http.ResponseWriter, r *http.Request) {
    go errornotifier.NotifyOnPanic(
        errornotifier.ForceContext(r.Context()),
        r,
        func() {
            panic(errors.New("error"))
        },
    )
}
```
