terraform {
  required_providers {
    aws = {
      source                = "hashicorp/aws"
      version               = "~> 5.0"
      configuration_aliases = [aws.us_east_1]
    }
  }
}

data "aws_route53_zone" "main" {
  name         = var.domain_name
  private_zone = false
}

locals {
  env_prefix = var.environment == "prod" ? "" : "${var.environment}."
}

# ─── ACM cert for CloudFront (must be us-east-1) ──────────────────────────────
# domain_name uses env-specific wildcard so "app.dev.opsnexus.site" is covered
# by "*.dev.opsnexus.site". A top-level "*.opsnexus.site" only covers one level
# deep and would NOT match two-level dev subdomains.

resource "aws_acm_certificate" "cloudfront" {
  provider                  = aws.us_east_1
  domain_name               = "*.${local.env_prefix}${var.domain_name}"
  subject_alternative_names = local.env_prefix == "" ? [var.domain_name] : ["*.${var.domain_name}", var.domain_name]
  validation_method         = "DNS"
  tags                      = { Name = "${var.project_name}-${var.environment}-cf-cert" }
  lifecycle { create_before_destroy = true }
}

# ─── ACM cert for API Gateway (regional) ─────────────────────────────────────

resource "aws_acm_certificate" "regional" {
  domain_name               = "${local.env_prefix}${var.domain_name}"
  subject_alternative_names = ["*.${local.env_prefix}${var.domain_name}"]
  validation_method         = "DNS"
  tags                      = { Name = "${var.project_name}-${var.environment}-regional-cert" }
  lifecycle { create_before_destroy = true }
}

# ─── DNS validation records ───────────────────────────────────────────────────
# Kept separate per cert (not merged) so for_each keys stay stable at plan time.
# allow_overwrite = true handles the case where both certs share the same CNAME
# (ACM wildcard + apex SAN reuse one record).

resource "aws_route53_record" "cloudfront_validation" {
  for_each = {
    for dvo in aws_acm_certificate.cloudfront.domain_validation_options :
    dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  ttl             = 60
  type            = each.value.type
  zone_id         = data.aws_route53_zone.main.zone_id
}

resource "aws_route53_record" "regional_validation" {
  for_each = {
    for dvo in aws_acm_certificate.regional.domain_validation_options :
    dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  ttl             = 60
  type            = each.value.type
  zone_id         = data.aws_route53_zone.main.zone_id
}

resource "aws_acm_certificate_validation" "cloudfront" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.cloudfront.arn
  validation_record_fqdns = [for r in aws_route53_record.cloudfront_validation : r.fqdn]
}

resource "aws_acm_certificate_validation" "regional" {
  certificate_arn         = aws_acm_certificate.regional.arn
  validation_record_fqdns = [for r in aws_route53_record.regional_validation : r.fqdn]
}
