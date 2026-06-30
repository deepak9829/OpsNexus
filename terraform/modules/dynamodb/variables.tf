variable "project_name" {
  description = "Name of the project, used as a prefix in resource naming."
  type        = string
}

variable "environment" {
  description = "Deployment environment (e.g. dev, staging, prod)."
  type        = string
}
