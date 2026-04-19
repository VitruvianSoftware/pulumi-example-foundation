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

	"github.com/VitruvianSoftware/pulumi-library/pkg/project"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadEnvConfig(ctx)

		// 1. Stack Reference to Organization stage
		orgStack, err := pulumi.NewStackReference(ctx, "organization", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.OrgStackName),
		})
		if err != nil {
			return err
		}

		// 2. For each environment, create per-env KMS and Secrets projects
		// under the environment folders created in Stage 1.
		envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}

		for env, code := range envCodes {
			// Resolve the environment folder ID from the Org stage outputs
			folderID := orgStack.GetStringOutput(pulumi.String(fmt.Sprintf("%s_folder_id", env)))

			// KMS Project — environment-level key management
			kmsProject, err := project.NewProject(ctx, fmt.Sprintf("env-kms-%s", env), &project.ProjectArgs{
				ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-kms", cfg.ProjectPrefix, code)),
				Name:            pulumi.String(fmt.Sprintf("%s-%s-kms", cfg.ProjectPrefix, code)),
				FolderID:        folderID,
				BillingAccount:  pulumi.String(cfg.BillingAccount),
				RandomProjectID: cfg.RandomSuffix,
				ActivateApis: []string{
					"cloudkms.googleapis.com",
					"billingbudgets.googleapis.com",
				},
			})
			if err != nil {
				return err
			}

			// Secrets Project — environment-level secret storage
			secretsProject, err := project.NewProject(ctx, fmt.Sprintf("env-secrets-%s", env), &project.ProjectArgs{
				ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-secrets", cfg.ProjectPrefix, code)),
				Name:            pulumi.String(fmt.Sprintf("%s-%s-secrets", cfg.ProjectPrefix, code)),
				FolderID:        folderID,
				BillingAccount:  pulumi.String(cfg.BillingAccount),
				RandomProjectID: cfg.RandomSuffix,
				ActivateApis: []string{
					"secretmanager.googleapis.com",
					"billingbudgets.googleapis.com",
				},
			})
			if err != nil {
				return err
			}

			// Exports
			ctx.Export(fmt.Sprintf("%s_kms_project_id", env), kmsProject.Project.ProjectId)
			ctx.Export(fmt.Sprintf("%s_secrets_project_id", env), secretsProject.Project.ProjectId)
			ctx.Export(fmt.Sprintf("%s_folder_id", env), folderID)
		}

		return nil
	})
}

// EnvConfig holds configuration for the environments stage.
type EnvConfig struct {
	OrgID          string
	BillingAccount string
	ProjectPrefix  string
	OrgStackName   string
	RandomSuffix   bool
}

func loadEnvConfig(ctx *pulumi.Context) *EnvConfig {
	conf := config.New(ctx, "")
	c := &EnvConfig{
		OrgID:          conf.Require("org_id"),
		BillingAccount: conf.Require("billing_account"),
		ProjectPrefix:  conf.Get("project_prefix"),
		OrgStackName:   conf.Require("org_stack_name"),
	}
	if c.ProjectPrefix == "" {
		c.ProjectPrefix = "prj"
	}
	randomSuffix := conf.Get("random_suffix")
	c.RandomSuffix = randomSuffix != "false"
	return c
}
