variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "github-event-system-eks"
}

variable "worker_security_group_id" {
  description = "Security group ID for the worker nodes"
  type        = string
}

variable "vpc_id" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}
