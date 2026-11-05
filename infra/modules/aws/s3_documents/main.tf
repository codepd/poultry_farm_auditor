resource "aws_s3_bucket" "this" {
  bucket = var.bucket_name
  tags   = var.tags
}

resource "aws_s3_bucket_ownership_controls" "this" {
  bucket = aws_s3_bucket.this.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "this" {
  bucket = aws_s3_bucket.this.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "this" {
  bucket = aws_s3_bucket.this.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "this" {
  bucket = aws_s3_bucket.this.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "this" {
  bucket = aws_s3_bucket.this.id

  rule {
    id     = "archive-old-noncurrent-versions"
    status = "Enabled"

    noncurrent_version_transition {
      noncurrent_days = 90
      storage_class   = "STANDARD_IA"
    }

    noncurrent_version_expiration {
      noncurrent_days = 730
    }
  }

  rule {
    id     = "transition-parsed-excels"
    status = "Enabled"

    filter {
      prefix = "parsed_excels/"
    }

    transition {
      days          = 365
      storage_class = "GLACIER_IR"
    }
  }
}

data "aws_iam_policy_document" "documents_rw" {
  statement {
    sid    = "AllowDocumentsBucketList"
    effect = "Allow"
    actions = [
      "s3:ListBucket",
      "s3:GetBucketLocation",
      "s3:ListBucketVersions"
    ]
    resources = [
      aws_s3_bucket.this.arn
    ]
  }

  statement {
    sid    = "AllowDocumentsObjectRW"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:PutObject",
      "s3:PutObjectTagging",
      "s3:GetObjectTagging",
      "s3:DeleteObject"
    ]
    resources = [
      "${aws_s3_bucket.this.arn}/*"
    ]
  }
}

resource "aws_iam_role_policy" "eks_node_documents_access" {
  count  = var.eks_node_role_name != "" ? 1 : 0
  name   = "${var.bucket_name}-eks-rw"
  role   = var.eks_node_role_name
  policy = data.aws_iam_policy_document.documents_rw.json
}

resource "aws_iam_role_policy" "github_actions_documents_access" {
  count  = var.github_actions_role_name != "" ? 1 : 0
  name   = "${var.bucket_name}-github-rw"
  role   = var.github_actions_role_name
  policy = data.aws_iam_policy_document.documents_rw.json
}
