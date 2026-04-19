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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CICDBuildOutputs holds the outputs from the CI/CD build provisioning.
type CICDBuildOutputs struct {
	// WIF outputs (only populated when github_owner is set)
	WIFPoolName     pulumi.StringOutput
	WIFProviderName pulumi.StringOutput
}

// deployGitHubActionsBuild provisions the Workload Identity Federation (WIF)
// resources in the CI/CD project. This is the Pulumi foundation's default
// CI/CD approach, equivalent to the Terraform foundation's Cloud Build default.
//
// When `github_owner` is configured, this creates:
//   - A Workload Identity Pool ("foundation-pool")
//   - A Workload Identity Pool OIDC Provider ("foundation-gh-provider")
//     configured for GitHub Actions' OIDC token issuer
//   - Per-SA attribute bindings so each GitHub repo can impersonate the
//     corresponding stage's service account
//
// This replaces the key-based GOOGLE_CREDENTIALS approach with short-lived
// tokens, following GCP security best practices.
func deployGitHubActionsBuild(ctx *pulumi.Context, cfg *Config, cicd *CICDProject, sas map[string]*serviceaccount.Account) (*CICDBuildOutputs, error) {
	outputs := &CICDBuildOutputs{}

	// If github_owner is not set, skip WIF provisioning.
	// The user can still use key-based auth (GOOGLE_CREDENTIALS).
	if cfg.GitHubOwner == "" {
		return outputs, nil
	}

	// ========================================================================
	// 1. Workload Identity Pool
	// A pool scoped to the CI/CD project that groups all GitHub-based
	// identity providers.
	// ========================================================================
	pool, err := iam.NewWorkloadIdentityPool(ctx, "foundation-wif-pool", &iam.WorkloadIdentityPoolArgs{
		Project:                cicd.ProjectID,
		WorkloadIdentityPoolId: pulumi.String("foundation-pool"),
		DisplayName:            pulumi.String("Foundation CI/CD Pool"),
		Description:            pulumi.String("Workload Identity Pool for GitHub Actions CI/CD pipeline. Managed by Pulumi."),
		Disabled:               pulumi.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// 2. Workload Identity Pool OIDC Provider
	// Configures GitHub Actions as an OIDC identity provider.
	// The attribute_condition restricts tokens to the configured GitHub owner.
	// ========================================================================
	attributeCondition := cfg.WIFAttributeCondition
	if attributeCondition == "" {
		// Default: restrict to the configured GitHub organization/owner
		attributeCondition = fmt.Sprintf("assertion.repository_owner=='%s'", cfg.GitHubOwner)
	}

	provider, err := iam.NewWorkloadIdentityPoolProvider(ctx, "foundation-wif-gh-provider", &iam.WorkloadIdentityPoolProviderArgs{
		Project:                        cicd.ProjectID,
		WorkloadIdentityPoolId:         pool.WorkloadIdentityPoolId,
		WorkloadIdentityPoolProviderId: pulumi.String("foundation-gh-provider"),
		DisplayName:                    pulumi.String("Foundation GitHub Provider"),
		Description:                    pulumi.String("GitHub Actions OIDC provider for foundation pipelines. Managed by Pulumi."),
		AttributeCondition:             pulumi.String(attributeCondition),
		AttributeMapping: pulumi.StringMap{
			"google.subject":       pulumi.String("assertion.sub"),
			"attribute.actor":      pulumi.String("assertion.actor"),
			"attribute.aud":        pulumi.String("assertion.aud"),
			"attribute.repository": pulumi.String("assertion.repository"),
		},
		Oidc: &iam.WorkloadIdentityPoolProviderOidcArgs{
			IssuerUri: pulumi.String("https://token.actions.githubusercontent.com"),
		},
	})
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// 3. SA → WIF Attribute Bindings
	// Map each granular SA to a specific GitHub repository so that only the
	// intended repo can impersonate the corresponding stage SA.
	// Uses the attribute mapping: attribute.repository/{owner}/{repo}
	// ========================================================================

	// Map stage keys to their GitHub repo config values
	stageRepos := map[string]string{
		"bootstrap": cfg.GitHubRepoBootstrap,
		"org":       cfg.GitHubRepoOrg,
		"env":       cfg.GitHubRepoEnv,
		"net":       cfg.GitHubRepoNet,
		"proj":      cfg.GitHubRepoProj,
	}

	for key, sa := range sas {
		repo := stageRepos[key]
		if repo == "" {
			// If no repo is configured for this stage, use a wildcard repo pattern
			// scoped to the owner. This allows any repo under the owner to impersonate.
			repo = "*"
		}

		var member pulumi.StringInput
		if repo == "*" {
			// Wildcard: any repo under this owner
			member = pulumi.Sprintf(
				"principalSet://iam.googleapis.com/%s/attribute.repository/%s",
				pool.Name, cfg.GitHubOwner,
			)
		} else {
			// Specific repo binding
			member = pulumi.Sprintf(
				"principalSet://iam.googleapis.com/%s/attribute.repository/%s/%s",
				pool.Name, cfg.GitHubOwner, repo,
			)
		}

		_, err := serviceaccount.NewIAMMember(ctx, fmt.Sprintf("wif-sa-binding-%s", key), &serviceaccount.IAMMemberArgs{
			ServiceAccountId: sa.Name,
			Role:             pulumi.String("roles/iam.workloadIdentityUser"),
			Member:           member,
		})
		if err != nil {
			return nil, err
		}
	}

	// ========================================================================
	// 4. Outputs
	// ========================================================================
	outputs.WIFPoolName = pool.Name
	outputs.WIFProviderName = provider.Name

	return outputs, nil
}
