data "aws_region" "current" {}

locals {
  oidc_url_stripped = replace(var.oidc_provider_url, "https://", "")
}

# IRSA role for the ESO controller — grants read access to Secrets Manager.
# The Helm release is installed by the deploy-k8s workflow, not Terraform.
resource "aws_iam_role" "eso" {
  name = "${var.project_name}-${var.environment}-eso-irsa"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = var.oidc_provider_arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_url_stripped}:sub" = "system:serviceaccount:external-secrets:external-secrets"
          "${local.oidc_url_stripped}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-eso-irsa" }
}

resource "aws_ssm_parameter" "eso_role_arn" {
  name  = "/${var.project_name}/${var.environment}/eso-irsa-role-arn"
  type  = "String"
  value = aws_iam_role.eso.arn
  tags  = { Name = "${var.project_name}-${var.environment}-eso-irsa-role-arn" }
}

resource "aws_iam_role_policy" "eso_secrets" {
  name = "secrets-manager-read"
  role = aws_iam_role.eso.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "ReadAppSecrets"
      Effect = "Allow"
      Action = [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret",
      ]
      Resource = "arn:aws:secretsmanager:${data.aws_region.current.name}:*:secret:${var.project_name}/${var.environment}/*"
    }]
  })
}

