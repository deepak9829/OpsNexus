variable "project_name" { type = string }
variable "environment" { type = string }
variable "oidc_provider_arn" { type = string }
variable "oidc_provider_url" { type = string }
variable "eso_version" {
  type    = string
  default = "0.14.0"
}
