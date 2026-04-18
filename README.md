# Vitruvian Software Pulumi Foundation (Public)

This repository provides a reference implementation of the Google Cloud Enterprise Foundation, built using Pulumi and Go.

## Overview
It demonstrates how to deploy the foundational stages of a secure GCP organization using the [Vitruvian Software Pulumi Library](https://github.com/VitruvianSoftware/pulumi-library).

### Stages
- **0-bootstrap**: Org-level bootstrapping.
- **1-org**: Organization resources and logging.
- **2-environments**: Environment folder structures.
- **3-networks**: Networking stack (SVPC or Hub-and-Spoke).
- **4-projects**: Business unit project management.
- **5-app-infra**: Sample application deployment.

## Public vs. Private
- This repo is a **Public Reference**.
- Real Vitruvian Software implementations are managed in **Private** `gcp-*` repositories that consume this foundation logic.
