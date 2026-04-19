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
		cfg := loadProjectsConfig(ctx)

		// 1. Stack References to resolve folder and network project IDs
		orgStack, err := pulumi.NewStackReference(ctx, "organization", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.OrgStackName),
		})
		if err != nil {
			return err
		}

		// 2. Resolve the environment folder from Stage 1 outputs
		folderID := orgStack.GetStringOutput(pulumi.String(fmt.Sprintf("%s_folder_id", cfg.Env)))

		// Resolve the SVPC host project for this environment
		networkProjectID := orgStack.GetStringOutput(pulumi.String(fmt.Sprintf("%s_network_project_id", cfg.Env)))

		// 3. Create the Business Unit folder under the environment folder
		buFolder, err := organizations.NewFolder(ctx, "bu-folder", &organizations.FolderArgs{
			DisplayName: folderID.ApplyT(func(_ string) string {
				return fmt.Sprintf("%s-%s-%s", cfg.FolderPrefix, cfg.Env, cfg.BusinessCode)
			}).(pulumi.StringOutput),
			Parent: folderID.ApplyT(func(id string) string {
				return "folders/" + id
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}

		// 4. Deploy Business Unit Projects
		projects, err := deployBusinessUnitProjects(ctx, cfg, buFolder.ID(), networkProjectID)
		if err != nil {
			return err
		}

		// 5. Deploy Infra Pipeline Project (under common folder)
		commonFolderID := orgStack.GetStringOutput(pulumi.String("common_folder_id"))
		infraPipeline, err := deployInfraPipelineProject(ctx, cfg, commonFolderID)
		if err != nil {
			return err
		}

		// 6. Exports
		ctx.Export("bu_folder_id", buFolder.ID())
		ctx.Export("svpc_project_id", projects.SVPCProjectID)
		ctx.Export("floating_project_id", projects.FloatingProjectID)
		ctx.Export("peering_project_id", projects.PeeringProjectID)
		ctx.Export("infra_pipeline_project_id", infraPipeline)
		ctx.Export("network_project_id", networkProjectID)

		return nil
	})
}

// ProjectsConfig holds configuration for the projects stage.
type ProjectsConfig struct {
	Env            string
	EnvCode        string
	BusinessCode   string
	BillingAccount string
	ProjectPrefix  string
	FolderPrefix   string
	OrgStackName   string
	RandomSuffix   bool
}

func loadProjectsConfig(ctx *pulumi.Context) *ProjectsConfig {
	conf := config.New(ctx, "")
	c := &ProjectsConfig{
		Env:            conf.Require("env"),
		BusinessCode:   conf.Require("business_code"),
		BillingAccount: conf.Require("billing_account"),
		ProjectPrefix:  conf.Get("project_prefix"),
		FolderPrefix:   conf.Get("folder_prefix"),
		OrgStackName:   conf.Require("org_stack_name"),
	}
	if c.ProjectPrefix == "" {
		c.ProjectPrefix = "prj"
	}
	if c.FolderPrefix == "" {
		c.FolderPrefix = "fldr"
	}

	// Derive env code (d/n/p) from environment name
	envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}
	c.EnvCode = envCodes[c.Env]
	if c.EnvCode == "" {
		c.EnvCode = c.Env[:1]
	}

	randomSuffix := conf.Get("random_suffix")
	c.RandomSuffix = randomSuffix != "false"

	return c
}
