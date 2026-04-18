# 0-Bootstrap - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **0-bootstrap** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
The purpose of this stage is to bootstrap an existing Google Cloud organization. It creates the foundational resources required to manage your infrastructure as code using Pulumi.

Key resources created:
- **Seed Project**: Houses the Pulumi state (if using a GCS backend) and the service accounts used by subsequent stages.
- **Service Accounts**: Granular service accounts for `bootstrap`, `org`, `env`, `net`, `proj`, and `apps` stages, each with the least privilege required.
- **Organization IAM**: Assigns the necessary roles at the organization level to the created service accounts.
- **State Bucket**: A Google Cloud Storage bucket for storing Pulumi or Terraform state.

## Transition from Terraform to Pulumi
This port translates the original CFT Bootstrap Terraform module into idiomatic Pulumi Go code.
Key differences:
- **Language**: HCL is replaced by Go, allowing for better testing and structured logic.
- **Stack References**: Subsequent stages use Pulumi Stack References to retrieve outputs from this stage (e.g., Service Account emails, Project IDs).
- **Automation**: Pulumi handles resource dependencies and ordering natively within the Go program.

## Prerequisites
- Pulumi CLI
- Go 1.21+
- Google Cloud SDK (`gcloud`)
- Organization Admin and Billing Account Admin permissions.

## Usage
1. Initialize the stack:
   ```bash
   pulumi stack init bootstrap
   ```
2. Configure required variables:
   ```bash
   pulumi config set org_id 1234567890
   pulumi config set billing_account 012345-678901-ABCDEF
   pulumi config set project_prefix my-prefix
   ```
3. Deploy:
   ```bash
   pulumi up
   ```

## Configuration
| Key | Description | Default |
|-----|-------------|---------|
| `org_id` | The GCP Organization ID | *Required* |
| `billing_account` | The Billing Account ID | *Required* |
| `project_prefix` | Prefix for project IDs | `prj` |
| `folder_prefix` | Prefix for folder names | `fldr` |
| `default_region` | Default region for resources | `us-central1` |
