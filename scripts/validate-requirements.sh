#!/usr/bin/env bash

# Copyright 2026 Vitruvian Software
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# validate-requirements.sh
#
# Pre-flight validation script for the Pulumi Go Foundation.
# Checks that all required tools are installed at the correct versions,
# validates IAM permissions, and ensures required APIs are enabled.
#
# Usage:
#   ./scripts/validate-requirements.sh -o <ORG_ID> -b <BILLING_ACCOUNT_ID> -u <USER_EMAIL>

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Minimum versions
PULUMI_VERSION="3.0.0"
GO_VERSION="1.21.0"
GCLOUD_VERSION="393.0.0"
GIT_VERSION="2.28.0"

# Parse arguments
ORG_ID=""
BILLING_ACCOUNT=""
USER_EMAIL=""

while getopts "o:b:u:" opt; do
  case ${opt} in
    o) ORG_ID="${OPTARG}" ;;
    b) BILLING_ACCOUNT="${OPTARG}" ;;
    u) USER_EMAIL="${OPTARG}" ;;
    *) echo "Usage: $0 -o <ORG_ID> -b <BILLING_ACCOUNT_ID> -u <USER_EMAIL>"; exit 1 ;;
  esac
done

if [[ -z "${ORG_ID}" || -z "${BILLING_ACCOUNT}" || -z "${USER_EMAIL}" ]]; then
  echo "Usage: $0 -o <ORG_ID> -b <BILLING_ACCOUNT_ID> -u <USER_EMAIL>"
  exit 1
fi

ERRORS=0
WARNINGS=0

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; ERRORS=$((ERRORS + 1)); }
warn() { echo -e "${YELLOW}!${NC} $1"; WARNINGS=$((WARNINGS + 1)); }

# Compare semantic versions: returns 0 if $1 >= $2
version_gte() {
  printf '%s\n%s' "$2" "$1" | sort -V -C
}

echo ""
echo "═══════════════════════════════════════════════════════════"
echo " Pulumi Go Foundation — Pre-flight Validation"
echo "═══════════════════════════════════════════════════════════"
echo ""

# --- Tool Checks ---
echo "▸ Checking required tools..."
echo ""

# Pulumi CLI
if command -v pulumi &> /dev/null; then
  INSTALLED_PULUMI=$(pulumi version 2>/dev/null | sed 's/^v//')
  if version_gte "${INSTALLED_PULUMI}" "${PULUMI_VERSION}"; then
    pass "Pulumi CLI ${INSTALLED_PULUMI} (>= ${PULUMI_VERSION})"
  else
    fail "Pulumi CLI ${INSTALLED_PULUMI} (need >= ${PULUMI_VERSION})"
  fi
else
  fail "Pulumi CLI not found. Install: https://www.pulumi.com/docs/install/"
fi

# Go
if command -v go &> /dev/null; then
  INSTALLED_GO=$(go version | awk '{print $3}' | sed 's/^go//')
  if version_gte "${INSTALLED_GO}" "${GO_VERSION}"; then
    pass "Go ${INSTALLED_GO} (>= ${GO_VERSION})"
  else
    fail "Go ${INSTALLED_GO} (need >= ${GO_VERSION})"
  fi
else
  fail "Go not found. Install: https://go.dev/dl/"
fi

# Google Cloud SDK
if command -v gcloud &> /dev/null; then
  INSTALLED_GCLOUD=$(gcloud version 2>/dev/null | head -1 | grep -oP '[\d.]+' || gcloud --version 2>/dev/null | head -1 | awk '{print $NF}')
  if [[ -n "${INSTALLED_GCLOUD}" ]]; then
    if version_gte "${INSTALLED_GCLOUD}" "${GCLOUD_VERSION}"; then
      pass "Google Cloud SDK ${INSTALLED_GCLOUD} (>= ${GCLOUD_VERSION})"
    else
      fail "Google Cloud SDK ${INSTALLED_GCLOUD} (need >= ${GCLOUD_VERSION})"
    fi
  else
    warn "Could not determine gcloud version"
  fi
