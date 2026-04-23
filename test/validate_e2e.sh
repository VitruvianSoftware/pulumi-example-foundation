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
# E2E Validation
# =============================================================================
# Validates that all foundation resources were created correctly by querying
# GCP APIs directly. This is the verification step that runs AFTER deploy.sh
# to ensure the foundation is structurally correct.
#
# Usage:
#   ./test/validate_e2e.sh
#
# Requires:
#   - .env file with E2E_ORG_ID, E2E_BILLING_ACCOUNT, E2E_DOMAIN
#   - gcloud authenticated as the sandbox admin
#   - Foundation deployed via deploy.sh
# =============================================================================

set -e

# Load .env if it exists
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0
WARN=0

assert_exists() {
  local description="$1"
  local result="$2"

  if [ -n "$result" ] && [ "$result" != "0" ] && [ "$result" != "null" ]; then
    echo -e "  ${GREEN}✓${NC} $description"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $description"
    FAIL=$((FAIL + 1))
  fi
}

assert_count() {
  local description="$1"
  local actual="$2"
  local expected="$3"

  if [ "$actual" -ge "$expected" ] 2>/dev/null; then
    echo -e "  ${GREEN}✓${NC} $description (found $actual, expected >= $expected)"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} $description (found $actual, expected >= $expected)"
    FAIL=$((FAIL + 1))
  fi
}

warn_missing() {
  local description="$1"
  echo -e "  ${YELLOW}⚠${NC} $description (skipped — stage not deployed)"
  WARN=$((WARN + 1))
}

echo "============================================="
echo " Foundation E2E Validation"
echo " Organization: ${E2E_ORG_ID}"
echo " Domain:       ${E2E_DOMAIN}"
echo "============================================="
echo ""

# =========================================================================
# Phase 0: Bootstrap
# =========================================================================
echo "━━━ Phase 0: Bootstrap ━━━"

# Check bootstrap folder exists
BOOTSTRAP_FOLDER=$(gcloud resource-manager folders list \
  --organization="${E2E_ORG_ID}" \
  --format="value(name)" \
  --filter="displayName:fldr-bootstrap" 2>/dev/null | head -1)
assert_exists "Bootstrap folder exists" "$BOOTSTRAP_FOLDER"

# Check seed project
SEED_PROJECT=$(gcloud projects list \
  --format="value(projectId)" \
  --filter="projectId:prj-b-seed*" 2>/dev/null | head -1)
assert_exists "Seed project exists (prj-b-seed-*)" "$SEED_PROJECT"

# Check CICD project
CICD_PROJECT=$(gcloud projects list \
  --format="value(projectId)" \
  --filter="projectId:prj-b-cicd*" 2>/dev/null | head -1)
assert_exists "CICD project exists (prj-b-cicd-*)" "$CICD_PROJECT"

# Check KMS key ring
if [ -n "$SEED_PROJECT" ]; then
  KMS_KEYRING=$(gcloud kms keyrings list \
    --location=us-central1 \
    --project="$SEED_PROJECT" \
    --format="value(name)" 2>/dev/null | head -1)
  assert_exists "KMS keyring exists in seed project" "$KMS_KEYRING"

  # Check state bucket
  STATE_BUCKET=$(gcloud storage buckets list \
    --project="$SEED_PROJECT" \
    --format="value(name)" 2>/dev/null | grep "tfstate" | head -1)
  assert_exists "State bucket exists" "$STATE_BUCKET"

  # Check service accounts
  SA_COUNT=$(gcloud iam service-accounts list \
    --project="$SEED_PROJECT" \
    --format="value(email)" \
    --filter="email:sa-terraform-*" 2>/dev/null | wc -l | tr -d ' ')
  assert_count "Pipeline service accounts created" "$SA_COUNT" 4
fi

echo ""

# =========================================================================
# Phase 1: Org
# =========================================================================
echo "━━━ Phase 1: Organization ━━━"

# Check org-level folders
COMMON_FOLDER=$(gcloud resource-manager folders list \
  --organization="${E2E_ORG_ID}" \
  --format="value(name)" \
  --filter="displayName:fldr-common" 2>/dev/null | head -1)

