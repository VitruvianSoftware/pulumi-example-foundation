# 4-projects

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

The purpose of this step is to set up the folder structure, projects, and infrastructure pipelines for applications that are connected as service projects to the Shared VPC created in the previous stage.

For each business unit, this stage creates:

- A **business unit subfolder** under each environment folder (e.g., `fldr-development-bu1`)
- **Three project types** per business unit:
  - **SVPC-attached** (`prj-{env}-{bu}-sample-svpc`) — Connected as a service project to the Shared VPC host, with VPC-SC perimeter attachment
  - **Floating** (`prj-{env}-{bu}-sample-floating`) — Standalone project not attached to any VPC
  - **Peering** (`prj-{env}-{bu}-sample-peering`) — Project with its own VPC, subnet, DNS policy, and bi-directional peering to the Shared VPC host, with a full firewall policy including IAP secure tags
- An **infrastructure pipeline project** (`prj-c-{bu}-infra-pipeline`) under the common folder with Cloud Build APIs
- **CMEK storage** — KMS keyring, crypto key, and CMEK-encrypted GCS bucket on the SVPC project
- **Budget alerts** on every project with configurable thresholds
- **Project labels** matching the upstream foundation's metadata convention (8 labels per project)

Running this code as-is should generate a structure as shown below:

```
example-organization/
└── fldr-development
    └── fldr-development-bu1
        ├── prj-d-bu1-sample-floating
        ├── prj-d-bu1-sample-svpc      (+ CMEK bucket, VPC-SC)
        └── prj-d-bu1-sample-peering   (+ VPC, subnet, DNS, peering, firewall)
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

1. (Optional) Override prefixes and feature toggles:

   ```bash
   # Prefixes
   pulumi config set project_prefix "prj"     # default: prj
   pulumi config set folder_prefix "fldr"      # default: fldr

   # Budget
   pulumi config set budget_amount 1000         # default: 1000
   pulumi config set budget_spend_basis "FORECASTED_SPEND"  # default

   # VPC-SC
   pulumi config set enforce_vpcsc true          # default: true

   # Peering network
   pulumi config set peering_enabled true        # default: true
   pulumi config set peering_iap_fw_rules_enabled true  # default: true
   pulumi config set subnet_region "us-central1"
   pulumi config set subnet_ip_range "10.3.64.0/21"

   # CMEK storage
   pulumi config set cmek_enabled true           # default: true
   pulumi config set keyring_name "bu1-sample-keyring"
   pulumi config set key_rotation_period "7776000s"

   # Metadata (applied as project labels)
   pulumi config set primary_contact "example@example.com"
   pulumi config set billing_code "1234"
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
| `budget_amount` | Budget amount per project (USD) | | `1000` |
| `budget_spend_basis` | Budget threshold basis: `CURRENT_SPEND` or `FORECASTED_SPEND` | | `"FORECASTED_SPEND"` |
| `enforce_vpcsc` | Attach SVPC project to VPC-SC perimeter | | `true` |
| `peering_enabled` | Deploy peering VPC infrastructure | | `true` |
| `peering_iap_fw_rules_enabled` | Create IAP SSH/RDP firewall rules + secure tags | | `true` |
| `subnet_region` | Region for the peering subnet | | `"us-central1"` |
| `subnet_ip_range` | IP CIDR range for the peering subnet | | `"10.3.64.0/21"` |
| `firewall_enable_logging` | Enable logging on firewall rules | | `true` |
| `windows_activation_enabled` | Enable Windows KMS activation egress rule | | `false` |
| `optional_fw_rules_enabled` | Enable load balancer health check firewall rules | | `false` |
| `cmek_enabled` | Create KMS keyring + CMEK-encrypted GCS bucket | | `true` |
| `location_kms` | KMS keyring location | | Same as `subnet_region` |
| `location_gcs` | GCS bucket location | | `"US"` |
| `keyring_name` | KMS keyring name | | `"{bu}-sample-keyring"` |
| `key_name` | KMS crypto key name | | `"crypto-key-example"` |
| `key_rotation_period` | Crypto key rotation period | | `"7776000s"` |
| `primary_contact` | Primary contact email (used in project labels) | | `"example@example.com"` |
| `secondary_contact` | Secondary contact email (used in project labels) | | `"example2@example.com"` |
| `billing_code` | Billing code for project labels | | `"1234"` |
| `folder_deletion_protection` | Prevent accidental folder deletion | | `true` |

## Outputs

| Name | Description |
|------|-------------|
| `bu_folder_id` | Business unit folder ID |
| `svpc_project_id` | SVPC-attached project ID |
| `floating_project_id` | Floating project ID |
| `peering_project_id` | Peering project ID |
| `peering_network` | Peering VPC network self-link |
| `infra_pipeline_project_id` | Infrastructure pipeline project ID |
| `network_project_id` | Network project ID (passed through from Stage 1) |
| `cmek_bucket` | CMEK-encrypted GCS bucket name |
| `cmek_keyring` | KMS keyring name |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Configuration loading, folder creation, orchestration, project labels helper |
| `business_unit.go` | Creates three project types (SVPC, floating, peering) with labels, budget, VPC-SC attachment; delegates to peering and CMEK modules |
| `peering.go` | Full peering network: VPC, subnet, DNS policy, bi-directional peering, firewall policy with IAP secure tags |
| `cmek.go` | KMS keyring, crypto key, GCS service account IAM, CMEK-encrypted GCS bucket |
