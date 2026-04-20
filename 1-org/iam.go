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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/folder"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// orgOrFolderIAMMember creates an IAM member at the organization level or at the
// parent-folder level depending on whether a parent_folder is configured.
// This fixes G11: previously the folder branch incorrectly used organizations.NewIAMMember
// with OrgId set to a folder ID. Now it correctly uses folder.NewIAMMember.
func orgOrFolderIAMMember(ctx *pulumi.Context, name string, cfg *OrgConfig, role, member string) error {
	if cfg.ParentFolder == "" {
		if _, err := organizations.NewIAMMember(ctx, name, &organizations.IAMMemberArgs{
			OrgId:  pulumi.String(cfg.OrgID),
			Role:   pulumi.String(role),
			Member: pulumi.String(member),
		}); err != nil {
			return err
		}
	} else {
		if _, err := folder.NewIAMMember(ctx, name+"-folder", &folder.IAMMemberArgs{
			Folder: pulumi.String(cfg.ParentFolder),
			Role:   pulumi.String(role),
			Member: pulumi.String(member),
		}); err != nil {
			return err
		}
	}
	return nil
}

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
	// 3. Security Reviewer Group — org or folder level (G11 FIX)
	// Previously used organizations.NewIAMMember with OrgId=ParentFolder,
	// which is wrong. Now correctly uses folder.NewIAMMember for folder case.
	// ========================================================================
	if cfg.GCPSecurityReviewer != "" {
		member := fmt.Sprintf("group:%s", cfg.GCPSecurityReviewer)
		if err := orgOrFolderIAMMember(ctx, "security-reviewer", cfg, "roles/iam.securityReviewer", member); err != nil {
			return err
		}
	}

	// ========================================================================
	// 4. Network Viewer Group — org or folder level (G11 FIX)
	// ========================================================================
	if cfg.GCPNetworkViewer != "" {
		member := fmt.Sprintf("group:%s", cfg.GCPNetworkViewer)
		if err := orgOrFolderIAMMember(ctx, "network-viewer", cfg, "roles/compute.networkViewer", member); err != nil {
			return err
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

	// ========================================================================
	// 8. KMS Org Service Agent IAM (G3)
	// Grants roles/cloudkms.orgServiceAgent to the KMS service agent at the
	// org level when KMS key usage tracking is enabled. This allows KMS to
	// track key usage across all projects in the organization.
	// Mirrors: google_organization_iam_member "kms_usage_tracking" in iam.tf
	// ========================================================================
	if cfg.EnableKMSKeyUsageTracking {
		kmsServiceAgent := fmt.Sprintf("serviceAccount:service-org-%s@gcp-sa-cloudkms.iam.gserviceaccount.com", cfg.OrgID)
		if _, err := organizations.NewIAMMember(ctx, "kms-usage-tracking", &organizations.IAMMemberArgs{
			OrgId:  pulumi.String(cfg.OrgID),
			Role:   pulumi.String("roles/cloudkms.orgServiceAgent"),
			Member: pulumi.String(kmsServiceAgent),
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// 9. Audit Viewer Group IAM (G7)
	// Separate from audit_data_users — this is an optional governance group
	// that gets read access to audit logs and BQ data.
	// Mirrors: google_project_iam_member "audit_log_viewer",
	//          "audit_private_logviewer", "audit_bq_data_viewer" in iam.tf
	// ========================================================================
	if cfg.GCPAuditViewer != "" {
		viewerGroup := fmt.Sprintf("group:%s", cfg.GCPAuditViewer)
		viewerRoles := []struct{ name, role string }{
			{"audit-viewer-log", "roles/logging.viewer"},
			{"audit-viewer-private-log", "roles/logging.privateLogViewer"},
			{"audit-viewer-bq-data", "roles/bigquery.dataViewer"},
		}
		for _, r := range viewerRoles {
			if _, err := projects.NewIAMMember(ctx, r.name, &projects.IAMMemberArgs{
				Project: proj.AuditLogsProjectID,
				Role:    pulumi.String(r.role),
				Member:  pulumi.String(viewerGroup),
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// 10. Hub-and-Spoke Network SA IAM (G8)
	// When enable_hub_and_spoke is true, grant the networks pipeline SA
	// elevated roles on the hub project to manage compute, SAs, and IAM.
	// Mirrors: google_project_iam_member "network_sa" in projects.tf
	// ========================================================================
	if cfg.EnableHubAndSpoke && cfg.NetworksSAEmail != "" {
		hubAndSpokeRoles := []string{
			"roles/compute.instanceAdmin",
			"roles/iam.serviceAccountAdmin",
			"roles/resourcemanager.projectIamAdmin",
			"roles/iam.serviceAccountUser",
		}
		networkSA := fmt.Sprintf("serviceAccount:%s", cfg.NetworksSAEmail)
		for _, role := range hubAndSpokeRoles {
			if _, err := projects.NewIAMMember(ctx, fmt.Sprintf("net-hub-sa-%s", role), &projects.IAMMemberArgs{
				Project: proj.NetHubProjectID,
				Role:    pulumi.String(role),
				Member:  pulumi.String(networkSA),
			}); err != nil {
				return err
			}
		}
	}

	// ========================================================================
	// 11. CAI Monitoring Builder SA + IAM (G4+G5)
	// Create a dedicated SA for the CAI monitoring Cloud Build pipeline
	// and grant it the roles needed to build and deploy Cloud Functions.
	// Mirrors: sa.tf + iam.tf cai_monitoring_builder in TF foundation
	// ========================================================================
	if cfg.EnableSCCResources {
		if _, err := serviceaccount.NewAccount(ctx, "cai-monitoring-builder", &serviceaccount.AccountArgs{
			Project:     proj.SCCProjectID,
			AccountId:   pulumi.String("cai-monitoring-builder"),
			Description: pulumi.String("Service account for Cloud Build to provision CAI monitoring Cloud Functions"),
		}); err != nil {
			return err
		}

		caiSAMember := proj.SCCProjectID.ApplyT(func(id string) string {
			return fmt.Sprintf("serviceAccount:cai-monitoring-builder@%s.iam.gserviceaccount.com", id)
		}).(pulumi.StringOutput)
		caiRoles := []struct{ name, role string }{
			{"cai-log-writer", "roles/logging.logWriter"},
			{"cai-storage-viewer", "roles/storage.objectViewer"},
			{"cai-ar-writer", "roles/artifactregistry.writer"},
		}
		for _, r := range caiRoles {
			if _, err := projects.NewIAMMember(ctx, r.name, &projects.IAMMemberArgs{
				Project: proj.SCCProjectID,
				Role:    pulumi.String(r.role),
				Member:  caiSAMember,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