if [ -n "$COMMON_FOLDER" ]; then
  assert_exists "Common folder exists" "$COMMON_FOLDER"

  # Check org-level projects
  ORG_AUDIT_PROJECT=$(gcloud projects list \
    --format="value(projectId)" \
    --filter="projectId:prj-c-logging*" 2>/dev/null | head -1)
  assert_exists "Audit logging project exists (prj-c-logging-*)" "$ORG_AUDIT_PROJECT"

  ORG_BILLING_PROJECT=$(gcloud projects list \
    --format="value(projectId)" \
    --filter="projectId:prj-c-billing*" 2>/dev/null | head -1)
  assert_exists "Billing export project exists (prj-c-billing-*)" "$ORG_BILLING_PROJECT"

  ORG_SCC_PROJECT=$(gcloud projects list \
    --format="value(projectId)" \
    --filter="projectId:prj-c-scc*" 2>/dev/null | head -1)
  assert_exists "SCC notifications project exists (prj-c-scc-*)" "$ORG_SCC_PROJECT"
else
  warn_missing "Phase 1 not deployed"
fi

echo ""

# =========================================================================
# Phase 2: Environments
# =========================================================================
echo "━━━ Phase 2: Environments ━━━"

ENV_FOLDERS=$(gcloud resource-manager folders list \
  --organization="${E2E_ORG_ID}" \
  --format="value(displayName)" \
  --filter="displayName:(fldr-development OR fldr-nonproduction OR fldr-production)" 2>/dev/null | wc -l | tr -d ' ')

if [ "$ENV_FOLDERS" -ge 1 ] 2>/dev/null; then
  assert_count "Environment folders created" "$ENV_FOLDERS" 3
else
  warn_missing "Phase 2 not deployed"
fi

echo ""

# =========================================================================
# Phase 3: Networks
# =========================================================================
echo "━━━ Phase 3: Networks ━━━"

NET_PROJECTS=$(gcloud projects list \
  --format="value(projectId)" \
  --filter="projectId:(prj-d-shared-base* OR prj-n-shared-base* OR prj-p-shared-base* OR prj-d-shared-restricted* OR prj-n-shared-restricted* OR prj-p-shared-restricted*)" 2>/dev/null | wc -l | tr -d ' ')

if [ "$NET_PROJECTS" -ge 1 ] 2>/dev/null; then
  assert_count "Network host projects created" "$NET_PROJECTS" 2
else
  warn_missing "Phase 3 not deployed"
fi

echo ""

# =========================================================================
# Phase 4: Projects
# =========================================================================
echo "━━━ Phase 4: Projects ━━━"

BU_PROJECTS=$(gcloud projects list \
  --format="value(projectId)" \
  --filter="projectId:(prj-d-bu1* OR prj-n-bu1* OR prj-p-bu1*)" 2>/dev/null | wc -l | tr -d ' ')

if [ "$BU_PROJECTS" -ge 1 ] 2>/dev/null; then
  assert_count "Business unit projects created" "$BU_PROJECTS" 3
else
  warn_missing "Phase 4 not deployed"
fi

echo ""

# =========================================================================
# Phase 5: App Infra
# =========================================================================
echo "━━━ Phase 5: App Infra ━━━"

APP_PROJECTS=$(gcloud projects list \
  --format="value(projectId)" \
  --filter="labels.application_name=app-infra-pipelines" 2>/dev/null | wc -l | tr -d ' ')

if [ "$APP_PROJECTS" -ge 1 ] 2>/dev/null; then
  assert_count "App infra projects created" "$APP_PROJECTS" 1
else
  warn_missing "Phase 5 not deployed"
fi

echo ""

# =========================================================================
# Summary
# =========================================================================
TOTAL=$((PASS + FAIL + WARN))
echo "============================================="
echo -e " Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}, ${YELLOW}${WARN} skipped${NC} / ${TOTAL} total"
echo "============================================="

if [ "$FAIL" -gt 0 ]; then
  echo -e "${RED}E2E VALIDATION FAILED${NC}"
  exit 1
else
  echo -e "${GREEN}E2E VALIDATION PASSED${NC}"
  exit 0
fi
