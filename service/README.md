This package provides a common harness for running HTTP apps, that
configures middleware that we use nearly all the time, and provides a
standard way to configure the different parts of the service.

Basic usage is to call `service.ListenAndServe`, and pass a function
that registers your HTTP handlers on the `*http.ServeMux` parameter
passed to your function.

There are some pre-configured common things available in the background context
passed to the function:

* [Logger](../log)

* [Monitor](../monitoring)

* [Error Notifier](../errornotifier)

* [Vault Client](../credentials/vault)

* [AWS Session](../credentials/aws)

Most of these are also made available via middleware, and *should be
accessed via the HTTP request context instead*.

## Middleware

In request-processing order:

1. `appkit/server` default middleware:

   1. HTTP status memoisation
   2. Taggin request with UUID
   3. Adding logger to request context
   4. Logging request/response
   5. `recover`ing from `panic`, and returning `500 Internal Server
      Error` if no other HTTP status has been "sent" from a later
      handler.

2. Request tracing via Opentracing/Jaeger
3. Notification of `panic`s to Airbrake
4. Logging request metrics in InfluxDB
5. Sending request information to New Relic
6. CORS handling
7. HTTP Basic Authentication
8. Adding AWS session to request context

# Configuration

## General Configuration

`SERVICE_NAME` can be set in the environment, this will be used by:

* Loggger: as `svc` field.

* Vault+AWS client credentials sourcing: as default role for Vault
  authn, and default AWS credentials path.

* Request Tracing (Opentracing/Jaeger): As service name for traced
  requests.

* New Relic Middleware: Application name reported to New Relic.

## HTTP Basic Authentication

Environment variables:

* `BASICAUTH_Username`
* `BASICAUTH_Password`
* `BASICAUTH_UserAgentWhitelistRegexp`: Regexp matched against HTTP
  User-Agent header to bypass HTTP Basic Authentication.
* `BASICAUTH_PathWhitelistRegexp`: Regexp matched against request path
  to bypass HTTP Basic Authentication.

## CORS

[CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
configuration of https://github.com/rs/cors

Environment variables:

* `CORS_RawAllowedOrigins`: comma-separated list of allowed values for
  [`Access-Control-Allow-Origin`](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#Access-Control-Allow-Origin)
  HTTP header.

* `CORS_AllowCredentials`: boolean-ish value for
  [`Access-Control-Allow-Credentials`](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#Access-Control-Allow-Credentials)
  HTTP header.

## Vault/AWS Session
    
Environment variables:
    
* `VAULT_AUTHN_ADDRESS`: base URL of Vault server.

* `VAULT_AUTH_PATH`: path to Vault K8s auth backend. By default this
  is set to `auth/kubernetes/login`.

* `VAULT_AUTH_ROLE`: role used when authenticating with Vault. By
default this is set to `$SERVICE_NAME`.

* `VAULT_AUTH_AUTORENEW`: flag to enable/disable automatic renewal of
  Vault lease in background goroutine. Defaults to `true`.

* `VAULT_AWSPATH`: When using AWS credentials sourced from Vault, the
Vault path of the credentials secret. By default this is set to
`aws/sts/<$SERVICE_NAME>`.

* `VAULT_AUTHN_DISABLED`: flag to enable/disable Vault authentication
  completely. Use this when the service does not need any access to
  Vault.

## HTTP Server

`PORT`: port number for HTTP server to bind to, defaults to `9800`. If
required, `ADDR` can be set instead to allow binding to a specific
interface/IP address using `[interface]:port` syntax.

## Monitor

`INFLUXDB_URL`: Set to URL of InfluxDB server. If blank, a logging
(via `appkit/log` monitor will be used instead of sending data to
InfluxDB.

If `INFLUXDB_URL`'s scheme is `vault` (vs `http` or `https`), then the
client will source InfluxDB credentials from Vault. In this case,
`INFLUXDB_URL` does not need any credentials. See the Vault+InfluxDB
client documentation in the [`credentials`
README](../credentials/README.md) for information about configuration
constraints.

If `INFLUXDB_URL` has no (or blank) `service-name` query parameter,
the parameter will be set to `SERVICE_NAME`.

## Error Notifier

* `AIRBRAKE_PROJECTID`
* `AIRBRAKE_TOKEN`
* `AIRBRAKE_ENVIRONMENT`

If notifier can't be created due to blank project ID or token, a
logging notifier will be used instead.

## New Relic

* `NEWRELIC_APIKey`
* `NEWRELIC_AppName`: If this is blank, `$SERVICE_NAME` will be used.

## Tracer

Configured as for [Jaeger Go
client](https://github.com/jaegertracing/jaeger-client-go). 

If `JAEGER_SERVICE_NAME` is blank, `$SERVICE_NAME` will be used.


# Design Background

Design is implemented via a configuration callback because the
lifecycle of the app is controlled by the serivce harness.

1. Create and configure service background context (including starting
   background goroutines used by some middleware).

2. Set up common middleware using background context.

3. Hand over to app-side code (passed function) to do app-specific
   configuration.

4. App-side code hands back to the service harness (by returning from
   the passed function).

5. Start HTTP server in background goroutine.

6. Wait for signal to terminate.

7. Ask HTTP server to shut down, and wait.

8. Once HTTP server has shut down (=> all requests have been handled
   or rejected, no more HTTP requests will be accepted), shut down
   background context goroutines.

9. Exit

The reason for this structure is that step 7 needs to finish *before*
step 8 is started, and managing this lifecycle is somewhat complex,
and tedious. So we don't want to implement it in *every* service.
