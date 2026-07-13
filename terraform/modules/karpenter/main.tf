################################################################################
# Karpenter Module — OpsNexus
#
# Installs Karpenter via Helm and configures a default NodePool +
# EC2NodeClass so the cluster can auto-provision worker nodes.
#
# Prerequisites (created by the EKS module):
#   - karpenter_controller IAM role (IRSA)
#   - karpenter_interruption SQS queue
#   - aws-auth entry granting the node role cluster access
################################################################################

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

data "aws_region" "current" {}

data "aws_iam_role" "node" {
  name = var.node_role_name
}

################################################################################
# Helm — install Karpenter into the karpenter namespace
################################################################################

resource "helm_release" "karpenter" {
  name             = "karpenter"
  namespace        = "karpenter"
  create_namespace = true

  repository = "oci://public.ecr.aws/karpenter"
  chart      = "karpenter"
  version    = var.karpenter_version

  set {
    name  = "settings.clusterName"
    value = var.cluster_name
  }

  set {
    name  = "settings.interruptionQueue"
    value = var.karpenter_interruption_queue_name
  }

  set {
    name  = "serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
    value = var.karpenter_role_arn
  }

  set {
    name  = "controller.resources.requests.cpu"
    value = "250m"
  }

  set {
    name  = "controller.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "controller.resources.limits.cpu"
    value = "1"
  }

  set {
    name  = "controller.resources.limits.memory"
    value = "1Gi"
  }
}

################################################################################
# EC2NodeClass — defines the AWS-side config for provisioned nodes
################################################################################

resource "kubernetes_manifest" "ec2_node_class" {
  manifest = {
    apiVersion = "karpenter.k8s.aws/v1"
    kind       = "EC2NodeClass"
    metadata = {
      name = "default"
    }
    spec = {
      amiSelectorTerms = [
        { alias = "al2023@latest" }
      ]
      role = var.node_role_name
      subnetSelectorTerms = [
        {
          tags = {
            "kubernetes.io/cluster/${var.cluster_name}" = "owned"
            "karpenter.sh/discovery"                    = var.cluster_name
          }
        }
      ]
      securityGroupSelectorTerms = [
        {
          tags = {
            "kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      ]
      tags = {
        Project     = var.project_name
        Environment = var.environment
        ManagedBy   = "Karpenter"
      }
    }
  }

  depends_on = [helm_release.karpenter]
}

################################################################################
# NodePool — defines scheduling constraints and limits
################################################################################

resource "kubernetes_manifest" "node_pool" {
  manifest = {
    apiVersion = "karpenter.sh/v1"
    kind       = "NodePool"
    metadata = {
      name = "default"
    }
    spec = {
      template = {
        spec = {
          nodeClassRef = {
            group = "karpenter.k8s.aws"
            kind  = "EC2NodeClass"
            name  = "default"
          }
          requirements = [
            {
              key      = "kubernetes.io/arch"
              operator = "In"
              values   = var.node_arch
            },
            {
              key      = "karpenter.sh/capacity-type"
              operator = "In"
              values   = var.node_capacity_types
            },
            {
              key      = "karpenter.k8s.aws/instance-family"
              operator = "In"
              values   = var.node_instance_families
            },
            {
              key      = "karpenter.k8s.aws/instance-size"
              operator = "In"
              values   = var.node_instance_sizes
            },
          ]
        }
      }
      limits = {
        cpu    = "100"
        memory = "400Gi"
      }
      disruption = {
        consolidationPolicy = "WhenEmptyOrUnderutilized"
        consolidateAfter    = "30s"
      }
    }
  }

  depends_on = [kubernetes_manifest.ec2_node_class]
}
