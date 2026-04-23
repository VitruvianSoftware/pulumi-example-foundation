# Copyright 2026 Vitruvian Software
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# =============================================================================
# Foundation Bootstrap — Containerized Environment
# =============================================================================
#
# This Dockerfile provides a reproducible environment for running the
# foundation bootstrap stage. It includes all required tools pre-installed.
#
# Usage:
#   docker build -t foundation-bootstrap .
#   docker run -it -v "$HOME/.config/gcloud:/root/.config/gcloud" \
#     -v "$(pwd):/workspace" -w /workspace foundation-bootstrap
#
# =============================================================================

FROM golang:1.23-bookworm

# Use ARGs so versions can be overridden at build time
ARG PULUMI_VERSION=3.165.0
ARG GCLOUD_VERSION=525.0.0

# Install system dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        curl \
        jq \
        unzip \
        git \
        ca-certificates \
        gnupg \
        python3 && \
    rm -rf /var/lib/apt/lists/*

# Install Pulumi CLI
RUN curl -fsSL https://get.pulumi.com | bash -s -- --version ${PULUMI_VERSION}
ENV PATH="/root/.pulumi/bin:${PATH}"

# Install Google Cloud SDK
RUN curl -sSL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz \
    | tar xz -C /opt && \
    /opt/google-cloud-sdk/install.sh --quiet --usage-reporting false && \
    /opt/google-cloud-sdk/bin/gcloud components install alpha beta --quiet
ENV PATH="/opt/google-cloud-sdk/bin:${PATH}"

# Set working directory
WORKDIR /workspace

# Default: show versions
ENTRYPOINT ["bash", "-c"]
CMD ["echo '=== Foundation Bootstrap ===' && go version && pulumi version && gcloud version --format=json | jq '.\"Google Cloud SDK\"'"]
