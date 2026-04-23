# Foundation Deployer

Automated multi-stage foundation deployment tool for the Pulumi foundation.

## Overview

This is the Pulumi equivalent of the Terraform foundation's `helpers/foundation-deployer/` (35-file Go CLI). Since Pulumi handles state management natively (no `backend.tf` manipulation needed), the same workflow is achievable with a single shell script.

## Usage

```bash
# Preview all stages (dry run)
./deploy.sh --action preview

# Deploy a specific stage
./deploy.sh --action up --stage 0

# Deploy all stages sequentially
./deploy.sh --action up

# Deploy a specific stack within a stage
./deploy.sh --action up --stage 3 --stack shared

# Destroy everything (reverse order)
./deploy.sh --destroy

# Show what would be executed
./deploy.sh --action up --dry-run
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--stage <N>` | Deploy a specific stage (0-5) | All stages |
| `--action <action>` | `preview` or `up` | `preview` |
| `--stack <stack>` | Target a specific stack name | All stacks |
| `--destroy` | Destroy in reverse order | `false` |
| `--dry-run` | Show commands without executing | `false` |

## Stage Execution Order

### Deploy (forward)
```
0-bootstrap → 1-org → 2-environments → 3-networks-svpc → 4-projects → 5-app-infra
```

### Destroy (reverse)
```
5-app-infra → 4-projects → 3-networks-svpc → 2-environments → 1-org → 0-bootstrap
```

## Differences from Terraform Foundation Deployer

| Feature | TF Deployer | Pulumi Deployer |
|---------|-------------|-----------------|
| **Implementation** | 35-file Go CLI | Single shell script |
| **Backend management** | Manipulates `backend.tf` files | Not needed (Pulumi manages state natively) |
| **State migration** | Handles remote state bucket creation | Handled by `pulumi login` |
| **Dependency install** | `terraform init` | `npm install` (TS) / `go mod download` (Go) |
| **Cloud Build integration** | Full trigger management | Separate CI/CD pipeline files |

## Prerequisites

- Pulumi CLI installed and authenticated (`pulumi login`)
- Google Cloud SDK authenticated (`gcloud auth application-default login`)
- Node.js 20+ (TypeScript) or Go 1.23+ (Go)
- All `Pulumi.<stack>.yaml` configs populated (copy from `.example` files)
