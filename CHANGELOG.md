# PR [#43](https://github.com/theplant/appkit/pull/43)

* Allow configure `Access-Control-Allow-Headers` via
  `CORS_RawAllowedHeaders` for CORS middleware in appkit/service
  package.

# PR [#39](https://github.com/theplant/appkit/pull/39)

* Add monitoring InfluxDB client that sources InfluxDB credentials
  from Vault.

* Allow creation of InfluxDB-backed monitoring client with custom
  InfluxDB transport client.

* Introduce public Vault client interface (instead of using
  `github.com/hashicorp/vault/api.Client` directly)

* Make Vault client available from service context

* Allow subscription to Vault (re-)authentication via
  `appkit/credentials/vault.Client.OnAuth`

* Revoke Vault auth lease when service context is closed

# PR [#38](https://github.com/theplant/appkit/pull/38)

* Add support for `VAULT_AUTHN_DISABLED` to stop appkit/service
  automatically attempting to authenticate with Vault.

# PR [#37](https://github.com/theplant/appkit/pull/37)

* Add package appkit/service

# PR [#19](https://github.com/theplant/appkit/pull/19)

* Add Package appkit/credentials

# PR [#36](https://github.com/theplant/appkit/pull/36)

## Breaking Changes

* `errornotifier.NewAirbrakeNotifier` func now returns `io.Closer` to
  wait send data to Aribrake.
* Add a new parameter `context` to `errornotifier.Notifier.Notify` allow
  add context when call `Notify`.

## Changes

* Upgrade Airbrake package.
* Change `airbrakeNotifier.Notify` func to async.
* Add log req_id and span_context to `errornotifier.NotifyOnPanic` and
  `Recover` middleware.

# PR [#33](https://github.com/theplant/appkit/pull/33)

* `monitoring.WithMonitor` middleware now collects `X-Via`
  request header value as `via` tag value.

# PR [#31](https://github.com/theplant/appkit/pull/31)

* Log request id in trace log.
* Log trace id in InfluxDB.
* Add `service-name` config for InfluxDB.

# PR [#30](https://github.com/theplant/appkit/pull/30)

## Breaking Changes

* `monitoring.NewInfluxdbMonitor` now returns a "close" `func()` to
  allow termination of InfluxDB buffer introduced in #25.

## Added

* `server.GoListenAndServe` gives control over shut-down of the HTTP
  server.


## Changed Behaviour

* `server.ListenAndServe` now delegates the HTTP server to this
  function and `server.GoListenAndServe` and *manually* blocks forever
  by deadlocking itself. This preserves the basic behaviour, but it
  might change how `ListenAndServe` responded to OS signals.

* Tracing on monitoring middleware was removed. Tracing now occurs on
  the InfluxDB monitor buffer goroutine that sends batches of points
  to InfluxDB.

# PR [#25](https://github.com/theplant/appkit/pull/25)

* Add batch write (buffer) for monitoring/influxdb.

# PR [#20](https://github.com/theplant/appkit/pull/20)

* Add `tracing` package

* Make `errornotifier` and `monitoring` middleware trace API requests

# PR [#27](https://github.com/theplant/appkit/pull/27)

* Change influxdb client import path. Detail in [issue](https://github.com/influxdata/influxdb/issues/11035)

# PR [#17](https://github.com/theplant/appkit/pull/17)

* Don't log SQL query *values* in Gorm log adapter

# PR [#16](https://github.com/theplant/appkit/pull/16)

* Replace the `sessions.Config` with `sessions.CookieStoreConfig`, the `sessions.CookieStoreConfig` is compatible with `jinzhu/configor`, change the struct to enable `Secure` and `HttpOnly` by default and extend the struct to make it general.

* Add `sessions.NewCookieStore` to easy new a `gorilla/sessions.CookieStore` with `sessions.Config`.

# PR [#14](https://github.com/theplant/appkit/pull/14)

* Add `server.SecureMiddleware` for CSRF/CORS API configuration.

* Add `log.NewTestLogger` for stable log output when testing (ie. no
  timestamps or stack references)

# PR [#12](https://github.com/theplant/appkit/pull/12)

* Upgrade [`influxdb.Client`](https://github.com/influxdata/influxdb/tree/master/client#description) to `v2` version to fix the monitoring/influxdb monitor broken [issue](https://github.com/theplant/appkit/issues/11) due to the old version of `influxdb.Client` being deprecated.

# PR [#9](https://github.com/theplant/appkit/pull/9)

* Fix `contexts.HTTPStatus` to assume that the response is `http.StatusOK`, rather than `0`.

# PR [#6](https://github.com/theplant/appkit/pull/6)

* Add `log.WrapError` to `log`

# PR [#2](https://github.com/theplant/appkit/pull/2)

* Add `encryptedbox` package

# PR [#43](https://github.com/theplant/appkit-private/pull/43)

* Add `APPKIT_LOG_HUMAN` environment flag to output logs with better formatted for human


# PR [#42](https://github.com/theplant/appkit-private/pull/42)

* Add Package appkit/sessions

# [Add support for logging fields via monitoring.Monitor (PR#37)](https://github.com/theplant/appkit-private/pull/37)

## Breaking changes

* [Changed monitoring interface](https://github.com/theplant/appkit-private/pull/37/commits/5dde9fa2bc77527f9760feecdda762864fa0572c#diff-2b8f43b8889cf5d451e1b7e74a89ae74L62) to accept data fields in addition to tags.

## Changed behaviour

* Monitoring middleware now logs `req_id` as a field, rather than a
tag, to avoid generating InfluxDB series with immense
[cardinality](https://docs.influxdata.com/influxdb/v1.2/concepts/glossary/#series-cardinality).

# PR [#35](https://github.com/theplant/appkit-private/pull/35)

## Added

* Add Package appkit/kerrs
* Add `log.WithError` to log to be able to log appkit errors with ease

# PR [#33](https://github.com/theplant/appkit-private/pull/33)

## Added

* [`log.NewNopLogger` function](https://github.com/theplant/appkit/blob/08b478e/log/log.go#L74-L78)
* [`log.Context` function](https://github.com/theplant/appkit/blob/08b478e/log/context.go#L29-L32)

## Fixed

* `log.FromContext` panic when receiving a nil context bug

# PR [#32](https://github.com/theplant/appkit-private/pull/32)

## Added

* [`errornotifier` package](errornotifier/README.md)

# PR [#30](https://github.com/theplant/appkit-private/pull/30)

## Added

* `monitoring` package


# PR [#29](https://github.com/theplant/appkit-private/pull/29) Context cleanup

## Breaking changes

* Move Gorm/DB context functions from `contexts` to `appkit/db`
* Move logging context functions from `contexts` to `appkit/log`
* Move tracing context functions from `contexts` to `contexts/trace`

