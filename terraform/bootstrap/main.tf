# Bootstrap: Creates S3 state bucket + GitHub OIDC role.
# Run ONCE before all other terraform environments.
# State for this config is local (bootstrap.tfstate — gitignored).
#
# Usage:
#   cd terraform/bootstrap
#   terraform init
#   terraform apply -var="github_org=YOUR_GITHUB_ORG" -var="github_repo=OpsNexus"
#
# AWS profile: defaults to "opsnexus". Override with -var="aws_profile=other".

terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile != "" ? var.aws_profile : null
}

data "aws_caller_identity" "current" {}

# ─── S3 state bucket ──────────────────────────────────────────────────────
resource "aws_s3_bucket" "tf_state" {
  bucket        = "dev-opsnexus-tf-state"
  force_destroy = false
  tags = {
    Name      = "dev-opsnexus-tf-state"
    ManagedBy = "Terraform"
    Purpose   = "Terraform state storage"
  }
}

resource "aws_s3_bucket_versioning" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tf_state" {
  bucket = aws_s3_bucket.tf_state.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "tf_state" {
  bucket                  = aws_s3_bucket.tf_state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Create plan folder structure
resource "aws_s3_object" "plans_folder" {
  bucket  = aws_s3_bucket.tf_state.id
  key     = "plans/"
  content = ""
}

# ─── GitHub Actions OIDC provider ─────────────────────────────────────────
data "aws_iam_openid_connect_provider" "github" {
  count = var.create_github_oidc_provider ? 0 : 1
  url   = "https://token.actions.githubusercontent.com"
}

resource "aws_iam_openid_connect_provider" "github" {
  count           = var.create_github_oidc_provider ? 1 : 0
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  tags = {
    Name      = "github-actions-oidc"
    ManagedBy = "Terraform"
  }
}

locals {
  github_oidc_provider_arn = var.create_github_oidc_provider ? (
    aws_iam_openid_connect_provider.github[0].arn
    ) : (
    data.aws_iam_openid_connect_provider.github[0].arn
  )
}

# ─── GitHub Actions IAM role (used by all workflows) ──────────────────────
resource "aws_iam_role" "github_actions" {
  name        = "opsnexus-github-actions"
  description = "Role assumed by GitHub Actions via OIDC for OpsNexus deployments"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.github_oidc_provider_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringLike = {
          "token.actions.githubusercontent.com:sub" = "repo:${var.github_org}/${var.github_repo}:*"
        }
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = {
    Name      = "opsnexus-github-actions"
    ManagedBy = "Terraform"
  }
}

# Policy: Terraform + ECR + EKS + S3 state
resource "aws_iam_role_policy" "github_actions" {
  name = "opsnexus-github-actions-policy"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "TerraformStateAccess"
        Effect = "Allow"
        Action = [
          "s3:GetObject", "s3:PutObject", "s3:DeleteObject",
          "s3:ListBucket", "s3:GetBucketVersioning",
        ]
        Resource = [
          aws_s3_bucket.tf_state.arn,
          "${aws_s3_bucket.tf_state.arn}/*",
        ]
      },
      {
        Sid    = "ECRAccess"
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload",
          "ecr:DescribeRepositories",
          "ecr:ListImages",
        ]
        Resource = "*"
      },
      {
        Sid    = "EKSAccess"
        Effect = "Allow"
        Action = [
          "eks:DescribeCluster",
          "eks:ListClusters",
          "eks:UpdateKubeconfig",
        ]
        Resource = "*"
      },
      {
        Sid    = "SSMParameterAccess"
        Effect = "Allow"
        Action = [
          "ssm:GetParameter", "ssm:GetParameters", "ssm:GetParametersByPath",
          "ssm:PutParameter", "ssm:DeleteParameter", "ssm:DeleteParameters",
          "ssm:GetParameterHistory", "ssm:AddTagsToResource", "ssm:ListTagsForResource",
        ]
        Resource = "arn:aws:ssm:*:${data.aws_caller_identity.current.account_id}:parameter/opsnexus/*"
      },
      {
        # DescribeParameters operates on the SSM service, not individual parameter ARNs
        Sid      = "SSMDescribeParameters"
        Effect   = "Allow"
        Action   = ["ssm:DescribeParameters"]
        Resource = "*"
      },
      {
        # s3:ListAllMyBuckets (used by data.aws_canonical_user_id) requires Resource = "*"
        # bucket-scoped s3:* does not cover service-level list actions
        Sid      = "S3ServiceLevelActions"
        Effect   = "Allow"
        Action   = ["s3:ListAllMyBuckets", "s3:GetAccountPublicAccessBlock"]
        Resource = "*"
      },
      {
        Sid      = "CloudFrontInvalidation"
        Effect   = "Allow"
        Action   = ["cloudfront:CreateInvalidation"]
        Resource = "*"
      },
      {
        Sid    = "S3BucketManagement"
        Effect = "Allow"
        Action = ["s3:*"]
        Resource = [
          "arn:aws:s3:::opsnexus-*",
          "arn:aws:s3:::opsnexus-*/*",
        ]
      },
      {
        Sid    = "TerraformProvisioning"
        Effect = "Allow"
        Action = [
          "ec2:*", "eks:*", "ecr:*", "rds:*", "dynamodb:*",
          "elasticloadbalancing:*", "autoscaling:*",
          "iam:CreateRole", "iam:DeleteRole", "iam:GetRole", "iam:ListRoles",
          "iam:AttachRolePolicy", "iam:DetachRolePolicy", "iam:PutRolePolicy",
          "iam:DeleteRolePolicy", "iam:GetRolePolicy", "iam:ListRolePolicies",
          "iam:ListAttachedRolePolicies", "iam:TagRole", "iam:UntagRole",
          "iam:PassRole", "iam:CreateInstanceProfile", "iam:DeleteInstanceProfile",
          "iam:AddRoleToInstanceProfile", "iam:RemoveRoleFromInstanceProfile",
          "iam:GetInstanceProfile", "iam:ListInstanceProfiles",
          "iam:ListInstanceProfilesForRole",
          "iam:CreateServiceLinkedRole",
          "iam:CreateOpenIDConnectProvider", "iam:GetOpenIDConnectProvider",
          "iam:DeleteOpenIDConnectProvider", "iam:TagOpenIDConnectProvider",
          "kms:*", "secretsmanager:*",
          "route53:*", "acm:*",
          "cloudfront:*", "apigateway:*",
          "lambda:*", "logs:*",
          "sqs:*", "events:*",
          "sts:GetCallerIdentity",
        ]
        Resource = "*"
      },
    ]
  })
}

variable "aws_region" {
  type    = string
  default = "ap-south-1"
}

variable "aws_profile" {
  type        = string
  default     = "opsnexus"
  description = "AWS CLI profile for local bootstrap run."
}

variable "github_org" {
  type        = string
  description = "GitHub organization or username"
}

variable "github_repo" {
  type        = string
  description = "GitHub repository name"
}

variable "create_github_oidc_provider" {
  type        = bool
  default     = true
  description = "Set false if OIDC provider already exists in your account"
}

output "state_bucket_name" {
  value = aws_s3_bucket.tf_state.id
}

output "github_actions_role_arn" {
  description = "Set this as AWS_ROLE_ARN in GitHub repository secrets"
  value       = aws_iam_role.github_actions.arn
}
