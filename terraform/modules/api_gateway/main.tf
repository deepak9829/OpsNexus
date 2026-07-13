################################################################################
# API Gateway Module — OpsNexus
#
# Flow:
#   Internet → REST API Gateway
#              ├── /auth/*  (public — no JWT, no API key)
#              └── /*       (JWT authorizer + API key required)
#                       │
#                  VPC Link → NLB → Traefik (NodePort on EKS nodes)
#
# SaaS billing: Usage Plans (basic / pro / enterprise) enforce per-tenant
# rate limits and quotas via API keys.
################################################################################

data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

locals {
  env_prefix = var.environment != "prod" ? "${var.environment}." : ""
  api_domain = "api.${local.env_prefix}${var.domain_name}"
}

# ─── Account-level: allow API Gateway to push logs to CloudWatch ──────────────
# This is a regional account setting (one per region). Must be set before any
# Stage resource enables logging_level, or UpdateStage returns 400.

resource "aws_iam_role" "api_gw_cloudwatch" {
  name = "${var.project_name}-${var.environment}-apigw-cw-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "apigateway.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-apigw-cw-role" }
}

resource "aws_iam_role_policy_attachment" "api_gw_cloudwatch" {
  role       = aws_iam_role.api_gw_cloudwatch.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
}

resource "aws_api_gateway_account" "main" {
  cloudwatch_role_arn = aws_iam_role.api_gw_cloudwatch.arn

  depends_on = [aws_iam_role_policy_attachment.api_gw_cloudwatch]
}

# ─── CloudWatch log group ─────────────────────────────────────────────────────
resource "aws_cloudwatch_log_group" "api_gw" {
  name              = "/aws/api-gateway/${var.project_name}-${var.environment}"
  retention_in_days = 30
  tags              = { Name = "${var.project_name}-${var.environment}-apigw-logs" }
}

################################################################################
# NLB — VPC Link target, routes directly to Traefik NodePort on EKS nodes
################################################################################

resource "aws_lb" "nlb" {
  name               = "${var.project_name}-${var.environment}-apigw-nlb"
  internal           = true
  load_balancer_type = "network"
  subnets            = var.private_eks_subnet_ids

  enable_cross_zone_load_balancing = true

  tags = { Name = "${var.project_name}-${var.environment}-apigw-nlb" }
}

# Target group: instance type, port = Traefik NodePort (default 30080)
# EKS nodes register automatically via ASG attachment (done in environment layer)
resource "aws_lb_target_group" "traefik" {
  name        = "${var.project_name}-${var.environment}-traefik-tg"
  port        = var.traefik_node_port
  protocol    = "TCP"
  target_type = "instance"
  vpc_id      = var.vpc_id

  health_check {
    enabled             = true
    protocol            = "HTTP"
    port                = var.traefik_node_port
    path                = "/ping"
    healthy_threshold   = 2
    unhealthy_threshold = 2
    interval            = 10
  }

  tags = { Name = "${var.project_name}-${var.environment}-traefik-tg" }
}

resource "aws_lb_listener" "nlb_http" {
  load_balancer_arn = aws_lb.nlb.arn
  port              = 80
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.traefik.arn
  }
}

################################################################################
# VPC Link — REST API → NLB
################################################################################

resource "aws_api_gateway_vpc_link" "main" {
  name        = "${var.project_name}-${var.environment}-vpc-link"
  description = "VPC Link: REST API → NLB → Traefik (EKS)"
  target_arns = [aws_lb.nlb.arn]
  tags        = { Name = "${var.project_name}-${var.environment}-vpc-link" }
}

################################################################################
# REST API
################################################################################

resource "aws_api_gateway_rest_api" "main" {
  name        = "${var.project_name}-${var.environment}-api"
  description = "OpsNexus SaaS API Gateway (${var.environment})"

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  minimum_compression_size = 1024

  # Tenants send their key in x-api-key header
  api_key_source = "HEADER"

  tags = { Name = "${var.project_name}-${var.environment}-api" }
}

# JWT Lambda Authorizer
resource "aws_api_gateway_authorizer" "jwt" {
  name                             = "jwt-authorizer"
  rest_api_id                      = aws_api_gateway_rest_api.main.id
  authorizer_uri                   = var.lambda_invoke_arn
  type                             = "TOKEN"
  identity_source                  = "method.request.header.Authorization"
  authorizer_result_ttl_in_seconds = 300
}

################################################################################
# Service routes
################################################################################

