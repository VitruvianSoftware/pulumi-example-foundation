# GitHub Actions Administration

This document outlines the administrative configuration required to support the GitHub Actions CI/CD pipelines, specifically focusing on the End-to-End (E2E) testing infrastructure.

## End-to-End (E2E) Testing Infrastructure

The E2E testing infrastructure is designed to deploy the complete 6-stage foundation against a sandbox Google Cloud Organization, validate resource creation via GCP APIs, and gracefully tear everything down.

Because full foundation deployments are expensive in terms of time and Google Cloud quotas, the E2E workflow is heavily gated.

### Workflow Triggers

The E2E workflow (`.github/workflows/e2e.yml`) operates under two primary modes:

1. **Manual Dispatch (Ad-hoc Validation)**
   - Always enabled.
   - Can be triggered manually via the GitHub Actions UI.
   - Accepts `action` (`up` or `destroy`) and an optional `stage` target.
2. **Automated (Push / Pull Request)**
   - **Gated by default.**
   - Only executes if the `E2E_ENABLED` repository variable is explicitly set to `"true"`.
   - Prevents accidental, long-running deployments on every minor documentation or code change.

## CI Credentials Strategy (Workload Identity Federation)

For automated pipelines to interact with Google Cloud and Pulumi Cloud, they require robust, keyless authentication. We strictly adhere to **Workload Identity Federation (WIF)**, eliminating the need for long-lived Service Account JSON keys.

### 1. GCP Service Account

Create a dedicated Google Cloud Service Account within the sandbox organization (e.g., `sa-e2e-foundation@prj-b-seed-XXXX.iam.gserviceaccount.com`). This identity requires organization-level administrative privileges:

*   `roles/resourcemanager.folderAdmin`
*   `roles/resourcemanager.projectCreator`
*   `roles/billing.admin`
*   `roles/orgpolicy.policyAdmin`
*   `roles/iam.organizationRoleAdmin`

### 2. Workload Identity Pool and Provider

Establish a Workload Identity Pool and Provider in your sandbox seed project configured to trust GitHub's OIDC tokens (`https://token.actions.githubusercontent.com`).

*   Scope the provider attribute mapping to your specific GitHub repository or organization to ensure isolation.
*   Grant the GitHub Actions identity the `roles/iam.workloadIdentityUser` role on the dedicated GCP Service Account created in Step 1.

## Required GitHub Configuration

To fully enable the E2E pipeline, configure the following Secrets and Variables within your GitHub Repository or Organization settings:

### Repository Secrets

These values are sensitive and must be stored securely.

*   `GCP_WORKLOAD_IDENTITY_PROVIDER`: The full resource name of the WIF provider (e.g., `projects/1234567890/locations/global/workloadIdentityPools/github-pool/providers/github-provider`).
*   `GCP_SERVICE_ACCOUNT`: The email address of the dedicated GCP Service Account.
*   `PULUMI_ACCESS_TOKEN`: A valid Pulumi Cloud access token for state management.

### Repository Variables

These non-secret values configure the E2E runtime environment.

*   `E2E_ORG_ID`: The Sandbox Google Cloud Organization ID.
*   `E2E_BILLING_ACCOUNT`: The Sandbox Google Cloud Billing Account ID.
*   `E2E_DOMAIN`: The Sandbox domain (e.g., `sandbox.example.com`).
*   `E2E_ENABLED`: Set to `"true"` to enable E2E runs on pushes and Pull Requests. Defaults to false if unset.
