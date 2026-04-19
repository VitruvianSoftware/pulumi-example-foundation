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
			DisplayName: pulumi.String(cfg.FolderPrefix + "-bootstrap"),
			Parent:      pulumi.String(cfg.Parent),
		})
		if err != nil {
			return err
		}

		// 3. Deploy the Seed Project (state storage and SA hosting)
		seed, err := deploySeedProject(ctx, cfg, bootstrapFolder.ID())
		if err != nil {
			return err
		}

		// 4. Deploy the CI/CD Project (pipeline hosting)
		cicd, err := deployCICDProject(ctx, cfg, bootstrapFolder.ID())
		if err != nil {
			return err
		}

		// 5. Deploy IAM: granular service accounts with least-privilege bindings
		sas, err := deployIAM(ctx, cfg, seed, cicd)
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
		ParentFolder:       conf.Get("parent_folder"),
		GroupOrgAdmins:     conf.Require("group_org_admins"),
		GroupBillingAdmins: conf.Require("group_billing_admins"),
		BillingDataUsers:   conf.Require("billing_data_users"),
		AuditDataUsers:     conf.Require("audit_data_users"),
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
