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

variable "vpc_cidr" { type = string }
variable "availability_zones" { type = list(string) }
variable "public_subnet_cidrs" { type = list(string) }
variable "private_eks_subnet_cidrs" { type = list(string) }
variable "private_db_subnet_cidrs" { type = list(string) }

variable "db_instance_class" {
  type    = string
  default = "db.t3.large"
}

variable "db_allocated_storage" {
  type    = number
  default = 100
}

variable "db_max_allocated_storage" {
  type    = number
  default = 500
}

variable "multi_az" {
  type    = bool
  default = true
}

variable "deletion_protection" {
  type    = bool
  default = true
}

variable "skip_final_snapshot" {
  type    = bool
  default = false
}

variable "enable_performance_insights" {
  type    = bool
  default = true
}

variable "backup_retention_period" {
  type    = number
  default = 30
}

variable "cluster_version" {
  type    = string
  default = "1.30"
}

variable "system_node_desired" {
  type    = number
  default = 3
}

variable "system_node_instance_types" {
  type    = list(string)
  default = ["t3.medium"]
}

variable "enable_public_endpoint" {
  type    = bool
  default = false
}

variable "public_access_cidrs" {
  type    = list(string)
  default = []
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
  default     = true
  description = "true for prod HA, false for dev/staging cost savings"
}

variable "tf_state_bucket" {
  type        = string
  description = "S3 bucket name for Terraform state"
}

variable "tf_state_key" {
  type        = string
  description = "S3 key path for this environment's state file"
}

variable "tf_state_region" {
  type        = string
  description = "AWS region where the state bucket lives"
}
