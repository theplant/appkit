provider "vault" {}
provider "kubernetes" {
  config_context = "docker-for-desktop"
}

locals {
  ns-name = "${var.k8s-namespace}.${var.name}"
}

resource "kubernetes_service_account" "serviceaccount" {
  metadata {
    namespace = "${var.k8s-namespace}"
    name = "${var.name}"
  }

  automount_service_account_token = true
}

resource "vault_kubernetes_auth_backend_role" "kubernetes" {
  backend                          = "${var.vault-k8s-backend}"
  role_name                        = "${local.ns-name}"
  bound_service_account_names      = ["${var.name}"]
  bound_service_account_namespaces = ["${var.k8s-namespace}"]
  policies                         = ["${local.ns-name}", "default"]
}

resource "vault_aws_secret_backend_role" "aws-role" {
  backend = "${var.vault-aws-backend}"
  name    = "${local.ns-name}"
  
  # Not supported yet by Vault Terraform backend
  # credential_type = "assumed_role"

  policy_arn = "${var.aws-role-arn}"

  # Vault automatically moves policy_arn to role_arn but Terraform
  # gets confused with the state change...
  lifecycle {
    ignore_changes = ["policy_arn"]
  }
}

resource "vault_policy" "k8s" {
  name = "${local.ns-name}"

  # vault_aws_secret_backend_role has a bug, it doesn't support
  # role_arns:
  # 
  # https://www.vaultproject.io/api/secret/aws/index.html#role_arns
  #
  # so we need to use `aws/sts/...` instead of `aws/creds/...` to
  # generate role credentials.
  
  policy = <<EOT
path "${var.vault-aws-backend}/sts/${local.ns-name}" {
  capabilities = ["read"]
}
EOT
}
