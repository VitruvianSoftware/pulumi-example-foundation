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

	"github.com/VitruvianSoftware/pulumi-library/pkg/policy"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/accesscontextmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployOrgPolicies enforces organization-level security policies.
// This mirrors the Terraform foundation's org_policy.tf, applying 14+ boolean
// constraints and list constraints that form the security baseline.
func deployOrgPolicies(ctx *pulumi.Context, cfg *OrgConfig) error {
	parentID := "organizations/" + cfg.OrgID
	if cfg.ParentFolder != "" {
		parentID = "folders/" + cfg.ParentFolder
	}

	// ========================================================================
	// Boolean Organization Policies
	// These are enforced across the entire org/folder hierarchy, preventing
	// common security misconfigurations at the infrastructure level.
	// ========================================================================
	booleanPolicies := []string{
		// Compute Engine hardening
		"compute.disableNestedVirtualization",
		"compute.disableSerialPortAccess",
		"compute.skipDefaultNetworkCreation",
		"compute.restrictXpnProjectLienRemoval",
		"compute.disableVpcExternalIpv6",
		"compute.setNewProjectDefaultToZonalDNSOnly",
		"compute.requireOsLogin",
		// Cloud SQL hardening
		"sql.restrictPublicIp",
		"sql.restrictAuthorizedNetworks",
		// IAM hardening — prevent SA key sprawl
		"iam.disableServiceAccountKeyCreation",
		"iam.automaticIamGrantsForDefaultServiceAccounts",
		"iam.disableServiceAccountKeyUpload",
		// Storage hardening
		"storage.uniformBucketLevelAccess",
		"storage.publicAccessPrevention",
	}

	for _, constraint := range booleanPolicies {
		if _, err := policy.NewOrgPolicy(ctx, fmt.Sprintf("policy-%s", constraint), &policy.OrgPolicyArgs{
			ParentID:   pulumi.String(parentID),
			Constraint: pulumi.String(fmt.Sprintf("constraints/%s", constraint)),
			Boolean:    pulumi.Bool(true),
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// List Organization Policies
	// ========================================================================

	// Deny all VM external IP access — enforce private networking
	if _, err := policy.NewOrgPolicy(ctx, "policy-vm-external-ip", &policy.OrgPolicyArgs{
		ParentID:   pulumi.String(parentID),
		Constraint: pulumi.String("constraints/compute.vmExternalIpAccess"),
		DenyAll:    pulumi.Bool(true),
	}); err != nil {
		return err
	}

	// Restrict protocol forwarding to internal only
	if _, err := policy.NewOrgPolicy(ctx, "policy-restrict-protocol-forwarding", &policy.OrgPolicyArgs{
		ParentID:    pulumi.String(parentID),
		Constraint:  pulumi.String("constraints/compute.restrictProtocolForwardingCreationForTypes"),
		AllowValues: pulumi.StringArray{pulumi.String("INTERNAL")},
	}); err != nil {
		return err
	}

	// Domain-restricted sharing — only allow specified domains
	if len(cfg.DomainsToAllow) > 0 {
		domainValues := make(pulumi.StringArray, len(cfg.DomainsToAllow))
		for i, d := range cfg.DomainsToAllow {
			domainValues[i] = pulumi.String(d)
		}
		if _, err := policy.NewOrgPolicy(ctx, "policy-domain-restricted-sharing", &policy.OrgPolicyArgs{
			ParentID:    pulumi.String(parentID),
			Constraint:  pulumi.String("constraints/iam.allowedPolicyMemberDomains"),
			AllowValues: domainValues,
		}); err != nil {
			return err
		}
	}

	// Essential Contacts domain restriction
	if len(cfg.EssentialContactsDomains) > 0 {
		contactDomains := make(pulumi.StringArray, len(cfg.EssentialContactsDomains))
		for i, d := range cfg.EssentialContactsDomains {
			// Ensure domain starts with "@"
			if d[0] != '@' {
				d = "@" + d
			}
			contactDomains[i] = pulumi.String(d)
		}
		if _, err := policy.NewOrgPolicy(ctx, "policy-essential-contacts-domains", &policy.OrgPolicyArgs{
			ParentID:    pulumi.String(parentID),
			Constraint:  pulumi.String("constraints/essentialcontacts.allowedContactDomains"),
			AllowValues: contactDomains,
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// Allowed Worker Pools (G1)
	// Restricts Cloud Build to only use the specified private worker pool.
	// Mirrors: module "allowed_worker_pools" in org_policy.tf
	// ========================================================================
	if cfg.EnforceAllowedWorkerPools && cfg.AllowedWorkerPoolID != "" {
		if _, err := policy.NewOrgPolicy(ctx, "policy-allowed-worker-pools", &policy.OrgPolicyArgs{
			ParentID:    pulumi.String(parentID),
			Constraint:  pulumi.String("constraints/cloudbuild.allowedWorkerPools"),
			AllowValues: pulumi.StringArray{pulumi.String(cfg.AllowedWorkerPoolID)},
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// Access Context Manager
	// ========================================================================
	if cfg.CreateAccessContextManagerPolicy {
		if _, err := accesscontextmanager.NewAccessPolicy(ctx, "access-policy", &accesscontextmanager.AccessPolicyArgs{
			Parent: pulumi.String("organizations/" + cfg.OrgID),
			Title:  pulumi.String("default policy"),
		}); err != nil {
			return err
		}
	}

	return nil
}
