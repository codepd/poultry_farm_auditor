# Staging Environment Setup Guide (Option A -> Option B Ready)

This repository uses **Option A** for staging: isolated Kubernetes namespace, PostgreSQL database, S3 buckets, and Route53/ACM domains sharing the existing `kolipannai-dev-eks` cluster and RDS instance. Because all GitHub Actions workflows target the `staging` **GitHub Environment**, you can migrate to a completely separate AWS stack (**Option B**) later by updating only the `staging` Environment variables/secrets.

## 1. One-Time AWS & Database Provisioning

1. **Create the isolated staging database on RDS** (via bastion/SSM tunnel):
   ```sql
   CREATE USER poultry_staging_user WITH PASSWORD '<STRONG_STAGING_DB_PASSWORD>';
   CREATE DATABASE poultry_farm_staging OWNER poultry_staging_user;
   ```
   Initialize the schema:
   ```bash
   grep -v -E '^\\(restrict|unrestrict)' database/schema.sql | \
     psql "postgresql://poultry_staging_user:<PASSWORD>@localhost:15432/poultry_farm_staging?sslmode=require"
   ```

2. **Create staging S3 buckets:**
   - Documents bucket: `kolipannai-staging-documents` (grant read/write to the EKS node role)
   - Frontend bucket: `app-staging.mykolipannai.com-frontend` (+ CloudFront distribution for `staging.mykolipannai.com`)

3. **Request ACM certificates & DNS:**
   - `api-staging.mykolipannai.com` in `ap-south-1` (for the EKS ALB Ingress)
   - `staging.mykolipannai.com` in `us-east-1` (for CloudFront)
   - Update `alb.ingress.kubernetes.io/certificate-arn` in `k8s/go-api-staging/ingress.yaml` with the `ap-south-1` ACM ARN.

4. **Apply Kubernetes manifests & secrets:**
   ```bash
   kubectl apply -f k8s/go-api-staging/namespace.yaml

   kubectl -n poultry-staging create secret generic poultry-db \
     --from-literal=DB_HOST=<RDS_ENDPOINT> \
     --from-literal=DB_PORT=5432 \
     --from-literal=DB_NAME=poultry_farm_staging \
     --from-literal=DB_USER=poultry_staging_user \
     --from-literal=DB_PASSWORD=<STRONG_STAGING_DB_PASSWORD> \
     --from-literal=DB_SSLMODE=require

   kubectl -n poultry-staging create secret generic poultry-app \
     --from-literal=ADMIN_KEY=<STAGING_ADMIN_KEY> \
     --from-literal=JWT_SECRET=<STAGING_JWT_SECRET> \
     --from-literal=OTP_ENCRYPTION_KEY=<32_HEX_OR_CHAR_KEY>

   kubectl apply -f k8s/go-api-staging/
   ```
   Point `api-staging.mykolipannai.com` in Route 53 (Alias) to the ALB created by `go-api-ingress`.

## 2. Configure GitHub Environment (`staging`)

In GitHub **Settings -> Environments -> New environment (`staging`)**:

### Variables
- `STAGING_ENABLED`: `true` (leave unset or `false` until AWS resources above are ready)
- `EKS_CLUSTER_NAME`: `kolipannai-dev-eks`
- `K8S_NAMESPACE`: `poultry-staging`
- `STAGING_API_URL`: `https://api-staging.mykolipannai.com`
- `STAGING_FRONTEND_S3_BUCKET`: `app-staging.mykolipannai.com-frontend`
- `STAGING_CLOUDFRONT_DISTRIBUTION_ID`: `<STAGING_CF_DIST_ID>`

### Secrets
- `AWS_ROLE_TO_ASSUME`: IAM OIDC role ARN
- `QA_ADMIN_KEY`: `<STAGING_ADMIN_KEY>`
- `QA_TEST_PASSWORD`: `<OPTIONAL_TEST_USER_PASSWORD>`
- `STAGING_DB_URL` *(optional, for full weekly schema reset)*: `postgresql://poultry_staging_user:<PASSWORD>@<RDS_ENDPOINT>:5432/poultry_farm_staging?sslmode=require`

## 3. Pipelines Included
- **PR CI (`pr-ci.yml`)**: Runs Go unit tests/build, React Jest tests/build, AND an **ephemeral Postgres + Go API functional E2E** (`scripts/qa_full_e2e.py` + `scripts/api-post-deploy-regression.sh`) on every PR at \$0 AWS cost.
- **Staging Deploy (`staging-deploy.yml`)**: Auto-deploys `main` to `poultry-staging` + staging S3 and runs `qa_full_e2e.py` against `STAGING_API_URL`.
- **Weekly Refresh (`weekly-staging-refresh.yml`)**: Runs every Sunday at `06:00 UTC` (or manually) to reset/re-seed `poultry_farm_staging` with synthetic rolling 24-month data + sample ledgers and verify all E2E checks.

## 4. Upgrading to Option B Later
When you want a completely separate AWS account/VPC/EKS/RDS for staging:
1. Provision `infra/envs/aws/staging` (new `kolipannai-staging-eks` + new RDS).
2. Apply `k8s/go-api-staging/` to the new cluster with the new `DB_HOST`.
3. Update `EKS_CLUSTER_NAME` (and `STAGING_DB_URL`) in the GitHub `staging` Environment. No workflow edits required.
