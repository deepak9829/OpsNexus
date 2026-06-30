variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "cluster_endpoint" {
  type = string
}

variable "cluster_ca_certificate" {
  type      = string
  sensitive = true
}

variable "karpenter_role_arn" {
  type = string
}

variable "karpenter_interruption_queue_name" {
  type = string
}

variable "node_role_name" {
  type = string
}

variable "karpenter_version" {
  type    = string
  default = "1.13.0"
}
