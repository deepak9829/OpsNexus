output "notification_role_arn" {
  description = "IRSA role ARN for notification-service"
  value       = aws_iam_role.notification.arn
}
