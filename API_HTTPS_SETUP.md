# API HTTPS Setup Guide

This guide explains how to set up HTTPS for the API endpoint using ACM certificate and Route53.

## Prerequisites

1. Route53 hosted zone for `mykolipannai.com` must exist in your AWS account
2. Terraform has been initialized and configured
3. Kubernetes cluster is running and Ingress controller is installed

## Steps

### Step 1: Apply Terraform to Create ACM Certificate

```bash
cd infra/envs/aws/dev
terraform plan
terraform apply
```

This will create:
- ACM certificate for `api-dev.mykolipannai.com` in `ap-south-1`
- Route53 validation records
- Wait for certificate validation

### Step 2: Get Certificate ARN

```bash
terraform output -raw api_certificate_arn
```

Copy the certificate ARN (e.g., `arn:aws:acm:ap-south-1:123456789012:certificate/abc-123-def-456`)

### Step 3: Update Ingress with Certificate ARN

Edit `k8s/go-api/ingress.yaml` and update the certificate ARN:

```yaml
alb.ingress.kubernetes.io/certificate-arn: "arn:aws:acm:ap-south-1:123456789012:certificate/abc-123-def-456"
```

Or use the helper script:

```bash
./scripts/setup-api-https.sh
```

### Step 4: Apply Updated Ingress

```bash
kubectl apply -f k8s/go-api/ingress.yaml
```

Wait for the ALB to be created (may take 2-3 minutes):

```bash
kubectl get ingress go-api-ingress -n poultry-dev -w
```

### Step 5: Get ALB DNS Name

Once the Ingress shows the ALB DNS name:

```bash
kubectl get ingress go-api-ingress -n poultry-dev -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
```

### Step 6: Create Route53 A Record

Update `terraform.tfvars` with the ALB DNS name:

```hcl
alb_dns_name = "k8s-poultryd-goapiing-xxxxx.ap-south-1.elb.amazonaws.com"
```

Then apply Terraform:

```bash
terraform apply
```

Or use the helper script which will guide you through this.

## Verification

After DNS propagation (may take a few minutes), test the HTTPS endpoint:

```bash
curl https://api-dev.mykolipannai.com/api/health
```

You should see `OK` response.

## Troubleshooting

### Certificate Validation Failed

- Check Route53 validation records were created
- Verify DNS records are correct: `aws route53 list-resource-record-sets --hosted-zone-id <ZONE_ID>`
- Wait for certificate validation (can take 5-10 minutes)

### ALB Not Created

- Check Ingress controller logs: `kubectl logs -n kube-system -l app.kubernetes.io/name=aws-load-balancer-controller`
- Verify IAM role has correct permissions
- Check Ingress status: `kubectl describe ingress go-api-ingress -n poultry-dev`

### DNS Not Resolving

- Verify Route53 A record exists: `aws route53 list-resource-record-sets --hosted-zone-id <ZONE_ID>`
- Check DNS propagation: `dig api-dev.mykolipannai.com`
- Wait for DNS propagation (can take up to 48 hours, usually much faster)

## Notes

- The ALB hosted zone ID for `ap-south-1` is `Z1D633PJN98FT9` (constant for all ALBs in this region)
- Health checks use HTTP (port 80) even though main traffic is HTTPS
- SSL redirect is enabled, so HTTP requests are automatically redirected to HTTPS
