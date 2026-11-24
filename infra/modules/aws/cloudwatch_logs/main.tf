resource "aws_cloudwatch_log_group" "application" {
  name              = "/aws/containerinsights/${var.cluster_name}/application"
  retention_in_days = var.log_retention_days
  tags              = var.tags
}

resource "aws_iam_role_policy_attachment" "node_cloudwatch" {
  role       = var.eks_node_role_name
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

resource "aws_eks_addon" "cloudwatch_observability" {
  cluster_name                = var.cluster_name
  addon_name                  = "amazon-cloudwatch-observability"
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"

  depends_on = [
    aws_cloudwatch_log_group.application,
    aws_iam_role_policy_attachment.node_cloudwatch,
  ]

  tags = var.tags
}

resource "aws_cloudwatch_query_definition" "api_errors" {
  name = "${var.cluster_name}/go-api-errors"

  log_group_names = [
    aws_cloudwatch_log_group.application.name,
  ]

  query_string = <<-EOT
    fields @timestamp, level, trace_id, correlation_id, method, path, status, duration_ms, user_id, tenant_id, msg
    | filter level = "ERROR" or status >= 500
    | sort @timestamp desc
    | limit 100
  EOT
}

resource "aws_cloudwatch_query_definition" "trace_lookup" {
  name = "${var.cluster_name}/go-api-trace-lookup"

  log_group_names = [
    aws_cloudwatch_log_group.application.name,
  ]

  query_string = <<-EOT
    fields @timestamp, level, trace_id, span_id, method, path, status, duration_ms, user_id, tenant_id, msg
    | filter trace_id = "REPLACE_WITH_TRACE_ID"
    | sort @timestamp asc
  EOT
}