else
  fail "Google Cloud SDK not found. Install: https://cloud.google.com/sdk/install"
fi

# Git
if command -v git &> /dev/null; then
  INSTALLED_GIT=$(git --version | awk '{print $3}')
  if version_gte "${INSTALLED_GIT}" "${GIT_VERSION}"; then
    pass "Git ${INSTALLED_GIT} (>= ${GIT_VERSION})"
  else
    fail "Git ${INSTALLED_GIT} (need >= ${GIT_VERSION})"
  fi
else
  fail "Git not found. Install: https://git-scm.com/"
fi

# jq (optional but recommended)
if command -v jq &> /dev/null; then
  pass "jq $(jq --version 2>/dev/null | sed 's/^jq-//')"
else
  warn "jq not found (optional but recommended). Install: https://jqlang.github.io/jq/download/"
fi

echo ""

# --- IAM Checks ---
echo "▸ Checking IAM permissions for ${USER_EMAIL}..."
echo ""

REQUIRED_ROLES=(
  "roles/resourcemanager.organizationAdmin"
  "roles/orgpolicy.policyAdmin"
  "roles/resourcemanager.projectCreator"
  "roles/resourcemanager.folderCreator"
  "roles/securitycenter.admin"
)

for role in "${REQUIRED_ROLES[@]}"; do
  if gcloud organizations get-iam-policy "${ORG_ID}" \
    --flatten="bindings[].members" \
    --filter="bindings.role=${role} AND bindings.members:user:${USER_EMAIL}" \
    --format="value(bindings.role)" 2>/dev/null | grep -q "${role}"; then
    pass "${role}"
  else
    warn "${role} — could not verify (user may be in a group with this role)"
  fi
done

# Billing admin check
if gcloud beta billing accounts get-iam-policy "${BILLING_ACCOUNT}" \
  --flatten="bindings[].members" \
  --filter="bindings.role=roles/billing.admin AND bindings.members:user:${USER_EMAIL}" \
  --format="value(bindings.role)" 2>/dev/null | grep -q "roles/billing.admin"; then
  pass "roles/billing.admin on billing account"
else
  warn "roles/billing.admin — could not verify on billing account ${BILLING_ACCOUNT}"
fi

echo ""

# --- API Checks ---
echo "▸ Checking required APIs..."
echo ""

REQUIRED_APIS=(
  "cloudresourcemanager.googleapis.com"
  "cloudbilling.googleapis.com"
  "iam.googleapis.com"
  "cloudkms.googleapis.com"
  "servicenetworking.googleapis.com"
)

for api in "${REQUIRED_APIS[@]}"; do
  if gcloud services list --enabled --filter="config.name=${api}" --format="value(config.name)" 2>/dev/null | grep -q "${api}"; then
    pass "${api}"
  else
    warn "${api} — not enabled in current project (may need: gcloud services enable ${api})"
  fi
done

echo ""

# --- Summary ---
echo "═══════════════════════════════════════════════════════════"
if [[ ${ERRORS} -gt 0 ]]; then
  echo -e " ${RED}FAILED${NC}: ${ERRORS} error(s), ${WARNINGS} warning(s)"
  echo " Please fix the errors above before proceeding."
  echo "═══════════════════════════════════════════════════════════"
  exit 1
elif [[ ${WARNINGS} -gt 0 ]]; then
  echo -e " ${YELLOW}PASSED WITH WARNINGS${NC}: ${WARNINGS} warning(s)"
  echo " The script could not verify some permissions."
  echo " You may proceed, but verify the warnings manually."
  echo "═══════════════════════════════════════════════════════════"
  exit 0
else
  echo -e " ${GREEN}ALL CHECKS PASSED${NC}"
  echo " Your environment is ready to deploy the foundation."
  echo "═══════════════════════════════════════════════════════════"
  exit 0
fi
