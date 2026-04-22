# 5-app-infra

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

The purpose of this step is to deploy sample application infrastructure in one of the business unit projects using the infra pipeline set up in [4-projects](../4-projects/README.md).

This stage deploys:

- A **Base Compute Instance** in the SVPC-attached project (`prj-{env}-{bu}-sample-svpc`) using the shared VPC subnet.
- A **Base Compute Instance** in the Peering project (`prj-{env}-{bu}-sample-peering`) using the peering VPC subnet, with attached IAP Secure Tags for firewall evaluation.
- An optional **Confidential Space VM** in the Confidential Space project (`prj-{env}-{bu}-conf-space`), utilizing a dedicated Workload Identity Pool and Provider for attestation, and running the `confidential-space` OS image.

This perfectly mirrors the upstream Terraform foundation's `env_base` and `confidential_space` modules, demonstrating how applications and hardware-secured workloads are provisioned within the connected environments.

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
   pulumi config set projects_stack_name "VitruvianSoftware/foundation-4-projects/development"
   ```

1. (Optional) Override the bootstrap stack reference (default derives from projects stack name):

   ```bash
   pulumi config set bootstrap_stack_name "VitruvianSoftware/foundation-0-bootstrap/shared"
   ```

1. (Optional) Override the default region:

   ```bash
   pulumi config set region "us-central1"   # default: us-central1
   ```

1. (Optional) Enable Confidential Space deployment by providing an image digest:

   ```bash
   pulumi config set confidential_image_digest "sha256:exampledigest"
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. **Repeat for each environment** (`nonproduction`, `production`).

### Running Pulumi Locally

Same process as above — navigate, initialize, configure, and deploy.

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `env` | Environment name (`development`, `nonproduction`, `production`) | ✅ | — |
| `business_code` | Business Unit code (e.g. `bu1`) | | `"bu1"` |
| `projects_stack_name` | Fully qualified Pulumi stack name of the 4-projects stage for this environment | | `VitruvianSoftware/foundation-4-projects/<env>` |
| `bootstrap_stack_name` | Fully qualified Pulumi stack name of the 0-bootstrap stage (shared) | | Derived from `projects_stack_name` |
| `region` | Region for the Compute Instances | | `"us-central1"` |
| `confidential_image_digest` | SHA256 digest of the Docker image to be used for running the workload in Confidential Space | | — |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Resolves outputs from previous stages, coordinates deployment of instances, and exports results. |
| `env_base.go` | Deploys standard Compute Instances with Service Accounts and IAP tag bindings. |
| `confidential_space.go` | Deploys Confidential Space VMs and Workload Identity components for attestation. |
