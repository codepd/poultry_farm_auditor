provider "aws" {
  region = var.aws_region
}

module "vpc" {
  source               = "../../../modules/aws/vpc"
  name                 = var.name
  cidr_block           = var.vpc_cidr
  public_subnet_cidrs  = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
  tags                 = var.tags
}

module "eks" {
  source             = "../../../modules/aws/eks"
  cluster_name       = "${var.name}-eks"
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  node_instance_type = var.eks_node_instance_type
  node_desired       = var.eks_node_desired
  node_min           = var.eks_node_min
  node_max           = var.eks_node_max
  tags               = var.tags
}

module "rds" {
  source             = "../../../modules/aws/rds"
  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  db_name            = var.db_name
  db_username        = var.db_username
  db_password        = var.db_password
  instance_class     = var.db_instance_class
  allocated_storage  = var.db_allocated_storage
  tags               = var.tags
}

module "s3_cdn" {
  source      = "../../../modules/aws/s3_cdn"
  bucket_name = var.frontend_bucket_name
  domain_name = var.frontend_domain_name
  tags        = var.tags
}
