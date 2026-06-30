################################################################################
# Frontend Module — OpsNexus
#
# Single S3 bucket with one folder per app:
#   opsnexus-{env}-frontend-{account}/
#     customer-portal/   ← served by customer-portal CloudFront distribution
#     admin-console/     ← served by admin-console CloudFront distribution
#
# CloudFront origin_path = "/{app_name}" transparently maps each distribution
# to its folder — no path rewriting needed in the SPA or deploy pipeline.
################################################################################

data "aws_caller_identity" "current" {}
data "aws_canonical_user_id" "current" {}
data "aws_cloudfront_log_delivery_canonical_user_id" "cloudfront" {}

locals {
  bucket_name      = "${var.project_name}-${var.environment}-frontend-${data.aws_caller_identity.current.account_id}"
  logs_bucket_name = "${var.project_name}-${var.environment}-frontend-logs-${data.aws_caller_identity.current.account_id}"
}

# ─── Shared content bucket ────────────────────────────────────────────────────

resource "aws_s3_bucket" "content" {
  bucket        = local.bucket_name
  force_destroy = true
  tags          = { Name = local.bucket_name }
}

resource "aws_s3_bucket_versioning" "content" {
  bucket = aws_s3_bucket.content.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "content" {
  bucket = aws_s3_bucket.content.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "content" {
  bucket                  = aws_s3_bucket.content.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# ─── Logs bucket ─────────────────────────────────────────────────────────────

resource "aws_s3_bucket" "logs" {
  bucket        = local.logs_bucket_name
  force_destroy = true
  tags          = { Name = local.logs_bucket_name }
}

# CloudFront log delivery writes objects with bucket-owner-full-control ACL.
# S3 now defaults to BucketOwnerEnforced (no ACLs) — must switch to ObjectWriter.
resource "aws_s3_bucket_ownership_controls" "logs" {
  bucket = aws_s3_bucket.logs.id
  rule {
    object_ownership = "ObjectWriter"
  }
}

resource "aws_s3_bucket_acl" "logs" {
  bucket     = aws_s3_bucket.logs.id
  depends_on = [aws_s3_bucket_ownership_controls.logs]

  access_control_policy {
    owner {
      id = data.aws_canonical_user_id.current.id
    }
    # CloudFront log delivery service needs FULL_CONTROL to write access logs.
    grant {
      grantee {
        type = "CanonicalUser"
        id   = data.aws_cloudfront_log_delivery_canonical_user_id.cloudfront.id
      }
      permission = "FULL_CONTROL"
    }
    grant {
      grantee {
        type = "CanonicalUser"
        id   = data.aws_canonical_user_id.current.id
      }
      permission = "FULL_CONTROL"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "logs" {
  bucket                  = aws_s3_bucket.logs.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
  depends_on              = [aws_s3_bucket_ownership_controls.logs]
}

resource "aws_s3_bucket_lifecycle_configuration" "logs" {
  bucket = aws_s3_bucket.logs.id
  rule {
    id     = "expire-logs"
    status = "Enabled"
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
    expiration { days = 90 }
    filter { prefix = "" }
  }
}

# ─── One OAC per app ─────────────────────────────────────────────────────────

resource "aws_cloudfront_origin_access_control" "app" {
  for_each = var.apps

  name                              = "${var.project_name}-${var.environment}-${each.key}-oac"
  description                       = "OAC for ${each.key} (${var.environment})"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# ─── One CloudFront distribution per app ─────────────────────────────────────

resource "aws_cloudfront_distribution" "app" {
  for_each = var.apps

  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  aliases             = length(each.value.domain_aliases) > 0 ? each.value.domain_aliases : []
  price_class         = "PriceClass_100"
  comment             = "${var.project_name}-${var.environment}-${each.key}"

  origin {
    domain_name = aws_s3_bucket.content.bucket_regional_domain_name
    origin_id   = "s3-${each.key}"
    # Routes this distribution to its folder in the shared bucket.
    # CloudFront prepends this to every S3 request path automatically.
    origin_path              = "/${each.key}"
    origin_access_control_id = aws_cloudfront_origin_access_control.app[each.key].id
  }

  default_cache_behavior {
    target_origin_id       = "s3-${each.key}"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true
    # CachingOptimized managed policy
    cache_policy_id = "658327ea-f89d-4fab-a63d-7e88639e58f6"
    # SecurityHeadersPolicy managed policy
    response_headers_policy_id = "67f7725c-6f97-4210-82d7-5512b31e9d03"
  }

  # SPA routing — 403/404 returns index.html so React Router handles the path.
  # With origin_path set, CloudFront translates /index.html → /{app}/index.html in S3.
  custom_error_response {
    error_code            = 403
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 0
  }
  custom_error_response {
    error_code            = 404
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 0
  }

  viewer_certificate {
    acm_certificate_arn            = var.acm_certificate_arn != "" ? var.acm_certificate_arn : null
    cloudfront_default_certificate = var.acm_certificate_arn == "" ? true : false
    ssl_support_method             = var.acm_certificate_arn != "" ? "sni-only" : null
    minimum_protocol_version       = var.acm_certificate_arn != "" ? "TLSv1.2_2021" : "TLSv1"
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }

  logging_config {
    bucket          = aws_s3_bucket.logs.bucket_domain_name
    prefix          = "${each.key}/"
    include_cookies = false
  }

  tags = { Name = "${var.project_name}-${var.environment}-${each.key}-cf" }
}

# ─── Bucket policy — each distribution may only read its own folder ───────────

resource "aws_s3_bucket_policy" "content" {
  bucket = aws_s3_bucket.content.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      for app_key, dist in aws_cloudfront_distribution.app : {
        Sid       = "AllowCloudFront${replace(title(replace(app_key, "-", " ")), " ", "")}"
        Effect    = "Allow"
        Principal = { Service = "cloudfront.amazonaws.com" }
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.content.arn}/${app_key}/*"
        Condition = {
          StringEquals = { "AWS:SourceArn" = dist.arn }
        }
      }
    ]
  })
}

# ─── SSM parameters — consumed by frontend-deploy.yml ────────────────────────
# Shared bucket name (same value for both apps)

resource "aws_ssm_parameter" "s3_bucket" {
  for_each = var.apps

  name  = "/opsnexus/${var.environment}/${replace(each.key, "-", "_")}/s3_bucket"
  type  = "String"
  value = aws_s3_bucket.content.id
  tags  = { Name = "${var.project_name}-${var.environment}-${each.key}-s3-param" }
}

# Folder prefix — deploy workflow syncs to s3://{bucket}/{prefix}/
resource "aws_ssm_parameter" "s3_prefix" {
  for_each = var.apps

  name  = "/opsnexus/${var.environment}/${replace(each.key, "-", "_")}/s3_prefix"
  type  = "String"
  value = each.key
  tags  = { Name = "${var.project_name}-${var.environment}-${each.key}-prefix-param" }
}

resource "aws_ssm_parameter" "cloudfront_id" {
  for_each = var.apps

  name  = "/opsnexus/${var.environment}/${replace(each.key, "-", "_")}/cloudfront_id"
  type  = "String"
  value = aws_cloudfront_distribution.app[each.key].id
  tags  = { Name = "${var.project_name}-${var.environment}-${each.key}-cf-param" }
}
