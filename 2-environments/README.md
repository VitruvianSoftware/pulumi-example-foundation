# 2-environments

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

The purpose of this step is to set up development, nonproduction, and production environments within the Google Cloud organization that you've created. For each environment, this stage creates:

- **`prj-{d,n,p}-kms`** — An environment-level [Cloud KMS](https://cloud.google.com/security-key-management) project for key management
- **`prj-{d,n,p}-secrets`** — An environment-level [Secret Manager](https://cloud.google.com/secret-manager) project for secret storage

This will create the following project structure under the environment folders created in Stage 1:

```
example-organization
└── fldr-development
    ├── prj-d-kms
    └── prj-d-secrets
└── fldr-nonproduction
    ├── prj-n-kms
    └── prj-n-secrets
└── fldr-production
    ├── prj-p-kms
    └── prj-p-secrets
```

## Prerequisites

1. [0-bootstrap](../0-bootstrap/README.md) executed successfully.
1. [1-org](../1-org/README.md) executed successfully.

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Usage

### Deploying with GitHub Actions

1. Navigate to the `2-environments` directory and initialize the stack:

   ```bash
   cd 2-environments
   pulumi stack init production
   ```

1. Set the required configuration:

   ```bash
   pulumi config set org_id "YOUR_ORG_ID"
   pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
   pulumi config set org_stack_name "organization/vitruvian/1-org/production"
   ```

1. (Optional) Override the project prefix:

   ```bash
   pulumi config set project_prefix "prj"  # default: prj
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. Commit and push to trigger the CI/CD pipeline:

   ```bash
   git add .
   git commit -m "Initialize environments stage"
   git push origin production
   ```

1. You can now move to the instructions in the network step. To use the [Shared VPC](https://cloud.google.com/architecture/security-foundations/networking#vpcsharedvpc-id7-1-shared-vpc-) network mode, go to [3-networks-svpc](../3-networks-svpc/README.md). To use the [Hub and Spoke](https://cloud.google.com/architecture/security-foundations/networking#hub-and-spoke) network mode, go to [3-networks-hub-and-spoke](../3-networks-hub-and-spoke/README.md).

### Running Pulumi Locally

1. Navigate to `2-environments`, initialize, and set configuration as described above.

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `org_id` | GCP Organization ID | ✅ | — |
| `billing_account` | Billing account ID | ✅ | — |
| `org_stack_name` | Fully qualified Pulumi stack name of the 1-org stage | ✅ | — |
| `project_prefix` | Project name prefix | | `"prj"` |

## Outputs

| Name | Description |
|------|-------------|
| `development_kms_project_id` | KMS project ID for development |
| `development_secrets_project_id` | Secrets project ID for development |
| `nonproduction_kms_project_id` | KMS project ID for nonproduction |
| `nonproduction_secrets_project_id` | Secrets project ID for nonproduction |
| `production_kms_project_id` | KMS project ID for production |
| `production_secrets_project_id` | Secrets project ID for production |
| `{env}_folder_id` | Folder ID for each environment (passed through from Stage 1) |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Creates per-environment KMS and Secrets projects under each environment folder via Stack References to Stage 1 |
