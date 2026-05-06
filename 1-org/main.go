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
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/accesscontextmanager"
)

// BootstrapOutputs holds resolved values from the 0-bootstrap StackReference.
type BootstrapOutputs struct {
	BootstrapFolderName string

	// Required groups
	GroupOrgAdmins     string
	GroupBillingAdmins string
	BillingDataUsers   string
	AuditDataUsers     string

	// Optional groups
	GCPSecurityReviewer    string
	GCPNetworkViewer       string
	GCPSCCAdmin            string
	GCPGlobalSecretsAdmin  string
	GCPKMSAdmin            string
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadOrgConfig(ctx)

		// 1. Stack Reference to Bootstrap (for cross-stage outputs)
		bootstrapRef, err := pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.BootstrapStackName),
		})
		if err != nil {
			return err
		}
		_ = bootstrapRef // Used for StackReference outputs in future enhancements

		// 2. Deploy Folders (Common, Network, Environment)
		folders, err := deployFolders(ctx, cfg)
		if err != nil {
			return err
		}

		// 3. Deploy all Org-level Projects
		projOutputs, err := deployOrgProjects(ctx, cfg, folders)
		if err != nil {
			return err
		}

		// 4. Deploy Centralized Logging (org sinks → Storage, Pub/Sub, BigQuery)
		// Must run BEFORE policies so domain-restricted sharing waits for sinks (Gap 3)
		logOutputs, err := deployCentralizedLogging(ctx, cfg, projOutputs.AuditLogsProjectID, projOutputs.BillingExportProjectID)
		if err != nil {
			return err
		}

		// 5. Deploy Organization Policies (14+ boolean + list)
		// The domain-restricted sharing policy depends on log sinks via loggingDeps
		var loggingDeps []pulumi.Resource
		if logOutputs.LastResource != nil {
			loggingDeps = append(loggingDeps, logOutputs.LastResource)
		}
		if err := deployOrgPolicies(ctx, cfg, loggingDeps); err != nil {
			return err
		}

		// 6. Deploy SCC Notifications
		if cfg.EnableSCCResources {
			if err := deploySCCNotification(ctx, cfg, projOutputs.SCCProjectID); err != nil {
				return err
			}
		}

		// 6b. Deploy CAI Monitoring infrastructure (Gap 2)
		var caiOutputs *CAIMonitoringOutputs
		if cfg.EnableSCCResources {
			caiOutputs, err = deployCAIMonitoring(ctx, cfg, projOutputs.SCCProjectID)
			if err != nil {
				return err
			}
		}

		// 7. Deploy Org-level Tags (with folder bindings)
		tagOutputs, err := deployTags(ctx, cfg, folders)
		if err != nil {
			return err
		}

		// 8. Deploy IAM bindings for groups
		if err := deployOrgIAM(ctx, cfg, projOutputs); err != nil {
			return err
		}

		// 9. Deploy Essential Contacts
		if err := deployEssentialContacts(ctx, cfg); err != nil {
			return err
		}

		// =================================================================
		// 10. Exports
		// =================================================================

		// Org/parent metadata
		ctx.Export("org_id", pulumi.String(cfg.OrgID))
		ctx.Export("parent_resource_id", pulumi.String(cfg.ParentID))
		ctx.Export("parent_resource_type", pulumi.String(cfg.ParentType))

		// Folders
		ctx.Export("common_folder_name", folders.Common.Name)
		ctx.Export("common_folder_id", folders.Common.ID())
		ctx.Export("network_folder_name", folders.Network.Name)
		ctx.Export("network_folder_id", folders.Network.ID())

		// Projects
		ctx.Export("org_audit_logs_project_id", projOutputs.AuditLogsProjectID)
		ctx.Export("org_billing_export_project_id", projOutputs.BillingExportProjectID)
		ctx.Export("scc_notifications_project_id", projOutputs.SCCProjectID)
		ctx.Export("common_kms_project_id", projOutputs.OrgKMSProjectID)
		ctx.Export("org_secrets_project_id", projOutputs.OrgSecretsProjectID)
		ctx.Export("interconnect_project_id", projOutputs.InterconnectProjectID)
		ctx.Export("interconnect_project_number", projOutputs.InterconnectProjectNumber)
		if cfg.EnableHubAndSpoke {
			ctx.Export("net_hub_project_id", projOutputs.NetHubProjectID)
			ctx.Export("net_hub_project_number", projOutputs.NetHubProjectNumber)
		}
		for env, id := range projOutputs.NetworkProjectIDs {
			ctx.Export(fmt.Sprintf("%s_network_project_id", env), id)
		}

		// Shared VPC projects grouped by environment (upstream: shared_vpc_projects)
		sharedVPCMap := pulumi.Map{}
		for env, id := range projOutputs.NetworkProjectIDs {
			sharedVPCMap[env] = id
		}
		ctx.Export("shared_vpc_projects", sharedVPCMap.ToMapOutput())

		// Logging
		ctx.Export("logs_export_storage_bucket_name", logOutputs.StorageBucketName)
		ctx.Export("logs_export_pubsub_topic", logOutputs.PubSubTopicName)
		ctx.Export("logs_export_project_logbucket_name", logOutputs.LogBucketName)
		ctx.Export("logs_export_project_linked_dataset_name", logOutputs.LinkedDatasetName)

		// SCC
		ctx.Export("scc_notification_name", pulumi.String(cfg.SCCNotificationName))

		// CAI Monitoring
		if caiOutputs != nil {
			ctx.Export("cai_monitoring_artifact_registry", caiOutputs.ArtifactRegistryName)
			ctx.Export("cai_monitoring_asset_feed", caiOutputs.AssetFeedName)
			ctx.Export("cai_monitoring_bucket", caiOutputs.BucketName)
			ctx.Export("cai_monitoring_topic", caiOutputs.TopicName)
		}

		// Tags
		ctx.Export("tags", tagOutputs)

		// Config passthrough
		ctx.Export("domains_to_allow", pulumi.ToStringArray(cfg.DomainsToAllow))

		// 9.5 Access Context Manager Policy
		var accessContextManagerPolicyID pulumi.StringOutput
		if cfg.CreateAccessContextManagerPolicy {
			accessPolicy, err := accesscontextmanager.NewAccessPolicy(ctx, "access-policy", &accesscontextmanager.AccessPolicyArgs{
				Parent: pulumi.Sprintf("organizations/%s", cfg.OrgID),
				Title:  pulumi.String("default policy"),
			})
			if err != nil {
				return err
			}
			accessContextManagerPolicyID = accessPolicy.Name
		} else {
			accessContextManagerPolicyID = pulumi.String("").ToStringOutput()
		}

		// ACM policy — mirrors TS port (available from VPC-SC module or org policy)
		ctx.Export("access_context_manager_policy_id", accessContextManagerPolicyID)
		// Billing sink names — stub for TF parity (billing sinks not deployed in base foundation)
		ctx.Export("billing_sink_names", pulumi.ToStringArray([]string{}))

		return nil
	})
}

