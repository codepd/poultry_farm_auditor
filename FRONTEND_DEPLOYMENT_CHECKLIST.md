# Frontend Deployment Checklist

Quick reference checklist for setting up frontend deployment.

## ✅ Pre-Deployment Setup

- [ ] Terraform infrastructure applied successfully
- [ ] S3 bucket created and configured
- [ ] CloudFront distribution created
- [ ] Route53 DNS configured
- [ ] GitHub OIDC module updated with S3/CloudFront permissions

## ✅ Get Deployment Values

Run after Terraform apply:
```bash
cd infra/envs/aws/dev
terraform output
```

Or use the helper script:
```bash
./scripts/get-frontend-deployment-values.sh
```

## ✅ Configure GitHub Secrets

Go to: **GitHub Repo → Settings → Secrets and variables → Actions**

Add these secrets:

- [ ] **`AWS_ROLE_TO_ASSUME`**
  - Value: `github_actions_role_arn` from Terraform output
  - Example: `arn:aws:iam::123456789012:role/poultry-dev-github-actions`

- [ ] **`FRONTEND_S3_BUCKET`**
  - Value: `frontend_bucket_name` from Terraform output
  - Example: `poultry-dev-frontend-replace-me`

- [ ] **`CLOUDFRONT_DISTRIBUTION_ID`**
  - Value: `frontend_cloudfront_distribution_id` from Terraform output
  - Example: `E1234567890ABC`

## ✅ Test Deployment

- [ ] Make a small change to `react_frontend/` (e.g., update a comment)
- [ ] Commit and push to `main` branch
- [ ] Check GitHub Actions tab - workflow should trigger
- [ ] Verify workflow completes successfully
- [ ] Visit your frontend domain to verify changes are live

## ✅ Verify Infrastructure

- [ ] S3 bucket has files (`index.html`, `static/` folder)
- [ ] CloudFront distribution is enabled
- [ ] Route53 record points to CloudFront
- [ ] HTTPS certificate is valid
- [ ] Frontend loads at your domain

## 📝 Quick Commands

**Manual deployment (if needed):**
```bash
cd react_frontend
npm install
npm run build
aws s3 sync build/ s3://<bucket-name> --delete
aws cloudfront create-invalidation --distribution-id <dist-id> --paths "/*"
```

**Get all deployment values:**
```bash
./scripts/get-frontend-deployment-values.sh
```

**Check Terraform outputs:**
```bash
cd infra/envs/aws/dev
terraform output
```

## 🔍 Troubleshooting

| Issue | Solution |
|-------|----------|
| Workflow fails with "Access Denied" | Check `AWS_ROLE_TO_ASSUME` secret matches Terraform output |
| S3 sync fails | Verify bucket name and IAM permissions |
| Old content shows | CloudFront invalidation takes 1-5 minutes |
| Build fails | Test locally: `cd react_frontend && npm run build` |

## 📚 Full Documentation

See `FRONTEND_DEPLOYMENT_GUIDE.md` for detailed instructions.
