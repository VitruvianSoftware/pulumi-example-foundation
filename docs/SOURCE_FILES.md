# Go Source File Reference

This document provides a comprehensive reference for all Go source files across
the foundation stages. It is the Go foundation's equivalent of the per-module
READMEs in the TypeScript foundation.

## 0-bootstrap (5 files, ~1,214 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~250 | Orchestrates bootstrap: config loading, folder creation, project/IAM coordination, WIF setup, output exports |
| `projects.go` | ~200 | Creates Seed project (KMS ring, crypto key, encrypted state bucket) and CI/CD project with labels and deletion protection |
| `iam.go` | ~400 | 5 granular SAs, least-privilege IAM at org/parent/seed/cicd/billing scopes, SA self-impersonation, editor removal |
| `build_github_actions.go` | ~200 | WIF pool, OIDC provider, per-SA repo bindings for GitHub Actions |
| `groups.go` | ~160 | Required and optional Cloud Identity group creation |

**Alternative CI/CD files (not compiled by default):**

| File | Description |
|------|-------------|
| `build_cloud_build.go.example` | Cloud Build provisioning (CSR, Artifact Registry, triggers) |
| `build_gitlab.go.example` | GitLab WIF OIDC provisioning |

## 1-org (10 files, ~1,870 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~180 | Config loading, orchestration, output exports (29 outputs) |
| `folders.go` | ~80 | Common, network, and 3 environment folders |
| `projects.go` | ~250 | 8+ shared projects: logging, billing-export, SCC, KMS, secrets, DNS, interconnect, network |
| `policies.go` | ~300 | 14+ boolean + list organization policies via `pkg/policy` |
| `logging.go` | ~200 | Org-level sinks to Storage, Pub/Sub, BigQuery; billing log sink |
| `scc.go` | ~100 | SCC notification with Pub/Sub topic and subscription |
| `tags.go` | ~80 | Org-level environment classification tags |
| `iam.go` | ~150 | Org admin IAM, Essential Contacts permissions |
| `essential_contacts.go` | ~80 | Essential Contacts notification channels |
| `cai_monitoring.go` | ~150 | Cloud Asset Inventory monitoring (Cloud Function v2, feeds, topics) |

**Supporting:**

| File | Description |
|------|-------------|
| `cai-monitoring-function/` | Node.js Cloud Function source for CAI monitoring |

## 2-environments (2 files, ~477 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~180 | Config, Stack Reference to Stage 1, iterates 3 environments |
| `env_baseline.go` | ~297 | Per-environment KMS + Secrets project creation with labels |

## 3-networks-svpc (1 file, ~364 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~364 | Complete network stack: SVPC host, VPC, subnets with GKE ranges, PSA, hierarchical firewall policies, DNS, NAT, restricted APIs routing |

## 3-networks-hub-and-spoke (1 file, ~493 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~493 | Hub VPC, spoke VPC, VPC peering with route exchange, subnets, firewall policies, DNS, NAT, restricted APIs routing |

## 4-projects (5 files, ~1,171 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~200 | Config, Stack References, folder creation, BU orchestration, project labels helper |
| `business_unit.go` | ~400 | 4 project types (SVPC, floating, peering, confidential space), labels, budgets, VPC-SC |
| `peering.go` | ~250 | Full peering network: VPC, subnet, DNS, bi-directional peering, firewall with IAP secure tags |
| `cmek.go` | ~150 | KMS keyring, crypto key, GCS SA IAM, CMEK-encrypted GCS bucket |
| `confidential_space.go` | ~170 | Confidential Space project, workload SA, IAM, SVPC + VPC-SC attachment |

## 5-app-infra (3 files, ~479 lines)

| File | Lines | Description |
|------|------:|-------------|
| `main.go` | ~150 | Stack References to Stages 0 + 4, compute instance coordination, output exports |
| `env_base.go` | ~200 | Standard Compute Instances with Service Accounts and IAP tag bindings |
| `confidential_space.go` | ~130 | Confidential Space VMs, Workload Identity Pool/Provider, attestation config |

## Total: 27 Go files, ~6,068 lines

## Related

- [Pulumi Library](https://github.com/VitruvianSoftware/pulumi-library/go) — Shared component library (13 Go packages)
- [TypeScript Foundation](https://github.com/VitruvianSoftware/pulumi_ts-example-foundation) — TS equivalent with 21 modules
- [Upstream Terraform](https://github.com/terraform-google-modules/terraform-example-foundation) — Reference architecture
