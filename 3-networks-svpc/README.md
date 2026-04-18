# 3-Networks-SVPC - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **3-networks-svpc** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
This stage implements a **Shared VPC** (SVPC) network topology. It creates a host project (if not already existing) and the Shared VPC network with subnets, firewall rules, and private service connectivity.

Key resources:
- **VPC Network**: A custom-mode VPC network designated as a Shared VPC.
- **Subnets**: Regional subnets for workloads.
- **Private Service Access**: Configures global internal IP ranges and peering for Google-managed services (like Cloud SQL).

## Transition from Terraform to Pulumi
- **Networking Logic**: Complex networking rules are defined using the `pulumi-gcp` SDK, allowing for programmatic calculation of CIDR ranges or firewall rules.
- **Service Networking**: The connection between your VPC and Google services is handled via the `servicenetworking` resource.

## Prerequisites
- Successful deployment of the `2-environments` stage.

## Usage
1. Initialize the stack:
   ```bash
   pulumi stack init dev-networking
   ```
2. Configure:
   ```bash
   pulumi config set env development
   pulumi config set project_id <host-project-id>
   pulumi config set region1 us-central1
   ```
3. Deploy:
   ```bash
   pulumi up
   ```

## Configuration
| Key | Description | Default |
|-----|-------------|---------|
| `env` | Environment name | *Required* |
| `project_id` | The host project ID | *Required* |
| `region1` | Primary region | `us-central1` |
