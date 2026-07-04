data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

# ─── KMS key for EKS secrets ───────────────────────────────────────────────
resource "aws_kms_key" "eks" {
  description             = "EKS secrets encryption for ${var.project_name}-${var.environment}"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  tags                    = { Name = "${var.project_name}-${var.environment}-eks-kms" }
}

resource "aws_kms_alias" "eks" {
  name          = "alias/${var.project_name}-${var.environment}-eks"
  target_key_id = aws_kms_key.eks.key_id
}

# ─── IAM: Cluster role ─────────────────────────────────────────────────────
resource "aws_iam_role" "cluster" {
  name = "${var.project_name}-${var.environment}-eks-cluster-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "eks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-eks-cluster-role" }
}

resource "aws_iam_role_policy_attachment" "cluster_policy" {
  role       = aws_iam_role.cluster.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonEKSClusterPolicy"
}

resource "aws_iam_role_policy_attachment" "cluster_vpc_resource" {
  role       = aws_iam_role.cluster.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonEKSVPCResourceController"
}

# ─── IAM: Node group role ──────────────────────────────────────────────────
resource "aws_iam_role" "node" {
  name = "${var.project_name}-${var.environment}-eks-node-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-eks-node-role" }
}

resource "aws_iam_role_policy_attachment" "node_worker" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}

resource "aws_iam_role_policy_attachment" "node_cni" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "node_ecr" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_role_policy_attachment" "node_ssm" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# ─── Security Group: additional node SG for Karpenter ─────────────────────
resource "aws_security_group" "eks_nodes" {
  name        = "${var.project_name}-${var.environment}-eks-nodes-sg"
  description = "Security group for EKS nodes managed by Karpenter"
  vpc_id      = var.vpc_id

  ingress {
    description = "Node to node all traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }


  egress {
    description = "All outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project_name}-${var.environment}-eks-nodes-sg" }
}

# Allow NLB health checks and traffic to reach Traefik on NodePort 30080.
# Targets the EKS cluster security group (auto-created by EKS and attached to
# all nodes) rather than the additional node SG, which is not auto-attached.
resource "aws_security_group_rule" "traefik_nodeport" {
  security_group_id = aws_eks_cluster.main.vpc_config[0].cluster_security_group_id
  type              = "ingress"
  from_port         = 30080
  to_port           = 30080
  protocol          = "tcp"
  cidr_blocks       = [var.vpc_cidr]
  description       = "NLB to Traefik NodePort 30080"

  depends_on = [aws_eks_cluster.main]
}

# ─── EKS Cluster ──────────────────────────────────────────────────────────
resource "aws_eks_cluster" "main" {
  name     = "${var.project_name}-${var.environment}-eks"
  version  = var.cluster_version
  role_arn = aws_iam_role.cluster.arn

  vpc_config {
    subnet_ids              = var.private_eks_subnet_ids
    endpoint_private_access = true
    endpoint_public_access  = var.enable_public_endpoint
    public_access_cidrs     = var.enable_public_endpoint ? var.public_access_cidrs : []
  }

  access_config {
    # API_AND_CONFIG_MAP keeps existing aws-auth ConfigMap entries working while
    # enabling the modern Access Entries API. Transition is one-way (can go up
    # to API-only later, but never back to CONFIG_MAP).
    # bootstrap=false so EKS does NOT auto-create an access entry for the cluster
    # creator — Terraform owns all entries via aws_eks_access_entry exclusively.
    authentication_mode                         = "API_AND_CONFIG_MAP"
    bootstrap_cluster_creator_admin_permissions = false
  }

  encryption_config {
    provider {
      key_arn = aws_kms_key.eks.arn
    }
    resources = ["secrets"]
  }

  enabled_cluster_log_types = [
    "api", "audit", "authenticator", "controllerManager", "scheduler"
  ]

  tags = { Name = "${var.project_name}-${var.environment}-eks" }

  depends_on = [
    aws_iam_role_policy_attachment.cluster_policy,
    aws_iam_role_policy_attachment.cluster_vpc_resource,
  ]

  lifecycle {
    # bootstrap_cluster_creator_admin_permissions is write-once (creation only).
    # Changing it after cluster creation forces replacement — ignore to prevent that.
    ignore_changes = [access_config[0].bootstrap_cluster_creator_admin_permissions]
  }
}

# ─── IRSA: EBS CSI Driver ──────────────────────────────────────────────────
resource "aws_iam_role" "ebs_csi_irsa" {
  name = "${var.project_name}-${var.environment}-ebs-csi-irsa"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.eks.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_provider_url_stripped}:sub" = "system:serviceaccount:kube-system:ebs-csi-controller-sa"
          "${local.oidc_provider_url_stripped}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-ebs-csi-irsa" }
}

