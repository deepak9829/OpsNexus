variable "project_name" {
  description = "Name of the project, used as a prefix in resource naming."
  type        = string
}

variable "environment" {
  description = "Deployment environment (e.g. dev, staging, prod)."
  type        = string
}

variable "service_names" {
  description = "List of service names for which ECR repositories will be created."
  type        = list(string)
  default = [
    "auth-service",
    "tenant-service",
    "workflow-service",
    "document-service",
    "notification-service",
  ]
}
