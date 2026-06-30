variable "project_name" {
  type        = string
  description = "Project name"
}

variable "environment" {
  type        = string
  description = "Environment (dev/staging/prod)"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID"
}

variable "private_eks_subnet_ids" {
  type        = list(string)
  description = "Private subnet IDs for EKS nodes"
}

variable "cluster_version" {
  type        = string
  description = "EKS Kubernetes version"
  default     = "1.35"
}

variable "enable_public_endpoint" {
  type        = bool
  description = "Enable public API server endpoint"
  default     = true
}

variable "public_access_cidrs" {
  type        = list(string)
  description = "CIDRs allowed to access public endpoint"
  default     = ["0.0.0.0/0"]
}

variable "system_node_instance_types" {
  type        = list(string)
  description = "Instance types for system node group"
  default     = ["t3.medium"]
}

variable "system_node_desired" {
  type        = number
  description = "Desired count for system node group"
  default     = 2
}
