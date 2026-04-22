# Per-Environment Deployment Guide

This document provides environment-specific deployment instructions for the
Pulumi Go Foundation. Unlike the TypeScript foundation which uses directory-based
environment separation (`envs/development/`, `envs/production/`), the Go
foundation uses **stack-based environment separation** where the same Go code
is deployed with different Pulumi stack configurations.

## Architecture: Stack-Based Environments

```
0-bootstrap/     → 1 stack:  production (shared)
1-org/           → 1 stack:  production (shared)
2-environments/  → 1 stack:  production (creates all 3 env folders)
3-networks-*/    → 3 stacks: development, nonproduction, production
4-projects/      → 3 stacks: development, nonproduction, production (per BU)
5-app-infra/     → 3 stacks: development, nonproduction, production (per BU)
```

## Environment Configuration: Development

### 3-networks (development stack)

```bash
cd 3-networks-svpc  # or 3-networks-hub-and-spoke
pulumi stack init development
pulumi config set env "development"
pulumi config set project_id "prj-d-svpc"          # from Stage 1 output
pulumi config set parent_id "organizations/YOUR_ORG_ID"
pulumi config set region1 "us-central1"             # default
pulumi config set region2 "us-west1"                # default
pulumi up
```

### 4-projects (development stack)

```bash
cd 4-projects
pulumi stack init development
pulumi config set env "development"
pulumi config set business_code "bu1"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
pulumi config set org_stack_name "organization/vitruvian/1-org/production"
pulumi up
```

### 5-app-infra (development stack)

```bash
cd 5-app-infra
pulumi stack init development
pulumi config set env "development"
pulumi config set projects_stack_name "VitruvianSoftware/foundation-4-projects/development"
pulumi up
```

## Environment Configuration: Non-Production

### 3-networks (nonproduction stack)

```bash
cd 3-networks-svpc
pulumi stack init nonproduction
pulumi config set env "nonproduction"
pulumi config set project_id "prj-n-svpc"
pulumi config set parent_id "organizations/YOUR_ORG_ID"
pulumi up
```

### 4-projects (nonproduction stack)

```bash
cd 4-projects
pulumi stack init nonproduction
pulumi config set env "nonproduction"
pulumi config set business_code "bu1"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
pulumi config set org_stack_name "organization/vitruvian/1-org/production"
pulumi up
```

### 5-app-infra (nonproduction stack)

```bash
cd 5-app-infra
pulumi stack init nonproduction
pulumi config set env "nonproduction"
pulumi config set projects_stack_name "VitruvianSoftware/foundation-4-projects/nonproduction"
pulumi up
```

## Environment Configuration: Production

### 3-networks (production stack)

> **Note:** Deploy **production first** as it includes the DNS Hub that other
> environments depend on.

```bash
cd 3-networks-svpc
pulumi stack init production
pulumi config set env "production"
pulumi config set project_id "prj-p-svpc"
pulumi config set parent_id "organizations/YOUR_ORG_ID"
pulumi up
```

### 4-projects (production stack)

```bash
cd 4-projects
pulumi stack init production
pulumi config set env "production"
pulumi config set business_code "bu1"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
pulumi config set org_stack_name "organization/vitruvian/1-org/production"
pulumi up
```

### 5-app-infra (production stack)

```bash
cd 5-app-infra
pulumi stack init production
pulumi config set env "production"
pulumi config set projects_stack_name "VitruvianSoftware/foundation-4-projects/production"
pulumi up
```

## Environment Inputs Summary

### 3-networks (all environments)

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `env` | Environment name | `string` | n/a | yes |
| `project_id` | Shared VPC host project ID | `string` | n/a | yes |
| `parent_id` | Parent scope for firewall policies | `string` | n/a | yes |
| `region1` | Primary region | `string` | `"us-central1"` | no |
| `region2` | Secondary region | `string` | `"us-west1"` | no |

### 4-projects (all environments)

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `env` | Environment name | `string` | n/a | yes |
| `business_code` | Business unit code | `string` | n/a | yes |
| `billing_account` | Billing account ID | `string` | n/a | yes |
| `org_stack_name` | Stack name of 1-org stage | `string` | n/a | yes |

### 5-app-infra (all environments)

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `env` | Environment name | `string` | n/a | yes |
| `projects_stack_name` | Stack name of 4-projects stage | `string` | n/a | yes |
| `region` | Region for compute instances | `string` | `"us-central1"` | no |

## Key Difference: Stack vs Directory

| Aspect | Terraform | Pulumi TS | Pulumi Go |
|--------|-----------|-----------|-----------|
| Env isolation | `envs/development/` directory | `envs/development/` directory | `development` stack |
| Config file | `terraform.tfvars` per dir | `Pulumi.development.yaml` per dir | `Pulumi.development.yaml` per stage |
| State isolation | Backend prefix per dir | Automatic per directory | Automatic per stack |
| Code duplication | Per-env `main.tf` | Per-env `index.ts` | **Single `main.go`** (zero duplication) |
