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

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// 1. Stack References — resolve outputs from 0-bootstrap and 1-org
		// Mirrors upstream remote.tf which reads common_config from bootstrap
		// and tags from org.
		cfg := loadEnvConfig(ctx)

		bootstrapStack, err := pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.BootstrapStackName),
		})
		if err != nil {
			return err
		}

		// Resolve common_config from bootstrap — upstream reads:
		// org_id, parent_id, billing_account, project_prefix, folder_prefix
		commonConfig := bootstrapStack.GetOutput(pulumi.String("common_config"))
		applyCommonConfig(cfg, commonConfig)

		orgStack, err := pulumi.NewStackReference(ctx, "organization", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.OrgStackName),
		})
		if err != nil {
			return err
		}

		// Resolve tag values from the 1-org stage for folder tag bindings.
		// The 1-org stage exports a "tags" map with keys like "environment_development".
		tagsOutput := orgStack.GetOutput(pulumi.String("tags"))

		// 2. Deploy per-environment baselines
		envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}

		for env, code := range envCodes {
			outputs, err := deployEnvBaseline(ctx, cfg, env, code, tagsOutput)
			if err != nil {
				return err
			}

			// Exports — matches upstream per-env outputs
			ctx.Export(fmt.Sprintf("%s_env_folder", env), outputs.FolderName)
			ctx.Export(fmt.Sprintf("%s_env_folder_id", env), outputs.FolderID)
			ctx.Export(fmt.Sprintf("%s_env_kms_project_id", env), outputs.KMSProjectID)
			ctx.Export(fmt.Sprintf("%s_env_kms_project_number", env), outputs.KMSProjectNumber)
			ctx.Export(fmt.Sprintf("%s_env_secrets_project_id", env), outputs.SecretsProjectID)

			// Assured Workload outputs (only when enabled)
			if cfg.AssuredWorkload.Enabled {
				ctx.Export(fmt.Sprintf("%s_assured_workload_id", env), outputs.AssuredWorkloadID)
			}
		}

		return nil
	})
}

// PerProjectBudget holds the budget configuration for a single project.
type PerProjectBudget struct {
	Amount             float64
	AlertSpentPercents []float64
	AlertPubSubTopic   string
	AlertSpendBasis    string
}

// EnvProjectBudgetConfig mirrors the upstream project_budget variable for 2-environments.
type EnvProjectBudgetConfig struct {
	KMS    PerProjectBudget
	Secret PerProjectBudget
}

// AssuredWorkloadConfig mirrors the upstream assured_workload_configuration variable.
type AssuredWorkloadConfig struct {
	Enabled          bool
	Location         string
	DisplayName      string
	ComplianceRegime string
	ResourceType     string
}

// EnvConfig holds all configuration for the environments stage.
// This mirrors all variables from the upstream Terraform foundation's
// 2-environments/modules/env_baseline/variables.tf and remote.tf.
//
// Core identifiers (org_id, billing_account, project_prefix, folder_prefix,
// parent_id) are resolved from the bootstrap stage's common_config output
// via stack reference, matching upstream remote.tf. They can also be
// overridden via direct config for testing or standalone deployments.
type EnvConfig struct {
	// Core identifiers (from bootstrap common_config via stack ref or direct config)
	OrgID          string
	BillingAccount string
	ProjectPrefix  string
	FolderPrefix   string
	Parent         string // "organizations/<id>" or "folders/<id>"

	// Stack references
	BootstrapStackName string
	OrgStackName       string

	// Project settings
	RandomSuffix             bool
	ProjectDeletionPolicy    string
	FolderDeletionProtection bool
	DefaultServiceAccount    string

	// Budgets
	ProjectBudget *EnvProjectBudgetConfig

	// Assured Workloads
	AssuredWorkload AssuredWorkloadConfig
}

