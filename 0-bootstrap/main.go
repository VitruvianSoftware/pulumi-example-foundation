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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// 1. Load Configuration
		cfg := loadConfig(ctx)

		// 2. Create the Bootstrap Folder
		bootstrapFolder, err := organizations.NewFolder(ctx, "bootstrap-folder", &organizations.FolderArgs{
			DisplayName:        pulumi.String(cfg.FolderPrefix + "-bootstrap"),
			Parent:             pulumi.String(cfg.Parent),
			DeletionProtection: pulumi.Bool(true),
		}, pulumi.Protect(true))
		if err != nil {
			return err
		}

		// Convert IDOutput to StringOutput for folder ID
		folderID := bootstrapFolder.ID().ApplyT(func(id pulumi.ID) string {
			return string(id)
		}).(pulumi.StringOutput)

		// 3. Deploy the Seed Project (state storage and SA hosting)
		seed, err := deploySeedProject(ctx, cfg, folderID)
		if err != nil {
			return err
		}

		// 4. Deploy the CI/CD Project (pipeline hosting)
		cicd, err := deployCICDProject(ctx, cfg, folderID)
		if err != nil {
			return err
		}

		// 5. Deploy IAM: granular service accounts with least-privilege bindings
		sas, err := deployIAM(ctx, cfg, seed, cicd)
		if err != nil {
			return err
		}

		// 5b. Deploy CI/CD Build Infrastructure (GitHub Actions WIF by default)
		buildOutputs, err := deployGitHubActionsBuild(ctx, cfg, cicd, sas)
		if err != nil {
			return err
		}

		// 6. Exports
		ctx.Export("seed_project_id", seed.ProjectID)
		ctx.Export("cicd_project_id", cicd.ProjectID)
		ctx.Export("bootstrap_folder_id", bootstrapFolder.ID())
		ctx.Export("tf_state_bucket", seed.StateBucketName)
		ctx.Export("state_bucket_kms_key_id", seed.KMSKeyID)
		for key, sa := range sas {
			ctx.Export(key+"_sa_email", sa.Email)
		}

		// 7. Common config — composite output consumed by all downstream
		// stages via Stack References. Mirrors Terraform's common_config output.
		ctx.Export("common_config", pulumi.Map{
			"org_id":                pulumi.String(cfg.OrgID),
			"parent_folder":         pulumi.String(cfg.ParentFolder),
			"billing_account":       pulumi.String(cfg.BillingAccount),
			"default_region":        pulumi.String(cfg.DefaultRegion),
			"default_region_2":      pulumi.String(cfg.DefaultRegion2),
			"default_region_gcs":    pulumi.String(cfg.DefaultRegionGCS),
			"default_region_kms":    pulumi.String(cfg.DefaultRegionKMS),
			"project_prefix":        pulumi.String(cfg.ProjectPrefix),
			"folder_prefix":         pulumi.String(cfg.FolderPrefix),
			"parent_id":             pulumi.String(cfg.Parent),
			"bootstrap_folder_name": bootstrapFolder.Name,
		})

		// 8. Group outputs — consumed by 1-org for IAM bindings.
		ctx.Export("required_groups", pulumi.Map{
			"group_org_admins":     pulumi.String(cfg.GroupOrgAdmins),
			"group_billing_admins": pulumi.String(cfg.GroupBillingAdmins),
			"billing_data_users":   pulumi.String(cfg.BillingDataUsers),
			"audit_data_users":     pulumi.String(cfg.AuditDataUsers),
		})
		ctx.Export("optional_groups", pulumi.Map{
			"gcp_security_reviewer":    pulumi.String(cfg.GCPSecurityReviewer),
			"gcp_network_viewer":       pulumi.String(cfg.GCPNetworkViewer),
			"gcp_scc_admin":            pulumi.String(cfg.GCPSCCAdmin),
			"gcp_global_secrets_admin": pulumi.String(cfg.GCPGlobalSecretsAdmin),
			"gcp_kms_admin":            pulumi.String(cfg.GCPKMSAdmin),
		})

		// 9. CI/CD build outputs (WIF)
		if cfg.GitHubOwner != "" {
			ctx.Export("wif_pool_name", buildOutputs.WIFPoolName)
			ctx.Export("wif_provider_name", buildOutputs.WIFProviderName)
		}

		return nil
	})
}

