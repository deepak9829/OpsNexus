output "nat_gateway_ids" {
  description = "List of NAT Gateway IDs (one per AZ if enable_nat_per_az=true, otherwise a single-element list)"
  value       = aws_nat_gateway.main[*].id
}

output "public_route_table_id" {
  description = "ID of the public route table"
  value       = aws_route_table.public.id
}

output "private_eks_route_table_ids" {
  description = "List of private EKS route table IDs (one per AZ)"
  value       = aws_route_table.private_eks[*].id
}
