# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest on `main` | ✅ |

## Reporting a Vulnerability

If you discover a security vulnerability in this foundation, please report it responsibly:

1. **Do NOT open a public GitHub issue.**
2. Email [security@vitruviansoftware.com](mailto:security@vitruviansoftware.com) with:
   - A description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Suggested fix (if any)
3. You will receive an acknowledgment within **48 hours**.
4. We will work with you on a fix and coordinate disclosure.

## Scope

This repository deploys enterprise Google Cloud infrastructure. Security issues of particular concern include:

- IAM misconfigurations that could escalate privileges
- Organization policy gaps that weaken security posture
- Service account key exposure
- Overly permissive firewall rules or network configurations
- Secrets or credentials exposed in Pulumi state or config

## Security Best Practices

When deploying this foundation:

- **Use [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation)** instead of service account keys for CI/CD authentication
- **Encrypt Pulumi state** using [Pulumi Cloud](https://www.pulumi.com/product/pulumi-cloud/) or a KMS-backed encryption provider
- **Never store secrets in `Pulumi.*.yaml`** — use `pulumi config set --secret` for sensitive values
- **Review `pulumi preview` output** before applying, especially for IAM and org policy changes
- **Request project quota increases** for the service accounts created in 0-bootstrap to avoid cascading deployment issues
