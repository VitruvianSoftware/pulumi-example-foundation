# 5-app-infra

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

The purpose of this step is to deploy sample application infrastructure in one of the business unit projects using the infra pipeline set up in [4-projects](../4-projects/README.md).

This stage deploys:

- A **[Cloud Run](https://cloud.google.com/run)** service (`chat-demo`) using the official `hello` container image in the SVPC-attached project — demonstrating how applications are deployed within the VPC-connected project
- A **[BigQuery](https://cloud.google.com/bigquery)** dataset (`airline-data`) in the floating project — demonstrating a data platform deployment using the shared library's `data` component

The SVPC-attached project (`prj-{env}-{bu}-sample-svpc`) is used for the application deployment because it has network connectivity through the Shared VPC created in Stage 3. The floating project (`prj-{env}-{bu}-sample-floating`) is used for the data platform because BigQuery does not require VPC connectivity.

## Prerequisites

1. [0-bootstrap](../0-bootstrap/README.md) executed successfully.
1. [1-org](../1-org/README.md) executed successfully.
1. [2-environments](../2-environments/README.md) executed successfully.
1. [3-networks](../3-networks-svpc/README.md) executed successfully.
1. [4-projects](../4-projects/README.md) executed successfully.

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Usage

### Deploying with GitHub Actions

1. Navigate to the `5-app-infra` directory and initialize a stack for each environment:

   ```bash
   cd 5-app-infra
   pulumi stack init development
   ```

1. Set the required configuration:

   ```bash
   pulumi config set env "development"
   pulumi config set projects_stack_name "organization/vitruvian/4-projects/development"
   ```

1. (Optional) Override the default region:

   ```bash
   pulumi config set region "us-central1"   # default: us-central1
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. **Repeat for each environment** (`nonproduction`, `production`).

### Running Pulumi Locally

Same process as above — navigate, initialize, configure, and deploy.

### Customizing the Application

The sample application is intentionally minimal. To deploy your own workloads:

1. Modify `main.go` to add your application-specific infrastructure
2. Use the shared library components:
   - `pkg/app` — For Cloud Run deployments
   - `pkg/data` — For BigQuery data platform setups
3. Add additional Pulumi config values as needed for your application

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `env` | Environment name (`development`, `nonproduction`, `production`) | ✅ | — |
| `projects_stack_name` | Fully qualified Pulumi stack name of the 4-projects stage for this environment | ✅ | — |
| `region` | Region for Cloud Run deployment | | `"us-central1"` |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Resolves project IDs from Stage 4 via Stack Reference, deploys Cloud Run app and BigQuery dataset using the shared library components |
