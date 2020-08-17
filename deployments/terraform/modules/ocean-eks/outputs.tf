output "cluster_endpoint" {
  description = "The endpoint for your EKS Kubernetes API"
  value       = module.eks.cluster_endpoint
}

output "cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster"
  value       = module.eks.cluster_security_group_id
}

output "kubeconfig" {
  description = "kubectl config file contents for this EKS cluster"
  value       = module.eks.kubeconfig
}

output "kubeconfig_filename" {
  description = "The filename of the generated kubectl config"
  value       = module.eks.kubeconfig_filename
}

output "config_map_aws_auth" {
  description = "A Kubernetes configuration to authenticate to this EKS cluster"
  value       = module.eks.config_map_aws_auth
}

output "worker_iam_role_arn" {
  description = "Default IAM role ARN for EKS worker groups"
  value       = module.eks.worker_iam_role_arn
}

output "cluster_id" {
  description = "The name/id of the EKS cluster"
  value       = module.eks.cluster_id
}
