/*
 * Copyright 2026 Vitruvian Software
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"github.com/VitruvianSoftware/pulumi-library/go/pkg/group"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployGroups optionally creates Google Workspace groups via Cloud Identity.
// This mirrors the Terraform foundation's 0-bootstrap/groups.tf which uses
// the terraform-google-modules/group/google module.
//
// Groups are only created when create_required_groups or create_optional_groups
// is set to true in the config. When disabled, the bootstrap assumes the
// groups already exist and uses their email addresses for IAM bindings.
func deployGroups(ctx *pulumi.Context, cfg *Config) error {
	if !cfg.CreateRequiredGroups && !cfg.CreateOptionalGroups {
		return nil // Groups are pre-existing; nothing to create.
	}

	// Look up the org's directory customer ID (needed to scope groups).
	org, err := organizations.GetOrganization(ctx, &organizations.GetOrganizationArgs{
		Organization: &cfg.OrgID,
	})
	if err != nil {
		return err
	}
	customerID := pulumi.String(org.DirectoryCustomerId)

	// ========================================================================
	// Required Groups
	// These are the minimum groups needed by the foundation. Mirrors the
	// TF foundation's module "required_group" for_each block.
	// ========================================================================
	if cfg.CreateRequiredGroups {
		requiredGroups := map[string]string{
			"group_org_admins":     cfg.GroupOrgAdmins,
			"group_billing_admins": cfg.GroupBillingAdmins,
			"billing_data_users":   cfg.BillingDataUsers,
			"audit_data_users":     cfg.AuditDataUsers,
		}

		for key, email := range requiredGroups {
			if email == "" {
				continue
			}
			if _, err := group.NewGroup(ctx, "required-"+key, &group.GroupArgs{
				ID:                 email,
				DisplayName:        key,
				Description:        key,
				CustomerID:         customerID,
				InitialGroupConfig: cfg.InitialGroupConfig,
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// Optional Groups
	// Governance groups consumed by 1-org stage. Only created if
	// create_optional_groups is true AND the email is non-empty.
	// Mirrors the TF foundation's module "optional_group".
	// ========================================================================
	if cfg.CreateOptionalGroups {
		optionalGroups := map[string]string{
			"gcp_security_reviewer":    cfg.GCPSecurityReviewer,
			"gcp_network_viewer":       cfg.GCPNetworkViewer,
			"gcp_scc_admin":            cfg.GCPSCCAdmin,
			"gcp_global_secrets_admin": cfg.GCPGlobalSecretsAdmin,
			"gcp_kms_admin":            cfg.GCPKMSAdmin,
		}

		for key, email := range optionalGroups {
			if email == "" {
				continue // Skip unconfigured optional groups
			}
			if _, err := group.NewGroup(ctx, "optional-"+key, &group.GroupArgs{
				ID:                 email,
				DisplayName:        key,
				Description:        key,
				CustomerID:         customerID,
				InitialGroupConfig: cfg.InitialGroupConfig,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
