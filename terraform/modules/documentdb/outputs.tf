output "cluster_endpoint" {
  description = "DocumentDB cluster endpoint"
  value       = aws_docdb_cluster.main.endpoint
}

output "secret_arn" {
  description = "Secrets Manager ARN for DocumentDB credentials"
  value       = aws_secretsmanager_secret.docdb.arn
}

output "security_group_id" {
  description = "DocumentDB security group ID"
  value       = aws_security_group.docdb.id
}
