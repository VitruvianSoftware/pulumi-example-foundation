# Errata: Deviations from Upstream Terraform Foundation

This document describes known differences between the
[Pulumi Example Foundation](../README.md) and the upstream
[Google Terraform Example Foundation](https://github.com/terraform-google-modules/terraform-example-foundation)
that it is ported from. These deviations are tracked here so operators
can evaluate whether they affect their deployment and plan accordingly.

## CI/CD Tool Support

**Upstream:** Provides full deployment instructions and infrastructure for five CI/CD tools:
- Google Cloud Build (default)
- GitHub Actions
- GitLab CI/CD
- Jenkins
- Terraform Cloud / Terraform Enterprise

**This port:** Supports **GitHub Actions only**. The deployment pipeline template
is provided in `build/pulumi-ci.yml` and is copied to the operator's repository
during onboarding. There are no equivalent `README-Jenkins.md`,
`README-GitLab.md`, or `README-Terraform-Cloud.md` files.

## Policy Validation

**Upstream:** Includes a `policy-library/` directory with OPA-based constraint
templates and uses `gcloud beta terraform vet` to validate plans against
organizational policies before apply.

**This port:** Does not include policy validation infrastructure. Pulumi does
not have a direct equivalent of `terraform vet`. Policy enforcement is handled
via organization policies deployed in Stage 1 and via Pulumi's
[CrossGuard](https://www.pulumi.com/docs/using-pulumi/crossguard/) policy-as-code
framework, which can be adopted separately.

## State Backend

**Upstream:** Uses a GCS-backed Terraform remote state with
`terraform_remote_state` data sources for cross-stage communication. Each stage
has a `backend.tf` file that must be updated with the state bucket name.

**This port:** Uses Pulumi's native
[Stack References](./GLOSSARY.md#pulumi-stack-reference) for cross-stage
communication. State can be stored in Pulumi Cloud (default), a local backend,
or a self-managed GCS backend. There is no `backend.tf` equivalent to update.

## Cloud Build Integration

**Upstream:** The bootstrap stage creates Cloud Build triggers, worker pools,
and Artifact Registry repositories as part of the CI/CD project. Cloud Build
is deeply integrated into the deployment workflow.

**This port:** The CI/CD project (`prj-b-cicd`) is created with the relevant
APIs enabled, but Cloud Build triggers are not provisioned. CI/CD is handled
entirely by GitHub Actions with Workload Identity Federation (WIF) or service
account key authentication.

## Networking: Interconnect and VPN

**Upstream:** Includes complete Terraform modules for:
- Dedicated Interconnect VLAN attachments
- Partner Interconnect VLAN attachments
- HA VPN configuration with BGP sessions

**This port:** Creates the Interconnect project but does not include
Interconnect or VPN resource definitions. The README provides guidance on
where to extend the code, but the resources must be added manually.

## VPC Service Controls

**Upstream:** Includes VPC Service Controls (VPC-SC) perimeter configuration
with Access Context Manager integration, including `access_context.auto.tfvars`
for perimeter member management.

**This port:** Access Context Manager policy creation is supported in Stage 1,
but VPC-SC perimeters and access levels are not configured. The firewall
policies and Private Google Access routes provide a baseline network security
posture.

## Assured Workloads

**Upstream:** Stage 2 (`2-environments`) includes optional Assured Workloads
configuration for compliance-regulated environments.

**This port:** Stage 2 creates per-environment KMS and Secrets projects but
does not include Assured Workloads resource definitions. The `assuredworkloads`
API is enabled in the seed project, but workload creation must be added
manually.

## App Infrastructure: Confidential Space

**Upstream:** Stage 5 (`5-app-infra`) deploys a Confidential Space workload
with integrity verification, token claim configuration, and Source Repository
integration.

**This port:** Stage 5 deploys a sample Cloud Run service and BigQuery dataset
using the shared library components. Confidential Space resources are not
included.

## Naming Conventions

**Upstream:** Uses a consistent naming scheme documented in the
[Security Foundations Guide](https://cloud.google.com/architecture/security-foundations/using-example-terraform#naming_conventions),
including random suffixes on project IDs to avoid collisions.

**This port:** ~~Does not append random suffixes.~~ **Resolved.** All project
IDs now receive a 4-character random hex suffix (e.g., `prj-b-seed-a1b2`) via
the `RandomProjectID` option on the shared `project.NewProject` component, and
the state bucket name receives an independent suffix. This matches the upstream
`terraform-google-project-factory` module's `random_project_id` feature. The
suffix is generated once by a `random.RandomId` resource and persisted in
Pulumi state.

## Essential Contacts and Tags

**Upstream:** Provides detailed configuration tables for Essential Contacts
(notification categories by group) and organization-level tags (environment
classification).

**This port:** Implements both Essential Contacts and Tags in Stage 1, but the
configuration is applied programmatically in Go rather than via `.tfvars` files.
Customization requires modifying the Go source.
