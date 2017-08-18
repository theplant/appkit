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

