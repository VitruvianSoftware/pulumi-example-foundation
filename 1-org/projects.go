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
)

// OrgProjects holds outputs from all org-level project deployments.
type OrgProjects struct {
	AuditLogsProjectID     pulumi.StringOutput
	BillingExportProjectID pulumi.StringOutput
	SCCProjectID           pulumi.StringOutput
	OrgKMSProjectID        pulumi.StringOutput
	OrgSecretsProjectID    pulumi.StringOutput
	InterconnectProjectID  pulumi.StringOutput
	InterconnectProjectNumber pulumi.StringOutput // upstream: interconnect_project_number
	NetHubProjectID        pulumi.StringOutput
	NetHubProjectNumber    pulumi.StringOutput // upstream: net_hub_project_number
	NetworkProjectIDs      map[string]pulumi.StringOutput
}

// createProject is a helper that creates a standardized project using the
// shared Project component from the Vitruvian Pulumi Library.
// Labels mirror the Terraform foundation's project labeling convention (D3).
// Budget and DefaultServiceAccount are optional — pass nil/empty to skip.
// Returns both the project ID and project number for cross-stage exports.
func createProject(ctx *pulumi.Context, name, projectID string, folderID pulumi.StringOutput, cfg *OrgConfig, apis []string, labels map[string]string, budget *project.BudgetConfig) (pulumi.StringOutput, pulumi.StringOutput, error) {
	// Convert labels to Pulumi StringMap
	pulumiLabels := pulumi.StringMap{}
	for k, v := range labels {
		pulumiLabels[k] = pulumi.String(v)
	}

	p, err := project.NewProject(ctx, name, &project.ProjectArgs{
		ProjectID:             pulumi.String(projectID),
		Name:                  pulumi.String(projectID),
		FolderID:              folderID,
		BillingAccount:        pulumi.String(cfg.BillingAccount),
		RandomProjectID:       cfg.RandomSuffix,
		ActivateApis:          apis,
		Labels:                pulumiLabels,
		DeletionPolicy:        pulumi.String(cfg.ProjectDeletionPolicy),
		Budget:                budget,
		DefaultServiceAccount: cfg.DefaultServiceAccount,
	})
	if err != nil {
		return pulumi.StringOutput{}, pulumi.StringOutput{}, err
	}
	return p.Project.ProjectId, p.Project.Number, nil
}

// budgetFor converts a PerProjectBudget to a BudgetConfig for the project
// library. Returns nil when pb is nil or the amount is 0.
// Matches upstream per-project budget variable shape where each project has
// independent amount, alert_spent_percents, alert_pubsub_topic, and
// budget_alert_spend_basis fields.
func budgetFor(pb *PerProjectBudget) *project.BudgetConfig {
	if pb == nil || pb.Amount == 0 {
		return nil
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
		Amount:             pb.Amount,
		AlertSpentPercents: alertPercents,
		AlertPubSubTopic:   pb.AlertPubSubTopic,
		AlertSpendBasis:    spendBasis,
	}
}

// getProjectBudget returns the per-project budget config for the named project.
// Returns nil when ProjectBudget is not configured.
func getProjectBudget(cfg *OrgConfig, name string) *PerProjectBudget {
	if cfg.ProjectBudget == nil {
		return nil
	}
	switch name {
	case "logging":
		return &cfg.ProjectBudget.OrgAuditLogs
	case "billing_export":
		return &cfg.ProjectBudget.OrgBillingExport
	case "scc":
		return &cfg.ProjectBudget.SCC
	case "kms":
		return &cfg.ProjectBudget.CommonKMS
	case "secrets":
		return &cfg.ProjectBudget.OrgSecrets
	case "interconnect":
		return &cfg.ProjectBudget.Interconnect
	case "net_hub":
		return &cfg.ProjectBudget.NetHub
	case "shared_network":
		return &cfg.ProjectBudget.SharedNetwork
	default:
		return nil
	}
}

