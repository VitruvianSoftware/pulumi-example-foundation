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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadProjectsConfig(ctx)

		// 1. Stack Reference to Environment
		envStack, err := pulumi.NewStackReference(ctx, "environment", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.EnvStackName),
		})
		if err != nil {
			return err
		}

		// 2. Resolve Folder ID from stack
		folderID := envStack.GetOutput(pulumi.String("folder_id")).(pulumi.StringOutput)

		// 3. Deploy Business Unit Projects
		projects, err := deployBusinessUnitProjects(ctx, cfg, folderID)
		if err != nil {
			return err
		}

		// 4. Exports
		ctx.Export("app_project_id", projects.AppProjectID)
		ctx.Export("data_project_id", projects.DataProjectID)

		return nil
	})
}

type ProjectsConfig struct {
	Env            string
	BusinessCode   string
	BillingAccount string
	ProjectPrefix  string
	EnvStackName   string
}

func loadProjectsConfig(ctx *pulumi.Context) *ProjectsConfig {
	conf := config.New(ctx, "")
	return &ProjectsConfig{
		Env:            conf.Require("env"),
		BusinessCode:   conf.Require("business_code"),
		BillingAccount: conf.Require("billing_account"),
		ProjectPrefix:  conf.Get("project_prefix"),
		EnvStackName:   conf.Require("env_stack_name"),
	}
}
