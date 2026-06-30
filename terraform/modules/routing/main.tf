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
# Elastic IPs for NAT Gateways
# One per AZ when enable_nat_per_az = true, otherwise a single shared EIP.
# ---------------------------------------------------------------------------
resource "aws_eip" "nat" {
  count  = var.enable_nat_per_az ? length(var.availability_zones) : 1
  domain = "vpc"

  tags = {
    Name = "${var.project_name}-${var.environment}-nat-eip-${count.index + 1}"
  }
}

# ---------------------------------------------------------------------------
# NAT Gateways
# Placed in the first public subnet (single) or spread across all public
# subnets (one per AZ). Explicitly depend on the IGW so the gateway is
# attached before NAT GW creation is attempted.
# ---------------------------------------------------------------------------
resource "aws_nat_gateway" "main" {
  count = var.enable_nat_per_az ? length(var.availability_zones) : 1

  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = var.enable_nat_per_az ? var.public_subnet_ids[count.index] : var.public_subnet_ids[0]

  tags = {
    Name = "${var.project_name}-${var.environment}-nat-${count.index + 1}"
  }

  depends_on = [var.igw_id]
}

# ---------------------------------------------------------------------------
# Public route table — 0.0.0.0/0 via IGW
# ---------------------------------------------------------------------------
resource "aws_route_table" "public" {
  vpc_id = var.vpc_id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = var.igw_id
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-public-rt"
  }
}

resource "aws_route_table_association" "public" {
  count = length(var.public_subnet_ids)

  subnet_id      = var.public_subnet_ids[count.index]
  route_table_id = aws_route_table.public.id
}

# ---------------------------------------------------------------------------
# Private EKS route tables — one per AZ, 0.0.0.0/0 via appropriate NAT GW
# ---------------------------------------------------------------------------
resource "aws_route_table" "private_eks" {
  count  = length(var.availability_zones)
  vpc_id = var.vpc_id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main[var.enable_nat_per_az ? count.index : 0].id
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-private-eks-rt-${count.index + 1}"
  }
}

resource "aws_route_table_association" "private_eks" {
  count = length(var.private_eks_subnet_ids)

  subnet_id      = var.private_eks_subnet_ids[count.index]
  route_table_id = aws_route_table.private_eks[count.index].id
}

# ---------------------------------------------------------------------------
# Private DB route tables — one per AZ, NO internet route (fully isolated)
# ---------------------------------------------------------------------------
resource "aws_route_table" "private_db" {
  count  = length(var.availability_zones)
  vpc_id = var.vpc_id

  # Intentionally no 0.0.0.0/0 route — DB subnets are fully isolated from
  # the internet. Only local VPC traffic is implicitly routed.

  tags = {
    Name = "${var.project_name}-${var.environment}-private-db-rt-${count.index + 1}"
  }
}

resource "aws_route_table_association" "private_db" {
  count = length(var.private_db_subnet_ids)

  subnet_id      = var.private_db_subnet_ids[count.index]
  route_table_id = aws_route_table.private_db[count.index].id
}
