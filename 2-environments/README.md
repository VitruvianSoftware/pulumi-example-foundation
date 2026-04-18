# 2-Environments - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **2-environments** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
The purpose of this stage is to refine the environment-specific folders and shared resources (like KMS or Secrets) for each environment (`development`, `nonproduction`, `production`).

## Transition from Terraform to Pulumi
- **Environment Isolation**: This stage can be run multiple times with different stack configurations to manage each environment independently.
- **Stack References**: Retreives the parent folder IDs from the `1-org` stack.

## Prerequisites
- Successful deployment of the `1-org` stage.

## Usage
1. Initialize a stack for an environment (e.g., development):
   ```bash
   pulumi stack init development
   ```
2. Configure required variables:
   ```bash
   pulumi config set org_stack_name <org>/<project>/organization
   pulumi config set env development
   pulumi config set org_id 1234567890
   ```
3. Deploy:
   ```bash
   pulumi up
   ```

## Configuration
| Key | Description | Default |
|-----|-------------|---------|
| `org_stack_name` | Full name of the organization stack | *Required* |
| `env` | Environment name (e.g., development) | *Required* |
| `org_id` | The GCP Organization ID | *Required* |
| `folder_prefix` | Prefix for folder names | `fldr` |
