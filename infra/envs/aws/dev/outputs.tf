output "vpc_id" {
  value = module.vpc.vpc_id
}

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "db_endpoint" {
  value = module.rds.db_endpoint
}

output "frontend_bucket_name" {
  value = module.s3_cdn.bucket_name
}

output "frontend_cdn_domain" {
  value = module.s3_cdn.cloudfront_domain_name
}

output "bastion_instance_id" {
  value = module.bastion.instance_id
}

output "ecr_repository_url" {
  value = module.ecr.repository_url
}
