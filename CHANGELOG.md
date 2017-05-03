# PR [#33](https://github.com/theplant/appkit/pull/32)

## Added

* [`log.NewNopLogger` function](https://github.com/theplant/appkit/blob/add-nop-logger/log/log.go#L60-L64)
* [`log.Context` function](https://github.com/theplant/appkit/blob/add-nop-logger/log/context.go#L29-L32)

## Fixed

* `log.FromContext` panic when receiving a nil context bug

# PR [#32](https://github.com/theplant/appkit/pull/32)

## Added

* [`errornotifier` package](errornotifier/README.md)

# PR [#30](https://github.com/theplant/appkit/pull/30)

## Added

* `monitoring` package


# Context cleanup (PR #29)

## Breaking changes

* Move Gorm/DB context functions from `contexts` to `appkit/db`
* Move logging context functions from `contexts` to `appkit/log`
* Move tracing context functions from `contexts` to `contexts/trace`

# Add Appkit Errors (PR #35)

* Add Package `appkit/kerrs`
* Add `WithError` to log to be able to log appkit errors with ease
