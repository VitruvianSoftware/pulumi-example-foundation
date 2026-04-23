#!/usr/bin/env bash

# Copyright 2026 Vitruvian Software
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# =============================================================================
# clean_org.sh — Clean up test resources from organization
# =============================================================================
#
# This script removes foundation resources created during integration testing.
# It is the Pulumi equivalent of terraform-example-foundation/test/clean_org.sh.
#
# WARNING: This script DESTROYS real infrastructure. Use with extreme caution.
#
# Usage:
#   ./test/clean_org.sh [--dry-run]
#
# Prerequisites:
#   - gcloud CLI authenticated with org-level permissions
#   - ORGANIZATION_ID environment variable set
#   - BILLING_ACCOUNT environment variable set
# =============================================================================

set -euo pipefail

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN=true
    echo "=== DRY RUN MODE — no resources will be deleted ==="
fi

# Load .env if it exists
if [ -f .env ]; then
  echo "Loading environment variables from .env..."
  set -a
  source .env
  set +a
fi

ORGANIZATION_ID=${ORGANIZATION_ID:-$E2E_ORG_ID}

# Validate required env vars
: "${ORGANIZATION_ID:?Environment variable ORGANIZATION_ID or E2E_ORG_ID is required}"

echo "=== Cleaning foundation resources from org: ${ORGANIZATION_ID} ==="

# ---------------------------------------------------------------------------
# 1. Find and destroy Pulumi stacks
# ---------------------------------------------------------------------------
echo ""
echo "--- Checking for active Pulumi stacks ---"

STAGES=(
    "5-app-infra"
    "4-projects"
    "3-networks-svpc"
    "3-networks-hub-and-spoke"
    "2-environments"
    "1-org"
    "0-bootstrap"
)

for stage in "${STAGES[@]}"; do
    if [ -d "$stage" ]; then
        echo "Checking $stage..."
        pushd "$stage" > /dev/null

        # Find all Pulumi stacks in this stage
        if command -v pulumi &> /dev/null; then
            stacks=$(pulumi stack ls --json 2>/dev/null | jq -r '.[].name' 2>/dev/null || true)
            for stack in $stacks; do
                echo "  Found stack: $stack"
                if [ "$DRY_RUN" = false ]; then
                    echo "  Unprotecting all resources in stack: $stack"
                    pulumi state unprotect --all -y -s "$stack" 2>/dev/null || true
                    echo "  Destroying stack: $stack"
                    pulumi destroy --yes --skip-preview -s "$stack" 2>/dev/null || true
                    echo "  Removing stack: $stack"
                    pulumi stack rm "$stack" --force --yes 2>/dev/null || true
                else
                    echo "  [DRY RUN] Would destroy stack: $stack"
                fi
            done
        fi

        popd > /dev/null
    fi
done

# ---------------------------------------------------------------------------
# 2. Clean up orphaned projects with foundation prefix
# ---------------------------------------------------------------------------
echo ""
echo "--- Checking for orphaned foundation projects ---"

PROJECT_PREFIXES=("prj-b-" "prj-d-" "prj-n-" "prj-p-" "prj-c-")

for prefix in "${PROJECT_PREFIXES[@]}"; do
    projects=$(gcloud projects list --filter="projectId:${prefix}*" --format="value(projectId)" 2>/dev/null || true)
    for project in $projects; do
        echo "  Found project: $project"
        if [ "$DRY_RUN" = false ]; then
            echo "  Deleting project: $project"
            gcloud projects delete "$project" --quiet 2>/dev/null || true
        else
            echo "  [DRY RUN] Would delete project: $project"
        fi
    done
done

# ---------------------------------------------------------------------------
# 3. Clean up foundation folders
# ---------------------------------------------------------------------------
echo ""
echo "--- Checking for foundation folders ---"

FOLDER_PREFIXES=("fldr-")

for prefix in "${FOLDER_PREFIXES[@]}"; do
    folders=$(gcloud alpha resource-manager folders search --query="displayName:${prefix}*" --format="value(name)" 2>/dev/null || true)
    for folder_name in $folders; do
        folder_id=${folder_name#folders/}
        echo "  Found folder: $folder_id"
        if [ "$DRY_RUN" = false ]; then
            echo "  Deleting folder: $folder_id"
            gcloud resource-manager folders delete "$folder_id" --quiet 2>/dev/null || true
        else
            echo "  [DRY RUN] Would delete folder: $folder_id"
        fi
    done
done

echo ""
echo "=== Cleanup complete ==="
