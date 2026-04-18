# 4-Projects - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **4-projects** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
This stage is responsible for creating service projects for specific business units and applications, and attaching them to the Shared VPC networks created in stage 3.

Key resources:
- **Service Projects**: Application and Data projects for a given business unit and environment.
- **IAM**: Sets up project-level permissions.

## Transition from Terraform to Pulumi
- **Dynamic Project Creation**: Projects are created dynamically based on business codes and environment names provided in the configuration.
- **Stack References**: Retrieves folder IDs from the `2-environments` stack.

## Usage
1. Initialize the stack:
   ```bash
   pulumi stack init bu1-dev-projects
   ```
2. Configure:
   ```bash
   pulumi config set env development
   pulumi config set business_code bu1
   pulumi config set billing_account 012345-678901-ABCDEF
   pulumi config set env_stack_name <org>/<project>/development
   ```
3. Deploy:
   ```bash
   pulumi up
   ```

## Configuration
| Key | Description | Default |
|-----|-------------|---------|
| `env` | Environment name | *Required* |
| `business_code` | Business unit code | *Required* |
| `billing_account` | Billing Account ID | *Required* |
| `env_stack_name` | Name of the environment stack | *Required* |
| `project_prefix` | Prefix for project IDs | `prj` |
