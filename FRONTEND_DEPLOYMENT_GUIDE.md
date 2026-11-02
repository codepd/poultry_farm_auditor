# Frontend Deployment Guide

This guide explains how to deploy the React frontend to AWS S3 + CloudFront using GitHub Actions.

## Architecture Overview

The frontend deployment uses:
- **S3 Bucket**: Stores the built React application files
- **CloudFront CDN**: Serves the frontend with HTTPS and global edge caching
- **Route53**: DNS routing to CloudFront
- **GitHub Actions**: Automated CI/CD pipeline

## Prerequisites

1. ✅ Terraform infrastructure applied (S3 bucket, CloudFront, Route53 configured)
2. ✅ GitHub repository with Actions enabled
3. ✅ GitHub OIDC configured in AWS (handled by Terraform)

## Step 1: Get Deployment Values

After Terraform apply completes, get the required values:

```bash
cd infra/envs/aws/dev
terraform output
```

You'll need these values:
- `frontend_bucket_name` - S3 bucket name
- `frontend_cloudfront_distribution_id` - CloudFront distribution ID
- `github_actions_role_arn` - IAM role ARN for GitHub Actions

## Step 2: Configure GitHub Secrets

Go to your GitHub repository → Settings → Secrets and variables → Actions → New repository secret

Add these secrets:

### Required Secrets

1. **`AWS_ROLE_TO_ASSUME`**
   - Value: The IAM role ARN from Terraform output (`github_actions_role_arn`)
   - Example: `arn:aws:iam::123456789012:role/poultry-dev-github-actions`

2. **`FRONTEND_S3_BUCKET`**
   - Value: S3 bucket name from Terraform output (`frontend_bucket_name`)
   - Example: `poultry-dev-frontend-replace-me`

3. **`CLOUDFRONT_DISTRIBUTION_ID`**
   - Value: CloudFront distribution ID from Terraform output (`frontend_cloudfront_distribution_id`)
   - Example: `E1234567890ABC`

## Step 3: Workflow Overview

The GitHub Actions workflow (`.github/workflows/fe-s3-deploy.yml`) automatically:

1. **Triggers** on:
   - Push to `main` branch
   - Changes in `react_frontend/` directory
   - Changes to the workflow file itself

2. **Builds** the React app:
   - Sets up Node.js 18
   - Installs dependencies (`npm ci`)
   - Builds production bundle (`npm run build`)

3. **Deploys** to S3:
   - Syncs `react_frontend/build/` to S3 bucket
   - Uses `--delete` flag to remove old files

4. **Invalidates** CloudFront cache:
   - Creates invalidation for all paths (`/*`)
   - Ensures users get the latest version immediately

## Step 4: Manual Deployment (Optional)

If you need to deploy manually:

```bash
# 1. Build the frontend
cd react_frontend
npm install
npm run build

# 2. Deploy to S3 (replace with your bucket name)
aws s3 sync build/ s3://poultry-dev-frontend-replace-me --delete

# 3. Invalidate CloudFront cache (replace with your distribution ID)
aws cloudfront create-invalidation \
  --distribution-id E1234567890ABC \
  --paths "/*"
```

## Step 5: Verify Deployment

1. **Check S3 bucket**:
   ```bash
   aws s3 ls s3://<bucket-name>/
   ```
   Should see: `index.html`, `static/` directory, etc.

2. **Check CloudFront**:
   - Visit your domain: `https://app-dev.mykolipannai.com`
   - Or CloudFront URL: `https://<distribution-id>.cloudfront.net`

3. **Check GitHub Actions**:
   - Go to Actions tab in GitHub
   - Verify workflow ran successfully
   - Check logs for any errors

## Troubleshooting

### Workflow fails with "Access Denied"

**Issue**: GitHub Actions can't assume the IAM role.

**Solution**:
1. Verify `AWS_ROLE_TO_ASSUME` secret is correct
2. Check GitHub OIDC provider is configured in AWS
3. Verify repository name and branch match Terraform config

### S3 sync fails

**Issue**: Permission denied when syncing to S3.

**Solution**:
1. Verify IAM role has S3 permissions (check Terraform applied correctly)
2. Check bucket name in secret matches actual bucket
3. Verify bucket policy allows CloudFront access

### CloudFront shows old content

**Issue**: Changes not visible after deployment.

**Solution**:
1. Check CloudFront invalidation completed (can take 1-5 minutes)
2. Verify `CLOUDFRONT_DISTRIBUTION_ID` secret is correct
3. Try hard refresh (Ctrl+Shift+R) or clear browser cache

### Build fails

**Issue**: `npm run build` fails.

**Solution**:
1. Test build locally: `cd react_frontend && npm run build`
2. Check for TypeScript/ESLint errors
3. Verify `package-lock.json` is committed
4. Check Node.js version compatibility

## Workflow Configuration

The workflow is configured in `.github/workflows/fe-s3-deploy.yml`:

```yaml
# Triggers on changes to frontend
on:
  push:
    branches: [main]
    paths:
      - "react_frontend/**"
      - ".github/workflows/fe-s3-deploy.yml"

# Uses GitHub OIDC for AWS authentication
permissions:
  id-token: write
  contents: read
```

## Environment Variables

The workflow uses these environment variables (set in workflow file):
- `AWS_REGION`: `ap-south-1`
- `S3_BUCKET`: From GitHub secret `FRONTEND_S3_BUCKET`
- `CLOUDFRONT_DISTRIBUTION_ID`: From GitHub secret `CLOUDFRONT_DISTRIBUTION_ID`

## IAM Permissions

The GitHub Actions IAM role needs:
- **S3**: `PutObject`, `GetObject`, `DeleteObject`, `ListBucket`
- **CloudFront**: `CreateInvalidation`, `GetInvalidation`, `ListInvalidations`

These are automatically configured by Terraform in the `github_oidc` module.

## Next Steps

After successful deployment:
1. ✅ Test the frontend at your domain
2. ✅ Verify API endpoints are accessible
3. ✅ Monitor CloudFront metrics in AWS Console
4. ✅ Set up monitoring/alerts if needed

## Related Files

- Workflow: `.github/workflows/fe-s3-deploy.yml`
- Infrastructure: `infra/modules/aws/s3_cdn/`
- GitHub OIDC: `infra/modules/aws/github_oidc/`
- Frontend: `react_frontend/`