// Config holds all configuration for the bootstrap stage, mirroring the
// Terraform foundation's variables.tf for full feature parity.
type Config struct {
	OrgID            string
	BillingAccount   string
	ProjectPrefix    string
	FolderPrefix     string
	BucketPrefix     string
	DefaultRegion    string
	DefaultRegion2   string
	DefaultRegionGCS string
	DefaultRegionKMS    string // Dedicated KMS region (default: "us"), matches upstream
	KMSKeyProtectionLevel string // "SOFTWARE" or "HSM" — matches upstream key_protection_level
	Parent           string // Full parent path: "organizations/123" or "folders/456"
	ParentFolder     string // Raw folder ID, empty if deploying at org root
	ParentType       string // "organization" or "folder"
	ParentID         string // The numeric ID for parent-level IAM bindings
	OrgPolicyAdminRole bool
	BucketForceDestroy bool
	RandomSuffix       bool // Append random hex suffix to project IDs (default: true)

	// Groups — required for org admin and billing workflows
	GroupOrgAdmins     string
	GroupBillingAdmins string
	BillingDataUsers   string
	AuditDataUsers     string

	// Optional groups — governance groups consumed by 1-org for IAM bindings.
	// These match the upstream Terraform foundation's optional_groups object.
	GCPSecurityReviewer    string
	GCPNetworkViewer       string
	GCPSCCAdmin            string
	GCPGlobalSecretsAdmin  string
	GCPKMSAdmin            string

	// GitHub Actions CI/CD — default CI/CD provider.
	// Set github_owner to enable Workload Identity Federation.
	GitHubOwner             string
	GitHubRepoBootstrap     string
	GitHubRepoOrg           string
	GitHubRepoEnv           string
	GitHubRepoNet           string
	GitHubRepoProj          string
	WIFAttributeCondition   string // Optional: override the default WIF attribute condition
}

func loadConfig(ctx *pulumi.Context) *Config {
	conf := config.New(ctx, "")
	c := &Config{
		OrgID:              conf.Require("org_id"),
		BillingAccount:     conf.Require("billing_account"),
		ProjectPrefix:      conf.Get("project_prefix"),
		FolderPrefix:       conf.Get("folder_prefix"),
		BucketPrefix:       conf.Get("bucket_prefix"),
		DefaultRegion:      conf.Get("default_region"),
		DefaultRegion2:     conf.Get("default_region_2"),
		DefaultRegionGCS:   conf.Get("default_region_gcs"),
		DefaultRegionKMS:     conf.Get("default_region_kms"),
		KMSKeyProtectionLevel: conf.Get("kms_key_protection_level"),
		ParentFolder:         conf.Get("parent_folder"),
		GroupOrgAdmins:     conf.Require("group_org_admins"),
		GroupBillingAdmins: conf.Require("group_billing_admins"),
		BillingDataUsers:   conf.Require("billing_data_users"),
		AuditDataUsers:     conf.Require("audit_data_users"),
		// Optional groups — empty string means not configured
		GCPSecurityReviewer:   conf.Get("gcp_security_reviewer"),
		GCPNetworkViewer:      conf.Get("gcp_network_viewer"),
		GCPSCCAdmin:           conf.Get("gcp_scc_admin"),
		GCPGlobalSecretsAdmin: conf.Get("gcp_global_secrets_admin"),
		GCPKMSAdmin:           conf.Get("gcp_kms_admin"),
		// GitHub Actions CI/CD
		GitHubOwner:           conf.Get("github_owner"),
		GitHubRepoBootstrap:   conf.Get("github_repo_bootstrap"),
		GitHubRepoOrg:         conf.Get("github_repo_org"),
		GitHubRepoEnv:         conf.Get("github_repo_env"),
		GitHubRepoNet:         conf.Get("github_repo_net"),
		GitHubRepoProj:        conf.Get("github_repo_proj"),
		WIFAttributeCondition: conf.Get("wif_attribute_condition"),
	}

	c.OrgPolicyAdminRole = conf.Get("org_policy_admin_role") == "true"
	c.BucketForceDestroy = conf.Get("bucket_force_destroy") == "true"

	// Random suffix defaults to true, matching upstream Terraform foundation.
	// Set to "false" to use deterministic project IDs.
	randomSuffix := conf.Get("random_suffix")
	c.RandomSuffix = randomSuffix != "false"

	// Apply defaults matching the Terraform foundation
	if c.ProjectPrefix == "" {
		c.ProjectPrefix = "prj"
	}
	if c.FolderPrefix == "" {
		c.FolderPrefix = "fldr"
	}
	if c.BucketPrefix == "" {
		c.BucketPrefix = "bkt"
	}
	if c.DefaultRegion == "" {
		c.DefaultRegion = "us-central1"
	}
	if c.DefaultRegion2 == "" {
		c.DefaultRegion2 = "us-west1"
	}
	if c.DefaultRegionGCS == "" {
		c.DefaultRegionGCS = "US"
	}
	if c.DefaultRegionKMS == "" {
		c.DefaultRegionKMS = "us"
	}
	if c.KMSKeyProtectionLevel == "" {
		c.KMSKeyProtectionLevel = "SOFTWARE"
	}

	// Determine parent: either a specific folder or the org root.
	// This controls where top-level folders and parent-level IAM are applied.
	if c.ParentFolder != "" {
		c.Parent = "folders/" + c.ParentFolder
		c.ParentType = "folder"
		c.ParentID = c.ParentFolder
	} else {
		c.Parent = "organizations/" + c.OrgID
		c.ParentType = "organization"
		c.ParentID = c.OrgID
	}

	return c
}
