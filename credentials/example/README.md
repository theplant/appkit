# Example System Configuration

## Overview

(assuming a k8s cluster exists, and `kubectl` works):

1. Deploy Vault onto the cluster from `vault.yaml`.

2. Use Terraform to

   * create an AWS IAM user for Vault's AWS backend and configure
     Vault to use AWS for secrets and K8s for authn,

   * create Vault k8s role, AWS secret and policy.

4. Run app on k8s cluster that:

   1. uses k8s service account token to
      authenticate with Vault,

   2. requests AWS credentials from Vault.

## Pre-requisites

* Make sure `kubectl` can access a *non-production* cluster.

* Terraform is installed.

## Running it

```
make test
```

