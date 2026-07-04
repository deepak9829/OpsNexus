data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

locals {
  oidc_url_stripped = replace(var.oidc_provider_url, "https://", "")
  account_id        = data.aws_caller_identity.current.account_id
  region            = data.aws_region.current.name
}

# ─── notification-service: DynamoDB read/write via IRSA ───────────────────────
# Pods assume this role automatically via the service account annotation.
# No AWS credentials are injected into the container.
resource "aws_iam_role" "notification" {
  name = "${var.project_name}-${var.environment}-notification-irsa"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = var.oidc_provider_arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_url_stripped}:sub" = "system:serviceaccount:opsnexus:notification-service"
          "${local.oidc_url_stripped}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-notification-irsa" }
}

resource "aws_iam_role_policy" "notification_dynamodb" {
  name = "dynamodb-notifications"
  role = aws_iam_role.notification.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "DynamoDBNotificationsAndAudit"
      Effect = "Allow"
      Action = [
        "dynamodb:PutItem",
        "dynamodb:GetItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:BatchWriteItem",
        "dynamodb:BatchGetItem",
        "dynamodb:DescribeTable",
      ]
      Resource = [
        "arn:aws:dynamodb:${local.region}:${local.account_id}:table/${var.project_name}-${var.environment}-notifications",
        "arn:aws:dynamodb:${local.region}:${local.account_id}:table/${var.project_name}-${var.environment}-notifications/index/*",
        "arn:aws:dynamodb:${local.region}:${local.account_id}:table/${var.project_name}-${var.environment}-audit-events",
        "arn:aws:dynamodb:${local.region}:${local.account_id}:table/${var.project_name}-${var.environment}-audit-events/index/*",
      ]
    }]
  })
}

# ─── Publish role ARN to SSM so deploy-k8s workflow can inject it ─────────────
resource "aws_ssm_parameter" "notification_role_arn" {
  name  = "/${var.project_name}/${var.environment}/notification-irsa-role-arn"
  type  = "String"
  value = aws_iam_role.notification.arn
  tags  = { Name = "${var.project_name}-${var.environment}-notification-irsa-role-arn" }
}