locals {
  service_paths = {
    auth = {
      path         = "auth"
      backend_port = 8081
      public       = true # login / register / refresh — no JWT, no API key
    }
    tenants = {
      path         = "tenants"
      backend_port = 8082
      public       = false
    }
    cases = {
      path         = "cases"
      backend_port = 8083
      public       = false
    }
    tasks = {
      path         = "tasks"
      backend_port = 8083
      public       = false
    }
    notifications = {
      path         = "notifications"
      backend_port = 8085
      public       = false
    }
  }
}

resource "aws_api_gateway_resource" "services" {
  for_each    = local.service_paths
  rest_api_id = aws_api_gateway_rest_api.main.id
  parent_id   = aws_api_gateway_rest_api.main.root_resource_id
  path_part   = each.value.path
}

resource "aws_api_gateway_resource" "proxy" {
  for_each    = local.service_paths
  rest_api_id = aws_api_gateway_rest_api.main.id
  parent_id   = aws_api_gateway_resource.services[each.key].id
  path_part   = "{proxy+}"
}

resource "aws_api_gateway_method" "proxy_any" {
  for_each = local.service_paths

  rest_api_id      = aws_api_gateway_rest_api.main.id
  resource_id      = aws_api_gateway_resource.proxy[each.key].id
  http_method      = "ANY"
  authorization    = each.value.public ? "NONE" : "CUSTOM"
  authorizer_id    = each.value.public ? null : aws_api_gateway_authorizer.jwt.id
  api_key_required = each.value.public ? false : true

  request_parameters = {
    "method.request.path.proxy"           = true
    "method.request.header.Authorization" = false
  }
}

locals {
  cors_allow_headers = "Accept,Authorization,Content-Type,X-Request-ID,X-Tenant-ID,X-User-ID,x-api-key"
  cors_allow_methods = "GET,POST,PUT,DELETE,OPTIONS,PATCH"
}

# OPTIONS method on parent resource (handles /cases, /tenants etc. with no trailing path)
resource "aws_api_gateway_method" "options_parent" {
  for_each = local.service_paths

  rest_api_id      = aws_api_gateway_rest_api.main.id
  resource_id      = aws_api_gateway_resource.services[each.key].id
  http_method      = "OPTIONS"
  authorization    = "NONE"
  api_key_required = false
}

resource "aws_api_gateway_integration" "options_parent" {
  for_each          = local.service_paths
  rest_api_id       = aws_api_gateway_rest_api.main.id
  resource_id       = aws_api_gateway_resource.services[each.key].id
  http_method       = aws_api_gateway_method.options_parent[each.key].http_method
  type              = "MOCK"
  request_templates = { "application/json" = "{\"statusCode\":200}" }
}

resource "aws_api_gateway_method_response" "options_parent_200" {
  for_each    = local.service_paths
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.services[each.key].id
  http_method = aws_api_gateway_method.options_parent[each.key].http_method
  status_code = "200"
  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = false
    "method.response.header.Access-Control-Allow-Methods" = false
    "method.response.header.Access-Control-Allow-Origin"  = false
    "method.response.header.Access-Control-Max-Age"       = false
  }
}

resource "aws_api_gateway_integration_response" "options_parent_200" {
  for_each    = local.service_paths
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.services[each.key].id
  http_method = aws_api_gateway_method.options_parent[each.key].http_method
  status_code = "200"
  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'${local.cors_allow_headers}'"
    "method.response.header.Access-Control-Allow-Methods" = "'${local.cors_allow_methods}'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
    "method.response.header.Access-Control-Max-Age"       = "'86400'"
  }
  depends_on = [aws_api_gateway_integration.options_parent]
}

# OPTIONS method on {proxy+} resource (handles /cases/123, /cases/list etc.)
resource "aws_api_gateway_method" "options_proxy" {
  for_each = local.service_paths

  rest_api_id      = aws_api_gateway_rest_api.main.id
  resource_id      = aws_api_gateway_resource.proxy[each.key].id
  http_method      = "OPTIONS"
  authorization    = "NONE"
  api_key_required = false
}

resource "aws_api_gateway_integration" "options_proxy" {
  for_each          = local.service_paths
  rest_api_id       = aws_api_gateway_rest_api.main.id
  resource_id       = aws_api_gateway_resource.proxy[each.key].id
  http_method       = aws_api_gateway_method.options_proxy[each.key].http_method
  type              = "MOCK"
  request_templates = { "application/json" = "{\"statusCode\":200}" }
}

