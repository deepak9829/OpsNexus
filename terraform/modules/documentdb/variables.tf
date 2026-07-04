variable "project_name" { type = string }
variable "environment" { type = string }
variable "vpc_id" { type = string }
variable "private_db_subnet_ids" { type = list(string) }
variable "eks_node_sg_id" { type = string }

variable "engine_version" {
  type    = string
  default = "5.0.0"
}

variable "instance_class" {
  type    = string
  default = "db.t3.medium"
}

variable "instance_count" {
  type    = number
  default = 1
}

variable "deletion_protection" {
  type    = bool
  default = false
}

variable "skip_final_snapshot" {
  type    = bool
  default = true
}

variable "backup_retention_period" {
  type    = number
  default = 7
}
