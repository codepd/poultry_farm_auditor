# Infrastructure (Terraform)

This directory contains Terraform scaffolding for AWS (dev environment first).

## Suggested DNS names
- Frontend: `app-dev.mykolipannai.com`
- API: `api-dev.mykolipannai.com`

Replace with your real domain as needed.

## Structure
- `envs/aws/dev`: dev environment root module
- `modules/aws/*`: reusable AWS modules (placeholders to be implemented)

## Next steps
1. Fill in module implementations in `modules/aws/*`.
2. Copy `envs/aws/dev/terraform.tfvars.example` to `terraform.tfvars`.
3. Run `terraform init`, `terraform plan`, `terraform apply` in `envs/aws/dev`.

## Frontend (S3 + CloudFront)

### Automated Deployment (Recommended)

The frontend is automatically deployed via GitHub Actions when changes are pushed to `main`.

**Setup:**
1. After Terraform apply, get deployment values:
   ```bash
   cd infra/envs/aws/dev
   terraform output
   ```
2. Configure GitHub Secrets (see `FRONTEND_DEPLOYMENT_GUIDE.md` for details):
   - `AWS_ROLE_TO_ASSUME`: IAM role ARN for GitHub Actions
   - `FRONTEND_S3_BUCKET`: S3 bucket name
   - `CLOUDFRONT_DISTRIBUTION_ID`: CloudFront distribution ID

3. Push changes to `react_frontend/` to trigger deployment

**Quick script to get values:**
```bash
./scripts/get-frontend-deployment-values.sh
```

### Manual Deployment

1. Build the frontend:
   ```bash
   cd react_frontend
   npm install
   npm run build
   ```

2. Upload the build output to S3:
   ```bash
   aws s3 sync build/ s3://<frontend_bucket_name> --delete
   ```

3. Invalidate the CloudFront cache:
   ```bash
   aws cloudfront create-invalidation --distribution-id <distribution_id> --paths "/*"
   ```

### Terraform Outputs

- `frontend_bucket_name`: S3 bucket name
- `frontend_cdn_domain`: CloudFront domain name
- `frontend_cloudfront_distribution_id`: CloudFront distribution ID for cache invalidation
- `github_actions_role_arn`: IAM role ARN for GitHub Actions

### Documentation

See `FRONTEND_DEPLOYMENT_GUIDE.md` for complete deployment instructions, troubleshooting, and workflow details.
