# Appkit monitoring

This package provides a basic interface for monitoring request times
and other arbitrary data, and recording data into InfluxDB.

# Configuration for InfluxDB

Include a field of type `monitoring.InfluxMonitorConfig` in your app config:

```
type AppConfig struct {
	// ...

	InfluxDB monitor.InfluxMonitorConfig
}
```

And configure this field with a value like: `https://<username>:<password>@<influxDB host>/<database>`.

Create an InfluxDB monitor in `main.go`:

```
if monitor, err := monitoring.NewInfluxdbMonitor(appConfig.InfluxDB, l); err != nil {
	// handle the error...
	l.Info().Log(
		"err", err,
		"msg", fmt.Sprintf("error configuring influxdb monitor: %v", err),
	)
}
```

# Middleware

Use included middleware in your stack:

```
func Routes(monitor monitoring.Monitor, ...) http.Handler {
    // ...

	middleware := server.Compose(
		// ... other middleware
    	monitoring.WithMonitor(monitor),
    	// ... more middleware
    )

    // ...
}
```

Now request data including request path, method, HTTP response status code, request duration, and request trace ID, will be sent to your InfluxDB instance in the `request` measurement.

## Path scrubbing

The middleware will convert any sequences of 1 or more digits into `:id`. Eg. `GET /api/users/123/comments/456` will be tagged with path of `/api/users/:id/comments/:id`.

## Turning off the `request` measurement

`monitoring.WithMonitorContextOnly` installs the Monitor in the request context
without recording the `request` measurement. Handlers that record their own
metrics keep working; only the per-request record goes away.

Use it when the Monitor is a log monitor. There the `request` measurement is one
log line per request, which on a busy service can be most of the log volume
while carrying data that request tracing already has.

Services built on `appkit/service` get this through an environment variable:
set `MONITORING_DISABLE_REQUEST_METRIC=true`.

# Recording other metrics

To record other metrics, eg counting subscriptions, measuring time of API calls to other services, retrieve the metric from the context with `monitoring.ForceContext`, and then call methods on the interface:

```
type Monitor interface {
	InsertRecord(measurement string, value interface{}, tags map[string]string, fields map[string]interface{}, time time.Time)
	Count(measurement string, value float64, tags map[string]string, fields map[string]interface{})
	CountError(measurement string, value float64, err error)
	CountSimple(measurement string, value float64)
}
```


For example, to count logins:

```
func Login(w http.ResponseWriter, r *http.Request) {
    monitor := monitoring.ForceContext(r.Context())
    if err := doLogin(); err == nil {
        // ... handle success case
        monitor.CountSimple("login_success", 1)
    } else {
        monitor.CountError("login_error", 1, err)
    }
}
```

# TODO

* Make metrics counted via the context monitor include request tags eg. request ID

* Provide simple wrapper API for context-based metrics:

  ```
  func InsertRecord(context.Context, string, interface{}, map[string]string, time.Time)
  func Count(context.Context, measurement string, value float64, tags map[string]string)
  ```

* Make path-scrubbing more flexible. Eg. scrub out product names: `/products/blue-winter-coat` -> `/products/:product_code`.
