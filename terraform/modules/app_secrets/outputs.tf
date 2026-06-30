output "jwt_secret_arn" {
  value       = aws_secretsmanager_secret.jwt.arn
  description = "ARN of the JWT secret in Secrets Manager"
}

output "jwt_secret_name" {
  value       = aws_secretsmanager_secret.jwt.name
  description = "Name of the JWT secret (for ESO ExternalSecret references)"
}
