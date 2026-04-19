# 0-bootstrap — GitHub Actions (Default)

This is the **default CI/CD configuration** for the Pulumi foundation. It uses
[GitHub Actions](https://docs.github.com/en/actions) with
[Workload Identity Federation (WIF)](https://cloud.google.com/iam/docs/workload-identity-federation)
for keyless authentication to Google Cloud.

> [!NOTE]
> This is equivalent to the Terraform foundation's `build_github.tf.example`, but
> activated by default. For Cloud Build, see [README-CloudBuild.md](README-CloudBuild.md).

## Architecture

```
┌──────────────────────────────────────────────────┐
│  GitHub Actions Runner                           │
│  ┌─────────────────────────────────────────────┐ │
│  │ GitHub OIDC Token (short-lived)             │ │
│  │   ↓                                         │ │
│  │ WIF Pool: foundation-pool                   │ │
│  │   ↓                                         │ │
│  │ WIF Provider: foundation-gh-provider        │ │
│  │   ↓ (attribute.repository/{owner}/{repo})   │ │
│  │ GCP SA: sa-terraform-{stage}                │ │
│  │   ↓                                         │ │
│  │ Pulumi preview / up                         │ │
│  └─────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

Each stage's GitHub repository is mapped to a specific service account via the
WIF attribute binding `attribute.repository/{owner}/{repo}`, ensuring that only
the intended repository can impersonate the corresponding SA.

## Prerequisites

1. Complete the [main bootstrap README](README.md) steps first.
2. Have your GitHub organization/user ready.
3. Know the repository names for each foundation stage.

## Configuration

Add these to your `Pulumi.<stack>.yaml`:

```yaml
config:
  # Required: GitHub organization or user name
  github_owner: "your-org"

  # Required: Repository names for each stage
  github_repo_bootstrap: "pulumi-foundation-bootstrap"
  github_repo_org: "pulumi-foundation-org"
  github_repo_env: "pulumi-foundation-environments"
  github_repo_net: "pulumi-foundation-networks"
  github_repo_proj: "pulumi-foundation-projects"

  # Optional: Override the WIF attribute condition (default restricts to github_owner)
  # wif_attribute_condition: "assertion.repository_owner=='your-org'"
```

Or via CLI:

```bash
pulumi config set github_owner "your-org"
pulumi config set github_repo_bootstrap "pulumi-foundation-bootstrap"
pulumi config set github_repo_org "pulumi-foundation-org"
pulumi config set github_repo_env "pulumi-foundation-environments"
pulumi config set github_repo_net "pulumi-foundation-networks"
pulumi config set github_repo_proj "pulumi-foundation-projects"
```

## What Gets Created

When `github_owner` is set, the bootstrap creates these resources in the CI/CD project:

| Resource | Name | Description |
|----------|------|-------------|
| Workload Identity Pool | `foundation-pool` | Groups all GitHub-based identity providers |
| WIF OIDC Provider | `foundation-gh-provider` | GitHub Actions OIDC token issuer |
| SA IAM Bindings (×5) | `wif-sa-binding-{stage}` | Maps each stage repo to its SA via `workloadIdentityUser` |

## Additional Outputs

| Name | Description |
|------|-------------|
| `wif_pool_name` | Full resource name of the Workload Identity Pool |
| `wif_provider_name` | Full resource name of the WIF OIDC provider |

## GitHub Actions Workflow Setup

After deploying bootstrap with WIF, configure your GitHub Actions workflows to
use keyless authentication:

```yaml
# .github/workflows/pulumi.yml
name: Pulumi
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  id-token: write   # Required for WIF
  contents: read

jobs:
  preview:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ secrets.WIF_PROVIDER_NAME }}
          service_account: ${{ secrets.SERVICE_ACCOUNT_EMAIL }}
      - uses: pulumi/actions@v5
        with:
          command: preview
          stack-name: production
```

### Required GitHub Secrets

Set these in each repository's Settings → Secrets:

| Secret | Value | Source |
|--------|-------|--------|
| `WIF_PROVIDER_NAME` | `projects/{project_number}/locations/global/workloadIdentityPools/foundation-pool/providers/foundation-gh-provider` | `pulumi stack output wif_provider_name` |
| `SERVICE_ACCOUNT_EMAIL` | `sa-terraform-{stage}@prj-b-seed-xxxx.iam.gserviceaccount.com` | `pulumi stack output {stage}_sa_email` |

## Migration from Key-Based Auth

If you're currently using `GOOGLE_CREDENTIALS` with a SA key:

1. Set the `github_owner` and repo configs, then run `pulumi up`
2. Update your GitHub Actions workflows to use `google-github-actions/auth@v2`
3. Copy the `wif_provider_name` output to each repo's secrets
4. Remove the `GOOGLE_CREDENTIALS` secret from your repos
5. Revoke and delete the SA key from GCP
