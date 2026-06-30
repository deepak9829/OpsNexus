output "vpc_id"           { value = module.vpc.vpc_id }
output "cluster_name"     { value = module.eks.cluster_name }

output "cluster_endpoint" {
  value     = module.eks.cluster_endpoint
  sensitive = true
}

output "ecr_repository_urls" { value = module.ecr.repository_urls }

output "rds_endpoint"   { value = module.rds.rds_endpoint }
output "rds_secret_arn" { value = module.rds.secret_arn }

output "notifications_table_name" { value = module.dynamodb.notifications_table_name }
output "audit_events_table_name"  { value = module.dynamodb.audit_events_table_name }

output "api_invoke_url"         { value = module.api_gateway.stage_invoke_url }
output "customer_portal_url"    { value = module.frontend.cloudfront_urls["customer-portal"] }
output "admin_console_url"      { value = module.frontend.cloudfront_urls["admin-console"] }
output "cloudfront_cf_cert_arn" { value = module.dns.cloudfront_certificate_arn }
