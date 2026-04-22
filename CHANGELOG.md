# Changelog

All notable changes to the Pulumi Example Foundation (Go) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial 6-stage foundation (0-bootstrap through 5-app-infra)
- Shared VPC and Hub-and-Spoke network topologies
- GitHub Actions CI/CD pipeline with Workload Identity Federation
- GitLab CI/CD pipeline alternative
- Comprehensive onboarding guide (`ONBOARDING.md`)
- Pre-flight validation script (`scripts/validate-requirements.sh`)
- Documentation suite: README, CONTRIBUTING, SECURITY, ERRATA, FAQ, GLOSSARY, TROUBLESHOOTING
- CrossGuard policy pack skeleton (`policy-library/`)
- Per-stage Configuration Reference and Outputs tables
- Resource hierarchy change guide (`docs/change_resource_hierarchy.md`)

### Changed

- Migrated shared components to [pulumi-library](https://github.com/VitruvianSoftware/pulumi-library)

### Security

- WIF-only authentication (no service account keys stored in CI/CD)
- KMS-encrypted Pulumi state bucket with configurable protection level
- Deletion protection on bootstrap folder, seed project, and CI/CD project
