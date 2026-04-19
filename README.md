# Vitruvian Software Pulumi Foundation

This repository provides a production-ready implementation of the [Google Cloud Enterprise Foundations Blueprint](https://cloud.google.com/architecture/security-foundations), built using **Pulumi** and **Go**.

It is a port of Google's [terraform-example-foundation](https://github.com/terraform-google-modules/terraform-example-foundation), providing the same security baseline, organizational structure, and operational patterns in a native programming language.

## Overview

This repo contains several distinct Pulumi projects, each within their own directory that must be applied separately, but in sequence. Stage `0-bootstrap` is manually executed, and subsequent stages are executed using the included GitHub Actions CI/CD pipeline.

### [0. Bootstrap](./0-bootstrap/)

Bootstraps the GCP organization by creating:
- **`prj-b-seed`** — Terraform/Pulumi state storage with KMS-encrypted bucket, and the service accounts used by the pipeline
- **`prj-b-cicd`** — CI/CD pipeline infrastructure (Artifact Registry, Cloud Build, Workload Identity)
- **Granular Service Accounts** — One per stage (`bootstrap`, `org`, `env`, `net`, `proj`) with least-privilege IAM at org, folder, project, and billing scopes

### [1. Organization](./1-org/)

Sets up the organizational folder structure and shared projects:
- **Folders**: `fldr-common`, `fldr-network`, plus environment folders
- **Common Projects**: `prj-c-logging`, `prj-c-billing-export`, `prj-c-scc`, `prj-c-kms`, `prj-c-secrets`
- **Network Projects**: `prj-net-dns`, `prj-net-interconnect`, `prj-{d,n,p}-svpc`
- **Org Policies**: 14+ boolean constraints (serial port, nested virtualization, OS Login, SA key creation, public access prevention, etc.) + list policies (VM external IP deny, domain-restricted sharing, protocol forwarding)
- **Centralized Logging**: Org sinks to Storage, Pub/Sub, and Logging buckets with BigQuery linked datasets
- **SCC Notifications**: Pub/Sub pipeline streaming all active Security Command Center findings
- **Tags**: Org-level environment classification tags

### [2. Environments](./2-environments/)

Creates per-environment KMS and Secrets projects:
- `prj-{d,n,p}-kms` — Environment-level Cloud KMS
- `prj-{d,n,p}-secrets` — Environment-level Secret Manager

### [3. Networks (Choose ONE)](./3-networks-svpc/)

**Shared VPC** (`3-networks-svpc/`):
- Multi-region subnets with GKE secondary ranges (pod + service CIDRs)
- Shared VPC host project designation
- Hierarchical firewall policies (IAP, health checks, Windows KMS)
- DNS policy with logging and inbound forwarding
- Cloud NAT on all regions with error logging
- Private Service Access for managed services
- Restricted Google APIs routing (`199.36.153.4/30`)
- Default internet routes removed

**Hub-and-Spoke** (`3-networks-hub-and-spoke/`):
- Hub VPC with central routing
- Spoke VPC with GKE secondary ranges
- Bidirectional VPC peering with custom route export/import
- Same firewall, DNS, NAT, and routing as SVPC

### [4. Projects](./4-projects/)

Creates business unit projects with:
- BU subfolder under each environment folder
- Three project types per BU: **SVPC-attached**, **floating**, **peering**
- SVPC service project attachment to the host network project
- Infrastructure pipeline project (`prj-c-{bu}-infra-pipeline`) under common folder

### [5. App Infrastructure](./5-app-infra/)

Sample application deployment:
- Cloud Run service (hello-world)
- BigQuery dataset (data platform)

## Final View

```
example-organization
└── fldr-common
    ├── prj-c-logging
    ├── prj-c-billing-export
    ├── prj-c-scc
    ├── prj-c-kms
    ├── prj-c-secrets
    ├── prj-c-bu1-infra-pipeline
    └── prj-c-bu2-infra-pipeline
└── fldr-network
    ├── prj-net-dns
    ├── prj-net-interconnect
    ├── prj-d-svpc
    ├── prj-n-svpc
    └── prj-p-svpc
└── fldr-development
    ├── prj-d-kms
    ├── prj-d-secrets
    └── fldr-development-bu1
        ├── prj-d-bu1-sample-svpc
        ├── prj-d-bu1-sample-floating
        └── prj-d-bu1-sample-peering
└── fldr-nonproduction
    ├── prj-n-kms
    ├── prj-n-secrets
    └── fldr-nonproduction-bu1
        ├── prj-n-bu1-sample-svpc
        ├── prj-n-bu1-sample-floating
        └── prj-n-bu1-sample-peering
└── fldr-production
    ├── prj-p-kms
    ├── prj-p-secrets
    └── fldr-production-bu1
        ├── prj-p-bu1-sample-svpc
        ├── prj-p-bu1-sample-floating
        └── prj-p-bu1-sample-peering
└── fldr-bootstrap
    ├── prj-b-cicd
    └── prj-b-seed
```

## Branching Strategy

The CI/CD pipeline (`.github/workflows/pulumi-ci.yml`) follows the same strategy as the Terraform foundation:

| Branch | Environment | Trigger |
|--------|-------------|---------|
| `development` | Development | `pulumi up` on merge |
| `nonproduction` | Non-production | `pulumi up` on merge |
| `production` | Production + Shared | `pulumi up` on merge |
| Feature branches | All (preview only) | `pulumi preview` on PR |

## Prerequisites

- Pulumi CLI installed
- Go 1.21+ installed
- Access to a Google Cloud Organization
- A Billing Account ID
- GitHub repository with `PULUMI_ACCESS_TOKEN` and `GOOGLE_CREDENTIALS` secrets

## Using the Shared Library

All stages use the [Vitruvian Software Pulumi Library](https://github.com/VitruvianSoftware/pulumi-library) for reusable components:
- `pkg/project` — Project factory with API activation
- `pkg/iam` — Multi-scope IAM member bindings
- `pkg/policy` — Organization policy enforcement
- `pkg/networking` — VPC and subnet management
- `pkg/app` — Cloud Run deployment
- `pkg/data` — BigQuery data platform

## Intended Usage and Support

This repository is intended as an example to be forked, tweaked, and maintained in the user's own version-control system; the modules within this repository are not intended for use as remote references.
Though this blueprint can help accelerate your foundation design and build, we assume that you have the engineering skills and teams to deploy and customize your own foundation based on your own requirements.

## Optional Variables

Some configuration values used to deploy the stages have defaults. Check those **before deployment** to ensure they match your requirements. For more information, see the configuration reference tables in each stage's README:

- Step 0-bootstrap: [Configuration Reference](./0-bootstrap/README.md#configuration-reference)
- Step 1-org: [Configuration Reference](./1-org/README.md#configuration-reference)
- Step 2-environments: [Configuration Reference](./2-environments/README.md#configuration-reference)
- Step 3-networks-svpc: [Configuration Reference](./3-networks-svpc/README.md#configuration-reference)
- Step 3-networks-hub-and-spoke: [Configuration Reference](./3-networks-hub-and-spoke/README.md#configuration-reference)
- Step 4-projects: [Configuration Reference](./4-projects/README.md#configuration-reference)
- Step 5-app-infra: [Configuration Reference](./5-app-infra/README.md#configuration-reference)

## Documentation

- [Troubleshooting Guide](./docs/TROUBLESHOOTING.md) — Solutions for common deployment issues
- [Glossary](./docs/GLOSSARY.md) — Definitions of key terms used throughout the documentation
- [FAQ](./docs/FAQ.md) — Frequently asked questions about the foundation
- [Onboarding Guide](./ONBOARDING.md) — Step-by-step quickstart for deploying your foundation

## Public vs. Private
- This repo is a **Public Reference**.
- Real Vitruvian Software implementations are managed in **Private** `gcp-*` repositories that consume this foundation logic.
