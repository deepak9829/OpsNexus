data "aws_region" "current" {}

locals {
  oidc_url_stripped = replace(var.oidc_provider_url, "https://", "")
}

# IRSA role for ESO controller
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

resource "helm_release" "eso" {
  name             = "external-secrets"
  repository       = "https://charts.external-secrets.io"
  chart            = "external-secrets"
  version          = var.eso_version
  namespace        = "external-secrets"
  create_namespace = true
  wait             = true
  timeout          = 300

  values = [yamlencode({
    serviceAccount = {
      annotations = {
        "eks.amazonaws.com/role-arn" = aws_iam_role.eso.arn
      }
    }
    webhook        = { serviceAccount = { annotations = { "eks.amazonaws.com/role-arn" = aws_iam_role.eso.arn } } }
    certController = { serviceAccount = { annotations = { "eks.amazonaws.com/role-arn" = aws_iam_role.eso.arn } } }
  })]
}
