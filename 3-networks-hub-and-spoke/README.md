# 3-networks-hub-and-spoke

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

This step configures the same network resources as [3-networks-svpc](../3-networks-svpc/README.md), but uses the [Hub and Spoke](https://cloud.google.com/architecture/security-foundations/networking#hub-and-spoke) reference network model instead of the Dual Shared VPC architecture.

### Architecture

In the Hub and Spoke model:

- A **Hub VPC** is created with central routing in the network hub project
- **Spoke VPCs** are created per environment with GKE secondary ranges
- **Bidirectional VPC peering** connects each spoke to the hub with custom route export/import
- The same baseline firewall, DNS, NAT, and routing configurations from the SVPC variant are applied

### What Gets Created

For each environment:

- **Hub VPC** with central routing
- **Spoke VPC** with GKE secondary ranges (pod + service CIDRs)
- **VPC Peering** between hub and spoke with custom route exchange
- **Hierarchical Firewall Policies** (IAP, health checks, Windows KMS)
- **DNS Policy** with logging and inbound forwarding
- **Cloud NAT** on all regions with error logging
- **Private Service Access** for managed services
- **Restricted Google APIs routing** (`199.36.153.4/30`)
- **Default internet routes removed**

## Prerequisites

1. [0-bootstrap](../0-bootstrap/README.md) executed successfully.
1. [1-org](../1-org/README.md) executed successfully.
1. [2-environments](../2-environments/README.md) executed successfully.

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Usage

### Deploying with GitHub Actions

1. Navigate to the `3-networks-hub-and-spoke` directory and initialize a stack for each environment:

   ```bash
   cd 3-networks-hub-and-spoke
   pulumi stack init development
   ```

1. Set the required configuration:

   ```bash
   pulumi config set env "development"
   pulumi config set hub_project_id "prj-net-hub-svpc"   # from Stage 1 output
   pulumi config set spoke_project_id "prj-d-svpc"       # from Stage 1 output
   pulumi config set parent_id "organizations/YOUR_ORG_ID"
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. **Repeat for each environment** (`nonproduction`, `production`).

   **Note:** Deploy **production first** as it includes the DNS Hub that other environments depend on.

1. Proceed to the [4-projects](../4-projects/README.md) step.

### Running Pulumi Locally

Same process as above — navigate, initialize, configure, and deploy.

### Known Issue: VPC Peering Rate Limit

When deploying the hub-and-spoke architecture, you may encounter a `CONCURRENT_OPERATIONS_QUOTA_EXCEEDED` error when adding network peering. This is a transient error; wait at least one minute and retry the deploy.

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `env` | Environment name (`development`, `nonproduction`, `production`) | ✅ | — |
| `hub_project_id` | Hub VPC host project ID | ✅ | — |
| `spoke_project_id` | Spoke VPC project ID | ✅ | — |
| `parent_id` | Parent scope for firewall policies | ✅ | — |
| `region1` | Primary region | | `"us-central1"` |
| `region2` | Secondary region | | `"us-west1"` |

## Outputs

| Name | Description |
|------|-------------|
| `hierarchical_fw` | Hierarchical firewall policy ID |
| `hub_vpc_id` | Hub VPC network resource ID |
| `spoke_vpc_id` | Spoke VPC network resource ID |
| `service_perimeter_name` | VPC Service Controls perimeter name (when enabled) |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Complete hub-and-spoke network stack: hub VPC, spoke VPC, VPC peering, subnets, firewall policies, DNS, NAT, restricted APIs routing |
