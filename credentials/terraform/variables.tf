variable "k8s-namespace" {
  description = "Created Vault role will only be accessible to apps in this namespace. Also included in Vault role names."
}

variable "name" {
  description = "Name of application/service. Used for k8s service account and included in Vault role names."
}

variable "aws-role-arn" {
  description = "AWS IAM role that will be assumed when requesting AWS credentials for this application/service"
}

variable "vault-k8s-backend" {
  default = "kubernetes"
  description = "configured path of Vault kubernetes auth backend"
}

variable "vault-aws-backend" {
  default = "aws"
  description = "configured path of Vault AWS secret backend"
}
