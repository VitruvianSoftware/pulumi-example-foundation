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
	"strings"

	"github.com/VitruvianSoftware/pulumi-library/pkg/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployIAM creates the granular service accounts and assigns least-privilege
// IAM roles at every scope (org, parent, seed project, CI/CD project, billing).
// This directly mirrors the Terraform foundation's sa.tf.
func deployIAM(ctx *pulumi.Context, cfg *Config, seed *SeedProject, cicd *CICDProject) (map[string]*serviceaccount.Account, error) {
	// ========================================================================
	// 1. Create Granular Service Accounts
	// Each foundation stage gets a dedicated SA for separation of duty.
	// ========================================================================
	granularSAs := map[string]string{
		"bootstrap": "Foundation Bootstrap SA. Managed by Pulumi.",
		"org":       "Foundation Organization SA. Managed by Pulumi.",
		"env":       "Foundation Environment SA. Managed by Pulumi.",
		"net":       "Foundation Network SA. Managed by Pulumi.",
		"proj":      "Foundation Projects SA. Managed by Pulumi.",
	}

	sas := make(map[string]*serviceaccount.Account)
	for key, desc := range granularSAs {
		sa, err := serviceaccount.NewAccount(ctx, fmt.Sprintf("sa-terraform-%s", key), &serviceaccount.AccountArgs{
			Project:     seed.ProjectID,
			AccountId:   pulumi.String(fmt.Sprintf("sa-terraform-%s", key)),
			DisplayName: pulumi.String(desc),
		})
		if err != nil {
			return nil, err
		}
		sas[key] = sa
	}

	// Helper: format a service account as an IAM member string
	memberOf := func(sa *serviceaccount.Account) pulumi.StringOutput {
		return sa.Email.ApplyT(func(email string) string {
			return fmt.Sprintf("serviceAccount:%s", email)
		}).(pulumi.StringOutput)
	}

	// Helper: create a short resource name from a role
	roleID := func(role string) string {
		return strings.ReplaceAll(strings.TrimPrefix(role, "roles/"), ".", "-")
	}

	// Helper: append common roles to a role list
	commonRoles := []string{"roles/browser"}
	withCommon := func(roles ...string) []string {
		return append(roles, commonRoles...)
	}

	// ========================================================================
	// 2. Organization-level IAM
	// ========================================================================
	orgRoles := map[string][]string{
		"bootstrap": withCommon(
			"roles/resourcemanager.organizationAdmin",
			"roles/accesscontextmanager.policyAdmin",
			"roles/serviceusage.serviceUsageConsumer",
		),
		"org": withCommon(
			"roles/orgpolicy.policyAdmin",
			"roles/logging.configWriter",
			"roles/resourcemanager.organizationAdmin",
			"roles/securitycenter.notificationConfigEditor",
			"roles/resourcemanager.organizationViewer",
			"roles/accesscontextmanager.policyAdmin",
			"roles/essentialcontacts.admin",
			"roles/resourcemanager.tagAdmin",
			"roles/resourcemanager.tagUser",
			"roles/cloudasset.owner",
			"roles/securitycenter.sourcesEditor",
		),
		"env": withCommon(
			"roles/resourcemanager.tagUser",
			"roles/assuredworkloads.admin",
		),
		"net": withCommon(
			"roles/accesscontextmanager.policyAdmin",
			"roles/compute.xpnAdmin",
		),
		"proj": withCommon(
			"roles/accesscontextmanager.policyAdmin",
			"roles/resourcemanager.organizationAdmin",
			"roles/serviceusage.serviceUsageConsumer",
			"roles/cloudkms.admin",
		),
	}

	for key, roles := range orgRoles {
		for _, role := range roles {
			if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("org-iam-%s-%s", key, roleID(role)), &iam.IAMMemberArgs{
				ParentID:   pulumi.String(cfg.OrgID),
				ParentType: "organization",
				Role:       pulumi.String(role),
				Member:     memberOf(sas[key]),
			}); err != nil {
				return nil, err
			}
		}
	}

	// ========================================================================
	// 3. Parent-level IAM (folder or organization scope)
	// When deploying under a parent folder, these roles are scoped to that
	// folder. At the org root, they apply at the organization level.
	// ========================================================================
	parentRoles := map[string][]string{
		"bootstrap": {
			"roles/resourcemanager.folderAdmin",
		},
		"org": {
			"roles/resourcemanager.folderAdmin",
		},
		"env": {
			"roles/resourcemanager.folderAdmin",
		},
		"net": {
			"roles/resourcemanager.folderViewer",
			"roles/compute.networkAdmin",
			"roles/compute.securityAdmin",
			"roles/compute.orgSecurityPolicyAdmin",
			"roles/compute.orgSecurityResourceAdmin",
			"roles/dns.admin",
		},
		"proj": {
			"roles/resourcemanager.folderAdmin",
			"roles/artifactregistry.admin",
			"roles/compute.networkAdmin",
			"roles/compute.xpnAdmin",
		},
	}

	for key, roles := range parentRoles {
		for _, role := range roles {
			if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("parent-iam-%s-%s", key, roleID(role)), &iam.IAMMemberArgs{
				ParentID:   pulumi.String(cfg.ParentID),
				ParentType: cfg.ParentType,
				Role:       pulumi.String(role),
				Member:     memberOf(sas[key]),
			}); err != nil {
				return nil, err
			}
		}
	}

	// ========================================================================
	// 4. Seed Project IAM
	// Roles required to manage resources in the Seed project itself.
	// ========================================================================
	seedProjectRoles := map[string][]string{
		"bootstrap": {
			"roles/storage.admin",
			"roles/iam.serviceAccountAdmin",
			"roles/resourcemanager.projectDeleter",
			"roles/cloudkms.admin",
		},
		"org":  {"roles/storage.objectAdmin"},
		"env":  {"roles/storage.objectAdmin"},
		"net":  {"roles/storage.objectAdmin"},
		"proj": {"roles/storage.objectAdmin", "roles/storage.admin"},
	}

	for key, roles := range seedProjectRoles {
		for _, role := range roles {
			if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("seed-iam-%s-%s", key, roleID(role)), &iam.IAMMemberArgs{
				ParentID:   seed.ProjectID,
				ParentType: "project",
				Role:       pulumi.String(role),
				Member:     memberOf(sas[key]),
			}); err != nil {
				return nil, err
			}
		}
	}

	// ========================================================================
	// 5. CI/CD Project IAM
	// Roles required to manage the CI/CD pipeline infrastructure.
	// ========================================================================
	cicdProjectRoles := map[string][]string{
		"bootstrap": {
			"roles/storage.admin",
			"roles/compute.networkAdmin",
			"roles/cloudbuild.builds.editor",
			"roles/cloudbuild.workerPoolOwner",
			"roles/artifactregistry.admin",
			"roles/source.admin",
			"roles/iam.serviceAccountAdmin",
			"roles/workflows.admin",
			"roles/cloudscheduler.admin",
			"roles/resourcemanager.projectDeleter",
			"roles/dns.admin",
			"roles/iam.workloadIdentityPoolAdmin",
		},
	}

	for key, roles := range cicdProjectRoles {
		for _, role := range roles {
			if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("cicd-iam-%s-%s", key, roleID(role)), &iam.IAMMemberArgs{
				ParentID:   cicd.ProjectID,
				ParentType: "project",
				Role:       pulumi.String(role),
				Member:     memberOf(sas[key]),
			}); err != nil {
				return nil, err
			}
		}
	}

	// ========================================================================
	// 6. Billing IAM
	// All SAs need billing.user to create projects with billing association.
	// All SAs also get billing.admin for full billing management.
	// The org SA additionally gets logging.configWriter for billing log sinks.
	// Now uses the library's billing scope instead of direct API calls.
	// ========================================================================
	for key := range granularSAs {
		if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("billing-user-%s", key), &iam.IAMMemberArgs{
			ParentID:   pulumi.String(cfg.BillingAccount),
			ParentType: "billing",
			Role:       pulumi.String("roles/billing.user"),
			Member:     memberOf(sas[key]),
		}); err != nil {
			return nil, err
		}

		if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("billing-admin-%s", key), &iam.IAMMemberArgs{
			ParentID:   pulumi.String(cfg.BillingAccount),
			ParentType: "billing",
			Role:       pulumi.String("roles/billing.admin"),
			Member:     memberOf(sas[key]),
		}); err != nil {
			return nil, err
		}
	}

	// Org SA: billing logging.configWriter for audit log sinks on billing
	if _, err := iam.NewIAMMember(ctx, "billing-logging-org", &iam.IAMMemberArgs{
		ParentID:   pulumi.String(cfg.BillingAccount),
		ParentType: "billing",
		Role:       pulumi.String("roles/logging.configWriter"),
		Member:     memberOf(sas["org"]),
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 7. Org Admins Group IAM
	// Grant the org admins group essential org-level roles. This mirrors
	// the Terraform bootstrap module's org_admins_org_iam_permissions which
	// grants organizationAdmin + billing.user (and optionally
	// orgpolicy.policyAdmin) to the group_org_admins group.
	// ========================================================================
	orgAdminGroupMember := pulumi.Sprintf("group:%s", cfg.GroupOrgAdmins)
	orgAdminRoles := []string{
		"roles/resourcemanager.organizationAdmin",
		"roles/billing.user",
	}
	if cfg.OrgPolicyAdminRole {
		orgAdminRoles = append(orgAdminRoles, "roles/orgpolicy.policyAdmin")
	}
	for _, role := range orgAdminRoles {
		if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("org-admins-%s", roleID(role)), &iam.IAMMemberArgs{
			ParentID:   pulumi.String(cfg.OrgID),
			ParentType: "organization",
			Role:       pulumi.String(role),
			Member:     orgAdminGroupMember,
		}); err != nil {
			return nil, err
		}
	}

	// ========================================================================
	// 8. SA Self-Impersonation
	// Each granular SA gets roles/iam.serviceAccountTokenCreator on itself.
	// This is required for Workload Identity Federation flows where the SA
	// needs to mint its own tokens. Mirrors the Terraform foundation's
	// build_github.tf self_impersonate resource.
	// ========================================================================
	for key, sa := range sas {
		if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("sa-self-impersonate-%s", key), &iam.IAMMemberArgs{
			ParentID:   sa.Name,
			ParentType: "serviceAccount",
			Role:       pulumi.String("roles/iam.serviceAccountTokenCreator"),
			Member:     memberOf(sa),
		}); err != nil {
			return nil, err
		}
	}

	// Grant org admins group the ability to impersonate all granular SAs.
	// This allows org admins to assume any pipeline SA identity for
	// local development and troubleshooting.
	// Mirrors: org_admin_sa_impersonate_permissions + org_admin_sa_user in TF
	for key, sa := range sas {
		if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("org-admins-impersonate-%s", key), &iam.IAMMemberArgs{
			ParentID:   sa.Name,
			ParentType: "serviceAccount",
			Role:       pulumi.String("roles/iam.serviceAccountTokenCreator"),
			Member:     orgAdminGroupMember,
		}); err != nil {
			return nil, err
		}
		// roles/iam.serviceAccountUser allows the admins to "act as" the SA
		// (not just mint tokens). Required for full impersonation workflows.
		if _, err := iam.NewIAMMember(ctx, fmt.Sprintf("org-admins-sa-user-%s", key), &iam.IAMMemberArgs{
			ParentID:   sa.Name,
			ParentType: "serviceAccount",
			Role:       pulumi.String("roles/iam.serviceAccountUser"),
			Member:     orgAdminGroupMember,
		}); err != nil {
			return nil, err
		}
	}

	// Grant org admins serviceusage.serviceUsageConsumer at the parent level
	// so they can consume API quota while impersonating pipeline SAs.
	// Mirrors: org_admin_serviceusage_consumer in TF bootstrap
	if _, err := iam.NewIAMMember(ctx, "org-admins-serviceusage", &iam.IAMMemberArgs{
		ParentID:   pulumi.String(cfg.ParentID),
		ParentType: cfg.ParentType,
		Role:       pulumi.String("roles/serviceusage.serviceUsageConsumer"),
		Member:     orgAdminGroupMember,
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 9. Org Project Creators Enforcement
	// Restrict the Project Creator role at the org/parent level to ONLY the
	// granular service accounts. This is authoritative — it removes any other
	// members from this role. Mirrors the Terraform foundation's
	// org_project_creators enforcement in the bootstrap module.
	// ========================================================================
	saMembers := make(pulumi.StringArray, 0, len(granularSAs))
	for _, sa := range sas {
		saMembers = append(saMembers, memberOf(sa))
	}
	// Also include the org admins group so they retain the ability to create projects
	saMembers = append(saMembers, pulumi.Sprintf("group:%s", cfg.GroupOrgAdmins))

	if _, err := iam.NewIAMBinding(ctx, "org-project-creators", &iam.IAMBindingArgs{
		ParentID:   pulumi.String(cfg.ParentID),
		ParentType: cfg.ParentType,
		Role:       pulumi.String("roles/resourcemanager.projectCreator"),
		Members:    saMembers,
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 10. Authoritative billing.creator Enforcement
	// Restricts roles/billing.creator at the org level to ONLY the billing
	// admins group. This prevents any other principal from creating new
	// billing accounts. Mirrors: google_organization_iam_binding
	// "billing_creator" in TF bootstrap main.tf.
	// ========================================================================
	if _, err := iam.NewIAMBinding(ctx, "org-billing-creator", &iam.IAMBindingArgs{
		ParentID:   pulumi.String(cfg.OrgID),
		ParentType: "organization",
		Role:       pulumi.String("roles/billing.creator"),
		Members: pulumi.StringArray{
			pulumi.Sprintf("group:%s", cfg.GroupBillingAdmins),
		},
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 11. Remove roles/editor from bootstrap projects
	// When projects are created, the Compute Engine default SA gets the
	// Editor role. This removes all editors from both projects to follow
	// least-privilege. Mirrors the Terraform foundation's
	// bootstrap_projects_remove_editor module.
	// ========================================================================
	bootstrapProjects := map[string]pulumi.StringOutput{
		"seed": seed.ProjectID,
		"cicd": cicd.ProjectID,
	}
	for projKey, projID := range bootstrapProjects {
		if _, err := iam.NewIAMBinding(ctx, fmt.Sprintf("remove-editor-%s", projKey), &iam.IAMBindingArgs{
			ParentID:   projID,
			ParentType: "project",
			Role:       pulumi.String("roles/editor"),
			Members:    pulumi.StringArray{}, // empty = remove all editors
		}); err != nil {
			return nil, err
		}
	}

	// ========================================================================
	// 12. State Bucket IAM
	// Grant each pipeline SA + org admins group roles/storage.admin on the
	// state bucket. This is done here (not in pkg/bootstrap) because the SAs
	// don't exist yet when the seed project is created.
	// Mirrors: google_storage_bucket_iam_member "org_terraform_state_iam"
	// and "orgadmins_state_iam" in TF bootstrap main.tf.
	// ========================================================================
	bucketIAMMembers := make([]pulumi.StringInput, 0, len(sas)+1)
	for _, sa := range sas {
		bucketIAMMembers = append(bucketIAMMembers, memberOf(sa))
	}
	bucketIAMMembers = append(bucketIAMMembers, orgAdminGroupMember)

	for i, member := range bucketIAMMembers {
		if _, err := storage.NewBucketIAMMember(ctx, fmt.Sprintf("state-bucket-iam-%d", i), &storage.BucketIAMMemberArgs{
			Bucket: seed.StateBucketName,
			Role:   pulumi.String("roles/storage.admin"),
			Member: member,
		}); err != nil {
			return nil, err
		}
	}

	return sas, nil
}
