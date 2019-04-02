output "k8s-service-account" {
  value = "${var.k8s-namespace}:${var.name}"

  description = "k8s service account that should be used by apps/services to access this role's credentials (`spec.serviceAccountName`)"
}

output "vault-authn-path" {
  value = "auth/${var.vault-k8s-backend}login"

  description = "Vault path to use for initial authentication"
}
output "vault-authn-role" {
  value = "${vault_kubernetes_auth_backend_role.kubernetes.role_name}"

  description = "Vault role to use for initial authentication"
}

output "vault-aws-path" {
  value = "${var.vault-aws-backend}/sts/${vault_aws_secret_backend_role.aws-role.name}"

  description = "Vault path to use to fetch AWS credentials"
}
