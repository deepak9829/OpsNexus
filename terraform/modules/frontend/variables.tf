variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "apps" {
  type = map(object({
    domain_aliases = list(string)
  }))
  description = "Map of app_name => {domain_aliases}. Each key becomes an S3 folder and CloudFront origin_path."
}

variable "acm_certificate_arn" {
  type        = string
  default     = ""
  description = "ACM cert ARN (us-east-1). Empty = use CloudFront default cert."
}
