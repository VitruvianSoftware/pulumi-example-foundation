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

	ghactions "github.com/pulumi/pulumi-github/sdk/v6/go/github"
	libcicd "github.com/VitruvianSoftware/pulumi-library/go/pkg/cicd"
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
func deployGitHubActionsBuild(ctx *pulumi.Context, cfg *Config, seed *SeedProject, cicd *CICDProject, sas map[string]*serviceaccount.Account) (*CICDBuildOutputs, error) {
	outputs := &CICDBuildOutputs{}

	// If github_owner is not set, skip WIF provisioning.
	// The user can still use key-based auth (GOOGLE_CREDENTIALS).
	if cfg.GitHubOwner == "" {
		return outputs, nil
	}

	// ========================================================================
	// 1-3. Workload Identity Pool, Provider, and Attribute Bindings
	// ========================================================================
	attributeCondition := cfg.WIFAttributeCondition
	if attributeCondition == "" {
		// Default: restrict to the configured GitHub organization/owner
		attributeCondition = fmt.Sprintf("assertion.repository_owner=='%s'", cfg.GitHubOwner)
	}

	stageRepos := map[string]string{
		"bootstrap": cfg.GitHubRepoBootstrap,
		"org":       cfg.GitHubRepoOrg,
		"env":       cfg.GitHubRepoEnv,
		"net":       cfg.GitHubRepoNet,
		"proj":      cfg.GitHubRepoProj,
	}

	saMappings := make(map[string]libcicd.SAMappingEntry)
	for key, sa := range sas {
		repo := stageRepos[key]
		var attr string
		if repo == "" || repo == "*" {
			// Wildcard: any repo under this owner
			attr = fmt.Sprintf("attribute.repository/%s", cfg.GitHubOwner)
		} else {
			// Specific repo binding
			attr = fmt.Sprintf("attribute.repository/%s/%s", cfg.GitHubOwner, repo)
		}

		saMappings[key] = libcicd.SAMappingEntry{
			SAName:    sa.Name,
			Attribute: pulumi.String(attr),
		}
	}

	githubOidc, err := libcicd.NewGitHubOIDC(ctx, "foundation-wif", &libcicd.GitHubOIDCArgs{
		ProjectID:          cicd.ProjectID,
		PoolID:             pulumi.String("foundation-pool"),
		ProviderID:         pulumi.String("foundation-gh-provider"),
		AttributeCondition: pulumi.String(attributeCondition),
		SAMapping:          saMappings,
	})
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// 4. GitHub Actions Secrets
	// Automatically provision secrets in each stage repo so the pipeline
	// templates (build/pulumi-preview.yml, build/pulumi-up.yml) work
	// out of the box with zero manual setup.
	// Mirrors: github_actions_secret "secrets" in build_github.tf.example
	//
	// Secrets created per repo:
	//   WIF_PROVIDER_NAME     — full WIF provider resource name for auth
	//   SERVICE_ACCOUNT_EMAIL — per-stage SA email for impersonation
	//   PROJECT_ID            — CI/CD project ID
	//   PULUMI_BACKEND_URL    — Backend GCS bucket URL (proj uses isolated bucket)
	//
	// Note: PULUMI_ACCESS_TOKEN is NOT provisioned here because it is a
	// Pulumi Cloud credential, not a GCP credential. Users must set it
	// manually or via their org-level GitHub secrets.
	// ========================================================================
	for key := range sas {
		repo := stageRepos[key]
		if repo == "" || repo == "*" {
			continue // No specific repo configured for this stage
		}

		// Determine the appropriate state bucket for the stage
		var backendBucket pulumi.StringOutput
		if key == "proj" {
			backendBucket = seed.ProjectsStateBucketName // Isolated state
		} else {
			backendBucket = seed.StateBucketName // Shared seed state
		}
		backendURL := backendBucket.ApplyT(func(name string) string {
			return fmt.Sprintf("gs://%s", name)
		}).(pulumi.StringOutput)

		// WIF_PROVIDER_NAME: the full provider resource name for google-github-actions/auth
		if _, err := ghactions.NewActionsSecret(ctx, fmt.Sprintf("gh-secret-%s-wif-provider", key), &ghactions.ActionsSecretArgs{
			Repository:     pulumi.String(repo),
			SecretName:     pulumi.String("WIF_PROVIDER_NAME"),
			PlaintextValue: githubOidc.Provider.Name,
		}); err != nil {
			return nil, err
		}

		// SERVICE_ACCOUNT_EMAIL: the SA this repo's pipeline should impersonate
		if _, err := ghactions.NewActionsSecret(ctx, fmt.Sprintf("gh-secret-%s-sa-email", key), &ghactions.ActionsSecretArgs{
			Repository:     pulumi.String(repo),
			SecretName:     pulumi.String("SERVICE_ACCOUNT_EMAIL"),
			PlaintextValue: sas[key].Email,
		}); err != nil {
			return nil, err
		}

		// PROJECT_ID: the CI/CD project for WIF auth
		if _, err := ghactions.NewActionsSecret(ctx, fmt.Sprintf("gh-secret-%s-project-id", key), &ghactions.ActionsSecretArgs{
			Repository:     pulumi.String(repo),
			SecretName:     pulumi.String("PROJECT_ID"),
			PlaintextValue: cicd.ProjectID,
		}); err != nil {
			return nil, err
		}

		// PULUMI_BACKEND_URL: the GCS bucket URL for self-managed state
		if _, err := ghactions.NewActionsSecret(ctx, fmt.Sprintf("gh-secret-%s-backend", key), &ghactions.ActionsSecretArgs{
			Repository:     pulumi.String(repo),
			SecretName:     pulumi.String("PULUMI_BACKEND_URL"),
			PlaintextValue: backendURL,
		}); err != nil {
			return nil, err
		}
	}

	// ========================================================================
	// 5. Outputs
	// ========================================================================
	outputs.WIFPoolName = githubOidc.Pool.Name
	outputs.WIFProviderName = githubOidc.Provider.Name

	return outputs, nil
}
