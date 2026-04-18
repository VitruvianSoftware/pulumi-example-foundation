# Onboarding Guide: Pulumi GCP Foundation

Welcome to the Pulumi port of the Google Cloud Enterprise Foundation. This guide walks you through deploying your foundation step-by-step.

## Architectural Overview

This foundation is designed to be modular and deployed in stages. Each stage builds upon the outputs of the previous one using **Pulumi Stack References**. The CI/CD pipeline automates deployment via GitHub Actions.

### Prerequisites
- Pulumi CLI installed (`curl -fsSL https://get.pulumi.com | sh`)
- Go 1.21+ installed
- `gcloud` CLI authenticated with Organization Admin permissions
- A Billing Account ID
- A Google Cloud Organization ID

---

## Step 0: Bootstrap

**Purpose**: Create the Seed project (state storage, KMS encryption, service accounts) and CI/CD project.

```bash
cd 0-bootstrap
pulumi stack init production
pulumi config set org_id "YOUR_ORG_ID"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT"
pulumi config set group_org_admins "org-admins@example.com"
pulumi config set group_billing_admins "billing-admins@example.com"
pulumi config set billing_data_users "billing-data@example.com"
pulumi config set audit_data_users "audit-data@example.com"
pulumi up
```

**Key decisions**:
- `project_prefix` (default: `prj`) — prefix for all project IDs
- `folder_prefix` (default: `fldr`) — prefix for all folder names
- `parent_folder` (optional) — deploy under a specific folder instead of org root

## Step 1: Organization

**Purpose**: Create the core folder structure, shared projects (logging, billing, SCC, KMS, Secrets, DNS, Interconnect), enforce org policies, and set up centralized logging.

```bash
cd 1-org
pulumi stack init production
pulumi config set org_id "YOUR_ORG_ID"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT"
pulumi config set bootstrap_stack_name "organization/vitruvian/0-bootstrap/production"
pulumi config set domains_to_allow "example.com"
pulumi config set create_access_context_manager_policy "true"
pulumi up
```

## Step 2: Environments

**Purpose**: Create per-environment KMS and Secrets projects under each environment folder.

```bash
cd 2-environments
pulumi stack init production
pulumi config set org_id "YOUR_ORG_ID"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT"
pulumi config set org_stack_name "organization/vitruvian/1-org/production"
pulumi up
```

## Step 3: Networking (Decision Point)

**Purpose**: Deploy the network infrastructure. **Choose ONE**:

### Option A: Shared VPC
```bash
cd 3-networks-svpc
pulumi stack init development
pulumi config set env "development"
pulumi config set project_id "prj-d-svpc"  # from Stage 1 output
pulumi config set parent_id "organizations/YOUR_ORG_ID"
pulumi up
```

### Option B: Hub-and-Spoke
```bash
cd 3-networks-hub-and-spoke
pulumi stack init development
pulumi config set env "development"
pulumi config set hub_project_id "prj-net-hub-svpc"
pulumi config set spoke_project_id "prj-d-svpc"
pulumi config set parent_id "organizations/YOUR_ORG_ID"
pulumi up
```

**Run once per environment** (development, nonproduction, production).

## Step 4: Projects

**Purpose**: Create Business Unit projects with SVPC attachment, floating, and peering variants.

```bash
cd 4-projects
pulumi stack init development
pulumi config set env "development"
pulumi config set business_code "bu1"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT"
pulumi config set org_stack_name "organization/vitruvian/1-org/production"
pulumi up
```

## Step 5: App Infrastructure

**Purpose**: Deploy sample application infrastructure (Cloud Run + BigQuery).

```bash
cd 5-app-infra
pulumi stack init development
pulumi config set env "development"
pulumi config set projects_stack_name "organization/vitruvian/4-projects/development"
pulumi up
```

---

## CI/CD Pipeline

After the initial bootstrap, all subsequent changes are deployed via the GitHub Actions pipeline (`.github/workflows/pulumi-ci.yml`):

1. **Feature branch** → Open PR → `pulumi preview` runs for all stages
2. **Merge to `development`** → `pulumi up` deploys to development
3. **Merge to `nonproduction`** → `pulumi up` deploys to non-production
4. **Merge to `production`** → `pulumi up` deploys to production + shared resources

### Required GitHub Secrets
- `PULUMI_ACCESS_TOKEN` — Pulumi Cloud access token
- `GOOGLE_CREDENTIALS` — GCP service account key JSON

## Using the Shared Library

All stages use the [Vitruvian Software Pulumi Library](https://github.com/VitruvianSoftware/pulumi-library). This library provides **ComponentResources** that ensure your infrastructure is standardized:

| Package | Description |
|---------|-------------|
| `pkg/project` | Project factory with API activation and billing |
| `pkg/iam` | Multi-scope IAM bindings (org, folder, project) |
| `pkg/policy` | Org policy enforcement (boolean + list) |
| `pkg/networking` | VPC and subnet management |
| `pkg/app` | Cloud Run app deployment |
| `pkg/data` | BigQuery data platform |
