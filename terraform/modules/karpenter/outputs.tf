output "node_pool_name" {
  description = "Name of the default Karpenter NodePool"
  value       = "default"
}

output "ec2_node_class_name" {
  description = "Name of the default Karpenter EC2NodeClass"
  value       = "default"
}

output "helm_release_version" {
  description = "Installed Karpenter Helm chart version"
  value       = helm_release.karpenter.version
}
