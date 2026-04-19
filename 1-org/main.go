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

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadOrgConfig(ctx)

		// 1. Stack Reference to Bootstrap (for cross-stage outputs)
		_, err := pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.BootstrapStackName),
		})
		if err != nil {
			return err
		}

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
		if err := deployCentralizedLogging(ctx, cfg, projOutputs.AuditLogsProjectID, projOutputs.BillingExportProjectID); err != nil {
			return err
		}

		// 6. Deploy SCC Notifications
		if err := deploySCCNotification(ctx, cfg, projOutputs.SCCProjectID); err != nil {
			return err
		}

		// 7. Deploy Org-level Tags
		if err := deployTags(ctx, cfg); err != nil {
			return err
		}

		// 8. Exports
		ctx.Export("common_folder_id", folders.Common.ID())
		ctx.Export("network_folder_id", folders.Network.ID())
		for env, f := range folders.Environments {
			ctx.Export(fmt.Sprintf("%s_folder_id", env), f.ID())
		}
		ctx.Export("audit_logs_project_id", projOutputs.AuditLogsProjectID)
		ctx.Export("billing_export_project_id", projOutputs.BillingExportProjectID)
		ctx.Export("scc_project_id", projOutputs.SCCProjectID)
		ctx.Export("org_kms_project_id", projOutputs.OrgKMSProjectID)
		ctx.Export("org_secrets_project_id", projOutputs.OrgSecretsProjectID)
		ctx.Export("dns_hub_project_id", projOutputs.DNSHubProjectID)
		ctx.Export("interconnect_project_id", projOutputs.InterconnectProjectID)
		for env, id := range projOutputs.NetworkProjectIDs {
			ctx.Export(fmt.Sprintf("%s_network_project_id", env), id)
		}

		return nil
	})
}

// OrgConfig holds all configuration for the organization stage.
type OrgConfig struct {
	OrgID                            string
	BillingAccount                   string
	ProjectPrefix                    string
	FolderPrefix                     string
	DefaultRegion                    string
	Parent                           string
	ParentFolder                     string
	BootstrapStackName               string
	DomainsToAllow                   []string
	EssentialContactsDomains         []string
	SCCNotificationFilter            string
	CreateAccessContextManagerPolicy bool
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
		SCCNotificationFilter:            conf.Get("scc_notification_filter"),
		CreateAccessContextManagerPolicy: conf.Get("create_access_context_manager_policy") == "true",
	}

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

	parentFolder := conf.Get("parent_folder")
	if parentFolder != "" {
		c.Parent = "folders/" + parentFolder
		c.ParentFolder = parentFolder
	} else {
		c.Parent = "organizations/" + c.OrgID
	}

	return c
}
