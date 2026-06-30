variable "project_name" {
  type        = string
  description = "Project name used for resource naming"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID for security group placement"
}

variable "private_db_subnet_ids" {
  type        = list(string)
  description = "Private DB subnet IDs for RDS subnet group"
}

variable "eks_node_sg_id" {
  type        = string
  description = "EKS node security group ID allowed to connect to RDS"
}

variable "db_instance_class" {
  type        = string
  description = "RDS instance class"
  default     = "db.t3.small"
}

variable "db_allocated_storage" {
  type        = number
  description = "Initial storage allocation in GB"
  default     = 20
}

variable "db_max_allocated_storage" {
  type        = number
  description = "Maximum autoscaled storage in GB"
  default     = 100
}

variable "multi_az" {
  type        = bool
  description = "Enable Multi-AZ deployment"
  default     = false
}

variable "backup_retention_period" {
  type        = number
  description = "Days to retain automated backups (0 to disable)"
  default     = 7
}

variable "deletion_protection" {
  type        = bool
  description = "Enable deletion protection"
  default     = false
}

variable "skip_final_snapshot" {
  type        = bool
  description = "Skip final snapshot on deletion"
  default     = true
}

variable "enable_performance_insights" {
  type        = bool
  description = "Enable Performance Insights"
  default     = false
}
