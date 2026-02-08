provider "aws" {
  region = var.aws_region
}

provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

module "vpc" {
  source                  = "../../../modules/aws/vpc"
  name                    = var.name
  cidr_block              = var.vpc_cidr
  public_subnet_cidrs     = var.public_subnet_cidrs
  private_subnet_cidrs    = var.private_subnet_cidrs
  kubernetes_cluster_name = "${var.name}-eks"
  tags                    = var.tags
}

module "eks" {
  source                 = "../../../modules/aws/eks"
  cluster_name           = "${var.name}-eks"
  endpoint_public_access = var.eks_endpoint_public_access
  public_access_cidrs    = var.eks_public_access_cidrs
  vpc_id                 = module.vpc.vpc_id
  private_subnet_ids     = module.vpc.private_subnet_ids
  node_instance_type     = var.eks_node_instance_type
  node_desired           = var.eks_node_desired
  node_min               = var.eks_node_min
  node_max               = var.eks_node_max
  tags                   = var.tags
}

module "rds" {
  source             = "../../../modules/aws/rds"
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  allowed_cidr       = var.vpc_cidr
  db_name            = var.db_name
  db_username        = var.db_username
  db_password        = var.db_password
  instance_class     = var.db_instance_class
  allocated_storage  = var.db_allocated_storage
  tags               = var.tags
}

module "s3_cdn" {
  source          = "../../../modules/aws/s3_cdn"
  bucket_name     = var.frontend_bucket_name
  domain_name     = "" # Empty = use CloudFront default domain
  route53_zone_id = "" # Not needed without custom domain
  tags            = var.tags
  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }
}

module "bastion" {
  source           = "../../../modules/aws/bastion"
  name             = var.name
  vpc_id           = module.vpc.vpc_id
  public_subnet_id = module.vpc.public_subnet_ids[0]
  instance_type    = var.bastion_instance_type
  tags             = var.tags
}

module "ecr" {
  source          = "../../../modules/aws/ecr"
  repository_name = var.ecr_repository_name
  tags            = var.tags
}

module "github_oidc" {
  source                     = "../../../modules/aws/github_oidc"
  github_repository          = var.github_repository
  github_branch              = var.github_branch
  role_name                  = var.github_role_name
  ecr_repository_arn         = module.ecr.repository_arn
  s3_bucket_arn              = module.s3_cdn.bucket_arn
  cloudfront_distribution_id = module.s3_cdn.cloudfront_distribution_id
}