// RetentionPolicy configures data retention on the log export storage bucket.
// When IsLocked is true, the retention policy cannot be shortened or removed.
type RetentionPolicy struct {
	IsLocked            bool
	RetentionPeriodDays int
}

// PerProjectBudget holds the budget configuration for a single project.
// Each field matches the upstream TF per-project budget variables:
//   - Amount:             e.g. org_audit_logs_budget_amount (default 1000)
//   - AlertSpentPercents: e.g. org_audit_logs_alert_spent_percents (default [1.2])
//   - AlertPubSubTopic:   e.g. org_audit_logs_alert_pubsub_topic (default null)
//   - AlertSpendBasis:    e.g. org_audit_logs_budget_alert_spend_basis (default "FORECASTED_SPEND")
type PerProjectBudget struct {
	Amount             float64
	AlertSpentPercents []float64
	AlertPubSubTopic   string
	AlertSpendBasis    string
}

// ProjectBudgetConfig mirrors the TF foundation's project_budget variable (H2).
// Each project has independent budget controls matching the upstream per-project
// variable shape with amount, alert_spent_percents, alert_pubsub_topic, and
// budget_alert_spend_basis.
type ProjectBudgetConfig struct {
	OrgAuditLogs     PerProjectBudget
	OrgBillingExport PerProjectBudget
	SCC              PerProjectBudget
	CommonKMS        PerProjectBudget
	OrgSecrets       PerProjectBudget
	Interconnect     PerProjectBudget
	NetHub           PerProjectBudget
	SharedNetwork    PerProjectBudget
}

