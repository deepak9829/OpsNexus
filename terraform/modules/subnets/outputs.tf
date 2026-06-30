output "public_subnet_ids" {
  description = "List of IDs of the public subnets"
  value       = aws_subnet.public[*].id
}

output "private_eks_subnet_ids" {
  description = "List of IDs of the private EKS subnets"
  value       = aws_subnet.private_eks[*].id
}

output "private_db_subnet_ids" {
  description = "List of IDs of the private DB subnets"
  value       = aws_subnet.private_db[*].id
}

output "db_subnet_group_name" {
  description = "Name of the RDS DB subnet group"
  value       = aws_db_subnet_group.main.name
}
