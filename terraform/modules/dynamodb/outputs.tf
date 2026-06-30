output "notifications_table_name" {
  description = "Name of the DynamoDB notifications table."
  value       = aws_dynamodb_table.notifications.name
}

output "notifications_table_arn" {
  description = "ARN of the DynamoDB notifications table."
  value       = aws_dynamodb_table.notifications.arn
}

output "audit_events_table_name" {
  description = "Name of the DynamoDB audit events table."
  value       = aws_dynamodb_table.audit_events.name
}

output "audit_events_table_arn" {
  description = "ARN of the DynamoDB audit events table."
  value       = aws_dynamodb_table.audit_events.arn
}
