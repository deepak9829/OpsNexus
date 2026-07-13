variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
}

variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
}

variable "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  type        = string
}

variable "cluster_ca_certificate" {
  description = "Base64-encoded cluster CA certificate"
  type        = string
  sensitive   = true
}

variable "karpenter_role_arn" {
  description = "IAM role ARN for the Karpenter controller (created by EKS module)"
  type        = string
}

variable "karpenter_interruption_queue_name" {
  description = "SQS queue name for Karpenter spot interruption handling"
  type        = string
}

variable "node_role_name" {
  description = "IAM role name attached to Karpenter-provisioned nodes"
  type        = string
}

variable "karpenter_version" {
  description = "Karpenter Helm chart version (e.g. 1.13.0)"
  type        = string
  default     = "1.13.0"
}

variable "node_instance_families" {
  description = "EC2 instance families allowed for Karpenter nodes"
  type        = list(string)
  default     = ["m", "c", "r"]
}

variable "node_instance_sizes" {
  description = "EC2 instance sizes allowed for Karpenter nodes"
  type        = list(string)
  default     = ["large", "xlarge", "2xlarge"]
}

variable "node_capacity_types" {
  description = "Capacity types for Karpenter nodes (on-demand, spot)"
  type        = list(string)
  default     = ["on-demand", "spot"]
}

variable "node_arch" {
  description = "CPU architectures for Karpenter nodes"
  type        = list(string)
  default     = ["amd64"]
}
