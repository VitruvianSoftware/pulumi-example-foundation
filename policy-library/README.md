# Policy Library

This directory contains [Pulumi CrossGuard](https://www.pulumi.com/docs/using-pulumi/crossguard/)
policy pack configuration for the Pulumi Go Foundation. These policies are the
Pulumi equivalent of the upstream Terraform foundation's `policy-library/`
directory which uses OPA constraints validated via `gcloud beta terraform vet`.

## Architecture

CrossGuard policy packs for Go-based Pulumi programs are written in TypeScript
(the policy SDK is TypeScript-only). The policies validate resource properties
at preview-time, before any API calls are made.

For the full policy implementation, see the
[TypeScript foundation's policy-library](https://github.com/VitruvianSoftware/pulumi_ts-example-foundation/tree/main/policy-library),
which contains the shared CrossGuard policy pack used by both foundations.

## Policies

| Policy | Severity | Description |
|--------|----------|-------------|
| `no-default-network` | mandatory | Projects must not create the default network |
| `project-labels-required` | mandatory | Projects must have `environment` and `application_name` labels |
| `no-sa-key-creation` | mandatory | Service account keys must not be created (use WIF) |
| `no-public-access` | mandatory | Resources must not grant `allUsers` or `allAuthenticatedUsers` |
| `no-public-ip` | mandatory | Compute instances must not have external IPs |
| `require-private-google-access` | mandatory | Subnets must enable Private Google Access |
| `require-flow-logs` | advisory | Subnets should have VPC Flow Logs enabled |

## Usage

### Running Locally

```bash
# Clone the policy pack (or copy from TS foundation)
git clone https://github.com/VitruvianSoftware/pulumi_ts-example-foundation.git
cd pulumi_ts-example-foundation/policy-library
npm install

# Run your Go stack with policy enforcement
cd /path/to/your-go-stage
pulumi preview --policy-pack /path/to/pulumi_ts-example-foundation/policy-library
```

### CI/CD Integration

Add the `--policy-pack` flag to your preview and apply steps:

```yaml
- name: Preview with policies
  run: pulumi preview --policy-pack ./policy-library --diff
```

## Relationship to Upstream Terraform Policies

See the [TypeScript foundation's policy-library README](https://github.com/VitruvianSoftware/pulumi_ts-example-foundation/tree/main/policy-library#relationship-to-upstream-terraform-policies) for a detailed mapping between upstream OPA constraints and CrossGuard equivalents.
