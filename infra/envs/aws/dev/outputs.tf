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

output "frontend_cloudfront_distribution_id" {
  value       = module.s3_cdn.cloudfront_distribution_id
  description = "CloudFront distribution ID for cache invalidation"
}

output "bastion_instance_id" {
  value = module.bastion.instance_id
}

output "ecr_repository_url" {
  value = module.ecr.repository_url
}

output "github_actions_role_arn" {
  value = module.github_oidc.role_arn
}

output "api_certificate_arn" {
  value       = var.api_domain_name != "" ? aws_acm_certificate_validation.api[0].certificate_arn : ""
  description = "ACM certificate ARN for API domain (use in Ingress annotation)"
}

output "api_domain_name" {
  value       = var.api_domain_name
  description = "API domain name"
}

output "route53_zone_id" {
  value       = var.api_domain_name != "" && length(data.aws_route53_zone.api) > 0 ? data.aws_route53_zone.api[0].zone_id : ""
  description = "Route53 zone ID for API domain"
}
