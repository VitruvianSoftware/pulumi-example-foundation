# 0-bootstrap — GitLab CI/CD (Alternative)

This document describes how to use **GitLab CI/CD** as the CI/CD provider
instead of the default GitHub Actions. This mirrors the Terraform foundation's
`build_gitlab.tf.example`.

> [!IMPORTANT]
> GitLab is NOT the default for the Pulumi foundation. The default is
> [GitHub Actions with WIF](README-GitHub.md). Follow the steps below to switch.

## Architecture

```
┌──────────────────────────────────────────────────┐
│  GitLab CI/CD Runner                             │
│  ┌─────────────────────────────────────────────┐ │
│  │ GitLab OIDC Token (CI_JOB_JWT_V2)           │ │
│  │   ↓                                         │ │
│  │ WIF Pool: foundation-pool                   │ │
│  │   ↓                                         │ │
│  │ WIF Provider: foundation-gl-provider        │ │
│  │   ↓ (attribute.project_path/{owner}/{repo}) │ │
│  │ GCP SA: sa-terraform-{stage}                │ │
│  │   ↓                                         │ │
│  │ Pulumi preview / up                         │ │
│  └─────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

### Key Differences from GitHub

| Aspect | GitHub Actions | GitLab CI/CD |
|--------|---------------|--------------|
| OIDC Issuer | `https://token.actions.githubusercontent.com` | `https://gitlab.com` (or self-hosted) |
| Identity Attribute | `attribute.repository` | `attribute.project_path` |
| Attribute Condition | `assertion.repository_owner=='{owner}'` | `assertion.project_path.startsWith('{owner}/')` |
| Provider ID | `foundation-gh-provider` | `foundation-gl-provider` |

## How to Switch

### Step 1: Activate the GitLab code

```bash
cd 0-bootstrap

# Deactivate the GitHub Actions default
mv build_github_actions.go build_github_actions.go.bak

# Activate GitLab
mv build_gitlab.go.example build_gitlab.go
```

### Step 2: Update Config struct in `main.go`

Replace the GitHub config fields with GitLab equivalents:

```diff
-   // GitHub Actions CI/CD — default CI/CD provider.
-   GitHubOwner             string
-   GitHubRepoBootstrap     string
-   ...
+   // GitLab CI/CD
+   GitLabOwner            string
+   GitLabRepoBootstrap    string
+   GitLabRepoOrg          string
+   GitLabRepoEnv          string
+   GitLabRepoNet          string
+   GitLabRepoProj         string
+   GitLabRepoCICDRunner   string
```

### Step 3: Update `loadConfig()` in `main.go`

```diff
-   GitHubOwner:           conf.Get("github_owner"),
-   ...
+   GitLabOwner:           conf.Get("gitlab_owner"),
+   GitLabRepoBootstrap:   conf.Get("gitlab_repo_bootstrap"),
+   GitLabRepoOrg:         conf.Get("gitlab_repo_org"),
+   GitLabRepoEnv:         conf.Get("gitlab_repo_env"),
+   GitLabRepoNet:         conf.Get("gitlab_repo_net"),
+   GitLabRepoProj:        conf.Get("gitlab_repo_proj"),
+   GitLabRepoCICDRunner:  conf.Get("gitlab_repo_cicd_runner"),
```

### Step 4: Update the build call

```diff
-   buildOutputs, err := deployGitHubActionsBuild(ctx, cfg, cicd, sas)
+   buildOutputs, err := deployGitLabBuild(ctx, cfg, cicd, sas)
```

### Step 5: Configure

```bash
pulumi config set gitlab_owner "your-gitlab-group"
pulumi config set gitlab_repo_bootstrap "pulumi-foundation-bootstrap"
pulumi config set gitlab_repo_org "pulumi-foundation-org"
pulumi config set gitlab_repo_env "pulumi-foundation-environments"
pulumi config set gitlab_repo_net "pulumi-foundation-networks"
pulumi config set gitlab_repo_proj "pulumi-foundation-projects"
pulumi config set gitlab_repo_cicd_runner "pulumi-cicd-runner"
```

### Step 6: Deploy

```bash
pulumi up
```

## What Gets Created

| Resource | Name | Description |
|----------|------|-------------|
| Workload Identity Pool | `foundation-pool` | Groups all GitLab-based identity providers |
| WIF OIDC Provider | `foundation-gl-provider` | GitLab OIDC token issuer (`https://gitlab.com`) |
| SA IAM Bindings (×5) | `wif-sa-binding-{stage}` | Maps each stage project to its SA via `workloadIdentityUser` |
| SA IAM Binding | `bootstrap-wif-pool-admin` | Bootstrap SA gets `workloadIdentityPoolAdmin` on CI/CD project |

## GitLab CI/CD Pipeline Setup

After deploying bootstrap with WIF, configure your `.gitlab-ci.yml`:

```yaml
# .gitlab-ci.yml
stages:
  - validate
  - deploy

variables:
  PULUMI_STACK: production

.gcp_auth: &gcp_auth
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: https://iam.googleapis.com/${WIF_PROVIDER_NAME}
  before_script:
    - |
      gcloud iam workload-identity-pools create-cred-config \
        ${WIF_PROVIDER_NAME} \
        --service-account=${SERVICE_ACCOUNT_EMAIL} \
        --output-file=credentials.json \
        --credential-source-type=text \
        --credential-source-file=/dev/stdin \
        <<< "$GITLAB_OIDC_TOKEN"
    - export GOOGLE_APPLICATION_CREDENTIALS=$(pwd)/credentials.json

preview:
  stage: validate
  <<: *gcp_auth
  script:
    - pulumi preview --stack $PULUMI_STACK --non-interactive
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

deploy:
  stage: deploy
  <<: *gcp_auth
  script:
    - pulumi up --stack $PULUMI_STACK --non-interactive --yes
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
```

### Required GitLab CI/CD Variables

Set these in each project's Settings → CI/CD → Variables:

| Variable | Value | Source |
|----------|-------|--------|
| `WIF_PROVIDER_NAME` | `projects/{number}/locations/global/workloadIdentityPools/foundation-pool/providers/foundation-gl-provider` | `pulumi stack output wif_provider_name` |
| `SERVICE_ACCOUNT_EMAIL` | `sa-terraform-{stage}@prj-b-seed-xxxx.iam.gserviceaccount.com` | `pulumi stack output {stage}_sa_email` |

## Self-Hosted GitLab

If using a self-hosted GitLab instance, update the OIDC issuer URI in
`build_gitlab.go`:

```diff
  Oidc: &iam.WorkloadIdentityPoolProviderOidcArgs{
-     IssuerUri: pulumi.String("https://gitlab.com"),
+     IssuerUri: pulumi.String("https://gitlab.your-domain.com"),
  },
```
