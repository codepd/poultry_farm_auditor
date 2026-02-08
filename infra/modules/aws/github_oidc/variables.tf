variable "github_repository" {
  type = string
}

variable "github_branch" {
  type = string
}

variable "role_name" {
  type = string
}

variable "ecr_repository_arn" {
  type = string
}

variable "s3_bucket_arn" {
  type        = string
  description = "ARN of the S3 bucket for frontend deployment"
  default     = ""
}

variable "cloudfront_distribution_id" {
  type        = string
  description = "CloudFront distribution ID for cache invalidation"
  default     = ""
}
