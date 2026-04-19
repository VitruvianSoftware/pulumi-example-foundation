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

	return sas, nil
}
