provider vault {}

provider aws {
  region = "us-west-1"
}

variable "kubernetes_ca_crt" {
  default = ""
}

variable "kubernetes_namespace" {}

resource "aws_iam_user" "vault" {
  name = "vault"

  tags = {
    environment = "test"
    client      = "theplant"
    project     = "appkit/credentials"
  }
}

resource "aws_iam_access_key" "vault" {
  user = "${aws_iam_user.vault.name}"
}

resource "aws_iam_role" "test" {
  name = "test"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "${aws_iam_user.vault.arn}"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
POLICY

  tags = {
    client      = "theplant"
    environment = "test"
    project     = "appkit/credentials"
  }
}

resource "vault_auth_backend" "kubernetes" {
  type = "kubernetes"
}

resource "vault_kubernetes_auth_backend_config" "kubernetes" {
  backend            = "${vault_auth_backend.kubernetes.path}"
  kubernetes_host    = "https://kubernetes.default.svc"
  kubernetes_ca_cert = "${var.kubernetes_ca_crt}"
}

resource "vault_aws_secret_backend" "aws" {
  access_key = "${aws_iam_access_key.vault.id}"
  secret_key = "${aws_iam_access_key.vault.secret}"
}

module "app-role" {
  source = "../terraform"

  k8s-namespace = "${var.kubernetes_namespace}"
  name          = "app"

  vault-k8s-backend = "${vault_auth_backend.kubernetes.path}"
  vault-aws-backend = "${vault_aws_secret_backend.aws.path}"

  aws-role-arn = "${aws_iam_role.test.arn}"
}

output "aws-role-arn" {
  value = "${aws_iam_role.test.arn}"
}

output "k8s-service-account" {
  value = "${module.app-role.k8s-service-account}"
}

output "vault-authn-path" {
  value = "${module.app-role.vault-authn-path}"
}

output "vault-authn-role" {
  value = "${module.app-role.vault-authn-role}"
}

output "vault-aws-path" {
  value = "${module.app-role.vault-aws-path}"
}
