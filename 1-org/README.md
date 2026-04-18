# 1-Org - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **1-org** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
This stage sets up organization-level resources and the folder structure that houses the rest of the foundation.

Key resources created:
- **Folders**: Creates the `common`, `network`, and environment-specific (`development`, `nonproduction`, `production`) folders.
- **Logging**: Sets up an organization-wide log sink and a dedicated project for audit logs.
- **Billing Export**: Creates a dedicated project for billing data exports.

## Transition from Terraform to Pulumi
- **Stack References**: This stage retrieves outputs from the `0-bootstrap` stack to ensure consistency across the foundation.
- **Folder Management**: Folders are managed as first-class resources in Go, enabling dynamic naming and parentage logic.
- **Organization Sinks**: Log sinks are configured at the organization level, targeting the audit logs project created in this stage.

## Prerequisites
- Successful deployment of the `0-bootstrap` stage.
- Pulumi CLI and Go environment.

## Usage
1. Initialize the stack:
   ```bash
   pulumi stack init organization
   ```
2. Configure required variables:
   ```bash
   pulumi config set bootstrap_stack_name <org>/<project>/bootstrap
   pulumi config set org_id 1234567890
   pulumi config set billing_account 012345-678901-ABCDEF
   ```
3. Deploy:
   ```bash
   pulumi up
   ```

## Configuration
| Key | Description | Default |
|-----|-------------|---------|
| `bootstrap_stack_name` | Full name of the bootstrap stack | *Required* |
| `org_id` | The GCP Organization ID | *Required* |
| `billing_account` | The Billing Account ID | *Required* |
| `project_prefix` | Prefix for project IDs | `prj` |
| `folder_prefix` | Prefix for folder names | `fldr` |
