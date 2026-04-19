# 0-bootstrap — Cloud Build (Alternative)

This document describes how to use **Google Cloud Build** as the CI/CD provider
instead of the default GitHub Actions. This mirrors the Terraform foundation's
default `build_cb.tf` configuration.

> [!IMPORTANT]
> Cloud Build is NOT the default for the Pulumi foundation. The default is
> [GitHub Actions with WIF](README-GitHub.md). Follow the steps below to switch.

## Architecture

```
┌──────────────────────────────────────────┐
│  CI/CD Project (prj-b-cicd)             │
│  ┌────────────────────────────────────┐  │
│  │ Cloud Source Repos (7 repos)       │  │
│  │   gcp-bootstrap, gcp-org, ...     │  │
│  ├────────────────────────────────────┤  │
│  │ Artifact Registry                  │  │
│  │   pulumi-builders (Docker)         │  │
│  ├────────────────────────────────────┤  │
│  │ Cloud Build Triggers (×10)         │  │
│  │   plan-{stage} / apply-{stage}     │  │
│  ├────────────────────────────────────┤  │
│  │ Private Worker Pool (optional)     │  │
│  │   VPC-peered, 10.3.0.0/24         │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

## How to Switch

### Step 1: Activate the Cloud Build code

```bash
cd 0-bootstrap

# Deactivate the GitHub Actions default
mv build_github_actions.go build_github_actions.go.bak

# Activate Cloud Build
mv build_cloud_build.go.example build_cloud_build.go
```

### Step 2: Update main.go

Replace the build call in `main.go`:

```diff
-       // 5b. Deploy CI/CD Build Infrastructure (GitHub Actions WIF by default)
-       buildOutputs, err := deployGitHubActionsBuild(ctx, cfg, cicd, sas)
+       // 5b. Deploy CI/CD Build Infrastructure (Cloud Build)
+       buildOutputs, err := deployCloudBuild(ctx, cfg, cicd, sas)
```

And update the exports section:

```diff
-       // 9. CI/CD build outputs (WIF)
-       if cfg.GitHubOwner != "" {
-           ctx.Export("wif_pool_name", buildOutputs.WIFPoolName)
-           ctx.Export("wif_provider_name", buildOutputs.WIFProviderName)
-       }
+       // 9. CI/CD build outputs (Cloud Build)
+       ctx.Export("cloudbuild_project_id", buildOutputs.CloudBuildProjectID)
+       ctx.Export("artifact_repo_name", buildOutputs.ArtifactRepoName)
```

### Step 3: Add Cloud Build APIs

Add these to the CI/CD project's `ActivateApis` in `projects.go`:

```go
"sourcerepo.googleapis.com",
```

### Step 4: Deploy

```bash
pulumi up
```

## What Gets Created

| Resource | Count | Description |
|----------|-------|-------------|
| Cloud Source Repos | 7 | One per stage + `gcp-policies` + `pulumi-cloudbuilder` |
| Artifact Registry | 1 | Docker repo for custom Pulumi builder images |
| AR IAM Bindings | 5 | `artifactregistry.reader` per SA |
| Cloud Build Plan Triggers | 5 | `plan-{stage}` — runs on any branch push |
| Cloud Build Apply Triggers | 5 | `apply-{stage}` — runs on main branch only |

## Cloud Build YAML Files

Place these in each stage's repository root:

### `cloudbuild-pulumi-plan.yaml`

```yaml
steps:
  - id: 'pulumi-preview'
    name: '${_GAR_REGION}-docker.pkg.dev/${_GAR_PROJECT_ID}/pulumi-builders/pulumi:latest'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        pulumi login gs://${_TF_BACKEND}
        pulumi preview --stack production --non-interactive
    env:
      - 'PULUMI_CONFIG_PASSPHRASE=${_PULUMI_CONFIG_PASSPHRASE}'
```

### `cloudbuild-pulumi-apply.yaml`

```yaml
steps:
  - id: 'pulumi-up'
    name: '${_GAR_REGION}-docker.pkg.dev/${_GAR_PROJECT_ID}/pulumi-builders/pulumi:latest'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        pulumi login gs://${_TF_BACKEND}
        pulumi up --stack production --non-interactive --yes
    env:
      - 'PULUMI_CONFIG_PASSPHRASE=${_PULUMI_CONFIG_PASSPHRASE}'
```

## Additional Outputs

| Name | Description |
|------|-------------|
| `cloudbuild_project_id` | Project ID of the CI/CD project |
| `artifact_repo_name` | Name of the Artifact Registry repository |
