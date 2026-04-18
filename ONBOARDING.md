# Onboarding Guide: Pulumi GCP Foundation

Welcome to the Pulumi port of the Google Cloud Enterprise Foundation. This guide will walk you through deploying your foundation step-by-step.

## Architectural Overview
This foundation is designed to be modular. You will deploy it in stages, where each stage builds upon the outputs of the previous one using **Pulumi Stack References**.

### Prerequisites
- Pulumi CLI installed.
- Go 1.21+ installed.
- Access to a Google Cloud Organization.
- A Billing Account ID.

---

## Step 0: Bootstrap
**Purpose**: Set up the Seed project and the Service Accounts that will run the rest of the foundation.
- Navigate to `0-bootstrap/`.
- Run `pulumi up`.
- **Decision**: Define your `project_prefix` and `folder_prefix`.

## Step 1: Organization
**Purpose**: Create the core folder structure and logging/billing export projects.
- Navigate to `1-org/`.
- Configure `bootstrap_stack_name` to point to your Stage 0 stack.
- Run `pulumi up`.

## Step 2: Environments
**Purpose**: Create environment folders (Dev, Non-Prod, Prod).
- Navigate to `2-environments/`.
- Run `pulumi up`.

## Step 3: Networking (Decision Point)
**Purpose**: Deploy the shared VPCs and subnets.
**Choose ONE**:
- **SVPC**: Standard Shared VPC architecture (`3-networks-svpc/`).
- **Hub-and-Spoke**: More complex transitive networking (`3-networks-hub-and-spoke/`).

## Step 4: Projects
**Purpose**: Create Business Unit projects (App and Data pairs).
- Navigate to `4-projects/`.
- Run `pulumi up`.

## Step 5: App Infrastructure
**Purpose**: Deploy a sample Cloud Run app and BigQuery datasets.
- Navigate to `5-app-infra/`.
- Run `pulumi up`.

---

## Using the Shared Library
All stages use the [Vitruvian Software Pulumi Library](https://github.com/VitruvianSoftware/pulumi-library). This library provides the "ComponentResources" (like the Project Factory and Networking modules) that ensure your infrastructure is standardized and production-grade.
