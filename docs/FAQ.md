# Frequently Asked Questions

## Why am I encountering a low quota with projects created via the Pulumi Example Foundation?

When you deploy the Pulumi Example Foundation with the Service Account created in step 0-bootstrap,
the project quota will be based on the reputation of your service account rather than your user identity.
In many cases, this quota is initially low.

We recommend that you request 50 additional projects for the service account created in step 0-bootstrap.
You can use the [Request Project Quota Increase](https://support.google.com/code/contact/project_quota_increase) form to request the quota increase.
In the support form, for **Email addresses that will be used to create projects**, use the `proj_sa_email` address from the bootstrap stage output:

```bash
cd 0-bootstrap
pulumi stack output proj_sa_email
```

If you see other quota errors, see the [Quota documentation](https://cloud.google.com/docs/quota).

## What is a "named" branch?

Certain branches in the pulumi-example-foundation are considered to be
_named branches_. Pushing to a named branch causes `pulumi up` (apply) to
run. Pushing to branches other than the named branches only runs `pulumi preview`.

Named branches:
- `development`
- `nonproduction`
- `production`

## Which Pulumi commands are run when I push to a branch?

If you pushed to a _named branch_, the following commands are run: `pulumi preview`, then `pulumi up`.

If you push to a branch that is not a named branch (e.g., a feature branch via pull request),
only `pulumi preview` is run. `pulumi up` is **not** run.

## How do Stack References work across stages?

Each stage after 0-bootstrap uses [Pulumi Stack References](./GLOSSARY.md#pulumi-stack-reference) to read outputs from previous stages. This allows configuration values (like organization ID, folder IDs, project IDs) to flow automatically between stages without manual re-entry.

The stack reference name must match the fully qualified name of the source stack, for example:
```
organization/vitruvian/0-bootstrap/production
```

If you're using a local backend, the format may differ. Check your backend documentation.

## How is this different from the Terraform foundation?

Key differences:

| Aspect | Terraform Foundation | Pulumi Foundation |
|--------|---------------------|-------------------|
| Language | HCL | Go |
| State | GCS backend via `terraform_remote_state` | Pulumi Cloud / GCS backend via Stack References |
| CI/CD | Cloud Build, Jenkins, GitHub Actions, GitLab, Terraform Cloud | GitHub Actions |
| Modules | Terraform Registry modules (CFT) | Go packages in `pulumi-library` |
| Config | `.tfvars` files | `pulumi config` (YAML) |
| Plan/Apply | `terraform plan` / `terraform apply` | `pulumi preview` / `pulumi up` |
