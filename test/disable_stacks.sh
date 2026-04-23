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
# disable_stacks.sh — Disable Pulumi stack configs for integration testing
# =============================================================================
#
# This script renames Pulumi.<env>.yaml files to .disabled so that integration
# tests can run with test-specific configurations without interference from
# production stack configs. This mirrors the Terraform foundation's
# test/disable_tf_files.sh which renames backend.tf files.
#
# Usage:
#   ./test/disable_stacks.sh [--stage <stage>]
#
# Examples:
#   ./test/disable_stacks.sh                    # All stages
#   ./test/disable_stacks.sh --stage 1-org      # Single stage
# =============================================================================

set -euo pipefail

TARGET_STAGE="${2:-all}"

disable_stage() {
    local stage="$1"
    echo "--- Disabling stack configs in $stage ---"

    find "$stage" -name "Pulumi.*.yaml" -not -name "Pulumi.yaml" -not -name "*.example" | while read -r f; do
        if [[ ! "$f" == *.disabled ]]; then
            echo "  $f → ${f}.disabled"
            mv "$f" "${f}.disabled"
        fi
    done
}

if [ "$TARGET_STAGE" = "all" ]; then
    for stage in 0-bootstrap 1-org 2-environments 3-networks-svpc 3-networks-hub-and-spoke 4-projects 5-app-infra; do
        [ -d "$stage" ] && disable_stage "$stage"
    done
else
    [ -d "$TARGET_STAGE" ] && disable_stage "$TARGET_STAGE"
fi

echo "=== Done ==="
