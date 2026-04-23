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
# restore_stacks.sh — Restore disabled Pulumi stack configs
# =============================================================================
#
# Reverses the effect of disable_stacks.sh by renaming .disabled files back
# to their original names. This mirrors the Terraform foundation's
# test/restore_tf_files.sh.
#
# Usage:
#   ./test/restore_stacks.sh [--stage <stage>]
# =============================================================================

set -euo pipefail

TARGET_STAGE="${2:-all}"

restore_stage() {
    local stage="$1"
    echo "--- Restoring stack configs in $stage ---"

    find "$stage" -name "Pulumi.*.yaml.disabled" | while read -r f; do
        restored="${f%.disabled}"
        echo "  $f → $restored"
        mv "$f" "$restored"
    done
}

if [ "$TARGET_STAGE" = "all" ]; then
    for stage in 0-bootstrap 1-org 2-environments 3-networks-svpc 3-networks-hub-and-spoke 4-projects 5-app-infra; do
        [ -d "$stage" ] && restore_stage "$stage"
    done
else
    [ -d "$TARGET_STAGE" ] && restore_stage "$TARGET_STAGE"
fi

echo "=== Done ==="
