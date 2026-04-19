# 0-bootstrap

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. The following table lists the stages of this deployment.

<table>
<tbody>
<tr>
<td>0-bootstrap (this file)</td>
<td>Bootstraps a Google Cloud organization, creating all the required resources
and permissions to start using Infrastructure as Code with Pulumi. This
step also configures a <a href="../docs/GLOSSARY.md#foundation-cicd-pipeline">CI/CD pipeline</a> for foundations code in subsequent
stages.</td>
</tr>
<tr>
<td><a href="../1-org">1-org</a></td>
<td>Sets up top-level shared folders, networking projects, and
organization-level logging, and sets baseline security settings through
organizational policy.</td>
</tr>
<tr>
<td><a href="../2-environments"><span style="white-space: nowrap;">2-environments</span></a></td>
<td>Sets up development, nonproduction, and production environments within the
Google Cloud organization that you've created.</td>
</tr>
<tr>
<td><a href="../3-networks-svpc">3-networks-svpc</a></td>
<td>Sets up shared VPCs with default DNS, NAT, Private Service networking,
and baseline firewall rules for each environment. It also sets up the
global DNS hub.</td>
</tr>
<tr>
<td><a href="../3-networks-hub-and-spoke">3-networks-hub-and-spoke</a></td>
<td>Sets up shared VPCs with all the default configuration
found on step 3-networks-svpc, but here the architecture will be based on the
Hub and Spoke network model. It also sets up the global DNS hub.</td>
</tr>
<tr>
<td><a href="../4-projects">4-projects</a></td>
<td>Sets up a folder structure, projects, and application infrastructure pipeline for applications,
 which are connected as service projects to the shared VPC created in the previous stage.</td>
</tr>
<tr>
<td><a href="../5-app-infra">5-app-infra</a></td>
<td>Deploys sample application infrastructure (Cloud Run, BigQuery) in one of the business unit projects using the infra pipeline set up in 4-projects.</td>
</tr>
</tbody>
</table>

For an overview of the architecture and the parts, see the
[pulumi-example-foundation README](../README.md).

## Purpose

