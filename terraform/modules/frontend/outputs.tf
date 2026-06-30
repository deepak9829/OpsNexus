output "bucket_id" {
  description = "Shared frontend S3 bucket name"
  value       = aws_s3_bucket.content.id
}

output "bucket_arn" {
  description = "Shared frontend S3 bucket ARN"
  value       = aws_s3_bucket.content.arn
}

output "distribution_ids" {
  description = "Map of app_name => CloudFront distribution ID"
  value       = { for k, d in aws_cloudfront_distribution.app : k => d.id }
}

output "distribution_domain_names" {
  description = "Map of app_name => CloudFront distribution domain name"
  value       = { for k, d in aws_cloudfront_distribution.app : k => d.domain_name }
}

output "distribution_hosted_zone_ids" {
  description = "Map of app_name => CloudFront hosted zone ID (for Route53 ALIAS records)"
  value       = { for k, d in aws_cloudfront_distribution.app : k => d.hosted_zone_id }
}

output "cloudfront_urls" {
  description = "Map of app_name => CloudFront HTTPS URL"
  value       = { for k, d in aws_cloudfront_distribution.app : k => "https://${d.domain_name}" }
}
