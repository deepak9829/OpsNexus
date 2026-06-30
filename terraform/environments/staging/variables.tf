variable "aws_region" {
  type    = string
  default = "ap-southeast-1"
}

variable "environment" {
  type = string
}

variable "project_name" {
  type    = string
  default = "opsnexus"
}

variable "vpc_cidr"                 { type = string }
variable "availability_zones"       { type = list(string) }
variable "public_subnet_cidrs"      { type = list(string) }
variable "private_eks_subnet_cidrs" { type = list(string) }
variable "private_db_subnet_cidrs"  { type = list(string) }

variable "db_instance_class" {
  type    = string
  default = "db.t3.small"
}

variable "db_allocated_storage" {
  type    = number
  default = 20
}

variable "db_max_allocated_storage" {
  type    = number
  default = 100
}

variable "multi_az" {
  type    = bool
  default = false
}

variable "deletion_protection" {
  type    = bool
  default = false
}

variable "skip_final_snapshot" {
  type    = bool
  default = true
}

variable "enable_performance_insights" {
  type    = bool
  default = false
}

variable "backup_retention_period" {
  type    = number
  default = 7
}

variable "cluster_version" {
  type    = string
  default = "1.30"
}

variable "system_node_desired" {
  type    = number
  default = 2
}

variable "system_node_instance_types" {
  type    = list(string)
  default = ["t3.medium"]
}

variable "enable_public_endpoint" {
  type    = bool
  default = true
}

variable "public_access_cidrs" {
  type    = list(string)
  default = ["0.0.0.0/0"]
}

variable "karpenter_version" {
  type    = string
  default = "1.0.0"
}

variable "domain_name" {
  type    = string
  default = "opsnexus.site"
}

variable "nlb_arn" {
  type        = string
  default     = ""
  description = "NLB ARN from Traefik install. Empty until Traefik deployed."
}

variable "jwt_secret" {
  type        = string
  sensitive   = true
  description = "JWT signing secret. Pass via TF_VAR_jwt_secret or CI/CD - never in tfvars."
}

variable "caylent_owner" {
  type    = string
  default = "deepak.saini@caylent.com"
}

variable "caylent_customer" {
  type    = string
  default = ""
}

variable "caylent_workload" {
  type    = string
  default = ""
}

variable "caylent_project" {
  type    = string
  default = ""
}

variable "enable_nat_per_az" {
  type        = bool
  default     = false
  description = "true for prod HA, false for dev/staging cost savings"
}
