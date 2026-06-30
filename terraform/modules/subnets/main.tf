terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# ---------------------------------------------------------------------------
# Public subnets (ALB, NAT GW)
# ---------------------------------------------------------------------------
resource "aws_subnet" "public" {
  count = length(var.availability_zones)

  vpc_id                  = var.vpc_id
  cidr_block              = var.public_subnet_cidrs[count.index]
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project_name}-${var.environment}-public-${substr(var.availability_zones[count.index], -2, -1)}"
  }
}

# ---------------------------------------------------------------------------
# Private EKS subnets (nodes + pods)
# ---------------------------------------------------------------------------
resource "aws_subnet" "private_eks" {
  count = length(var.availability_zones)

  vpc_id            = var.vpc_id
  cidr_block        = var.private_eks_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]

  tags = {
    Name                                                               = "${var.project_name}-${var.environment}-private-eks-${substr(var.availability_zones[count.index], -2, -1)}"
    "kubernetes.io/role/internal-elb"                                  = "1"
    "kubernetes.io/cluster/${var.project_name}-${var.environment}-eks" = "owned"
  }
}

# ---------------------------------------------------------------------------
# Private DB subnets
# ---------------------------------------------------------------------------
resource "aws_subnet" "private_db" {
  count = length(var.availability_zones)

  vpc_id            = var.vpc_id
  cidr_block        = var.private_db_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]

  tags = {
    Name = "${var.project_name}-${var.environment}-private-db-${substr(var.availability_zones[count.index], -2, -1)}"
  }
}

# ---------------------------------------------------------------------------
# RDS subnet group
# ---------------------------------------------------------------------------
resource "aws_db_subnet_group" "main" {
  name        = "${var.project_name}-${var.environment}-db-subnet-group"
  description = "Subnet group for ${var.project_name}-${var.environment} RDS instances"
  subnet_ids  = aws_subnet.private_db[*].id

  tags = {
    Name = "${var.project_name}-${var.environment}-db-subnet-group"
  }
}

# ---------------------------------------------------------------------------
# Public NACL
# ---------------------------------------------------------------------------
resource "aws_network_acl" "public" {
  vpc_id     = var.vpc_id
  subnet_ids = aws_subnet.public[*].id

  # Inbound HTTP
  ingress {
    rule_no    = 100
    protocol   = "tcp"
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 80
    to_port    = 80
  }

  # Inbound HTTPS
  ingress {
    rule_no    = 110
    protocol   = "tcp"
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 443
    to_port    = 443
  }

  # Inbound ephemeral ports (return traffic)
  ingress {
    rule_no    = 120
    protocol   = "tcp"
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 1024
    to_port    = 65535
  }

  # Allow all outbound
  egress {
    rule_no    = 100
    protocol   = "-1"
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-public-nacl"
  }
}

# ---------------------------------------------------------------------------
# EKS NACL — all inbound/outbound within VPC CIDR (flexible for pod networking)
# ---------------------------------------------------------------------------
resource "aws_network_acl" "eks" {
  vpc_id     = var.vpc_id
  subnet_ids = aws_subnet.private_eks[*].id

  # Allow all inbound from VPC CIDR
  ingress {
    rule_no    = 100
    protocol   = "-1"
    action     = "allow"
    cidr_block = var.vpc_cidr
    from_port  = 0
    to_port    = 0
  }

  # Allow all outbound to VPC CIDR
  egress {
    rule_no    = 100
    protocol   = "-1"
    action     = "allow"
    cidr_block = var.vpc_cidr
    from_port  = 0
    to_port    = 0
  }

  # Allow outbound to internet (NAT GW traffic exits to 0.0.0.0/0)
  egress {
    rule_no    = 110
    protocol   = "-1"
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  # Allow inbound ephemeral return traffic from internet (via NAT)
  ingress {
    rule_no    = 110
    protocol   = "tcp"
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 1024
    to_port    = 65535
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-eks-nacl"
  }
}

# ---------------------------------------------------------------------------
# DB NACL — MySQL inbound from VPC only, ephemeral outbound to VPC only
# ---------------------------------------------------------------------------
resource "aws_network_acl" "db" {
  vpc_id     = var.vpc_id
  subnet_ids = aws_subnet.private_db[*].id

  # Inbound MySQL from VPC CIDR
  ingress {
    rule_no    = 100
    protocol   = "tcp"
    action     = "allow"
    cidr_block = var.vpc_cidr
    from_port  = 3306
    to_port    = 3306
  }

  # Outbound ephemeral ports back to VPC CIDR (return traffic)
  egress {
    rule_no    = 100
    protocol   = "tcp"
    action     = "allow"
    cidr_block = var.vpc_cidr
    from_port  = 1024
    to_port    = 65535
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-db-nacl"
  }
}
