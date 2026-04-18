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

		// 3. Deploy the Seed Infrastructure
		seed, err := deploySeedProject(ctx, cfg, bootstrapFolder.ID())
		if err != nil {
			return err
		}

		// 4. Deploy IAM and Service Accounts
		sas, err := deployIAM(ctx, cfg, seed.ProjectID)
		if err != nil {
			return err
		}

		// 5. Exports
		ctx.Export("seed_project_id", seed.ProjectID)
		ctx.Export("tf_state_bucket", seed.StateBucketName)
		for key, sa := range sas {
			ctx.Export(key+"_sa_email", sa.Email)
		}

		return nil
	})
}

type Config struct {
	OrgID          string
	BillingAccount string
	ProjectPrefix  string
	FolderPrefix   string
	BucketPrefix   string
	DefaultRegion  string
	Parent         string
}

func loadConfig(ctx *pulumi.Context) *Config {
	conf := config.New(ctx, "")
	c := &Config{
		OrgID:          conf.Require("org_id"),
		BillingAccount: conf.Require("billing_account"),
		ProjectPrefix:  conf.Get("project_prefix"),
		FolderPrefix:   conf.Get("folder_prefix"),
		BucketPrefix:   conf.Get("bucket_prefix"),
		DefaultRegion:  conf.Get("default_region"),
	}

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

	parentFolder := conf.Get("parent_folder")
	if parentFolder != "" {
		c.Parent = "folders/" + parentFolder
	} else {
		c.Parent = "organizations/" + c.OrgID
	}

	return c
}
