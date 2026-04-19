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
func createProject(ctx *pulumi.Context, name, projectID string, folderID pulumi.StringOutput, billingAccount string, randomSuffix bool, apis []string, labels map[string]string) (pulumi.StringOutput, error) {
	// Convert labels to Pulumi StringMap
	pulumiLabels := pulumi.StringMap{}
	for k, v := range labels {
		pulumiLabels[k] = pulumi.String(v)
	}

	p, err := project.NewProject(ctx, name, &project.ProjectArgs{
		ProjectID:       pulumi.String(projectID),
		Name:            pulumi.String(projectID),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(billingAccount),
		RandomProjectID: randomSuffix,
		ActivateApis:    apis,
		Labels:          pulumiLabels,
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}
	return p.Project.ProjectId, nil
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
		commonFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"logging.googleapis.com", "bigquery.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-logging",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
	)
	if err != nil {
		return nil, err
	}

	// Billing Export — BigQuery dataset for billing data
	billingExportProjectID, err := createProject(ctx, "org-billing-export",
		fmt.Sprintf("%s-c-billing-export", cfg.ProjectPrefix),
		commonFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"logging.googleapis.com", "bigquery.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-billing-export",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
	)
	if err != nil {
		return nil, err
	}

	// Security Command Center — SCC notifications via Pub/Sub
	sccProjectID, err := createProject(ctx, "org-scc",
		fmt.Sprintf("%s-c-scc", cfg.ProjectPrefix),
		commonFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"logging.googleapis.com", "securitycenter.googleapis.com", "pubsub.googleapis.com", "billingbudgets.googleapis.com", "cloudkms.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-scc",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
	)
	if err != nil {
		return nil, err
	}

	// KMS — org-level key management
	orgKMSProjectID, err := createProject(ctx, "org-kms",
		fmt.Sprintf("%s-c-kms", cfg.ProjectPrefix),
		commonFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"logging.googleapis.com", "cloudkms.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-kms",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
	)
	if err != nil {
		return nil, err
	}

	// Secrets — org-level secret storage
	orgSecretsProjectID, err := createProject(ctx, "org-secrets",
		fmt.Sprintf("%s-c-secrets", cfg.ProjectPrefix),
		commonFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"logging.googleapis.com", "secretmanager.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "common",
			"application_name": "org-secrets",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "c",
			"vpc":              "none",
		},
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
		networkFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"dns.googleapis.com", "compute.googleapis.com", "servicenetworking.googleapis.com", "logging.googleapis.com", "billingbudgets.googleapis.com"},
		map[string]string{
			"environment":      "network",
			"application_name": "org-dns-hub",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "net",
			"vpc":              "none",
		},
	)
	if err != nil {
		return nil, err
	}

	// Interconnect — Dedicated/Partner Interconnect connections
	interconnectProjectID, err := createProject(ctx, "org-interconnect",
		fmt.Sprintf("%s-net-interconnect", cfg.ProjectPrefix),
		networkFolderID, cfg.BillingAccount, cfg.RandomSuffix,
		[]string{"billingbudgets.googleapis.com", "compute.googleapis.com"},
		map[string]string{
			"environment":      "network",
			"application_name": "org-interconnect",
			"billing_code":     "1234",
			"business_code":    "shared",
			"env_code":         "net",
			"vpc":              "none",
		},
	)
	if err != nil {
		return nil, err
	}

	// Network Hub — conditional on hub-and-spoke architecture (D5)
	var netHubProjectID pulumi.StringOutput
	if cfg.EnableHubAndSpoke {
		netHubProjectID, err = createProject(ctx, "org-net-hub",
			fmt.Sprintf("%s-net-hub", cfg.ProjectPrefix),
			networkFolderID, cfg.BillingAccount, cfg.RandomSuffix,
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
		)
		if err != nil {
			return nil, err
		}
	}

	// Per-environment Shared VPC host projects under the Network folder
	envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}
	networkProjectIDs := make(map[string]pulumi.StringOutput)
	for env, code := range envCodes {
		netProjectID, err := createProject(ctx,
			fmt.Sprintf("org-net-%s", env),
			fmt.Sprintf("%s-%s-svpc", cfg.ProjectPrefix, code),
			networkFolderID, cfg.BillingAccount, cfg.RandomSuffix,
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
