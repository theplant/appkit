# Vault-based K8s/AWS Authn/z

Creates resources for apps running on k8s to use their service account
to authenticate with Vault, and fetch AWS IAM credentials from Vault.

## Requirements

1. Vault with pre-configured:

   * AWS secret backend

   * K8s auth backend

2. AWS IAM Role to assume

## Usage

Use module outputs to populate app configuration:

* Vault Authn path (`VAULT_AUTHN_PATH`)

* Vault Authn role (`VAULT_AUTHN_ROLE`)

* Vault AWS secret path (`VAULT_AWSPATH`)
