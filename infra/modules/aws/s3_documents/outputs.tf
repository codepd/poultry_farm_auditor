output "bucket_name" {
  value       = aws_s3_bucket.this.id
  description = "Name of the S3 documents bucket"
}

output "bucket_arn" {
  value       = aws_s3_bucket.this.arn
  description = "ARN of the S3 documents bucket"
}

output "bucket_regional_domain_name" {
  value       = aws_s3_bucket.this.bucket_regional_domain_name
  description = "Regional domain name of the S3 documents bucket"
}
