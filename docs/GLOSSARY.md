# Glossary

Defined terms in the documentation for the Pulumi Example Foundation are capitalized and have
specific meaning within the domain of knowledge.

## Pulumi Service Accounts

The email addresses for the privileged service accounts created in the seed project of step 0-bootstrap.
These service accounts are used to run Pulumi by the CI/CD pipeline (GitHub Actions). A service account is created for each stage of the foundation:

- `sa-terraform-bootstrap` — Bootstrap stage
- `sa-terraform-org` — Organization stage
- `sa-terraform-env` — Environments stage
- `sa-terraform-net` — Networks stage
- `sa-terraform-proj` — Projects stage

## Seed Project

The Seed Project (`prj-b-seed`) created in the 0-bootstrap step. It is the project where the Pulumi Service Accounts are created and hosts the GCS bucket used to store Pulumi state.

## Foundation CI/CD Pipeline

A project (`prj-b-cicd`) created in step 0-bootstrap to manage infrastructure **within the organization**.
The pipeline uses **GitHub Actions** and Pulumi is executed using the seed project service accounts.
Also known as the CI/CD project.
It is located under the `fldr-bootstrap` folder.

## App Infra Pipeline

A project created in step 4-projects to manage application infrastructure **within projects**.
A separate pipeline can exist for each business unit, configured to use a service account with limited permissions to deploy into certain projects.
They are located under the `fldr-common` folder.

## Pulumi Stack Reference

A Pulumi mechanism that retrieves output values from another stack's state. In the Pulumi Example Foundation context, it reads output values from previous stages like `0-bootstrap` so that users don't need to provide values that were already given as inputs in previous steps or find the values/attributes of resources created earlier.

Stack References are the Pulumi equivalent of Terraform's `terraform_remote_state` data source.

Usage example in Go:
```go
bootstrapStack, err := pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
    Name: pulumi.String("organization/vitruvian/0-bootstrap/production"),
})
seedProjectID := bootstrapStack.GetStringOutput(pulumi.String("seed_project_id"))
```

## Named Branch

An environment-specific branch in the Git repository that triggers `pulumi up` (apply) when pushed to. The named branches are:

- `development` — Applies to the development environment
- `nonproduction` — Applies to the nonproduction environment
- `production` — Applies to the production environment (and shared resources)

Pushing to non-named branches (e.g., feature branches) triggers `pulumi preview` only.
