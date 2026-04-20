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
	DNSHubProjectID        pulumi.StringOutput
	InterconnectProjectID  pulumi.StringOutput
	NetHubProjectID        pulumi.StringOutput
	NetworkProjectIDs      map[string]pulumi.StringOutput
}

// createProject is a helper that creates a standardized project using the
// shared Project component from the Vitruvian Pulumi Library.
// Labels mirror the Terraform foundation's project labeling convention (D3).
// Budget and DefaultServiceAccount are optional — pass nil/empty to skip.
func createProject(ctx *pulumi.Context, name, projectID string, folderID pulumi.StringOutput, cfg *OrgConfig, apis []string, labels map[string]string, budget *project.BudgetConfig) (pulumi.StringOutput, error) {
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
		return pulumi.StringOutput{}, err
	}
	return p.Project.ProjectId, nil
}

// budgetFor returns a BudgetConfig for a given project budget amount, using
// the shared alert thresholds and spend basis from the ProjectBudgetConfig.
// Returns nil when the ProjectBudget is not configured or the amount is 0.
func budgetFor(cfg *OrgConfig, amount float64, pubsubTopic string) *project.BudgetConfig {
	if cfg.ProjectBudget == nil || amount == 0 {
		return nil
	}
	alertPercents := cfg.ProjectBudget.AlertSpentPercents
	if len(alertPercents) == 0 {
		alertPercents = []float64{1.2} // TF default
	}
	// Gap 5: default to FORECASTED_SPEND matching upstream
	spendBasis := cfg.ProjectBudget.AlertSpendBasis
	if spendBasis == "" {
		spendBasis = "FORECASTED_SPEND"
	}
	return &project.BudgetConfig{
		Amount:             amount,
		AlertSpentPercents: alertPercents,
		AlertPubSubTopic:   pubsubTopic,
		AlertSpendBasis:    spendBasis,
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
	auditLogsProjectID, err := createProject(ctx, "org-logging",
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
		budgetFor(cfg, budgetAmount(cfg, "logging"), budgetPubSub(cfg, "logging")),
	)
	if err != nil {
		return nil, err
	}

	// Billing Export — BigQuery dataset for billing data
	billingExportProjectID, err := createProject(ctx, "org-billing-export",
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
		budgetFor(cfg, budgetAmount(cfg, "billing_export"), ""),
	)
	if err != nil {
		return nil, err
	}

	// Security Command Center — SCC notifications via Pub/Sub
	sccProjectID, err := createProject(ctx, "org-scc",
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
		budgetFor(cfg, budgetAmount(cfg, "scc"), budgetPubSub(cfg, "scc")),
	)
	if err != nil {
		return nil, err
	}

	// KMS — org-level key management
	orgKMSProjectID, err := createProject(ctx, "org-kms",
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
		budgetFor(cfg, budgetAmount(cfg, "kms"), ""),
	)
	if err != nil {
		return nil, err
	}

	// Secrets — org-level secret storage
	orgSecretsProjectID, err := createProject(ctx, "org-secrets",
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
		budgetFor(cfg, budgetAmount(cfg, "secrets"), ""),
	)
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// Network Folder Projects
	// ========================================================================

	// DNS Hub — centralized DNS management
	dnsHubProjectID, err := createProject(ctx, "org-dns-hub",
		fmt.Sprintf("%s-net-dns", cfg.ProjectPrefix),
		networkFolderID, cfg,
		[]string{"dns.googleapis.com", "compute.googleapis.com", "servicenetworking.googleapis.com", "logging.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "network",
			"application_name": "org-dns-hub",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "net",
			"vpc":              "none",
		},
		budgetFor(cfg, budgetAmount(cfg, "dns_hub"), ""),
	)
	if err != nil {
		return nil, err
	}

	// Interconnect — Dedicated/Partner Interconnect connections
	interconnectProjectID, err := createProject(ctx, "org-interconnect",
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
		budgetFor(cfg, budgetAmount(cfg, "interconnect"), ""),
	)
	if err != nil {
		return nil, err
	}

	// Network Hub — conditional on hub-and-spoke architecture (D5)
	var netHubProjectID pulumi.StringOutput
	if cfg.EnableHubAndSpoke {
		netHubProjectID, err = createProject(ctx, "org-net-hub",
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
			budgetFor(cfg, budgetAmount(cfg, "net_hub"), budgetPubSub(cfg, "net_hub")),
		)
		if err != nil {
			return nil, err
		}
	}

	// Per-environment Shared VPC host projects under the Network folder
	envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}
	networkProjectIDs := make(map[string]pulumi.StringOutput)
	for env, code := range envCodes {
		// Per-env shared network budget (Gap 6)
		var networkBudget *project.BudgetConfig
		if cfg.ProjectBudget != nil && cfg.ProjectBudget.SharedNetworkBudgetAmount > 0 {
			spendBasis := cfg.ProjectBudget.AlertSpendBasis
			if spendBasis == "" {
				spendBasis = "FORECASTED_SPEND"
			}
			networkBudget = &project.BudgetConfig{
				Amount:            cfg.ProjectBudget.SharedNetworkBudgetAmount,
				AlertSpentPercents: cfg.ProjectBudget.AlertSpentPercents,
				AlertPubSubTopic:  cfg.ProjectBudget.SharedNetworkAlertPubSubTopic,
				AlertSpendBasis:   spendBasis,
			}
		}

		netProjectID, err := createProject(ctx,
			fmt.Sprintf("org-net-%s", env),
			fmt.Sprintf("%s-%s-svpc", cfg.ProjectPrefix, code),
			networkFolderID, cfg,
			[]string{
				"compute.googleapis.com",
				"dns.googleapis.com",
				"servicenetworking.googleapis.com",
				"container.googleapis.com",
				"logging.googleapis.com",
				"billingbudgets.googleapis.com",
			},
			map[string]string{
				"environment":      env,
				"application_name": fmt.Sprintf("org-net-%s", env),
				"billing_code":     "1234",
				"business_code":    "shared",
				"env_code":         code,
				"vpc":              "svpc",
			},
			networkBudget,
		)
		if err != nil {
			return nil, err
		}
		networkProjectIDs[env] = netProjectID
	}

	return &OrgProjects{
		AuditLogsProjectID:     auditLogsProjectID,
		BillingExportProjectID: billingExportProjectID,
		SCCProjectID:           sccProjectID,
		OrgKMSProjectID:        orgKMSProjectID,
		OrgSecretsProjectID:    orgSecretsProjectID,
		DNSHubProjectID:        dnsHubProjectID,
		InterconnectProjectID:  interconnectProjectID,
		NetHubProjectID:        netHubProjectID,
		NetworkProjectIDs:      networkProjectIDs,
	}, nil
}

