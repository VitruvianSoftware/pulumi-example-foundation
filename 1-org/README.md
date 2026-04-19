# 1-org

This repo is part of a multi-part guide that shows how to configure and deploy
the example.com reference architecture described in
[Google Cloud security foundations guide](https://cloud.google.com/architecture/security-foundations), implemented using **Pulumi** and **Go**. The following table lists the stages of this deployment.

<table>
<tbody>
<tr>
<td><a href="../0-bootstrap">0-bootstrap</a></td>
<td>Bootstraps a Google Cloud organization, creating all the required resources
and permissions to start using Infrastructure as Code with Pulumi. This
step also configures a <a href="../docs/GLOSSARY.md#foundation-cicd-pipeline">CI/CD pipeline</a> for foundations code in subsequent
stages.</td>
</tr>
<tr>
<td>1-org (this file)</td>
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
and baseline firewall rules for each environment.</td>
</tr>
<tr>
<td><a href="../3-networks-hub-and-spoke">3-networks-hub-and-spoke</a></td>
<td>Alternative to 3-networks-svpc using the Hub and Spoke network model.</td>
</tr>
<tr>
<td><a href="../4-projects">4-projects</a></td>
<td>Sets up a folder structure, projects, and application infrastructure pipeline for applications.</td>
</tr>
<tr>
<td><a href="../5-app-infra">5-app-infra</a></td>
<td>Deploys sample application infrastructure in one of the business unit projects.</td>
</tr>
</tbody>
</table>

For an overview of the architecture and the parts, see the
[pulumi-example-foundation README](../README.md).

## Purpose

The purpose of this step is to set up the common folder used to house projects that contain shared resources such as Security Command Center notification, Cloud Key Management Service (KMS), org-level secrets, and org-level logging.
This stage also sets up the network folder used to house network-related projects such as DNS Hub, Interconnect, and shared VPC host projects for each environment (`development`, `nonproduction`, `production`).

This will create the following folder and project structure:

```
example-organization
└── fldr-common
    ├── prj-c-logging
    ├── prj-c-billing-export
    ├── prj-c-scc
    ├── prj-c-kms
    └── prj-c-secrets
└── fldr-network
    ├── prj-net-dns
    ├── prj-net-interconnect
    ├── prj-d-svpc
    ├── prj-n-svpc
    └── prj-p-svpc
└── fldr-development
└── fldr-nonproduction
└── fldr-production
```

### Key Resources

#### Logs

Under the common folder, a project `prj-c-logging` is used as the destination for organization-wide sinks. This includes admin activity audit logs from all projects in your organization and the billing account.

Logs are collected into a logging bucket with a linked BigQuery dataset for ad-hoc investigations, a Cloud Storage bucket for long-term archival, and Pub/Sub for streaming to external systems.

**Notes:**

- The various audit log types being captured in BigQuery are retained for 30 days.
- For billing data, a BigQuery dataset is created with permissions attached, however you will need to configure a billing export [manually](https://cloud.google.com/billing/docs/how-to/export-data-bigquery).

#### Security Command Center Notification

A project created under the common folder (`prj-c-scc`) hosts Security Command Center notification resources at the organization level. This includes a Pub/Sub topic, subscription, and an [SCC notification](https://cloud.google.com/security-command-center/docs/how-to-notifications) configured to stream all active findings. You can adjust the filter via the `scc_notification_filter` config value.

#### KMS

A project (`prj-c-kms`) allocated for [Cloud Key Management](https://cloud.google.com/security-key-management) for KMS resources shared by the organization.

#### Secrets

A project (`prj-c-secrets`) allocated for [Secret Manager](https://cloud.google.com/secret-manager) for secrets shared by the organization.

#### DNS Hub

A project (`prj-net-dns`) created under the network folder to host the DNS hub for the organization.

#### Interconnect

A project (`prj-net-interconnect`) created under the network folder to host the Dedicated Interconnect [connection](https://cloud.google.com/network-connectivity/docs/interconnect/concepts/terminology#elements) for the organization.

#### Networking

Under the network folder, one Shared VPC project is created per environment (`development`, `nonproduction`, `production`) intended to be used as a [Shared VPC host project](https://cloud.google.com/vpc/docs/shared-vpc). This stage only creates the projects and enables the correct APIs. The following network stages create the actual VPC networks.

#### Organization Policies

14+ boolean constraints are enforced including: serial port access, nested virtualization, OS Login, SA key creation, public access prevention, etc. List policies include: VM external IP deny, domain-restricted sharing, and protocol forwarding restrictions.

#### Tags

Org-level environment classification tags are created and applied to the bootstrap, common, and network folders.

## Prerequisites

1. [0-bootstrap](../0-bootstrap/README.md) executed successfully.
1. To enable Security Command Center notifications, choose a Security Command Center tier and create and grant permissions for the Security Command Center service account as described in [Setting up Security Command Center](https://cloud.google.com/security-command-center/docs/quickstart-security-command-center).

### Troubleshooting

See [troubleshooting](../docs/TROUBLESHOOTING.md) if you run into issues during this step.

## Usage

Consider the following:

- This stage creates sinks to export logs to Cloud Logging, BigQuery, Pub/Sub, and Cloud Storage. This will result in additional charges for those copies of logs.
- This stage implements but does not enable [bucket policy retention](https://cloud.google.com/storage/docs/bucket-lock) for organization logs.
- To use the **hub-and-spoke** architecture, you will select the `3-networks-hub-and-spoke` stage instead of `3-networks-svpc` in the networking step.
- This stage manages contacts for notifications using [Essential Contacts](https://cloud.google.com/resource-manager/docs/managing-notification-contacts).

### Deploying with GitHub Actions

1. Ensure the `0-bootstrap` stage has been deployed and the CI/CD pipeline is configured.

1. Navigate to the `1-org` directory and initialize the stack:

   ```bash
   cd 1-org
   pulumi stack init production
   ```

1. Set the required configuration:

   ```bash
   pulumi config set org_id "YOUR_ORG_ID"
   pulumi config set billing_account "YOUR_BILLING_ACCOUNT_ID"
   pulumi config set bootstrap_stack_name "organization/vitruvian/0-bootstrap/production"
   pulumi config set domains_to_allow "example.com"
   ```

1. (Optional) Check if your organization already has an Access Context Manager policy:

   ```bash
   export ORGANIZATION_ID="YOUR_ORG_ID"
   export ACCESS_CONTEXT_MANAGER_ID=$(gcloud access-context-manager policies list \
     --organization ${ORGANIZATION_ID} --format="value(name)")
   echo "access_context_manager_policy_id = ${ACCESS_CONTEXT_MANAGER_ID}"

   # If the above returns a value, set:
   pulumi config set create_access_context_manager_policy "false"
   ```

1. (Optional) Check if an SCC notification already exists:

   ```bash
   gcloud scc notifications describe "scc-notify" \
     --organization=${ORGANIZATION_ID} --location=global
   ```

   If it exists, set a different name:
   ```bash
   pulumi config set scc_notification_name "my-scc-notify"
   ```

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

1. Commit and push to trigger the CI/CD pipeline:

   ```bash
   git add .
   git commit -m "Initialize org stage"
   git push origin production
   ```

1. Proceed to the [2-environments](../2-environments/README.md) step.

### Running Pulumi Locally

1. Navigate to the `1-org` directory:

   ```bash
   cd pulumi-example-foundation/1-org
   ```

1. Initialize the stack and set configuration as described above.

1. Preview and deploy:

   ```bash
   pulumi preview
   pulumi up
   ```

## Configuration Reference

| Name | Description | Required | Default |
|------|-------------|:--------:|---------|
| `org_id` | GCP Organization ID | ✅ | — |
| `billing_account` | Billing account ID | ✅ | — |
| `bootstrap_stack_name` | Fully qualified Pulumi stack name of the 0-bootstrap stage | ✅ | — |
| `project_prefix` | Project name prefix | | `"prj"` |
| `folder_prefix` | Folder name prefix | | `"fldr"` |
| `default_region` | Default region | | `"us-central1"` |
| `domains_to_allow` | Comma-separated list of domains for domain-restricted sharing org policy | | `""` |
| `essential_contacts_domains` | Comma-separated list of domains for Essential Contacts | | `""` |
| `scc_notification_filter` | SCC notification filter expression | | `"state=\"ACTIVE\""` |
| `create_access_context_manager_policy` | Whether to create an Access Context Manager policy | | `"true"` |
| `parent_folder` | Deploy under a specific folder instead of org root | | `""` |

## File Structure

| File | Description |
|------|-------------|
| `main.go` | Orchestrates the org stage: loads config, deploys folders, projects, policies, logging, SCC, tags |
| `folders.go` | Creates the common, network, and environment folders |
| `projects.go` | Creates shared projects (logging, billing-export, scc, kms, secrets, dns, interconnect, network) |
| `policies.go` | Enforces 14+ boolean and list organization policies |
| `logging.go` | Sets up centralized logging with org sinks to Storage, Pub/Sub, and BigQuery |
| `scc.go` | Configures SCC notification with Pub/Sub pipeline |
| `tags.go` | Creates org-level environment classification tags |
