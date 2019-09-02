# Credentials

Package to provide single interface for acquiring credentials for apps
running on different platforms.

# AWS

* Use k8s service-account-based authentication with Vault when running
  on k8s. This will automatically renew Vault and AWS credetials as
  they get close to expiring.

* Otherwise use AWS session management

## Local Development

Set `AWS_SDK_LOAD_CONFIG=1` and `AWS_PROFILE=<a profile that works
with aws --profile name sts get-caller-identity>`

This will delegate authentication to your AWS configuration.

## K8s+Vault Provider

Provider will auth against Vault using the pod's [*Kubernetes Service
Account
Token*](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/):

1. Check if there's a token at
   `/var/run/secrets/kubernetes.io/serviceaccount/token`.

2. Fetch service account token from filesystem.

3. Send token to Vault `kubernetes` auth method to acquire Vault authn
   token.

4. Use Vault authn token to request AWS credentials from Vault.

# Example usage

```
var config credentials.Config

err := configor.New(&configor.Config{ENVPrefix: "VAULT"}).Load(&config)
if err != nil {
	// ...
}

vault, err := vault.NewVaultClient(logger, config.Authn)
if err != nil {
	// ...
}

session, err := aws.NewSession(logger, vault, config.AWSPath)
if err != nil {
	// ...
}

svc := sts.New(session)
result, err := svc.GetCallerIdentity(nil)
// ...
```

# InfluxDB

A [`monitoring`](../monitoring/README.md) client that uses an internal
InfluxDB client with credentials sourced from Vault.

The client handles Vault credential expiry (to be more precise, *renewal*):

1. When the client is initially created with `NewInfluxDBMonitor`, it
   starts with a "null" internal client that responds to all requests
   with an error (because we have no Vault credentials yet, we can't
   fetch any InfluxDB credentials).

2. When the passed Vault client authenticates (signalled via
   `appkit/credentials/vault.Client.OnAuth`), the monitoring client
   fetches new InfluxDB credentials and swaps out the internal
   InfluxDB HTTP client for a new one with the updated credentials.

3. Whenever the Vault client-reauthenticates in the future, step 2 is
   repeated.

Assumptions and constraints:

* `https` is hardcoded, the scheme of URL passed to
  `NewInfluxDBMonitor` is ignored.

* The path of the URL is used as the database name. All query
  attributes used by the InfluxDB monitor are respected.

* Vault has a database role for InfluxDB credentials at
  `database/creds/<name-from-path>-influxdb`,
  (eg. `vault://influxdb.example.com/the-database` =>
  `database/creds/the-database-influxdb`).

# Terraform Module

A Terraform module is provided to create the necessary resources for
the system to work when run in a k8s cluster.

# Example

the `example/` directory contains configuration and code for an
example app that runs in a pre-configured k8s cluster.

# Todo

- [ ] Extract Vault initial configuration from example app Terraform
      config into separate module.

- [ ] Support Postgres/Database/other types of expiring credentials.

- [ ] Document how to use Vault kv secrets for non-expiring credentials.