resource "aws_iam_role_policy_attachment" "ebs_csi_irsa" {
  role       = aws_iam_role.ebs_csi_irsa.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

# ─── IRSA: VPC CNI ─────────────────────────────────────────────────────────
resource "aws_iam_role" "vpc_cni_irsa" {
  name = "${var.project_name}-${var.environment}-vpc-cni-irsa"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.eks.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_provider_url_stripped}:sub" = "system:serviceaccount:kube-system:aws-node"
          "${local.oidc_provider_url_stripped}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-vpc-cni-irsa" }
}

resource "aws_iam_role_policy_attachment" "vpc_cni_irsa" {
  role       = aws_iam_role.vpc_cni_irsa.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonEKS_CNI_Policy"
}

# ─── EKS Add-ons ──────────────────────────────────────────────────────────
# IRSA roles wired to the add-ons that make AWS API calls.
# Without service_account_role_arn, pods fall back to IMDS — blocked by
# http_put_response_hop_limit = 1 on our launch template (IMDSv2 enforcement).
locals {
  eks_addons = {
    vpc-cni = {
      service_account_role_arn = aws_iam_role.vpc_cni_irsa.arn
    }
    coredns = {
      service_account_role_arn = null
    }
    kube-proxy = {
      service_account_role_arn = null
    }
    aws-ebs-csi-driver = {
      service_account_role_arn = aws_iam_role.ebs_csi_irsa.arn
    }
  }
}

resource "aws_eks_addon" "addons" {
  for_each = local.eks_addons

  cluster_name                = aws_eks_cluster.main.name
  addon_name                  = each.key
  service_account_role_arn    = each.value.service_account_role_arn
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"

  depends_on = [aws_eks_node_group.system]
}

# ─── Launch template for system nodes (IMDSv2 + encrypted EBS) ────────────
resource "aws_launch_template" "system" {
  name_prefix = "${var.project_name}-${var.environment}-system-"

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
    instance_metadata_tags      = "enabled"
  }

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size           = 50
      volume_type           = "gp3"
      encrypted             = true
      delete_on_termination = true
    }
  }

  tag_specifications {
    resource_type = "instance"
    tags          = { Name = "${var.project_name}-${var.environment}-system-node" }
  }

  lifecycle {
    create_before_destroy = true
  }
}

# ─── Managed Node Group: system ───────────────────────────────────────────
resource "aws_eks_node_group" "system" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "${var.project_name}-${var.environment}-system"
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = var.private_eks_subnet_ids
  instance_types  = var.system_node_instance_types
  capacity_type   = "ON_DEMAND"

  scaling_config {
    min_size     = 2
    max_size     = 4
    desired_size = var.system_node_desired
  }

  update_config {
    max_unavailable = 1
  }

  launch_template {
    id      = aws_launch_template.system.id
    version = aws_launch_template.system.latest_version
  }

  labels = {
    "node-type" = "system"
  }

  tags = { Name = "${var.project_name}-${var.environment}-system-ng" }

  depends_on = [
    aws_iam_role_policy_attachment.node_worker,
    aws_iam_role_policy_attachment.node_cni,
    aws_iam_role_policy_attachment.node_ecr,
    aws_iam_role_policy_attachment.node_ssm,
  ]

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }
}

# ─── OIDC Provider for IRSA ───────────────────────────────────────────────
data "tls_certificate" "eks" {
  url = aws_eks_cluster.main.identity[0].oidc[0].issuer
}

resource "aws_iam_openid_connect_provider" "eks" {
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.eks.certificates[0].sha1_fingerprint]
  url             = aws_eks_cluster.main.identity[0].oidc[0].issuer
  tags            = { Name = "${var.project_name}-${var.environment}-eks-oidc" }
}

# ─── Karpenter IAM role (IRSA) ─────────────────────────────────────────────
locals {
  oidc_provider_url_stripped = replace(aws_eks_cluster.main.identity[0].oidc[0].issuer, "https://", "")
}

resource "aws_iam_role" "karpenter_controller" {
  name = "${var.project_name}-${var.environment}-karpenter-controller"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_openid_connect_provider.eks.arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_provider_url_stripped}:sub" = "system:serviceaccount:karpenter:karpenter"
          "${local.oidc_provider_url_stripped}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = { Name = "${var.project_name}-${var.environment}-karpenter-controller" }
}

