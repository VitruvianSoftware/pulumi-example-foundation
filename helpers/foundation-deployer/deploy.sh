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
# Foundation Deployer
# =============================================================================
#
# Automated multi-stage foundation deployment tool. This is the Pulumi
# equivalent of the Terraform foundation's `helpers/foundation-deployer/`.
#
# The TF version is a 35-file Go CLI. This Pulumi equivalent provides the
# same sequential deployment workflow in a single script, since Pulumi
# handles state management natively (no backend.tf manipulation needed).
#
# Usage:
#   ./helpers/foundation-deployer/deploy.sh [OPTIONS]
#
# Options:
#   --stage <N>       Deploy a specific stage (0-5). Default: all stages.
#   --action <action> "preview" or "up". Default: preview.
#   --stack <stack>    Stack name (e.g., "production"). Default: all stacks.
#   --destroy         Destroy in reverse order instead of deploying.
#   --dry-run         Show what would be executed without running.
#   --help            Show this help message.
#
# Examples:
#   ./deploy.sh --action preview                  # Preview all stages
#   ./deploy.sh --action up --stage 0             # Deploy bootstrap only
#   ./deploy.sh --action up                       # Deploy all stages
#   ./deploy.sh --destroy                         # Tear down everything
#
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
ACTION="preview"
TARGET_STAGE=""
TARGET_STACK=""
DESTROY=false
DRY_RUN=false

# Load .env file if present
if [ -f ".env" ]; then
    set -a
    source ".env"
    set +a
elif [ -f "../.env" ]; then
    set -a
    source "../.env"
    set +a
fi

# Ordered list of stages (forward deployment order)
STAGES=(
    "0-bootstrap"
    "1-org"
    "2-environments"
    "3-networks-svpc"    # Change to 3-networks-hub-and-spoke if using that mode
    "4-projects"
    "5-app-infra"
)

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case $1 in
        --stage)   TARGET_STAGE="$2"; shift 2 ;;
        --action)  ACTION="$2"; shift 2 ;;
        --stack)   TARGET_STACK="$2"; shift 2 ;;
        --destroy) DESTROY=true; shift ;;
        --dry-run) DRY_RUN=true; shift ;;
        --help)
            head -44 "$0" | tail -30
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Find all Pulumi config files to determine directories and stacks
find_stacks() {
    local stage="$1"
    local stacks=()

    while read -r config_file; do
        if [[ -z "$config_file" ]]; then continue; fi
        
        local dir=$(dirname "$config_file")
        local file=$(basename "$config_file")
        
        # Extract stack name from Pulumi.<stack_name>.yaml
        local stack_name="${file#Pulumi.}"
        stack_name="${stack_name%.yaml}"
        
        if [[ "$stack_name" != "" && "$stack_name" != "Pulumi" ]]; then
            stacks+=("$dir:$stack_name")
        fi
    done < <(find "$stage" -type f -name "Pulumi.*.yaml" | grep -v "\.example$" | sort)

    echo "${stacks[@]}"
}

# Deploy a single Pulumi project directory
deploy_stack() {
    local target="$1"
    local dir="${target%:*}"
    local stack_name="${target#*:}"

    if [ -n "$TARGET_STACK" ] && [ "$stack_name" != "$TARGET_STACK" ]; then
        log_warn "Skipping $dir (stack $stack_name != $TARGET_STACK)"
        return 0
    fi

    log_info "Deploying: $dir (stack: $stack_name)"

    if [ "$DRY_RUN" = true ]; then
        echo "  [DRY RUN] cd $dir && npm install && pulumi $ACTION --stack $stack_name --yes"
        return 0
    fi

    pushd "$dir" > /dev/null

    # Install dependencies (TS-specific; skip for Go)
    if [ -f "package.json" ]; then
        npm install --silent 2>/dev/null || true
    fi

    # Select or create stack
    pulumi stack select "$stack_name" 2>/dev/null || pulumi stack init "$stack_name" 2>/dev/null || true

    # Execute action
    if [ "$DESTROY" = true ]; then
        log_warn "DESTROYING stack: $stack_name"
        pulumi destroy --yes --skip-preview
    elif [ "$ACTION" = "preview" ]; then
        pulumi preview
    elif [ "$ACTION" = "up" ]; then
        pulumi up --yes --skip-preview
    fi

    popd > /dev/null
}

