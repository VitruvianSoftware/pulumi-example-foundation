# pulumi-example-foundation

This example repository shows how Pulumi with Go can build a secure Google Cloud foundation, following the [Google Cloud Enterprise Foundations Blueprint](https://cloud.google.com/architecture/security-foundations).
It is a port of the [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation) to Pulumi and Go.

## Overview

This repo contains several distinct Pulumi projects, each within their own directory that should be applied in sequence.
Stage `0-bootstrap` is manually executed, and subsequent stages can be executed using your preferred CI/CD tool.

Each of these Pulumi projects are layered on top of each other using Pulumi Stack References.

### [0. bootstrap](./0-bootstrap/)
Bootstraps the GCP organization, creating the seed project, state bucket (for Terraform compatibility or just GCS storage), and the granular service accounts for each subsequent stage.

### [1. org](./1-org/)
Sets up organization-level resources: folders (common, network), projects for logging, billing export, SCC, KMS, and secrets. Configures organization-wide log sinks.

### [2. environments](./2-environments/)
Creates environment-specific folders (development, nonproduction, production) and shared resources within those environments.

### [3. networks-svpc](./3-networks-svpc/) or [3. networks-hub-and-spoke](./3-networks-hub-and-spoke/)
Deploys the networking stack. You can choose between a Shared VPC (SVPC) model or a Hub-and-Spoke model.

### [4. projects](./4-projects/)
Creates service projects for business units and attaches them to the Shared VPCs.

### [5. app-infra](./5-app-infra/)
Deploys application-level infrastructure (like Cloud Run or GKE) into the service projects.

## Prerequisites

- [Pulumi CLI](https://www.pulumi.com/docs/get-started/install/)
- [Go](https://golang.org/doc/install) (version 1.21+)
- Google Cloud SDK (`gcloud`)
- Administrative access to a Google Cloud Organization and Billing Account.

## Usage

1.  Navigate to each stage in order (0, 1, 2...).
2.  Initialize the stack: `pulumi stack init <stack-name>`.
3.  Configure required variables: `pulumi config set gcp:project <seed-project-id>`, etc.
4.  Preview changes: `pulumi preview`.
5.  Deploy changes: `pulumi up`.

Refer to the README in each subdirectory for stage-specific instructions.