resource "aws_api_gateway_method_response" "options_proxy_200" {
  for_each    = local.service_paths
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.proxy[each.key].id
  http_method = aws_api_gateway_method.options_proxy[each.key].http_method
  status_code = "200"
  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = false
    "method.response.header.Access-Control-Allow-Methods" = false
    "method.response.header.Access-Control-Allow-Origin"  = false
    "method.response.header.Access-Control-Max-Age"       = false
  }
}

resource "aws_api_gateway_integration_response" "options_proxy_200" {
  for_each    = local.service_paths
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.proxy[each.key].id
  http_method = aws_api_gateway_method.options_proxy[each.key].http_method
  status_code = "200"
  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'${local.cors_allow_headers}'"
    "method.response.header.Access-Control-Allow-Methods" = "'${local.cors_allow_methods}'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
    "method.response.header.Access-Control-Max-Age"       = "'86400'"
  }
  depends_on = [aws_api_gateway_integration.options_proxy]
}

# Gateway responses — add CORS headers to all 4xx/5xx so browser isn't blocked on auth errors
resource "aws_api_gateway_gateway_response" "cors_4xx" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  response_type = "DEFAULT_4XX"
  response_parameters = {
    "gatewayresponse.header.Access-Control-Allow-Origin"  = "'*'"
    "gatewayresponse.header.Access-Control-Allow-Headers" = "'${local.cors_allow_headers}'"
  }
}

resource "aws_api_gateway_gateway_response" "cors_5xx" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  response_type = "DEFAULT_5XX"
  response_parameters = {
    "gatewayresponse.header.Access-Control-Allow-Origin"  = "'*'"
    "gatewayresponse.header.Access-Control-Allow-Headers" = "'${local.cors_allow_headers}'"
  }
}

resource "aws_api_gateway_integration" "proxy_any" {
  for_each = local.service_paths

  rest_api_id             = aws_api_gateway_rest_api.main.id
  resource_id             = aws_api_gateway_resource.proxy[each.key].id
  http_method             = aws_api_gateway_method.proxy_any[each.key].http_method
  integration_http_method = "ANY"
  type                    = "HTTP_PROXY"
  connection_type         = "VPC_LINK"
  connection_id           = aws_api_gateway_vpc_link.main.id

  # NLB DNS → Traefik → service. Traefik routes by path prefix via IngressRoute CRDs.
  uri = "http://${aws_lb.nlb.dns_name}/api/v1/${each.value.path}/{proxy}"

  request_parameters = {
    "integration.request.path.proxy" = "method.request.path.proxy"
    # Forward authorizer context as trusted internal headers
    "integration.request.header.X-User-Id"    = "context.authorizer.userId"
    "integration.request.header.X-Tenant-Id"  = "context.authorizer.tenantId"
    "integration.request.header.X-User-Roles" = "context.authorizer.roles"
  }
}

################################################################################
# Deployment + Stage
################################################################################

resource "aws_api_gateway_deployment" "main" {
  rest_api_id = aws_api_gateway_rest_api.main.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.services,
      aws_api_gateway_resource.proxy,
      aws_api_gateway_method.proxy_any,
      aws_api_gateway_integration.proxy_any,
      aws_api_gateway_method.options_parent,
      aws_api_gateway_method.options_proxy,
      aws_api_gateway_gateway_response.cors_4xx,
      aws_api_gateway_gateway_response.cors_5xx,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [aws_api_gateway_integration.proxy_any]
}

resource "aws_api_gateway_stage" "main" {
  deployment_id = aws_api_gateway_deployment.main.id
  rest_api_id   = aws_api_gateway_rest_api.main.id
  stage_name    = var.environment

  xray_tracing_enabled = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gw.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      ip             = "$context.identity.sourceIp"
      requestTime    = "$context.requestTime"
      httpMethod     = "$context.httpMethod"
      resourcePath   = "$context.resourcePath"
      status         = "$context.status"
      responseLength = "$context.responseLength"
      protocol       = "$context.protocol"
      # SaaS billing / audit fields
      apiKeyId = "$context.identity.apiKeyId"
      tenantId = "$context.authorizer.tenantId"
      userId   = "$context.authorizer.userId"
      # Diagnostics
      integrationError  = "$context.integration.error"
      integrationStatus = "$context.integration.integrationStatus"
      authorizerError   = "$context.authorizer.error"
    })
  }

  tags = { Name = "${var.project_name}-${var.environment}-api-stage" }
}

resource "aws_api_gateway_method_settings" "all" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  stage_name  = aws_api_gateway_stage.main.stage_name
  method_path = "*/*"

  depends_on = [aws_api_gateway_account.main]

  settings {
    throttling_burst_limit = 5000
    throttling_rate_limit  = 2000
    logging_level          = "INFO"
    data_trace_enabled     = var.environment != "prod"
    metrics_enabled        = true
  }
}