// budgetAmount returns the configured budget amount for a given project name.
// Returns 0 when no ProjectBudget is set (which tells budgetFor to return nil).
func budgetAmount(cfg *OrgConfig, projectName string) float64 {
	if cfg.ProjectBudget == nil {
		return 0
	}
	switch projectName {
	case "logging":
		return cfg.ProjectBudget.OrgLoggingBudgetAmount
	case "billing_export":
		return cfg.ProjectBudget.OrgBillingExportAmount
	case "scc":
		return cfg.ProjectBudget.OrgSCCBudgetAmount
	case "kms":
		return cfg.ProjectBudget.OrgKMSBudgetAmount
	case "secrets":
		return cfg.ProjectBudget.OrgSecretsBudgetAmount
	case "dns_hub":
		return cfg.ProjectBudget.OrgDNSHubBudgetAmount
	case "interconnect":
		return cfg.ProjectBudget.OrgInterconnectBudgetAmount
	case "net_hub":
		return cfg.ProjectBudget.OrgNetHubBudgetAmount
	default:
		return 0
	}
}

// budgetPubSub returns the configured Pub/Sub topic for budget alerts, if any.
func budgetPubSub(cfg *OrgConfig, projectName string) string {
	if cfg.ProjectBudget == nil {
		return ""
	}
	switch projectName {
	case "logging":
		return cfg.ProjectBudget.OrgLoggingAlertPubSubTopic
	case "scc":
		return cfg.ProjectBudget.OrgSCCAlertPubSubTopic
	case "net_hub":
		return cfg.ProjectBudget.OrgNetHubAlertPubSubTopic
	default:
		return ""
	}
}
