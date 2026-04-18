# Stage 0: Bootstrap

This stage bootstraps your Google Cloud organization and prepares it for Infrastructure as Code management with Pulumi.

## Purpose
The Bootstrap stage is the "root" of your foundation. It creates the administrative resources that all subsequent stages rely on.

### Key Resources
- **Seed Project**: The administrative project that hosts the state storage and service accounts.
- **Service Accounts**: Granular identities for each foundation stage (Org, Network, Projects, etc.).
- **Permissions**: Minimum necessary IAM roles at the organization level for those service accounts.
- **State Bucket**: A GCS bucket used by Pulumi to store infrastructure state securely.

## Decisions: CI/CD Strategy
In the original foundation, you chose between Cloud Build and Jenkins. With Pulumi, the recommended approach is to use **Pulumi Service** or a **GCS/S3 Backend** combined with **GitHub Actions** or **GitLab CI**.

This reference architecture assumes you are using GitHub Actions.

## Onboarding / Usage

1.  **Initialize the Stack**:
    ```bash
    pulumi stack init bootstrap
    ```

2.  **Set Configuration**:
    Configure your organization and billing details:
    ```bash
    pulumi config set org_id <YOUR_ORG_ID>
    pulumi config set billing_account <YOUR_BILLING_ACCOUNT_ID>
    pulumi config set project_prefix vit-
    ```

3.  **Deployment**:
    ```bash
    pulumi up
    ```

4.  **Post-Deployment**:
    Record the outputs. These will be automatically consumed by Stage 1 via Stack References.

## File Structure
- `main.go`: Orchestrates the bootstrap process.
- `projects.go`: Uses the `project` component from the library to create the Seed project.
- `iam.go`: Uses the `iam` component from the library to setup granular permissions.
