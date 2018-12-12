# Credentials

Package to provide single interface for acquiring AWS (-only, for now)
credentials for apps running on different platforms:

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
