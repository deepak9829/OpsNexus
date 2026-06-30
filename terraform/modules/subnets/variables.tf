variable "project_name" {
  description = "Name of the project, used as a prefix in all resource names"
  type        = string
}

variable "environment" {
  description = "Deployment environment (e.g. dev, staging, prod)"
  type        = string
}

variable "vpc_id" {
  description = "ID of the VPC in which to create subnets"
  type        = string
}

variable "availability_zones" {
  description = "List of availability zones in which to create subnets (must have exactly 3 entries)"
  type        = list(string)
}

variable "public_subnet_cidrs" {
  description = "List of CIDR blocks for public subnets (one per AZ)"
  type        = list(string)
}

variable "private_eks_subnet_cidrs" {
  description = "List of CIDR blocks for private EKS subnets (one per AZ)"
  type        = list(string)
}

variable "private_db_subnet_cidrs" {
  description = "List of CIDR blocks for private DB subnets (one per AZ)"
  type        = list(string)
}

variable "vpc_cidr" {
  description = "VPC CIDR block, used in NACL rules to restrict DB and EKS traffic"
  type        = string
}
