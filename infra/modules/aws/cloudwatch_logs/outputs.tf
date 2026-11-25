output "log_group_name" {
  value = aws_cloudwatch_log_group.application.name
}

output "log_retention_days" {
  value = aws_cloudwatch_log_group.application.retention_in_days
}
