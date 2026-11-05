variable "bucket_name" {
  type        = string
  description = "Name of the S3 bucket for storing ledgers, feed price sheets, and parsed audit documents"
}

variable "eks_node_role_name" {
  type        = string
  description = "IAM role name of the EKS worker nodes to grant S3 read/write permissions"
  default     = ""
}

variable "github_actions_role_name" {
  type        = string
  description = "IAM role name of GitHub Actions OIDC to grant S3 read/write permissions"
  default     = ""
}

variable "tags" {
  type    = map(string)
  default = {}
}