The purpose of this step is to bootstrap a Google Cloud organization, creating all the required resources and permissions to start using the Pulumi Foundation. This step also configures a [CI/CD Pipeline](../docs/GLOSSARY.md#foundation-cicd-pipeline) for foundations code in subsequent stages using GitHub Actions.

The bootstrap step creates:

- The **`prj-b-seed`** project, which contains the following:
  - A KMS-encrypted GCS bucket for Pulumi state storage
  - Custom service accounts used by Pulumi to create new resources in Google Cloud (one per stage: `bootstrap`, `org`, `env`, `net`, `proj`)
- The **`prj-b-cicd`** project, which contains the following:
  - CI/CD pipeline infrastructure (Artifact Registry, Cloud Build, Workload Identity)
- The **`fldr-bootstrap`** folder under your organization (or parent folder) that contains both projects

It is a best practice to separate concerns by having two projects here: one for the Pulumi state and one for the CI/CD tool.
- The `prj-b-seed` project stores Pulumi state and has the service accounts that can create or modify infrastructure.
- The `prj-b-cicd` project holds the CI/CD tool infrastructure that coordinates the deployment.

To further separate concerns at the IAM level, a distinct service account is created for each stage. These service accounts are granted the minimum IAM permissions required to build the foundation at organization, folder, project, and billing scopes.

After executing this step, you will have the following structure:

```
example-organization/
└── fldr-bootstrap
    ├── prj-b-cicd
    └── prj-b-seed
```

## Intended Usage and Support

This repository is intended as an example to be forked, tweaked, and maintained in the user's own version-control system; the modules within this repository are not intended for use as remote references.
Though this blueprint can help accelerate your foundation design and build, we assume that you have the engineering skills and teams to deploy and customize your own foundation based on your own requirements.

## Prerequisites

To run the commands described in this document, install the following:

- [Google Cloud SDK](https://cloud.google.com/sdk/install) version 393.0.0 or later
- [Pulumi CLI](https://www.pulumi.com/docs/install/) version 3.0 or later
- [Go](https://go.dev/dl/) version 1.21 or later
- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) version 2.28.0 or later
- [jq](https://jqlang.github.io/jq/download/) version 1.6.0 or later

Also make sure that you've done the following:

1. Set up a Google Cloud
   [organization](https://cloud.google.com/resource-manager/docs/creating-managing-organization).
1. Set up a Google Cloud
   [billing account](https://cloud.google.com/billing/docs/how-to/manage-billing-account).
1. Create Cloud Identity or Google Workspace groups as defined in [groups for access control](https://cloud.google.com/architecture/security-foundations/authentication-authorization#groups_for_access_control).
1. For the user who will run the procedures in this document, grant the following roles:
   - The `roles/resourcemanager.organizationAdmin` role on the Google Cloud organization.
   - The `roles/orgpolicy.policyAdmin` role on the Google Cloud organization.
   - The `roles/resourcemanager.projectCreator` role on the Google Cloud organization.
   - The `roles/billing.admin` role on the billing account.
   - The `roles/resourcemanager.folderCreator` role.
   - The `roles/securitycenter.admin` role.

     ```bash
     # Example: grant roles to a user
     export ORG_ID="YOUR_ORG_ID"
     export SUPER_ADMIN_EMAIL="admin@example.com"

     gcloud organizations add-iam-policy-binding ${ORG_ID} \
       --member=user:${SUPER_ADMIN_EMAIL} \
       --role=roles/resourcemanager.organizationAdmin --quiet > /dev/null 2>&1
     gcloud organizations add-iam-policy-binding ${ORG_ID} \
       --member=user:${SUPER_ADMIN_EMAIL} \
       --role=roles/orgpolicy.policyAdmin --quiet > /dev/null 2>&1
     gcloud organizations add-iam-policy-binding ${ORG_ID} \
       --member=user:${SUPER_ADMIN_EMAIL} \
       --role=roles/resourcemanager.projectCreator --quiet > /dev/null 2>&1
     gcloud organizations add-iam-policy-binding ${ORG_ID} \
       --member=user:${SUPER_ADMIN_EMAIL} \
       --role=roles/resourcemanager.folderCreator --quiet > /dev/null 2>&1
     gcloud organizations add-iam-policy-binding ${ORG_ID} \
       --member=user:${SUPER_ADMIN_EMAIL} \
       --role=roles/securitycenter.admin --quiet > /dev/null 2>&1
     ```

1. Enable the following additional services on your current project:

     ```bash
     gcloud services enable cloudresourcemanager.googleapis.com
     gcloud services enable cloudbilling.googleapis.com
     gcloud services enable iam.googleapis.com
     gcloud services enable cloudkms.googleapis.com
     gcloud services enable servicenetworking.googleapis.com
     ```

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Deploying with GitHub Actions

This is the recommended CI/CD approach for the Pulumi foundation. After the initial bootstrap is applied manually, all subsequent changes are deployed via the GitHub Actions pipeline defined in [`.github/workflows/pulumi-ci.yml`](../.github/workflows/pulumi-ci.yml).

### Requirements for GitHub Actions

- A [GitHub account](https://docs.github.com/en/get-started/onboarding/getting-started-with-your-github-account) for your user or [Organization](https://docs.github.com/en/organizations/collaborating-with-groups-in-organizations/creating-a-new-organization-from-scratch).
- A **private** [GitHub repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-new-repository) to host this foundation code.
- The following [repository secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets) configured:
    - **`PULUMI_ACCESS_TOKEN`** — A Pulumi Cloud access token for state management. Generate one at [app.pulumi.com](https://app.pulumi.com/account/tokens).
    - **`GOOGLE_CREDENTIALS`** — GCP service account key JSON with the necessary permissions (typically the bootstrap SA created in this step).

### Instructions

1. Clone the [pulumi-example-foundation](https://github.com/VitruvianSoftware/pulumi-example-foundation) into your local environment and navigate to the `0-bootstrap` folder.

   ```bash
   git clone https://github.com/VitruvianSoftware/pulumi-example-foundation.git

   cd pulumi-example-foundation/0-bootstrap
   ```

1. Login to Pulumi and initialize the bootstrap stack:

   ```bash
   pulumi login  # or pulumi login --local for local state

   pulumi stack init production
   ```

1. Set the required configuration values for your environment:

   ```bash
   # Required configuration
   pulumi config set org_id "YOUR_ORG_ID"
   pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
   pulumi config set group_org_admins "org-admins@example.com"
   pulumi config set group_billing_admins "billing-admins@example.com"
   pulumi config set billing_data_users "billing-data@example.com"
   pulumi config set audit_data_users "audit-data@example.com"
   ```

1. (Optional) Set additional configuration to override defaults:

   ```bash
   # Optional — these have sensible defaults
   pulumi config set project_prefix "prj"          # default: prj
   pulumi config set folder_prefix "fldr"           # default: fldr
   pulumi config set bucket_prefix "bkt"            # default: bkt
   pulumi config set default_region "us-central1"   # default: us-central1
   pulumi config set default_region_2 "us-west1"    # default: us-west1
   pulumi config set default_region_gcs "US"         # default: US

   # Deploy under a specific folder instead of org root
   pulumi config set parent_folder "FOLDER_NUMERIC_ID"
   ```

1. Run `pulumi preview` to review the planned changes:

   ```bash
   pulumi preview
   ```

1. Run `pulumi up` to deploy:

   ```bash
   pulumi up
   ```

1. Record the outputs. These will be automatically consumed by Stage 1 and subsequent stages via [Stack References](../docs/GLOSSARY.md#pulumi-stack-reference):

   ```bash
   # View all outputs
   pulumi stack output

   # Key outputs:
   export SEED_PROJECT_ID=$(pulumi stack output seed_project_id)
   export CICD_PROJECT_ID=$(pulumi stack output cicd_project_id)
   export STATE_BUCKET=$(pulumi stack output tf_state_bucket)

   echo "Seed Project ID: ${SEED_PROJECT_ID}"
   echo "CI/CD Project ID: ${CICD_PROJECT_ID}"
   echo "State Bucket: ${STATE_BUCKET}"
   ```

1. Push the foundation code to your GitHub repository and configure the CI/CD pipeline:

   ```bash
   cd ..  # back to pulumi-example-foundation root

   git remote set-url origin git@github.com:<YOUR-ORG>/<YOUR-REPO>.git
   git add .
   git commit -m "Initialize Pulumi foundation"
   git push --set-upstream origin production
   ```

1. Configure GitHub repository secrets. Navigate to your GitHub repository → Settings → Secrets and variables → Actions, and add:
   - `PULUMI_ACCESS_TOKEN`
   - `GOOGLE_CREDENTIALS`

1. Continue with the instructions in the [1-org](../1-org/README.md) step.

**Note 1:** The stages after `0-bootstrap` use [Pulumi Stack References](../docs/GLOSSARY.md#pulumi-stack-reference) to read common configuration like the organization ID from the output of the `0-bootstrap` stage. They will fail if the bootstrap stack has not been successfully deployed.

**Note 2:** After the deploy, we recommend that you request 50 additional projects for the **projects step service account** (`sa-terraform-proj@prj-b-seed.iam.gserviceaccount.com`) created in this step to avoid project quota errors in later stages.

## Running Pulumi Locally

The following steps guide you through deploying without using the CI/CD pipeline. This is useful for initial setup and testing.

1. Clone [pulumi-example-foundation](https://github.com/VitruvianSoftware/pulumi-example-foundation) into your local environment:

   ```bash
   git clone https://github.com/VitruvianSoftware/pulumi-example-foundation.git

   cd pulumi-example-foundation/0-bootstrap
   ```

1. Authenticate with Google Cloud:

   ```bash
   gcloud auth application-default login
   gcloud config set project YOUR_EXISTING_PROJECT  # any project for initial API access
   ```

1. Login to Pulumi (use local backend if you don't have Pulumi Cloud):

   ```bash
   # Option A: Pulumi Cloud (recommended)
   pulumi login

   # Option B: Local backend (state stored on disk)
   pulumi login --local

   # Option C: GCS backend (state stored in GCS)
   pulumi login gs://YOUR_STATE_BUCKET
   ```

1. Initialize the bootstrap stack:

   ```bash
   pulumi stack init production
   ```

1. Set the required configuration (see [Configuration Reference](#configuration-reference) below for all options):

   ```bash
   pulumi config set org_id "YOUR_ORG_ID"
   pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
   pulumi config set group_org_admins "org-admins@example.com"
   pulumi config set group_billing_admins "billing-admins@example.com"
   pulumi config set billing_data_users "billing-data@example.com"
   pulumi config set audit_data_users "audit-data@example.com"
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. Record outputs for use in subsequent stages:

   ```bash
   pulumi stack output --json | jq .
   ```

1. After a successful deploy, initialize a local Git repository to track changes:

   ```bash
   cd ..
   git init
   git add .
   git commit -m "Initial Pulumi foundation bootstrap"
   ```

## Configuration Reference

### Required Configuration

| Name | Description | Example |
|------|-------------|---------|
| `org_id` | GCP Organization ID | `"123456789"` |
| `billing_account` | The ID of the billing account to associate projects with | `"XXXXXX-XXXXXX-XXXXXX"` |
| `group_org_admins` | Google Workspace group for organization admins | `"org-admins@example.com"` |
| `group_billing_admins` | Google Workspace group for billing admins | `"billing-admins@example.com"` |
| `billing_data_users` | Google Workspace group for billing data users | `"billing-data@example.com"` |
| `audit_data_users` | Google Workspace group for audit data users | `"audit-data@example.com"` |

### Optional Configuration

| Name | Description | Default |
|------|-------------|---------|
| `project_prefix` | Name prefix for projects created. Max 3 characters. | `"prj"` |
| `folder_prefix` | Name prefix for folders created. | `"fldr"` |
| `bucket_prefix` | Name prefix for state bucket created. | `"bkt"` |
| `default_region` | Default region for resource creation. | `"us-central1"` |
| `default_region_2` | Secondary default region for resource creation. | `"us-west1"` |
| `default_region_gcs` | Case-sensitive default region for GCS resources. | `"US"` |
| `parent_folder` | Numeric folder ID to deploy under instead of org root. | `""` (org root) |
| `org_policy_admin_role` | Grant additional Org Policy Admin role to admin group. | `"false"` |
| `bucket_force_destroy` | Allow deletion of state bucket even if it contains objects. | `"false"` |

## Outputs

| Name | Description |
|------|-------------|
| `seed_project_id` | Project ID of the Seed project (`prj-b-seed`) |
| `cicd_project_id` | Project ID of the CI/CD project (`prj-b-cicd`) |
| `bootstrap_folder_id` | Folder ID of the bootstrap folder |
| `tf_state_bucket` | Name of the GCS bucket for Pulumi state |
| `state_bucket_kms_key_id` | KMS key ID used for state bucket encryption |
| `bootstrap_sa_email` | Bootstrap stage service account email |
| `org_sa_email` | Organization stage service account email |
| `env_sa_email` | Environment stage service account email |
| `net_sa_email` | Network stage service account email |
| `proj_sa_email` | Projects stage service account email |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Orchestrates the bootstrap: loads config, creates the folder, coordinates projects and IAM |
| `projects.go` | Creates the Seed project (KMS key ring, crypto key, encrypted state bucket) and CI/CD project using the `pkg/project` library component |
| `iam.go` | Creates 5 granular service accounts and assigns least-privilege IAM at org, parent, seed project, CI/CD project, and billing scopes |
| `Pulumi.yaml` | Pulumi project configuration |
| `go.mod` / `go.sum` | Go module dependencies |
