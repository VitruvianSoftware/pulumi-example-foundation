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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/assuredworkloads"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/tags"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// EnvBaselineOutputs holds all outputs from a single environment's baseline deployment.
type EnvBaselineOutputs struct {
	FolderName       pulumi.StringOutput
	FolderID         pulumi.IDOutput
	KMSProjectID     pulumi.StringOutput
	KMSProjectNumber pulumi.StringOutput
	SecretsProjectID pulumi.StringOutput
	AssuredWorkloadID pulumi.StringOutput
	AssuredWorkloadResources assuredworkloads.WorkloadResourceArrayOutput
}

// deployEnvBaseline creates all per-environment resources for a single environment.
// This mirrors the upstream Terraform foundation's 2-environments/modules/env_baseline.
//
// Resources created per environment:
//   - Environment folder ({folder_prefix}-{env}) under parent
//   - Folder tag binding (environment_{env} tag from 1-org)
//   - KMS project ({project_prefix}-{code}-kms) with 8 labels, 3 APIs, budget
//   - Secrets project ({project_prefix}-{code}-secrets) with 8 labels, 2 APIs, budget
//   - (Optional) Assured Workload for FedRAMP compliance
func deployEnvBaseline(ctx *pulumi.Context, cfg *EnvConfig, env, envCode string, tagsOutput pulumi.Output) (*EnvBaselineOutputs, error) {
	outputs := &EnvBaselineOutputs{}

	// ========================================================================
	// 1. Environment Folder
	// Mirrors: folders.tf — google_folder.env
	// ========================================================================
	var folderOpts []pulumi.ResourceOption
	if cfg.FolderDeletionProtection {
		folderOpts = append(folderOpts, pulumi.Protect(true))
	}

	envFolder, err := organizations.NewFolder(ctx, fmt.Sprintf("env-folder-%s", env), &organizations.FolderArgs{
		DisplayName:        pulumi.String(fmt.Sprintf("%s-%s", cfg.FolderPrefix, env)),
		Parent:             pulumi.String(cfg.Parent),
		DeletionProtection: pulumi.Bool(cfg.FolderDeletionProtection),
	}, folderOpts...)
	if err != nil {
		return nil, err
	}
	outputs.FolderName = envFolder.Name
	outputs.FolderID = envFolder.ID()

	// Convert IDOutput to StringOutput for folder ID usage in project creation
	folderID := envFolder.ID().ApplyT(func(id pulumi.ID) string {
		return string(id)
	}).(pulumi.StringOutput)

	// ========================================================================
	// 2. Folder Tag Binding
	// Mirrors: folders.tf — google_tags_tag_binding.folder_env
	// Binds the environment_{env} tag value from the 1-org stage to this folder.
	// ========================================================================
	if tagsOutput != nil {
		tagValueID := tagsOutput.ApplyT(func(v interface{}) string {
			if m, ok := v.(map[string]interface{}); ok {
				key := fmt.Sprintf("environment_%s", env)
				if val, exists := m[key]; exists {
					return val.(string)
				}
			}
			return ""
		}).(pulumi.StringOutput)

		_, err := tags.NewTagBinding(ctx, fmt.Sprintf("tag-binding-%s", env), &tags.TagBindingArgs{
			Parent: envFolder.Name.ApplyT(func(name string) string {
				return fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s", name)
			}).(pulumi.StringOutput),
			TagValue: tagValueID,
		}, pulumi.DependsOn([]pulumi.Resource{envFolder}))
		if err != nil {
			return nil, err
		}
	}

	// ========================================================================
	// 3. KMS Project
	// Mirrors: kms.tf — module "env_kms"
	//
	// Upstream TF parity notes:
	// - TF uses time_sleep.wait_60_seconds (destroy_duration=60s) between folder
	//   creation and project creation. In Pulumi, DependsOn on envFolder provides
	//   create-time ordering; the 60s delay is destroy-time only and is not
	//   replicated since Pulumi handles destroy ordering via its dependency graph.
	// - TF sets disable_services_on_destroy=false. Pulumi's project component
	//   does not disable services on destroy by default, which matches.
	// ========================================================================
	kmsProject, err := project.NewProject(ctx, fmt.Sprintf("env-kms-%s", env), &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-%s-kms", cfg.ProjectPrefix, envCode)),
		Name:           pulumi.String(fmt.Sprintf("%s-%s-kms", cfg.ProjectPrefix, envCode)),
		FolderID:       folderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		RandomProjectID:       cfg.RandomSuffix,
		DeletionPolicy:        pulumi.String(cfg.ProjectDeletionPolicy),
		DefaultServiceAccount: cfg.DefaultServiceAccount,
		ActivateApis: []string{
			"logging.googleapis.com",
			"cloudkms.googleapis.com",
			"billingbudgets.googleapis.com",
		},
		Labels: pulumi.StringMap{
			"environment":       pulumi.String(env),
			"application_name":  pulumi.String("env-kms"),
			"billing_code":      pulumi.String("1234"),
			"primary_contact":   pulumi.String("example1"),
			"secondary_contact": pulumi.String("example2"),
			"business_code":     pulumi.String("shared"),
			"env_code":          pulumi.String(envCode),
			"vpc":               pulumi.String("none"),
		},
		Budget: budgetFor(getEnvProjectBudget(cfg, "kms")),
	}, pulumi.DependsOn([]pulumi.Resource{envFolder}))
	if err != nil {
		return nil, err
	}
	outputs.KMSProjectID = kmsProject.Project.ProjectId
	outputs.KMSProjectNumber = kmsProject.Project.Number

	// ========================================================================
	// 4. Secrets Project
	// Mirrors: secrets.tf — module "env_secrets"
	//
	// Same upstream TF parity notes as KMS project above:
	// - destroy_duration=60s from time_sleep not replicated (Pulumi graph ordering)
	// - disable_services_on_destroy=false matches Pulumi default
	// ========================================================================
	secretsProject, err := project.NewProject(ctx, fmt.Sprintf("env-secrets-%s", env), &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-%s-secrets", cfg.ProjectPrefix, envCode)),
		Name:           pulumi.String(fmt.Sprintf("%s-%s-secrets", cfg.ProjectPrefix, envCode)),
		FolderID:       folderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		RandomProjectID:       cfg.RandomSuffix,
		DeletionPolicy:        pulumi.String(cfg.ProjectDeletionPolicy),
		DefaultServiceAccount: cfg.DefaultServiceAccount,
		ActivateApis: []string{
			"logging.googleapis.com",
			"secretmanager.googleapis.com",
		},
		Labels: pulumi.StringMap{
			"environment":       pulumi.String(env),
			"application_name":  pulumi.String("env-secrets"),
			"billing_code":      pulumi.String("1234"),
			"primary_contact":   pulumi.String("example1"),
			"secondary_contact": pulumi.String("example2"),
			"business_code":     pulumi.String("shared"),
			"env_code":          pulumi.String(envCode),
			"vpc":               pulumi.String("none"),
		},
		Budget: budgetFor(getEnvProjectBudget(cfg, "secret")),
	}, pulumi.DependsOn([]pulumi.Resource{envFolder}))
	if err != nil {
		return nil, err
	}
	outputs.SecretsProjectID = secretsProject.Project.ProjectId

	// ========================================================================
	// 5. Assured Workload (optional — FedRAMP compliance)
	// Mirrors: assured_workload.tf — google_assured_workloads_workload.workload
	// ========================================================================
	if cfg.AssuredWorkload.Enabled {
		workload, err := assuredworkloads.NewWorkload(ctx, fmt.Sprintf("assured-workload-%s", env), &assuredworkloads.WorkloadArgs{
			Organization:                pulumi.String(cfg.OrgID),
			BillingAccount:              pulumi.String(fmt.Sprintf("billingAccounts/%s", cfg.BillingAccount)),
			ProvisionedResourcesParent: envFolder.ID().ApplyT(func(id pulumi.ID) string {
				return string(id)
			}).(pulumi.StringOutput),
			ComplianceRegime: pulumi.String(cfg.AssuredWorkload.ComplianceRegime),
			DisplayName:      pulumi.String(cfg.AssuredWorkload.DisplayName),
			Location:         pulumi.String(cfg.AssuredWorkload.Location),
			ResourceSettings: assuredworkloads.WorkloadResourceSettingArray{
				&assuredworkloads.WorkloadResourceSettingArgs{
					ResourceType: pulumi.String(cfg.AssuredWorkload.ResourceType),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{envFolder}))
		if err != nil {
			return nil, err
		}
		outputs.AssuredWorkloadID = workload.ID().ApplyT(func(id pulumi.ID) string {
			return string(id)
		}).(pulumi.StringOutput)
		outputs.AssuredWorkloadResources = workload.Resources
	}

	return outputs, nil
}

func budgetFor(pb *PerProjectBudget) *project.BudgetConfig {
	if pb == nil {
		return nil
	}
	amount := pb.Amount
	if amount == 0 {
		amount = 1000 // upstream default
	}
	alertPercents := pb.AlertSpentPercents
	if len(alertPercents) == 0 {
		alertPercents = []float64{1.2} // upstream default
	}
	spendBasis := pb.AlertSpendBasis
	if spendBasis == "" {
		spendBasis = "FORECASTED_SPEND" // upstream default
	}
	return &project.BudgetConfig{
		Amount:             amount,
		AlertSpentPercents: alertPercents,
		AlertPubSubTopic:   pb.AlertPubSubTopic,
		AlertSpendBasis:    spendBasis,
	}
}

// getEnvProjectBudget returns the per-project budget config for the named project type.
func getEnvProjectBudget(cfg *EnvConfig, projectType string) *PerProjectBudget {
	if cfg.ProjectBudget == nil {
		return nil
	}
	switch projectType {
	case "shared_network":
		return &cfg.ProjectBudget.SharedNetwork
	case "kms":
		return &cfg.ProjectBudget.KMS
	case "secret":
		return &cfg.ProjectBudget.Secret
	default:
		return nil
	}
}
