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

		// 4. Deploy Organization Policies (14+ boolean + list)
		if err := deployOrgPolicies(ctx, cfg); err != nil {
			return err
		}

		// 5. Deploy Centralized Logging (org sinks → Storage, Pub/Sub, BigQuery)
		logOutputs, err := deployCentralizedLogging(ctx, cfg, projOutputs.AuditLogsProjectID, projOutputs.BillingExportProjectID)
		if err != nil {
			return err
		}

		// 6. Deploy SCC Notifications
		if cfg.EnableSCCResources {
			if err := deploySCCNotification(ctx, cfg, projOutputs.SCCProjectID); err != nil {
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
		for env, f := range folders.Environments {
			ctx.Export(fmt.Sprintf("%s_folder_id", env), f.ID())
		}

		// Projects
		ctx.Export("audit_logs_project_id", projOutputs.AuditLogsProjectID)
		ctx.Export("billing_export_project_id", projOutputs.BillingExportProjectID)
		ctx.Export("scc_project_id", projOutputs.SCCProjectID)
		ctx.Export("org_kms_project_id", projOutputs.OrgKMSProjectID)
		ctx.Export("org_secrets_project_id", projOutputs.OrgSecretsProjectID)
		ctx.Export("dns_hub_project_id", projOutputs.DNSHubProjectID)
		ctx.Export("interconnect_project_id", projOutputs.InterconnectProjectID)
		if cfg.EnableHubAndSpoke {
			ctx.Export("net_hub_project_id", projOutputs.NetHubProjectID)
		}
		for env, id := range projOutputs.NetworkProjectIDs {
			ctx.Export(fmt.Sprintf("%s_network_project_id", env), id)
		}

		// Logging
		ctx.Export("logs_export_storage_bucket_name", logOutputs.StorageBucketName)
		ctx.Export("logs_export_pubsub_topic", logOutputs.PubSubTopicName)

		// Tags
		ctx.Export("tags", tagOutputs)

		// Config passthrough
		ctx.Export("domains_to_allow", pulumi.ToStringArray(cfg.DomainsToAllow))

		return nil
	})
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
	AuditDataUsers        string
	BillingDataUsers      string
	GCPSecurityReviewer   string
	GCPNetworkViewer      string
	GCPSCCAdmin           string
	GCPGlobalSecretsAdmin string
	GCPKMSAdmin           string

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

	// KMS
	EnableKMSKeyUsageTracking bool

	// Projects
	RandomSuffix             bool
	ProjectDeletionPolicy    string
	FolderDeletionProtection bool
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
		AuditDataUsers:        conf.Get("audit_data_users"),
		BillingDataUsers:      conf.Get("billing_data_users"),
		GCPSecurityReviewer:   conf.Get("gcp_security_reviewer"),
		GCPNetworkViewer:      conf.Get("gcp_network_viewer"),
		GCPSCCAdmin:           conf.Get("gcp_scc_admin"),
		GCPGlobalSecretsAdmin: conf.Get("gcp_global_secrets_admin"),
		GCPKMSAdmin:           conf.Get("gcp_kms_admin"),

		// SCC
		SCCNotificationName:   conf.Get("scc_notification_name"),
		SCCNotificationFilter: conf.Get("scc_notification_filter"),
		EnableSCCResources:    conf.Get("enable_scc_resources") != "false",

		// Policies
		CreateAccessContextManagerPolicy: conf.Get("create_access_context_manager_policy") != "false",
		EnforceAllowedWorkerPools:        conf.Get("enforce_allowed_worker_pools") == "true",
		EnableHubAndSpoke:                conf.Get("enable_hub_and_spoke") == "true",

		// KMS
		EnableKMSKeyUsageTracking: conf.Get("enable_kms_key_usage_tracking") != "false",

		// Projects
		ProjectDeletionPolicy:    conf.Get("project_deletion_policy"),
		FolderDeletionProtection: conf.Get("folder_deletion_protection") != "false",

		// Bootstrap
		BootstrapFolderName: conf.Get("bootstrap_folder_name"),
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
		c.SCCNotificationFilter = "state=\"ACTIVE\""
	}
	if c.SCCNotificationName == "" {
		c.SCCNotificationName = "scc-notify"
	}
	if c.ProjectDeletionPolicy == "" {
		c.ProjectDeletionPolicy = "PREVENT"
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