// OrgConfig holds all configuration for the organization stage.
// This mirrors all variables from the Terraform foundation's 1-org/envs/shared/variables.tf.
type OrgConfig struct {
	// Core identifiers
	OrgID          string
	BillingAccount string
	ProjectPrefix  string
	FolderPrefix   string
	DefaultRegion  string
	Parent         string
	ParentFolder   string
	ParentID       string // Numeric ID (folder or org)
	ParentType     string // "organization" or "folder"

	// Bootstrap cross-reference
	BootstrapStackName  string
	BootstrapFolderName string // Resolved from StackReference or config

	// Governance groups (from bootstrap required_groups/optional_groups)
	GroupOrgAdmins        string // email of org admins group
	GroupBillingAdmins    string // email of billing admins group
	AuditDataUsers        string
	BillingDataUsers      string
	GCPSecurityReviewer   string
	GCPNetworkViewer      string
	GCPSCCAdmin           string
	GCPGlobalSecretsAdmin string
	GCPKMSAdmin           string
	GCPAuditViewer        string // G7: separate audit viewer group

	// Domain restrictions
	DomainsToAllow           []string
	EssentialContactsDomains []string

	// SCC
	SCCNotificationName   string
	SCCNotificationFilter string
	EnableSCCResources    bool

	// Policies
	CreateAccessContextManagerPolicy bool
	EnforceAllowedWorkerPools        bool
	EnableHubAndSpoke                bool
	AllowedWorkerPoolID              string // G1: private worker pool for cloudbuild policy

	// Cross-stage references
	NetworksSAEmail string // G8: networks pipeline SA email for hub-and-spoke IAM

	// KMS
	EnableKMSKeyUsageTracking bool

	// Projects
	RandomSuffix             bool
	ProjectDeletionPolicy    string
	FolderDeletionProtection bool
	DefaultServiceAccount    string

	// Project Budgets (H2)
	ProjectBudget *ProjectBudgetConfig

	// Logging — storage options (H6, H7, H8)
	LogExportStorageLocation        string
	LogExportStorageForceDestroy    bool
	LogExportStorageVersioning      bool
	LogExportStorageRetentionPolicy *RetentionPolicy

	// Logging — billing account sink (matches upstream enable_billing_account_sink)
	EnableBillingAccountSink bool

	// Logging — billing export
	BillingExportDatasetLocation string

	// Essential Contacts (H9)
	EssentialContactsLanguage string

	// Tags (H14)
	CreateUniqueTagKey bool
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

		// Governance groups — pulled from bootstrap outputs or overridden locally
		GroupOrgAdmins:        conf.Get("group_org_admins"),
		GroupBillingAdmins:    conf.Get("group_billing_admins"),
		AuditDataUsers:        conf.Get("audit_data_users"),
		BillingDataUsers:      conf.Get("billing_data_users"),
		GCPSecurityReviewer:   conf.Get("gcp_security_reviewer"),
		GCPNetworkViewer:      conf.Get("gcp_network_viewer"),
		GCPSCCAdmin:           conf.Get("gcp_scc_admin"),
		GCPGlobalSecretsAdmin: conf.Get("gcp_global_secrets_admin"),
		GCPKMSAdmin:           conf.Get("gcp_kms_admin"),
		GCPAuditViewer:        conf.Get("gcp_audit_viewer"),

		// SCC
		SCCNotificationName:   conf.Get("scc_notification_name"),
		SCCNotificationFilter: conf.Get("scc_notification_filter"),
		EnableSCCResources:       conf.Get("enable_scc_resources") != "false",
		EnableBillingAccountSink: conf.Get("enable_billing_account_sink") != "false",

		// Policies
		CreateAccessContextManagerPolicy: conf.Get("create_access_context_manager_policy") != "false",
		EnforceAllowedWorkerPools:        conf.Get("enforce_allowed_worker_pools") == "true",
		EnableHubAndSpoke:                conf.Get("enable_hub_and_spoke") == "true",
		AllowedWorkerPoolID:              conf.Get("allowed_worker_pool_id"),

		// Cross-stage references
		NetworksSAEmail: conf.Get("networks_sa_email"),

		// KMS
		EnableKMSKeyUsageTracking: conf.Get("enable_kms_key_usage_tracking") != "false",

		// Projects
		ProjectDeletionPolicy:    conf.Get("project_deletion_policy"),
		FolderDeletionProtection: conf.Get("folder_deletion_protection") != "false",
		DefaultServiceAccount:    conf.Get("default_service_account"),

		// Logging storage options (H6, H7)
		LogExportStorageLocation:     conf.Get("log_export_storage_location"),
		LogExportStorageForceDestroy: conf.Get("log_export_storage_force_destroy") == "true",
		LogExportStorageVersioning:   conf.Get("log_export_storage_versioning") != "false",

		// Logging billing export
		BillingExportDatasetLocation: conf.Get("billing_export_dataset_location"),

		// Essential Contacts (H9)
		EssentialContactsLanguage: conf.Get("essential_contacts_language"),

		// Tags
		CreateUniqueTagKey: conf.Get("create_unique_tag_key") == "true",

		// Bootstrap
		BootstrapFolderName: conf.Get("bootstrap_folder_name"),
	}

	// Parse structured config for ProjectBudget
	var pb ProjectBudgetConfig
	if err := conf.GetObject("project_budget", &pb); err == nil {
		c.ProjectBudget = &pb
	}

	// Random suffix defaults to true, matching upstream Terraform foundation.
	randomSuffix := conf.Get("random_suffix")
	c.RandomSuffix = randomSuffix != "false"

	// Parse comma-separated domain lists
	if domainsStr := conf.Get("domains_to_allow"); domainsStr != "" {
		c.DomainsToAllow = strings.Split(domainsStr, ",")
	}
	if contactsDomains := conf.Get("essential_contacts_domains"); contactsDomains != "" {
		c.EssentialContactsDomains = strings.Split(contactsDomains, ",")
	}

	// Apply defaults
	if c.ProjectPrefix == "" {
		c.ProjectPrefix = "prj"
	}
	if c.FolderPrefix == "" {
		c.FolderPrefix = "fldr"
	}
	if c.DefaultRegion == "" {
		c.DefaultRegion = "us-central1"
	}
	if c.SCCNotificationFilter == "" {
		c.SCCNotificationFilter = `state = "ACTIVE"`
	}
	if c.SCCNotificationName == "" {
		c.SCCNotificationName = "scc-notify"
	}
	if c.ProjectDeletionPolicy == "" {
		c.ProjectDeletionPolicy = "PREVENT"
	}
	if c.DefaultServiceAccount == "" {
		c.DefaultServiceAccount = "deprivilege"
	}
	if c.EssentialContactsLanguage == "" {
		c.EssentialContactsLanguage = "en"
	}

	// Log storage retention policy (H8)
	if retentionDays := conf.GetInt("log_export_storage_retention_days"); retentionDays > 0 {
		c.LogExportStorageRetentionPolicy = &RetentionPolicy{
			IsLocked:            conf.Get("log_export_storage_retention_locked") == "true",
			RetentionPeriodDays: retentionDays,
		}
	}

	// Storage location defaults to DefaultRegion when not set
	if c.LogExportStorageLocation == "" {
		c.LogExportStorageLocation = c.DefaultRegion
	}
	if c.BillingExportDatasetLocation == "" {
		c.BillingExportDatasetLocation = c.DefaultRegion
	}

	parentFolder := conf.Get("parent_folder")
	if parentFolder != "" {
		c.Parent = "folders/" + parentFolder
		c.ParentFolder = parentFolder
		c.ParentID = parentFolder
		c.ParentType = "folder"
	} else {
		c.Parent = "organizations/" + c.OrgID
		c.ParentID = c.OrgID
		c.ParentType = "organization"
	}

	return c
}