# ---------------------------------------------------------------------------
# Main execution
# ---------------------------------------------------------------------------
echo "============================================="
echo " Foundation Deployer"
echo " Action:  $ACTION"
echo " Destroy: $DESTROY"
echo " Stage:   ${TARGET_STAGE:-all}"
echo " Stack:   ${TARGET_STACK:-all}"
echo "============================================="
echo ""

# Determine stage order (reverse for destroy)
if [ "$DESTROY" = true ]; then
    ordered=()
    for ((i=${#STAGES[@]}-1; i>=0; i--)); do
        ordered+=("${STAGES[$i]}")
    done
else
    ordered=("${STAGES[@]}")
fi

for stage in "${ordered[@]}"; do
    # Filter by target stage if specified
    if [ -n "$TARGET_STAGE" ]; then
        stage_num="${stage:0:1}"
        if [ "$stage_num" != "$TARGET_STAGE" ]; then
            continue
        fi
    fi

    if [ ! -d "$stage" ]; then
        log_warn "Stage directory not found: $stage (skipping)"
        continue
    fi

    echo ""
    log_info "========== Stage: $stage =========="

    stack_dirs=$(find_stacks "$stage")
    if [ -z "$stack_dirs" ]; then
        log_warn "No Pulumi projects found in $stage"
        continue
    fi

    ordered_targets=()
    # 1. First prioritize 'shared' stacks
    for target in $stack_dirs; do
        if [[ "$target" == *:shared ]]; then
            ordered_targets+=("$target")
        fi
    done
    # 2. Then add 'development'
    for target in $stack_dirs; do
        if [[ "$target" == *:development ]]; then
            ordered_targets+=("$target")
        fi
    done
    # 3. Then 'nonproduction'
    for target in $stack_dirs; do
        if [[ "$target" == *:nonproduction ]]; then
            ordered_targets+=("$target")
        fi
    done
    # 4. Finally 'production' (and anything else like bootstrap/org)
    for target in $stack_dirs; do
        if [[ "$target" != *:shared && "$target" != *:development && "$target" != *:nonproduction ]]; then
            ordered_targets+=("$target")
        fi
    done

    # If destroying, reverse the order
    if [ "$DESTROY" = true ]; then
        reversed=()
        for ((i=${#ordered_targets[@]}-1; i>=0; i--)); do
            reversed+=("${ordered_targets[$i]}")
        done
        ordered_targets=("${reversed[@]}")
    fi

    for target in "${ordered_targets[@]}"; do
        deploy_stack "$target"
    done

    # Automatically set GOOGLE_BILLING_PROJECT for subsequent stages when running locally
    if [[ "$stage" == "0-bootstrap" && "$ACTION" == "up" && "$DESTROY" == false ]]; then
        pushd "0-bootstrap" > /dev/null
        SEED_PROJECT=$(pulumi stack output seed_project_id -s production 2>/dev/null || true)
        popd > /dev/null
        if [ -n "$SEED_PROJECT" ]; then
            export GOOGLE_BILLING_PROJECT="$SEED_PROJECT"
            export GOOGLE_PROJECT="$SEED_PROJECT"
            log_info "Auto-configured GOOGLE_PROJECT=$GOOGLE_PROJECT for downstream stages."
        fi
    fi

    log_info "Stage $stage complete"
done

echo ""
log_info "============================================="
log_info " Foundation deployment complete"
log_info "============================================="