resource "aws_iam_role_policy" "karpenter_controller" {
  name = "karpenter-controller-policy"
  role = aws_iam_role.karpenter_controller.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowEC2Actions"
        Effect = "Allow"
        Action = [
          "ec2:CreateFleet",
          "ec2:CreateLaunchTemplate",
          "ec2:CreateTags",
          "ec2:DeleteLaunchTemplate",
          "ec2:DescribeAvailabilityZones",
          "ec2:DescribeImages",
          "ec2:DescribeInstances",
          "ec2:DescribeInstanceTypeOfferings",
          "ec2:DescribeInstanceTypes",
          "ec2:DescribeLaunchTemplates",
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeSpotPriceHistory",
          "ec2:DescribeSubnets",
          "ec2:RunInstances",
          "ec2:TerminateInstances",
        ]
        Resource = "*"
      },
      {
        Sid      = "AllowPassRole"
        Effect   = "Allow"
        Action   = ["iam:PassRole"]
        Resource = aws_iam_role.node.arn
      },
      {
        Sid    = "AllowSQS"
        Effect = "Allow"
        Action = [
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
          "sqs:ReceiveMessage",
        ]
        Resource = aws_sqs_queue.karpenter_interruption.arn
      },
      {
        Sid    = "AllowEKS"
        Effect = "Allow"
        Action = [
          "eks:DescribeCluster",
        ]
        Resource = aws_eks_cluster.main.arn
      },
    ]
  })
}

# ─── Karpenter SQS queue for spot interruption handling ────────────────────
resource "aws_sqs_queue" "karpenter_interruption" {
  name                      = "${var.project_name}-${var.environment}-karpenter-interruption"
  message_retention_seconds = 300
  sqs_managed_sse_enabled   = true
  tags                      = { Name = "${var.project_name}-${var.environment}-karpenter-interruption" }
}

resource "aws_sqs_queue_policy" "karpenter_interruption" {
  queue_url = aws_sqs_queue.karpenter_interruption.url
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = ["events.amazonaws.com", "sqs.amazonaws.com"] }
      Action    = "sqs:SendMessage"
      Resource  = aws_sqs_queue.karpenter_interruption.arn
    }]
  })
}

# ─── EventBridge rules for spot interruption ───────────────────────────────
locals {
  interruption_events = {
    spot_interruption        = { source = ["aws.ec2"], detail_type = ["EC2 Spot Instance Interruption Warning"] }
    rebalance_recommendation = { source = ["aws.ec2"], detail_type = ["EC2 Instance Rebalance Recommendation"] }
    instance_state_change    = { source = ["aws.ec2"], detail_type = ["EC2 Instance State-change Notification"] }
  }
}

resource "aws_cloudwatch_event_rule" "karpenter" {
  for_each    = local.interruption_events
  name        = "${var.project_name}-${var.environment}-karpenter-${each.key}"
  description = "Karpenter interruption handling: ${each.key}"

  event_pattern = jsonencode({
    source      = each.value.source
    detail-type = each.value.detail_type
  })

  tags = { Name = "${var.project_name}-${var.environment}-karpenter-${each.key}" }
}

resource "aws_cloudwatch_event_target" "karpenter" {
  for_each  = aws_cloudwatch_event_rule.karpenter
  rule      = each.value.name
  target_id = "karpenter-interruption-sqs"
  arn       = aws_sqs_queue.karpenter_interruption.arn
}

# ─── EKS Access Entries — IAM principals with cluster-admin ───────────────────
# Uses the modern EKS Access Entries API (EKS 1.23+) instead of aws-auth ConfigMap.
# Each entry grants AmazonEKSClusterAdminPolicy (equivalent to cluster-admin RBAC).
resource "aws_eks_access_entry" "admins" {
  for_each = toset(var.admin_iam_arns)

  cluster_name  = aws_eks_cluster.main.name
  principal_arn = each.value
  type          = "STANDARD"

  tags = { Name = "${var.project_name}-${var.environment}-eks-admin-entry" }
}

resource "aws_eks_access_policy_association" "admins" {
  for_each = toset(var.admin_iam_arns)

  cluster_name  = aws_eks_cluster.main.name
  principal_arn = each.value
  policy_arn    = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"

  access_scope {
    type = "cluster"
  }

  depends_on = [aws_eks_access_entry.admins]
}