// deployOrgProjects creates all organization-level projects under the Common
// and Network folders. This mirrors the Terraform foundation's 1-org projects.tf.
func deployOrgProjects(ctx *pulumi.Context, cfg *OrgConfig, folders *Folders) (*OrgProjects, error) {
	// Convert IDOutput to StringOutput for folder IDs
	commonFolderID := folders.Common.ID().ApplyT(func(id pulumi.ID) string {
		return string(id)
	}).(pulumi.StringOutput)
	networkFolderID := folders.Network.ID().ApplyT(func(id pulumi.ID) string {
		return string(id)
	}).(pulumi.StringOutput)

	// ========================================================================
	// Common Folder Projects
	// ========================================================================

	// Audit Logs — centralized logging destination
	auditLogsProjectID, _, err := createProject(ctx, "org-logging",
		fmt.Sprintf("%s-c-logging", cfg.ProjectPrefix),
		commonFolderID, cfg,
		[]string{"logging.googleapis.com", "bigquery.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-logging",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
		budgetFor(getProjectBudget(cfg, "logging")),
	)
	if err != nil {
		return nil, err
	}

	// Billing Export — BigQuery dataset for billing data
	billingExportProjectID, _, err := createProject(ctx, "org-billing-export",
		fmt.Sprintf("%s-c-billing-export", cfg.ProjectPrefix),
		commonFolderID, cfg,
		[]string{"logging.googleapis.com", "bigquery.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-billing-export",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
		budgetFor(getProjectBudget(cfg, "billing_export")),
	)
	if err != nil {
		return nil, err
	}

	// Security Command Center — SCC notifications via Pub/Sub
	sccProjectID, _, err := createProject(ctx, "org-scc",
		fmt.Sprintf("%s-c-scc", cfg.ProjectPrefix),
		commonFolderID, cfg,
		[]string{"logging.googleapis.com", "securitycenter.googleapis.com", "pubsub.googleapis.com", "billingbudgets.googleapis.com", "cloudkms.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-scc",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
		budgetFor(getProjectBudget(cfg, "scc")),
	)
	if err != nil {
		return nil, err
	}

	// KMS — org-level key management
	orgKMSProjectID, _, err := createProject(ctx, "org-kms",
		fmt.Sprintf("%s-c-kms", cfg.ProjectPrefix),
		commonFolderID, cfg,
		[]string{"logging.googleapis.com", "cloudkms.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-kms",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
		budgetFor(getProjectBudget(cfg, "kms")),
	)
	if err != nil {
		return nil, err
	}

	// Secrets — org-level secret storage
	orgSecretsProjectID, _, err := createProject(ctx, "org-secrets",
		fmt.Sprintf("%s-c-secrets", cfg.ProjectPrefix),
		commonFolderID, cfg,
		[]string{"logging.googleapis.com", "secretmanager.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-secrets",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
		budgetFor(getProjectBudget(cfg, "secrets")),
	)
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// Network Folder Projects
	// ========================================================================

	// Interconnect — Dedicated/Partner Interconnect connections
	interconnectProjectID, interconnectProjectNumber, err := createProject(ctx, "org-interconnect",
		fmt.Sprintf("%s-net-interconnect", cfg.ProjectPrefix),
		networkFolderID, cfg,
		[]string{"billingbudgets.googleapis.com", "compute.googleapis.com"},
		map[string]string{
			"environment":      "network",
			"application_name": "org-interconnect",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "net",
			"vpc":              "none",
		},
		budgetFor(getProjectBudget(cfg, "interconnect")),
	)
	if err != nil {
		return nil, err
	}

	// Network Hub — conditional on hub-and-spoke architecture (D5)
	var netHubProjectID pulumi.StringOutput
	var netHubProjectNumber pulumi.StringOutput
	if cfg.EnableHubAndSpoke {
		netHubProjectID, netHubProjectNumber, err = createProject(ctx, "org-net-hub",
			fmt.Sprintf("%s-net-hub", cfg.ProjectPrefix),
			networkFolderID, cfg,
			[]string{
				"compute.googleapis.com",
				"dns.googleapis.com",
				"servicenetworking.googleapis.com",
				"logging.googleapis.com",
				"cloudresourcemanager.googleapis.com",
				"billingbudgets.googleapis.com",
			},
			map[string]string{
				"environment":      "network",
				"application_name": "org-net-hub",
				"billing_code":     "1234",
				"business_code":    "shared",
				"env_code":         "net",
				"vpc":              "svpc",
			},
			budgetFor(getProjectBudget(cfg, "net_hub")),
		)
		if err != nil {
			return nil, err
		}
	}

	// Per-environment Shared VPC host projects under the Network folder
	// Mirrors: module "environment_network" in upstream projects.tf
	envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}
	networkProjectIDs := make(map[string]pulumi.StringOutput)
	for env, code := range envCodes {
		netProjectID, _, err := createProject(ctx,
			fmt.Sprintf("org-net-%s", env),
			fmt.Sprintf("%s-%s-svpc", cfg.ProjectPrefix, code),
			networkFolderID, cfg,
			[]string{
				"compute.googleapis.com",
				"dns.googleapis.com",
				"servicenetworking.googleapis.com",
				"container.googleapis.com",
				"logging.googleapis.com",
				"cloudresourcemanager.googleapis.com",  // Gap 2: matches upstream network module
				"accesscontextmanager.googleapis.com", // Gap 2: needed for VPC Service Controls
				"billingbudgets.googleapis.com",
			},
			map[string]string{
				"environment":      env,
				"application_name": "shared-vpc-host", // upstream label value
				"billing_code":     "1234",
				"business_code":    "shared",
				"env_code":         code,
			},
			budgetFor(getProjectBudget(cfg, "shared_network")),
		)
		if err != nil {
			return nil, err
		}
		networkProjectIDs[env] = netProjectID
	}

	return &OrgProjects{
		AuditLogsProjectID:        auditLogsProjectID,
		BillingExportProjectID:    billingExportProjectID,
		SCCProjectID:              sccProjectID,
		OrgKMSProjectID:           orgKMSProjectID,
		OrgSecretsProjectID:       orgSecretsProjectID,
		InterconnectProjectID:     interconnectProjectID,
		InterconnectProjectNumber: interconnectProjectNumber,
		NetHubProjectID:           netHubProjectID,
		NetHubProjectNumber:       netHubProjectNumber,
		NetworkProjectIDs:         networkProjectIDs,
	}, nil
}


