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
