output "cloudfront_certificate_arn" {
  description = "ACM cert ARN for CloudFront (us-east-1) — pass to frontend module"
  value       = aws_acm_certificate_validation.cloudfront.certificate_arn
}

output "regional_certificate_arn" {
  description = "ACM cert ARN for API Gateway / ALB (regional)"
  value       = aws_acm_certificate_validation.regional.certificate_arn
}

output "hosted_zone_id" {
  description = "Route53 hosted zone ID — use to create A records in the environment"
  value       = data.aws_route53_zone.main.zone_id
}

output "env_prefix" {
  description = "DNS prefix: empty for prod, '{env}.' for others"
  value       = local.env_prefix
}
