# Cursor Rules

## Workflow
- After code changes, rerun the backend.
- After Terraform changes, create a plan, ask for confirmation, then apply.
- After testing, push code changes with only non-sensitive info.

## Terraform (macOS)
- Use the arm64 Terraform binary: `/opt/homebrew/bin/terraform`.
- Use the AWS SSO profile for plans/applies:
  - `AWS_PROFILE=AdminAccess-Pradeep`
  - `AWS_SDK_LOAD_CONFIG=1`
