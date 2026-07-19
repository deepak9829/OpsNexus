terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
  backend "s3" {}
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile != "" ? var.aws_profile : null
  default_tags { tags = local.default_tags }
}

provider "aws" {
  alias   = "us_east_1"
  region  = "us-east-1"
  profile = var.aws_profile != "" ? var.aws_profile : null
  default_tags { tags = local.default_tags }
}


locals {
  default_tags = merge(
    { Owner = var.owner },
    var.customer != "" ? { Customer = var.customer } : {},
    var.workload != "" ? { Workload = var.workload } : {},
    var.project != "" ? { Project = var.project } : {},
    {
      Environment = var.environment
      ManagedBy   = "Terraform"
      Project     = var.project_name
    }
  )
}

# ─── Networking ───────────────────────────────────────────────────────────
module "vpc" {
  source       = "../../modules/vpc"
  project_name = var.project_name
  environment  = var.environment
  vpc_cidr     = var.vpc_cidr
}

module "subnets" {
  source                   = "../../modules/subnets"
  project_name             = var.project_name
  environment              = var.environment
  vpc_id                   = module.vpc.vpc_id
  vpc_cidr                 = var.vpc_cidr
  availability_zones       = var.availability_zones
  public_subnet_cidrs      = var.public_subnet_cidrs
  private_eks_subnet_cidrs = var.private_eks_subnet_cidrs
  private_db_subnet_cidrs  = var.private_db_subnet_cidrs
}

module "routing" {
  source                 = "../../modules/routing"
  project_name           = var.project_name
  environment            = var.environment
  vpc_id                 = module.vpc.vpc_id
  igw_id                 = module.vpc.igw_id
  availability_zones     = var.availability_zones
  public_subnet_ids      = module.subnets.public_subnet_ids
  private_eks_subnet_ids = module.subnets.private_eks_subnet_ids
  private_db_subnet_ids  = module.subnets.private_db_subnet_ids
  enable_nat_per_az      = var.enable_nat_per_az
}

# ─── Compute: EKS ─────────────────────────────────────────────────────────
module "eks" {
  source                     = "../../modules/eks"
  project_name               = var.project_name
  environment                = var.environment
  vpc_id                     = module.vpc.vpc_id
  vpc_cidr                   = var.vpc_cidr
  private_eks_subnet_ids     = module.subnets.private_eks_subnet_ids
  cluster_version            = var.cluster_version
  enable_public_endpoint     = var.enable_public_endpoint
  public_access_cidrs        = var.public_access_cidrs
  system_node_instance_types = var.system_node_instance_types
  system_node_desired        = var.system_node_desired
  admin_iam_arns             = var.admin_iam_arns

  depends_on = [module.routing]
}


# ─── Data: ECR, RDS, DynamoDB ─────────────────────────────────────────────
module "ecr" {
  source       = "../../modules/ecr"
  project_name = var.project_name
  environment  = var.environment
}

module "dynamodb" {
  source       = "../../modules/dynamodb"
  project_name = var.project_name
  environment  = var.environment
}

module "documentdb" {
  source                  = "../../modules/documentdb"
  project_name            = var.project_name
  environment             = var.environment
  vpc_id                  = module.vpc.vpc_id
  private_db_subnet_ids   = module.subnets.private_db_subnet_ids
  eks_node_sg_id          = module.eks.cluster_sg_id
  instance_class          = var.docdb_instance_class
  instance_count          = var.docdb_instance_count
  deletion_protection     = var.deletion_protection
  skip_final_snapshot     = var.skip_final_snapshot
  backup_retention_period = var.backup_retention_period

  depends_on = [module.eks]
}

module "app_irsa" {
  source            = "../../modules/app_irsa"
  project_name      = var.project_name
  environment       = var.environment
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url

  depends_on = [module.eks]
}

module "rds" {
  source                      = "../../modules/rds"
  project_name                = var.project_name
  environment                 = var.environment
  vpc_id                      = module.vpc.vpc_id
  private_db_subnet_ids       = module.subnets.private_db_subnet_ids
  eks_node_sg_id              = module.eks.cluster_sg_id
  db_instance_class           = var.db_instance_class
  db_allocated_storage        = var.db_allocated_storage
  db_max_allocated_storage    = var.db_max_allocated_storage
  multi_az                    = var.multi_az
  deletion_protection         = var.deletion_protection
  skip_final_snapshot         = var.skip_final_snapshot
  backup_retention_period     = var.backup_retention_period
  enable_performance_insights = var.enable_performance_insights

  depends_on = [module.eks]
}

module "app_secrets" {
  source       = "../../modules/app_secrets"
  project_name = var.project_name
  environment  = var.environment
}

