# Contributing

Thank you for your interest in contributing to the Pulumi Example Foundation!

## Development Workflow

This repository follows a **trunk-based development** workflow. For guidance
on how changes flow through environments, refer to the upstream
[Google Terraform Example Foundation](https://github.com/terraform-google-modules/terraform-example-foundation)
for the reference architecture and branching strategy context.

### Branch Structure

The branching model varies by stage:

| Stage | Branches | Rationale |
|-------|----------|-----------|
| `0-bootstrap` | `production` | Shared infrastructure — single environment |
| `1-org` | `production` | Organization-wide resources — single environment |
| `2-environments` | `development`, `nonproduction`, `production` | Per-environment resources |
| `3-networks-svpc` | `development`, `nonproduction`, `production` | Per-environment networks |
| `3-networks-hub-and-spoke` | `development`, `nonproduction`, `production` | Per-environment networks |
| `4-projects` | `development`, `nonproduction`, `production` | Per-environment projects |
| `5-app-infra` | `development`, `nonproduction`, `production` | Per-environment app infra |

### Submitting Changes

1. Fork this repository.
2. Create a feature branch from `production` (or the appropriate environment branch).
3. Make your changes and ensure they pass `pulumi preview` locally.
4. Open a Pull Request against the appropriate branch.
5. The CI pipeline will run `pulumi preview` on your PR.
6. Once approved and merged, the CI pipeline will run `pulumi up`.

### Code Style

- **Go**: Follow [Effective Go](https://go.dev/doc/effective_go) conventions.
- **Comments**: All exported functions and types must have doc comments.
- **Error handling**: Always return errors; do not use `panic` or `log.Fatal`.
- **Naming**: Follow the [naming conventions](https://cloud.google.com/architecture/security-foundations/using-example-terraform#naming_conventions) from the Security Foundations Guide.

### Shared Library

Reusable components live in the separate
[pulumi-library](https://github.com/VitruvianSoftware/pulumi-library)
repository. If your change involves a new reusable pattern, consider whether
it belongs in the library rather than in this foundation repo.

### Testing

- Run `pulumi preview` to validate your changes before submitting.
- Ensure `go build ./...` succeeds for any stage you modify.
- Check for lint issues with `go vet ./...`.

## Reporting Issues

Please open a GitHub Issue with:
- Which stage is affected
- Steps to reproduce
- Expected vs. actual behavior
- Relevant Pulumi/Go version information
