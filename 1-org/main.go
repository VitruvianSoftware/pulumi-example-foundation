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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadOrgConfig(ctx)

		// 1. Deploy Folders
		folders, err := deployFolders(ctx, cfg)
		if err != nil {
			return err
		}

		// 2. Deploy Projects (Logging, Billing, etc.)
		projOutputs, err := deployOrgProjects(ctx, cfg, folders.Common.ID())
		if err != nil {
			return err
		}

		// 3. Deploy Org-level Sinks and IAM
		err = deployOrgPoliciesAndSinks(ctx, cfg, projOutputs.AuditLogsProjectID)
		if err != nil {
			return err
		}

		// 4. Exports
		ctx.Export("common_folder_id", folders.Common.ID())
		ctx.Export("network_folder_id", folders.Network.ID())
		for env, f := range folders.Environments {
			ctx.Export(fmt.Sprintf("%s_folder_id", env), f.ID())
		}
		ctx.Export("audit_logs_project_id", projOutputs.AuditLogsProjectID)
		ctx.Export("billing_export_project_id", projOutputs.BillingExportProjectID)

		return nil
	})
}

type OrgConfig struct {
	OrgID               string
	BillingAccount      string
	ProjectPrefix       string
	FolderPrefix        string
	DefaultRegion       string
	Parent              string
	BootstrapStackName  string
}

func loadOrgConfig(ctx *pulumi.Context) *OrgConfig {
	conf := config.New(ctx, "")
	c := &OrgConfig{
		OrgID:              conf.Require("org_id"),
		BillingAccount:     conf.Require("billing_account"),
		ProjectPrefix:      conf.Get("project_prefix"),
		FolderPrefix:       conf.Get("folder_prefix"),
		DefaultRegion:      conf.Get("default_region"),
		BootstrapStackName: conf.Require("bootstrap_stack_name"),
	}

	if c.ProjectPrefix == "" {
		c.ProjectPrefix = "prj"
	}
	if c.FolderPrefix == "" {
		c.FolderPrefix = "fldr"
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