################################################################################
# Usage Plans — SaaS subscription tiers
#
# Tenant lifecycle:
#   1. Tenant signs up → chooses plan (basic / pro / enterprise)
#   2. tenant-service calls AWS SDK: CreateApiKey → CreateUsagePlanKey
#   3. API key value stored encrypted in tenant's record
#   4. Tenant includes x-api-key header on every API request
#   5. API Gateway enforces quota/throttle BEFORE Lambda authorizer runs
#      (saves Lambda invocations on over-quota requests)
################################################################################

resource "aws_api_gateway_usage_plan" "basic" {
  name        = "${var.project_name}-${var.environment}-basic"
  description = "Basic — small teams, 10k req/day, 10 req/s"

  api_stages {
    api_id = aws_api_gateway_rest_api.main.id
    stage  = aws_api_gateway_stage.main.stage_name
  }

  throttle_settings {
    rate_limit  = 10
    burst_limit = 20
  }

  quota_settings {
    limit  = 10000
    period = "DAY"
  }

  tags = { Name = "${var.project_name}-${var.environment}-basic-plan" }
}

resource "aws_api_gateway_usage_plan" "pro" {
  name        = "${var.project_name}-${var.environment}-pro"
  description = "Pro — growing businesses, 100k req/day, 100 req/s"

  api_stages {
    api_id = aws_api_gateway_rest_api.main.id
    stage  = aws_api_gateway_stage.main.stage_name
  }

  throttle_settings {
    rate_limit  = 100
    burst_limit = 200
  }

  quota_settings {
    limit  = 100000
    period = "DAY"
  }

  tags = { Name = "${var.project_name}-${var.environment}-pro-plan" }
}

resource "aws_api_gateway_usage_plan" "enterprise" {
  name        = "${var.project_name}-${var.environment}-enterprise"
  description = "Enterprise — unlimited quota, 1000 req/s"

  api_stages {
    api_id = aws_api_gateway_rest_api.main.id
    stage  = aws_api_gateway_stage.main.stage_name
  }

  throttle_settings {
    rate_limit  = 1000
    burst_limit = 2000
  }

  # No quota_settings = unlimited

  tags = { Name = "${var.project_name}-${var.environment}-enterprise-plan" }
}

# Internal key — health checks and service-to-service calls
resource "aws_api_gateway_api_key" "internal" {
  name        = "${var.project_name}-${var.environment}-internal"
  description = "Internal key for health checks and service-to-service calls"
  enabled     = true
  tags        = { Name = "${var.project_name}-${var.environment}-internal-key" }
}

resource "aws_api_gateway_usage_plan_key" "internal" {
  key_id        = aws_api_gateway_api_key.internal.id
  key_type      = "API_KEY"
  usage_plan_id = aws_api_gateway_usage_plan.enterprise.id
}

# Store internal key in Secrets Manager for Go services to retrieve
resource "aws_secretsmanager_secret" "internal_api_key" {
  name                    = "${var.project_name}/${var.environment}/api-gateway/internal-key"
  description             = "Internal API Gateway key for ${var.project_name}-${var.environment}"
  recovery_window_in_days = 0
  tags                    = { Name = "${var.project_name}-${var.environment}-internal-api-key" }
}

resource "aws_secretsmanager_secret_version" "internal_api_key" {
  secret_id     = aws_secretsmanager_secret.internal_api_key.id
  secret_string = aws_api_gateway_api_key.internal.value
}

################################################################################
# Custom Domain — api.{env.}opsnexus.site
################################################################################

resource "aws_api_gateway_domain_name" "api" {
  domain_name              = local.api_domain
  regional_certificate_arn = var.regional_certificate_arn
  security_policy          = "TLS_1_2"

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  tags = { Name = "${var.project_name}-${var.environment}-api-domain" }
}

# Maps the stage to the root path of the custom domain.
# After this, api.dev.opsnexus.site/auth/* works the same as the execute-api URL.
resource "aws_api_gateway_base_path_mapping" "api" {
  api_id      = aws_api_gateway_rest_api.main.id
  stage_name  = aws_api_gateway_stage.main.stage_name
  domain_name = aws_api_gateway_domain_name.api.domain_name
}

################################################################################
# Allow API GW to invoke Lambda authorizer
################################################################################

resource "aws_lambda_permission" "api_gw_authorizer" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.lambda_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.main.execution_arn}/*/*"
}
