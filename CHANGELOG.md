# Changelog

All notable changes to the Pulumi Example Foundation (Go) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 1.0.0 (2026-04-23)


### Features

* **0-bootstrap:** add billing.creator, SA impersonation, and bucket IAM ([57c5ce5](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/57c5ce56c20b25ce503986e54d38198d7c725612))
* **0-bootstrap:** add optional Google Workspace group creation ([d15f3cf](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/d15f3cfa9464a0924a3ed4796df222f4f3772e0f))
* **1-org:** achieve full IAM/policy parity with Terraform foundation ([fb62524](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/fb6252484fac8fa23b7c4d84e55875e80433003a))
* **1-org:** achieve full parity with Terraform Enterprise Foundation ([#7](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/7)) ([937c14f](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/937c14fe84d806972c54685abe697955c2ec0bee))
* **2-environments:** add bootstrap stack reference for common_config ([11e3ee6](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/11e3ee63c8a09efb556fc41b222808b4e5e964a1))
* **2-environments:** implement full upstream parity ([#16](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/16)) ([7121723](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/7121723afca2748b18e4d63789e883ccdea3d27a))
* **4-projects:** full upstream parity — peering, CMEK, VPC-SC, labels, budgets ([9b09420](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/9b09420f04c01790d06efd7d4c77854b9cbd584d))
* achieve parity with upstream 5-app-infra ([#28](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/28)) ([10fa891](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/10fa891fcd399be70fd933d792cbcb8fa398580d))
* add E2E testing infrastructure ([#37](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/37)) ([96fac02](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/96fac0201547b66a94d2b9c6b1892e23baca823b))
* add Pulumi stack configuration templates and documentation for environment-based deployments ([01d03dc](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/01d03dcdfbe3e0e2f8befd4fe927d0820ce9c416))
* add release-please automation for proper versioning ([#32](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/32)) ([91c301f](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/91c301f3f6df04ae5712ef0fd5723ea4617e36a4))
* **bootstrap:** achieve full parity with Terraform Enterprise Foundation ([#6](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/6)) ([ddae105](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/ddae1056225ff9bca760990e10989c95f38bf4ff))
* complete parity remediation for 4-projects ([#27](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/27)) ([2578ff3](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/2578ff349adc3981e4d83f8794cf52f442116a9d))
* e2e testing ([#39](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/39)) ([1db4eb7](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/1db4eb762efd00fb0f3fb810c3d92af28116b8c2))
* enable random project ID suffixes across all stages ([#4](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/4)) ([03ad067](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/03ad067b2f03b2121c7d5a2e5df6cfd9e37ece0d))
* implement 1-org gaps (billing sink, CAI monitoring, dependency ordering, KMS agent, budget) ([#9](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/9)) ([33bd281](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/33bd2818d960dbcf555e00c138f4aa2f1bbad449))
* initialize Pulumi Go foundation with CI/CD, documentation, and policy library scaffolding ([7f39e0d](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/7f39e0d327055a0930e713465a864e28bf6fd70c))
* migrate to restructured pulumi-library go monorepo ([#31](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/31)) ([65c2c43](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/65c2c431379865b627384c9997f1dff55daa5fea))
* **networks:** align SVPC and hub-spoke with upstream terraform foundation ([#21](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/21)) ([50bc0a1](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/50bc0a1dbb15856fd065686bd4d9b683dd990e9d))
* Plumb network component gaps for svpc and hub-and-spoke ([#26](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/26)) ([d862fa0](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/d862fa0aace4afce27b5d13025cc50b8d03a4479))
* remediate foundation parity gaps in phases 0-2 ([#18](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/18)) ([acb98b9](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/acb98b9b13b9f53bae85b5dbf37b42448929eacf))


### Bug Fixes

* **0-bootstrap:** add missing iamcredentials API ([#15](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/15)) ([3f7c697](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/3f7c697b53270283b41eb7be8afe24aa25088330))
* **0-bootstrap:** align project labels, config, and exports with upstream ([#13](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/13)) ([600a4c5](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/600a4c5455afa3ad2b95feb21b1bfa34f6ea2084))
* **0-bootstrap:** correct KMS region and add KMS prevention parity ([#14](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/14)) ([5792b6f](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/5792b6f186a492bea4c81149c10977e7921ed2e7))
* **1-org:** add missing labels, random suffixes, and default corrections ([#12](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/12)) ([18d0576](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/18d0576d93e7ba66c8c672e712c6a8a50d9f48bc))
* **1-org:** align export names with terraform foundation ([#11](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/11)) ([90a0365](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/90a0365b972faef1999068b7a3308056ec5066d4))
* **1-org:** align pulumi 1-org foundation with upstream terraform ([#10](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/10)) ([4a1a35d](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/4a1a35d57754a66c607abeaaaedfae358bb7c649))
* **1-org:** refactor CAIMonitoring to use library component ([#24](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/24)) ([7f497dc](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/7f497dc88829c394815dad95ab639faeebfb7a51))
* **2-environments:** add shared_network to budget config for parity ([#25](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/25)) ([fef602e](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/fef602e6fa31e835fcb22b6adc4a753e4493c7d3))
* **2-environments:** remove broken async applyCommonConfig ([babf3b1](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/babf3b1dd7d0333b079ff90aad80a5aab7701e1a))
* **5-app-infra:** address secondary audit parity gaps ([#30](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/30)) ([f6e7caf](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/f6e7cafd15406811756b65e3e44d792f4dae713c))
* **5-app-infra:** remediate review findings ([#29](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/29)) ([1e37b05](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/1e37b054bf466e61ae055e184a8eef23f42d49ff))
* API and IAM bindings usage of pulumi-library ([#1](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/1)) ([a1b5f36](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/a1b5f36a665931e6f399ca86755865f9ade20936))
* **deps:** resolve packages from go module proxy ([#35](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/35)) ([2dd327f](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/2dd327fe31b9a2bdbc7a74fd91a8893b944dae1a))
* e2e folder support ([#40](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/40)) ([f433413](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/f433413781ae534b2afa8a362cdb738192c9c8a2))
* e2e folder support and robust clean script ([#38](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/38)) ([0b5b224](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/0b5b224457dbfe823de1e6d3315455c552e03653))
* **go:** update module paths for workspace migration ([00877a4](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/00877a419fa96ed27c8584b6a41cb17ac42a5863))
* **go:** update module paths for workspace migration ([545d669](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/545d66955725fad422744a489c46377694a48f39))
* remove deprecated sourcerepo API and align KMS regions ([#41](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/41)) ([79d1aa8](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/79d1aa8ea4f54c0f8b016c31c42e4a6f33bd5a68))
* resolve phase 3 architectural gaps ([#22](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/22)) ([5d712c6](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/5d712c6dfe46eebb1b976e7e88fd71a4db0768cd))
* update pulumi-library dependency to include kms region parity ([#42](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/issues/42)) ([42cba4c](https://github.com/VitruvianSoftware/pulumi_go-example-foundation/commit/42cba4cf7525f0999d59a29b353d9a2900ad7094))

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

- Migrated shared components to [pulumi-library](https://github.com/VitruvianSoftware/pulumi-library/go)

### Security

- WIF-only authentication (no service account keys stored in CI/CD)
- KMS-encrypted Pulumi state bucket with configurable protection level
- Deletion protection on bootstrap folder, seed project, and CI/CD project
