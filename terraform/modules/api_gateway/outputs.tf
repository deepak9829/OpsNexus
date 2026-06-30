output "api_id" {
  description = "REST API ID"
  value       = aws_api_gateway_rest_api.main.id
}

output "api_execution_arn" {
  description = "REST API execution ARN"
  value       = aws_api_gateway_rest_api.main.execution_arn
}

output "stage_invoke_url" {
  description = "Stage invoke URL — base URL for all tenant API calls"
  value       = aws_api_gateway_stage.main.invoke_url
}

output "vpc_link_id" {
  description = "VPC Link ID"
  value       = aws_api_gateway_vpc_link.main.id
}

output "nlb_arn" {
  description = "NLB ARN"
  value       = aws_lb.nlb.arn
}

output "nlb_dns_name" {
  description = "NLB DNS name"
  value       = aws_lb.nlb.dns_name
}

output "traefik_target_group_arn" {
  description = "NLB target group ARN for Traefik — attach EKS node ASG to this"
  value       = aws_lb_target_group.traefik.arn
}

output "usage_plan_ids" {
  description = "Map of plan tier to usage plan ID — store in SSM for tenant-service to use when issuing API keys"
  value = {
    basic      = aws_api_gateway_usage_plan.basic.id
    pro        = aws_api_gateway_usage_plan.pro.id
    enterprise = aws_api_gateway_usage_plan.enterprise.id
  }
}

output "usage_plan_basic_id"      { value = aws_api_gateway_usage_plan.basic.id }
output "usage_plan_pro_id"        { value = aws_api_gateway_usage_plan.pro.id }
output "usage_plan_enterprise_id" { value = aws_api_gateway_usage_plan.enterprise.id }

output "internal_api_key_secret_arn" {
  description = "Secrets Manager ARN for the internal API key value"
  value       = aws_secretsmanager_secret.internal_api_key.arn
}

output "api_custom_domain" {
  description = "Custom domain name for API Gateway (e.g. api.dev.opsnexus.site)"
  value       = aws_api_gateway_domain_name.api.domain_name
}

output "api_custom_domain_regional_name" {
  description = "Regional DNS target for the API custom domain — use as Route53 alias target"
  value       = aws_api_gateway_domain_name.api.regional_domain_name
}

output "api_custom_domain_regional_zone_id" {
  description = "Hosted zone ID of the API custom domain — use for Route53 alias"
  value       = aws_api_gateway_domain_name.api.regional_zone_id
}