func loadEnvConfig(ctx *pulumi.Context) *EnvConfig {
	conf := config.New(ctx, "")
	c := &EnvConfig{
		// Core identifiers — these can be overridden via direct config,
		// but are normally resolved from bootstrap's common_config output.
		OrgID:          conf.Get("org_id"),
		BillingAccount: conf.Get("billing_account"),
		ProjectPrefix:  conf.Get("project_prefix"),
		FolderPrefix:   conf.Get("folder_prefix"),

		// Stack references
		BootstrapStackName: conf.Require("bootstrap_stack_name"),
		OrgStackName:       conf.Require("org_stack_name"),

		// Project settings
		ProjectDeletionPolicy: conf.Get("project_deletion_policy"),
		DefaultServiceAccount: conf.Get("default_service_account"),
	}

	// Boolean config with defaults
	c.FolderDeletionProtection = conf.Get("folder_deletion_protection") != "false"
	randomSuffix := conf.Get("random_suffix")
	c.RandomSuffix = randomSuffix != "false"

	// Parse structured config for ProjectBudget
	var pb EnvProjectBudgetConfig
	if err := conf.GetObject("project_budget", &pb); err == nil {
		c.ProjectBudget = &pb
	} else {
		// Default budget values matching upstream tf variables.tf
		c.ProjectBudget = &EnvProjectBudgetConfig{
			KMS: PerProjectBudget{
				Amount:             1000,
				AlertSpentPercents: []float64{1.2},
				AlertSpendBasis:    "FORECASTED_SPEND",
			},
			Secret: PerProjectBudget{
				Amount:             1000,
				AlertSpentPercents: []float64{1.2},
				AlertSpendBasis:    "FORECASTED_SPEND",
			},
		}
	}

	// Parse Assured Workload configuration
	c.AssuredWorkload = AssuredWorkloadConfig{
		Enabled:          conf.Get("assured_workload_enabled") == "true",
		Location:         conf.Get("assured_workload_location"),
		DisplayName:      conf.Get("assured_workload_display_name"),
		ComplianceRegime: conf.Get("assured_workload_compliance_regime"),
		ResourceType:     conf.Get("assured_workload_resource_type"),
	}

	// Apply defaults matching the upstream Terraform foundation
	if c.ProjectPrefix == "" {
		c.ProjectPrefix = "prj"
	}
	if c.FolderPrefix == "" {
		c.FolderPrefix = "fldr"
	}
	if c.ProjectDeletionPolicy == "" {
		c.ProjectDeletionPolicy = "PREVENT"
	}
	if c.DefaultServiceAccount == "" {
		c.DefaultServiceAccount = "deprivilege"
	}
	if c.AssuredWorkload.Location == "" {
		c.AssuredWorkload.Location = "us-central1"
	}
	if c.AssuredWorkload.DisplayName == "" {
		c.AssuredWorkload.DisplayName = "FEDRAMP-MODERATE"
	}
	if c.AssuredWorkload.ComplianceRegime == "" {
		c.AssuredWorkload.ComplianceRegime = "FEDRAMP_MODERATE"
	}
	if c.AssuredWorkload.ResourceType == "" {
		c.AssuredWorkload.ResourceType = "CONSUMER_FOLDER"
	}

	// Determine parent path — may be overridden by common_config later
	parentFolder := conf.Get("parent_folder")
	if parentFolder != "" {
		c.Parent = "folders/" + parentFolder
	} else if c.OrgID != "" {
		c.Parent = "organizations/" + c.OrgID
	}
	// If neither is set, Parent will be resolved from common_config

	return c
}

// applyCommonConfig merges bootstrap's common_config output into EnvConfig.
// Only fills in fields that weren't explicitly set via direct config,
// matching the upstream pattern where remote.tf locals override variables.
func applyCommonConfig(cfg *EnvConfig, commonConfig pulumi.Output) {
	commonConfig.ApplyT(func(v interface{}) string {
		m, ok := v.(map[string]interface{})
		if !ok {
			return ""
		}
		if cfg.OrgID == "" {
			if val, exists := m["org_id"]; exists {
				cfg.OrgID = val.(string)
			}
		}
		if cfg.BillingAccount == "" {
			if val, exists := m["billing_account"]; exists {
				cfg.BillingAccount = val.(string)
			}
		}
		if cfg.ProjectPrefix == "" || cfg.ProjectPrefix == "prj" {
			if val, exists := m["project_prefix"]; exists && val.(string) != "" {
				cfg.ProjectPrefix = val.(string)
			}
		}
		if cfg.FolderPrefix == "" || cfg.FolderPrefix == "fldr" {
			if val, exists := m["folder_prefix"]; exists && val.(string) != "" {
				cfg.FolderPrefix = val.(string)
			}
		}
		if cfg.Parent == "" {
			if val, exists := m["parent_id"]; exists && val.(string) != "" {
				cfg.Parent = val.(string)
			} else if cfg.OrgID != "" {
				cfg.Parent = "organizations/" + cfg.OrgID
			}
		}
		return ""
	})
}
