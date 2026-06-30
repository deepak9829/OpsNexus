variable "project_name" {
  description = "Name of the project, used as a prefix in all resource names"
  type        = string
}

variable "environment" {
  description = "Deployment environment (e.g. dev, staging, prod)"
  type        = string
}

variable "vpc_id" {
  description = "ID of the VPC"
  type        = string
}

variable "igw_id" {
  description = "ID of the Internet Gateway attached to the VPC"
  type        = string
}

variable "availability_zones" {
  description = "List of availability zones that match the subnet lists"
  type        = list(string)
}

variable "public_subnet_ids" {
  description = "List of public subnet IDs (one per AZ, ordered to match availability_zones)"
  type        = list(string)
}

variable "private_eks_subnet_ids" {
  description = "List of private EKS subnet IDs (one per AZ, ordered to match availability_zones)"
  type        = list(string)
}

variable "private_db_subnet_ids" {
  description = "List of private DB subnet IDs (one per AZ, ordered to match availability_zones)"
  type        = list(string)
}

variable "enable_nat_per_az" {
  description = "When true, provision one NAT Gateway per AZ for high availability. When false, a single shared NAT Gateway is used (lower cost, suitable for non-prod)."
  type        = bool
  default     = false
}
