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
  description = "VPC ID for NLB target group"
}

variable "private_eks_subnet_ids" {
  type        = list(string)
  description = "Private EKS subnet IDs for NLB placement"
}

variable "lambda_invoke_arn" {
  type        = string
  description = "Lambda authorizer invoke ARN"
}

variable "lambda_function_name" {
  type        = string
  description = "Lambda authorizer function name (for invoke permission)"
}

variable "traefik_node_port" {
  type        = number
  description = "NodePort on EKS nodes where Traefik HTTP is exposed"
  default     = 30080
}

variable "regional_certificate_arn" {
  type        = string
  description = "ACM regional certificate ARN — attached to NLB TLS listener and API Gateway custom domain"
}

variable "domain_name" {
  type        = string
  description = "Base domain (e.g. opsnexus.site) — API custom domain is api.{env.}domain_name"
}
