terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
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
  region = var.aws_region
  default_tags { tags = local.default_tags }
}

provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
  default_tags { tags = local.default_tags }
}

provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_ca_certificate)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--region", var.aws_region]
  }
}

provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_ca_certificate)
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name, "--region", var.aws_region]
    }
  }
}

locals {
  default_tags = merge(
    { "caylent:owner" = var.caylent_owner },
    var.caylent_customer != "" ? { "caylent:customer" = var.caylent_customer } : {},
    var.caylent_workload != "" ? { "caylent:workload" = var.caylent_workload } : {},
    var.caylent_project  != "" ? { "caylent:project"  = var.caylent_project  } : {},
    {
      Environment = var.environment
      ManagedBy   = "Terraform"
      Project      = var.project_name
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
  private_eks_subnet_ids     = module.subnets.private_eks_subnet_ids
  cluster_version            = var.cluster_version
  enable_public_endpoint     = var.enable_public_endpoint
  public_access_cidrs        = var.public_access_cidrs
  system_node_instance_types = var.system_node_instance_types
  system_node_desired        = var.system_node_desired

  depends_on = [module.routing]
}

module "karpenter" {
  source                            = "../../modules/karpenter"
  project_name                      = var.project_name
  environment                       = var.environment
  cluster_name                      = module.eks.cluster_name
  cluster_endpoint                  = module.eks.cluster_endpoint
  cluster_ca_certificate            = module.eks.cluster_ca_certificate
  karpenter_role_arn                = module.eks.karpenter_role_arn
  karpenter_interruption_queue_name = module.eks.karpenter_interruption_queue_name
  node_role_name                    = module.eks.node_role_name
  karpenter_version                 = var.karpenter_version

  depends_on = [module.eks]
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

module "rds" {
  source                      = "../../modules/rds"
  project_name                = var.project_name
  environment                 = var.environment
  vpc_id                      = module.vpc.vpc_id
  private_db_subnet_ids       = module.subnets.private_db_subnet_ids
  eks_node_sg_id              = module.eks.node_sg_id
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

module "eso" {
  source            = "../../modules/eso"
  project_name      = var.project_name
  environment       = var.environment
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url

  depends_on = [module.karpenter]
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

resource "aws_autoscaling_attachment" "traefik_nlb" {
  autoscaling_group_name = module.eks.system_node_group_asg_name
  lb_target_group_arn    = module.api_gateway.traefik_target_group_arn
}

# ─── ACM Certificates ─────────────────────────────────────────────────────
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

# ─── Route53 A records ────────────────────────────────────────────────────
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
