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
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployOrgIAM creates all IAM bindings for governance groups on
// org-level projects. This mirrors the Terraform foundation's iam.tf.
func deployOrgIAM(ctx *pulumi.Context, cfg *OrgConfig, proj *OrgProjects) error {
	// ========================================================================
	// 1. Audit Logs Project — IAM for audit_data_users
	// ========================================================================
	if cfg.AuditDataUsers != "" {
		auditGroup := fmt.Sprintf("group:%s", cfg.AuditDataUsers)
		auditRoles := []struct{ name, role string }{
			{"audit-log-viewer", "roles/logging.viewer"},
			{"audit-bq-user", "roles/bigquery.user"},
			{"audit-bq-data-viewer", "roles/bigquery.dataViewer"},
		}
		for _, r := range auditRoles {
			if _, err := projects.NewIAMMember(ctx, r.name, &projects.IAMMemberArgs{
				Project: proj.AuditLogsProjectID,
				Role:    pulumi.String(r.role),
				Member:  pulumi.String(auditGroup),
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// 2. Billing Export Project — IAM for billing_data_users
	// ========================================================================
	if cfg.BillingDataUsers != "" {
		billingGroup := fmt.Sprintf("group:%s", cfg.BillingDataUsers)

		// Project-level: BQ user + data viewer
		billingRoles := []struct{ name, role string }{
			{"billing-bq-user", "roles/bigquery.user"},
			{"billing-bq-data-viewer", "roles/bigquery.dataViewer"},
		}
		for _, r := range billingRoles {
			if _, err := projects.NewIAMMember(ctx, r.name, &projects.IAMMemberArgs{
				Project: proj.BillingExportProjectID,
				Role:    pulumi.String(r.role),
				Member:  pulumi.String(billingGroup),
			}); err != nil {
				return err
			}
		}

		// Org-level: billing viewer
		if _, err := organizations.NewIAMMember(ctx, "billing-viewer", &organizations.IAMMemberArgs{
			OrgId:  pulumi.String(cfg.OrgID),
			Role:   pulumi.String("roles/billing.viewer"),
			Member: pulumi.String(billingGroup),
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// 3. Security Reviewer Group — org or folder level
	// ========================================================================
	if cfg.GCPSecurityReviewer != "" {
		member := fmt.Sprintf("group:%s", cfg.GCPSecurityReviewer)
		if cfg.ParentFolder == "" {
			if _, err := organizations.NewIAMMember(ctx, "security-reviewer", &organizations.IAMMemberArgs{
				OrgId:  pulumi.String(cfg.OrgID),
				Role:   pulumi.String("roles/iam.securityReviewer"),
				Member: pulumi.String(member),
			}); err != nil {
				return err
			}
		} else {
			if _, err := organizations.NewIAMMember(ctx, "security-reviewer-folder", &organizations.IAMMemberArgs{
				OrgId:  pulumi.String(cfg.ParentFolder),
				Role:   pulumi.String("roles/iam.securityReviewer"),
				Member: pulumi.String(member),
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// 4. Network Viewer Group — org or folder level
	// ========================================================================
	if cfg.GCPNetworkViewer != "" {
		member := fmt.Sprintf("group:%s", cfg.GCPNetworkViewer)
		if cfg.ParentFolder == "" {
			if _, err := organizations.NewIAMMember(ctx, "network-viewer", &organizations.IAMMemberArgs{
				OrgId:  pulumi.String(cfg.OrgID),
				Role:   pulumi.String("roles/compute.networkViewer"),
				Member: pulumi.String(member),
			}); err != nil {
				return err
			}
		} else {
			if _, err := organizations.NewIAMMember(ctx, "network-viewer-folder", &organizations.IAMMemberArgs{
				OrgId:  pulumi.String(cfg.ParentFolder),
				Role:   pulumi.String("roles/compute.networkViewer"),
				Member: pulumi.String(member),
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// 5. SCC Admin Group
	// ========================================================================
	if cfg.GCPSCCAdmin != "" {
		member := fmt.Sprintf("group:%s", cfg.GCPSCCAdmin)

		// Org-level: SCC admin editor (only when not under parent_folder)
		if cfg.ParentFolder == "" {
			if _, err := organizations.NewIAMMember(ctx, "org-scc-admin", &organizations.IAMMemberArgs{
				OrgId:  pulumi.String(cfg.OrgID),
				Role:   pulumi.String("roles/securitycenter.adminEditor"),
				Member: pulumi.String(member),
			}); err != nil {
				return err
			}
		}

		// Project-level: SCC admin editor on SCC project (when SCC resources enabled)
		if cfg.EnableSCCResources {
			if _, err := projects.NewIAMMember(ctx, "project-scc-admin", &projects.IAMMemberArgs{
				Project: proj.SCCProjectID,
				Role:    pulumi.String("roles/securitycenter.adminEditor"),
				Member:  pulumi.String(member),
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// 6. Global Secrets Admin Group
	// ========================================================================
	if cfg.GCPGlobalSecretsAdmin != "" {
		if _, err := projects.NewIAMMember(ctx, "global-secrets-admin", &projects.IAMMemberArgs{
			Project: proj.OrgSecretsProjectID,
			Role:    pulumi.String("roles/secretmanager.admin"),
			Member:  pulumi.String(fmt.Sprintf("group:%s", cfg.GCPGlobalSecretsAdmin)),
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// 7. KMS Admin Group
	// ========================================================================
	if cfg.GCPKMSAdmin != "" {
		kmsGroup := fmt.Sprintf("group:%s", cfg.GCPKMSAdmin)

		// Project-level: KMS viewer on KMS project
		if _, err := projects.NewIAMMember(ctx, "kms-viewer", &projects.IAMMemberArgs{
			Project: proj.OrgKMSProjectID,
			Role:    pulumi.String("roles/cloudkms.viewer"),
			Member:  pulumi.String(kmsGroup),
		}); err != nil {
			return err
		}

		// Org-level: KMS protected resources viewer (when tracking enabled)
		if cfg.EnableKMSKeyUsageTracking {
			if _, err := organizations.NewIAMMember(ctx, "kms-protected-resources-viewer", &organizations.IAMMemberArgs{
				OrgId:  pulumi.String(cfg.OrgID),
				Role:   pulumi.String("roles/cloudkms.protectedResourcesViewer"),
				Member: pulumi.String(kmsGroup),
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