module "eso_irsa" {
  source            = "../../modules/eso_irsa"
  project_name      = var.project_name
  environment       = var.environment
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url

  depends_on = [module.eks]
}

# ─── API Gateway + Lambda Authorizer ──────────────────────────────────────
module "lambda_authorizer" {
  source         = "../../modules/lambda_authorizer"
  project_name   = var.project_name
  environment    = var.environment
  jwt_secret_arn = module.app_secrets.jwt_secret_arn
}

module "api_gateway" {
  source                   = "../../modules/api_gateway"
  project_name             = var.project_name
  environment              = var.environment
  vpc_id                   = module.vpc.vpc_id
  private_eks_subnet_ids   = module.subnets.private_eks_subnet_ids
  lambda_invoke_arn        = module.lambda_authorizer.lambda_invoke_arn
  lambda_function_name     = module.lambda_authorizer.lambda_function_name
  traefik_node_port        = 30080
  regional_certificate_arn = module.dns.regional_certificate_arn
  domain_name              = var.domain_name

  depends_on = [module.eks]
}

# Wire EKS node ASG → NLB target group so nodes auto-register/deregister
# as Karpenter scales the cluster. This runs outside the api_gateway module
# to avoid a circular dependency (api_gateway doesn't need to know EKS internals).
resource "aws_autoscaling_attachment" "traefik_nlb" {
  autoscaling_group_name = module.eks.system_node_group_asg_name
  lb_target_group_arn    = module.api_gateway.traefik_target_group_arn
}

# ─── Karpenter node → database access ────────────────────────────────────
# dev/main.tf passes module.eks.cluster_sg_id to the documentdb and rds
# modules, which covers system managed-node-group pods. Karpenter-provisioned
# nodes use the custom eks_nodes SG (selected by EC2NodeClass tag), so they
# need their own ingress rules on both database security groups.
resource "aws_security_group_rule" "docdb_from_karpenter_nodes" {
  security_group_id        = module.documentdb.security_group_id
  type                     = "ingress"
  from_port                = 27017
  to_port                  = 27017
  protocol                 = "tcp"
  source_security_group_id = module.eks.node_sg_id
  description              = "Karpenter nodes to DocumentDB"
}

resource "aws_security_group_rule" "rds_from_karpenter_nodes" {
  security_group_id        = module.rds.rds_sg_id
  type                     = "ingress"
  from_port                = 3306
  to_port                  = 3306
  protocol                 = "tcp"
  source_security_group_id = module.eks.node_sg_id
  description              = "Karpenter nodes to RDS MySQL"
}

# ─── ACM Certificates ─────────────────────────────────────────────────────
# Created before frontend so the cert ARN can be passed to CloudFront.
# Route53 A records are created AFTER frontend to avoid the cycle:
#   dns needs CF domain → frontend needs cert ARN → dns
module "dns" {
  source       = "../../modules/dns"
  project_name = var.project_name
  environment  = var.environment
  domain_name  = var.domain_name

  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }
}

# ─── Frontends: S3 + CloudFront ───────────────────────────────────────────
# Single bucket with two folder-scoped CloudFront distributions.
module "frontend" {
  source              = "../../modules/frontend"
  project_name        = var.project_name
  environment         = var.environment
  acm_certificate_arn = module.dns.cloudfront_certificate_arn

  apps = {
    "customer-portal" = { domain_aliases = ["app.${var.environment}.${var.domain_name}"] }
    "admin-console"   = { domain_aliases = ["admin.${var.environment}.${var.domain_name}"] }
  }
}

# ─── Route53 A records — created after frontend to break the cycle ────────
resource "aws_route53_record" "app" {
  zone_id = module.dns.hosted_zone_id
  name    = "app.${module.dns.env_prefix}${var.domain_name}"
  type    = "A"
  alias {
    name                   = module.frontend.distribution_domain_names["customer-portal"]
    zone_id                = module.frontend.distribution_hosted_zone_ids["customer-portal"]
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "admin" {
  zone_id = module.dns.hosted_zone_id
  name    = "admin.${module.dns.env_prefix}${var.domain_name}"
  type    = "A"
  alias {
    name                   = module.frontend.distribution_domain_names["admin-console"]
    zone_id                = module.frontend.distribution_hosted_zone_ids["admin-console"]
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "api" {
  zone_id = module.dns.hosted_zone_id
  name    = "api.${module.dns.env_prefix}${var.domain_name}"
  type    = "A"
  alias {
    name                   = module.api_gateway.api_custom_domain_regional_name
    zone_id                = module.api_gateway.api_custom_domain_regional_zone_id
    evaluate_target_health = false
  }
}
