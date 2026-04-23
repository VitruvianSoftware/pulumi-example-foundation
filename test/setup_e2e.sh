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
# E2E Test Setup 
# =============================================================================
# This script injects your Organization ID, Billing ID, and Identity domain
# into all Pulumi.<stack>.yaml configuration files across the foundation.
#
# Usage:
#   export E2E_ORG_ID="1234567890"
#   export E2E_BILLING_ACCOUNT="XXXXXX-XXXXXX-XXXXXX"
#   export E2E_DOMAIN="example.com"
#   ./test/setup_e2e.sh
# =============================================================================

set -e

# Load .env if it exists
if [ -f .env ]; then
  echo "Loading environment variables from .env..."
  set -a
  source .env
  set +a
fi

# Validate required variables
if [ -z "$E2E_ORG_ID" ]; then
  echo "Error: E2E_ORG_ID environment variable is required."
  exit 1
fi

if [ -z "$E2E_BILLING_ACCOUNT" ]; then
  echo "Error: E2E_BILLING_ACCOUNT environment variable is required."
  exit 1
fi

if [ -z "$E2E_DOMAIN" ]; then
  echo "Error: E2E_DOMAIN environment variable is required."
  exit 1
fi

E2E_FOLDER_ID=${E2E_FOLDER_ID:-""}

echo "================================================="
echo " Configuring Foundation for E2E Deployment"
echo "================================================="
echo " Organization:  $E2E_ORG_ID"
if [ -n "$E2E_FOLDER_ID" ]; then
  echo " Parent Folder: $E2E_FOLDER_ID"
fi
echo " Billing ID:    $E2E_BILLING_ACCOUNT"
echo " Domain:        $E2E_DOMAIN"

PULUMI_USER=$(pulumi whoami 2>/dev/null || echo "organization/vitruvian")
echo " Pulumi User:   $PULUMI_USER"
echo "================================================="

# Generate exact group emails based on the provided domain
ORG_ADMINS="gcp-organization-admins@${E2E_DOMAIN}"
BILLING_ADMINS="gcp-billing-admins@${E2E_DOMAIN}"
NETWORK_ADMINS="gcp-network-admins@${E2E_DOMAIN}"
SECURITY_ADMINS="gcp-security-admins@${E2E_DOMAIN}"
BILLING_DATA="gcp-billing-data@${E2E_DOMAIN}"
AUDIT_DATA="gcp-audit-data@${E2E_DOMAIN}"

# Find all example config files and inject variables
find . -type f -name "Pulumi.*.yaml.example" | while read -r example_file; do
  target_file="${example_file%.example}"
  echo "Generating $target_file..."
  
  # Use awk to do the replacement safely across MacOS/Linux
  awk -v org="$E2E_ORG_ID" \
      -v billing="$E2E_BILLING_ACCOUNT" \
      -v org_admin="$ORG_ADMINS" \
      -v bill_admin="$BILLING_ADMINS" \
      -v net_admin="$NETWORK_ADMINS" \
      -v sec_admin="$SECURITY_ADMINS" \
      -v bill_data="$BILLING_DATA" \
      -v audit_data="$AUDIT_DATA" \
      -v pulumi_user="$PULUMI_USER" \
      -v domain="$E2E_DOMAIN" \
      -v folder_id="$E2E_FOLDER_ID" \
      '{
        gsub(/YOUR_ORG_ID/, org);
        gsub(/XXXXXX-XXXXXX-XXXXXX/, billing);
        gsub(/org-admins@example.com/, org_admin);
        gsub(/billing-admins@example.com/, bill_admin);
        gsub(/network-admins@example.com/, net_admin);
        gsub(/security-admins@example.com/, sec_admin);
        gsub(/billing-data@example.com/, bill_data);
        gsub(/audit-data@example.com/, audit_data);
        gsub(/organization\/vitruvian/, pulumi_user);
        gsub(/domains_to_allow: "example.com"/, "domains_to_allow: \"" domain "\"");
        
        if (folder_id != "") {
          gsub(/# [a-zA-Z0-9_-]+:parent_folder: ""/, "0-bootstrap:parent_folder: \"" folder_id "\"");
        }
        
        print
      }' "$example_file" > "$target_file"
done

echo ""
echo "✅ E2E Configuration complete."
echo ""
echo "Next Steps:"
echo "1. Run the foundation deployer:"
echo "   ./helpers/foundation-deployer/deploy.sh --action up"
echo ""
echo "2. When finished, tear down the environment:"
echo "   ./test/clean_org.sh"
