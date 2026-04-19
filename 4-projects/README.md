# 4-projects

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

The purpose of this step is to set up the folder structure, projects, and infrastructure pipelines for applications that are connected as service projects to the Shared VPC created in the previous stage.

For each business unit, this stage creates:

- A **business unit subfolder** under each environment folder (e.g., `fldr-development-bu1`)
- **Three project types** per business unit:
  - **SVPC-attached** (`prj-{env}-{bu}-sample-svpc`) — Connected as a service project to the Shared VPC host
  - **Floating** (`prj-{env}-{bu}-sample-floating`) — Standalone project not attached to any VPC
  - **Peering** (`prj-{env}-{bu}-sample-peering`) — Project with VPC peering to the Shared VPC
- An **infrastructure pipeline project** (`prj-c-{bu}-infra-pipeline`) under the common folder

Running this code as-is should generate a structure as shown below:

```
example-organization/
└── fldr-development
    └── fldr-development-bu1
        ├── prj-d-bu1-sample-floating
        ├── prj-d-bu1-sample-svpc
        └── prj-d-bu1-sample-peering
└── fldr-nonproduction
    └── fldr-nonproduction-bu1
        ├── prj-n-bu1-sample-floating
        ├── prj-n-bu1-sample-svpc
        └── prj-n-bu1-sample-peering
└── fldr-production
    └── fldr-production-bu1
        ├── prj-p-bu1-sample-floating
        ├── prj-p-bu1-sample-svpc
        └── prj-p-bu1-sample-peering
└── fldr-common
    └── prj-c-bu1-infra-pipeline
```

## Prerequisites

1. [0-bootstrap](../0-bootstrap/README.md) executed successfully.
1. [1-org](../1-org/README.md) executed successfully.
1. [2-environments](../2-environments/README.md) executed successfully.
1. [3-networks](../3-networks-svpc/README.md) executed successfully.

**Note:** As mentioned in the [0-bootstrap README](../0-bootstrap/README.md), make sure that you have requested at least 50 additional projects for the **projects step service account** (`sa-terraform-proj`), otherwise you may face a project quota exceeded error.

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Usage

### Deploying with GitHub Actions

1. Navigate to the `4-projects` directory and initialize a stack for each environment:

   ```bash
   cd 4-projects
   pulumi stack init development
   ```

1. Set the required configuration:

   ```bash
   pulumi config set env "development"
   pulumi config set business_code "bu1"
   pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
   pulumi config set org_stack_name "organization/vitruvian/1-org/production"
   ```

1. (Optional) Override prefixes:

   ```bash
   pulumi config set project_prefix "prj"     # default: prj
   pulumi config set folder_prefix "fldr"      # default: fldr
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. **Repeat for each environment** (`nonproduction`, `production`).

1. Proceed to the [5-app-infra](../5-app-infra/README.md) step.

### Adding Additional Business Units

To create a new business unit (e.g., `bu2`), deploy additional stacks with different `business_code` values:

```bash
pulumi stack init development-bu2
pulumi config set env "development"
pulumi config set business_code "bu2"
pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
pulumi config set org_stack_name "organization/vitruvian/1-org/production"
pulumi up
```

Repeat for each environment and business unit combination.

### Running Pulumi Locally

Same process as above — navigate, initialize, configure, and deploy.

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `env` | Environment name (`development`, `nonproduction`, `production`) | ✅ | — |
| `business_code` | Short business unit identifier (e.g., `bu1`, `bu2`) | ✅ | — |
| `billing_account` | Billing account ID | ✅ | — |
| `org_stack_name` | Fully qualified Pulumi stack name of the 1-org stage | ✅ | — |
| `project_prefix` | Project name prefix | | `"prj"` |
| `folder_prefix` | Folder name prefix | | `"fldr"` |

## Outputs

| Name | Description |
|------|-------------|
| `bu_folder_id` | Business unit folder ID |
| `svpc_project_id` | SVPC-attached project ID |
| `floating_project_id` | Floating project ID |
| `peering_project_id` | Peering project ID |
| `infra_pipeline_project_id` | Infrastructure pipeline project ID |
| `network_project_id` | Network project ID (passed through from Stage 1) |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Orchestrates project creation: resolves folder/network IDs via Stack References, creates BU folder, deploys projects and infra pipeline |
| `business_unit.go` | Creates the three project types (SVPC, floating, peering) with appropriate APIs and Shared VPC service project attachment |
