# 5-App-Infra - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **5-app-infra** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
This stage demonstrates the deployment of actual application infrastructure into the service projects created in stage 4.

Example resources:
- **BigQuery Dataset**: Data warehouse resources in the data project.
- **Cloud Run Service**: A sample web application deployed to the app project.

## Transition from Terraform to Pulumi
This stage showcases how application teams can use Pulumi to deploy their own resources while leveraging the foundation created by the platform team via Stack References.

## Usage
1. Initialize the stack:
   ```bash
   pulumi stack init my-app
   ```
2. Configure:
   ```bash
   pulumi config set projects_stack_name <org>/<project>/bu1-dev-projects
   ```
3. Deploy:
   ```bash
   pulumi up
   ```
