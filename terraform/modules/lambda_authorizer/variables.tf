variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "jwt_secret" {
  type        = string
  sensitive   = true
  description = "JWT signing secret"
}

variable "reserved_concurrent_executions" {
  type        = number
  default     = -1
  description = "Reserved concurrency for the Lambda function. -1 means unreserved (no limit)."
}
