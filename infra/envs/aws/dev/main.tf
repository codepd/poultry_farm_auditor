provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
}

provider "aws" {
  alias   = "us_east_1"
  region  = "us-east-1"
  profile = var.aws_profile
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
  domain_name     = var.frontend_domain_name
  route53_zone_id = var.frontend_domain_name != "" ? data.aws_route53_zone.api[0].zone_id : ""
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
  eks_cluster_arn            = module.eks.cluster_arn
}

# Route53 zone for API domain (use existing or create new)
data "aws_route53_zone" "api" {
  count = var.api_domain_name != "" ? 1 : 0
  name  = var.route53_zone_name
}

# ACM certificate for API domain (in ap-south-1 for ALB)
resource "aws_acm_certificate" "api" {
  count             = var.api_domain_name != "" ? 1 : 0
  domain_name       = var.api_domain_name
  validation_method = "DNS"
  tags              = var.tags

  lifecycle {
    create_before_destroy = true
  }
}

# Route53 validation records for ACM certificate
resource "aws_route53_record" "api_cert_validation" {
  for_each = var.api_domain_name != "" && length(data.aws_route53_zone.api) > 0 ? {
    for dvo in aws_acm_certificate.api[0].domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  } : {}

  zone_id = data.aws_route53_zone.api[0].zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 60
  records = [each.value.record]
}

# Wait for certificate validation
resource "aws_acm_certificate_validation" "api" {
  count                   = var.api_domain_name != "" ? 1 : 0
  certificate_arn         = aws_acm_certificate.api[0].arn
  validation_record_fqdns = [for record in aws_route53_record.api_cert_validation : record.fqdn]
}

# Route53 A record pointing to ALB (set alb_dns_name after ALB is created)
resource "aws_route53_record" "api_alias" {
  count   = var.api_domain_name != "" && var.alb_dns_name != "" && length(data.aws_route53_zone.api) > 0 ? 1 : 0
  zone_id = data.aws_route53_zone.api[0].zone_id
  name    = var.api_domain_name
  type    = "A"

  alias {
    name                   = var.alb_dns_name
    zone_id                = var.alb_hosted_zone_id
    evaluate_target_health = false
  }
}
