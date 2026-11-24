variable "cluster_name" {
  type        = string
  description = "EKS cluster name"
}

variable "eks_node_role_name" {
  type        = string
  description = "IAM role name attached to EKS worker nodes"
}

variable "log_retention_days" {
  type        = number
  description = "CloudWatch Logs retention in days (e.g. 14 or 30)"
  default     = 30
}

variable "tags" {
  type    = map(string)
  default = {}
}
