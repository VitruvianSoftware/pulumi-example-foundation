# 3-networks-svpc

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. See the [stage navigation table](../0-bootstrap/README.md) for an overview of all stages.

## Purpose

The purpose of this step is to:

- Set up the global [DNS Hub](https://cloud.google.com/blog/products/networking/cloud-forwarding-peering-and-zones).
- Configure Shared VPCs with default DNS, NAT, Private Service networking, and baseline firewall rules for each environment.

This step makes use of the **Dual Shared VPC** architecture described in the **Networking** section of the [Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations/networking). To use the **Hub and Spoke** model instead, see [3-networks-hub-and-spoke](../3-networks-hub-and-spoke/README.md).

### What Gets Created

For each environment (`development`, `nonproduction`, `production`):

- **Shared VPC Host Project** designation on the network project created in Stage 1
- **VPC Network** with auto-create subnets disabled and default routes removed
- **Multi-region subnets** with GKE secondary ranges (pod + service CIDRs):
  - Region 1 (`us-central1`): `10.0.64.0/21` with pod `100.64.64.0/21` and svc `100.64.72.0/21`
  - Region 2 (`us-west1`): `10.1.64.0/21` with pod `100.65.64.0/21` and svc `100.65.72.0/21`
- **Private Google Access** enabled on all subnets with flow logging
- **Private Service Access** (PSA) for managed services like Cloud SQL and Memorystore
- **Hierarchical Firewall Policies** with baseline rules:
  - Allow [IAP TCP forwarding](https://cloud.google.com/iap/docs/using-tcp-forwarding) (SSH/RDP without public IPs) from `35.235.240.0/20`
  - Allow [load balancer health checks](https://cloud.google.com/load-balancing/docs/health-checks#firewall_rules) from `130.211.0.0/22` and `35.191.0.0/16`
  - Allow [Windows KMS activation](https://cloud.google.com/compute/docs/instances/windows/creating-managing-windows-instances#kms-server) to `35.190.247.13/32`
- **DNS Policy** with logging and [inbound query forwarding](https://cloud.google.com/dns/docs/overview#dns-server-policy-in) enabled
- **Cloud NAT** on all regions with error-only logging
- **Restricted Google APIs routing** via `199.36.153.4/30` (Private Google Access for `restricted.googleapis.com`)
- **Default internet routes removed** — VMs require tag-based routes for internet access

## Prerequisites

1. [0-bootstrap](../0-bootstrap/README.md) executed successfully.
1. [1-org](../1-org/README.md) executed successfully.
1. [2-environments](../2-environments/README.md) executed successfully.

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Usage

### Deploying with GitHub Actions

1. Navigate to the `3-networks-svpc` directory and initialize a stack for each environment:

   ```bash
   cd 3-networks-svpc
   pulumi stack init development
   ```

1. Set the required configuration:

   ```bash
   pulumi config set env "development"
   pulumi config set project_id "prj-d-svpc"        # from Stage 1 output
   pulumi config set parent_id "organizations/YOUR_ORG_ID"
   ```

1. (Optional) Override default regions:

   ```bash
   pulumi config set region1 "us-central1"   # default
   pulumi config set region2 "us-west1"      # default
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. **Repeat for each environment** (`nonproduction`, `production`):

   ```bash
   pulumi stack init nonproduction
   pulumi config set env "nonproduction"
   pulumi config set project_id "prj-n-svpc"
   pulumi config set parent_id "organizations/YOUR_ORG_ID"
   pulumi up

   pulumi stack init production
   pulumi config set env "production"
   pulumi config set project_id "prj-p-svpc"
   pulumi config set parent_id "organizations/YOUR_ORG_ID"
   pulumi up
   ```

   **Note:** The **production** environment should be deployed first as it includes the DNS Hub that other environments depend on.

1. Commit and push:

   ```bash
   git add .
   git commit -m "Initialize networks stage"
   git push origin production
   ```

1. Proceed to the [4-projects](../4-projects/README.md) step.

### Using Dedicated Interconnect

If you have provisioned Dedicated Interconnect prerequisites, you will need to extend the `main.go` to create VLAN attachments and BGP sessions. Refer to the [Dedicated Interconnect documentation](https://cloud.google.com/network-connectivity/docs/interconnect/how-to/dedicated/creating-vlan-attachments) for details.

### Using Partner Interconnect

For Partner Interconnect, VLAN attachments are placed directly into the corresponding hub projects. Refer to the [Partner Interconnect documentation](https://cloud.google.com/network-connectivity/docs/interconnect/how-to/partner/creating-vlan-attachments) for details.

### Using HA VPN (Optional)

If you cannot use Dedicated or Partner Interconnect, you can use an HA Cloud VPN to access on-premises resources. You will need to extend the `main.go` to create the VPN gateway, tunnels, and BGP configuration.

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `env` | Environment name (`development`, `nonproduction`, `production`) | ✅ | — |
| `project_id` | Shared VPC host project ID | ✅ | — |
| `parent_id` | Parent scope for firewall policies (`organizations/123` or `folders/456`) | ✅ | — |
| `region1` | Primary region for subnets and NAT | | `"us-central1"` |
| `region2` | Secondary region for subnets and NAT | | `"us-west1"` |

## Outputs

| Name | Description |
|------|-------------|
| `network_id` | VPC network resource ID |
| `network_name` | VPC network name |
| `network_self_link` | VPC network self link |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Complete network stack: SVPC host, VPC, subnets with GKE ranges, PSA, firewall policies, DNS, NAT, restricted APIs routing |
