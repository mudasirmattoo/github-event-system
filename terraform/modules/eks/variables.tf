variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "github-event-system-eks"
}

variable "vpc_id" {
  description = "VPC ID where the cluster will be deployed"
  type        = string
}

variable "private_subnets" {
  description = "List of private subnet IDs for the EKS cluster"
  type        = list(string)
}

variable "worker_security_group_id" {
  description = "Security group ID for the worker nodes"
  type        = string
}
