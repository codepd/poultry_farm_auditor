variable "aws_region" {
  type    = string
  default = "ap-south-1"
}

variable "name" {
  type    = string
  default = "poultry-dev"
}

variable "vpc_cidr" {
  type    = string
  default = "10.10.0.0/16"
}

variable "public_subnet_cidrs" {
  type    = list(string)
  default = ["10.10.0.0/20", "10.10.16.0/20"]
}

variable "private_subnet_cidrs" {
  type    = list(string)
  default = ["10.10.32.0/20", "10.10.48.0/20"]
}

variable "eks_node_instance_type" {
  type    = string
  default = "t3.small"
}

variable "eks_node_desired" {
  type    = number
  default = 1
}

variable "eks_node_min" {
  type    = number
  default = 1
}

variable "eks_node_max" {
  type    = number
  default = 2
}

variable "db_name" {
  type    = string
  default = "poultry_farm"
}

variable "db_username" {
  type    = string
  default = "poultry_admin"
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_allocated_storage" {
  type    = number
  default = 20
}

variable "frontend_bucket_name" {
  type    = string
  default = "poultry-dev-frontend-replace-me"
}

variable "frontend_domain_name" {
  type    = string
  default = "app-dev.mykolipannai.com"
}

variable "api_domain_name" {
  type    = string
  default = "api-dev.mykolipannai.com"
}

variable "tags" {
  type = map(string)
  default = {
    environment = "dev"
    project     = "poultry"
  }
}

variable "bastion_instance_type" {
  type    = string
  default = "t3.micro"
}

variable "ecr_repository_name" {
  type    = string
  default = "poultry-dev-go-api"
}

variable "github_repository" {
  type    = string
  default = "codepd/poultry_farm_auditor"
}

variable "github_branch" {
  type    = string
  default = "main"
}

variable "github_role_name" {
  type    = string
  default = "poultry-dev-github-actions"
}

variable "eks_endpoint_public_access" {
  type    = bool
  default = false
}

variable "eks_public_access_cidrs" {
  type    = list(string)
  default = []
}
