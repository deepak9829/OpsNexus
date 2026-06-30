output "repository_urls" {
  description = "Map of service name to ECR repository URL."
  value       = { for k, v in aws_ecr_repository.services : k => v.repository_url }
}

output "repository_arns" {
  description = "Map of service name to ECR repository ARN."
  value       = { for k, v in aws_ecr_repository.services : k => v.arn }
}

output "kms_key_arn" {
  description = "ARN of the KMS key used to encrypt ECR repositories."
  value       = aws_kms_key.ecr.arn
}

output "kms_key_id" {
  description = "ID of the KMS key used to encrypt ECR repositories."
  value       = aws_kms_key.ecr.key_id
}
