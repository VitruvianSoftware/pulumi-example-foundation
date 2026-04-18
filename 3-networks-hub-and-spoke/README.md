# 3-Networks-Hub-and-Spoke - Pulumi GCP Go Port

This directory contains the Pulumi Go implementation of the **3-networks-hub-and-spoke** stage from the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation).

## Purpose
This stage implements a **Hub-and-Spoke** network topology, which is often used for centralized egress/ingress and cross-environment connectivity.

*Note: The current implementation in this directory is a placeholder template. It demonstrates the structural setup for a Pulumi Go network project.*

## Transition from Terraform to Pulumi
This port aims to replace the complex HCL peering and VPN logic with structured Go code, facilitating better management of complex peering relationships and routing.

## Prerequisites
- Successful deployment of the `2-environments` stage.

## Usage
1. Initialize the stack:
   ```bash
   pulumi stack init hub-and-spoke
   ```
2. Deploy:
   ```bash
   pulumi up
   ```
