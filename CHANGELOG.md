# PR [#35](https://github.com/theplant/appkit/pull/35)

## Added

* Add Package appkit/kerrs
* Add `log.WithError` to log to be able to log appkit errors with ease

# PR [#33](https://github.com/theplant/appkit/pull/33)

## Added

* [`log.NewNopLogger` function](https://github.com/theplant/appkit/blob/08b478e/log/log.go#L74-L78)
* [`log.Context` function](https://github.com/theplant/appkit/blob/08b478e/log/context.go#L29-L32)

## Fixed

* `log.FromContext` panic when receiving a nil context bug

# PR [#32](https://github.com/theplant/appkit/pull/32)

## Added

* [`errornotifier` package](errornotifier/README.md)

# PR [#30](https://github.com/theplant/appkit/pull/30)

## Added

* `monitoring` package


# PR [#29](https://github.com/theplant/appkit/pull/29) Context cleanup

## Breaking changes

* Move Gorm/DB context functions from `contexts` to `appkit/db`
* Move logging context functions from `contexts` to `appkit/log`
* Move tracing context functions from `contexts` to `contexts/trace`

