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
	"github.com/VitruvianSoftware/pulumi-library/pkg/logging"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/bigquery"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// LoggingOutputs holds resource references for downstream exports.
type LoggingOutputs struct {
	StorageBucketName pulumi.StringOutput
	PubSubTopicName   pulumi.StringOutput
	LogBucketName     pulumi.StringOutput // upstream: logs_export_project_logbucket_name
	LinkedDatasetName pulumi.StringOutput // upstream: logs_export_project_linked_dataset_name
	// LastResource is the last resource created by the logging deployment,
	// used for dependency ordering (e.g., policies must wait for sinks).
	LastResource pulumi.Resource
}

// deployCentralizedLogging creates the centralized logging infrastructure
// by delegating to the pkg/logging.CentralizedLogging library component.
//
// This mirrors the Terraform foundation's log_sinks.tf which calls
// module "logs_export" (source = "../../modules/centralized-logging").
//
// The library component handles:
//   - Dynamic org/folder sink routing based on cfg.ParentType
//   - Destination resource creation (storage, pubsub, log bucket)
//   - IAM grants for each sink writer identity
//   - Billing account sinks (when enabled)
//   - Internal project sink (prevents the audit project from being a blind spot)
//
// The log filter covers all audit and network logs, matching the upstream.
func deployCentralizedLogging(ctx *pulumi.Context, cfg *OrgConfig, auditProjectID, billingExportProjectID pulumi.StringOutput) (*LoggingOutputs, error) {
	// Comprehensive log filter covering all audit and network logs
	// Matches upstream log_sinks.tf local.logs_filter
	logFilter := `logName: /logs/cloudaudit.googleapis.com%2Factivity OR
logName: /logs/cloudaudit.googleapis.com%2Fsystem_event OR
logName: /logs/cloudaudit.googleapis.com%2Fdata_access OR
logName: /logs/cloudaudit.googleapis.com%2Faccess_transparency OR
logName: /logs/cloudaudit.googleapis.com%2Fpolicy OR
logName: /logs/compute.googleapis.com%2Fvpc_flows OR
logName: /logs/compute.googleapis.com%2Ffirewall OR
logName: /logs/dns.googleapis.com%2Fdns_queries`

	// Build storage retention config
	var retentionEnabled bool
	var retentionLocked bool
	var retentionDays int
	if cfg.LogExportStorageRetentionPolicy != nil {
		retentionEnabled = true
		retentionLocked = cfg.LogExportStorageRetentionPolicy.IsLocked
		retentionDays = cfg.LogExportStorageRetentionPolicy.RetentionPeriodDays
	}

	// Delegate to the library component — this replaces ~250 lines of
	// inline sink orchestration with a single, tested, reusable call.
	cl, err := logging.NewCentralizedLogging(ctx, "logs-export", &logging.CentralizedLoggingArgs{
		Resources:                   map[string]string{"resource": cfg.ParentID},
		ResourceType:                cfg.ParentType, // "organization" or "folder"
		LoggingDestinationProjectID: auditProjectID,
		BillingAccount:              cfg.BillingAccount,
		EnableBillingAccountSink:    cfg.EnableBillingAccountSink,

		StorageOptions: &logging.StorageOptions{
			LoggingSinkName:           "sk-c-logging-bkt",
			LoggingSinkFilter:         logFilter,
			Location:                  cfg.LogExportStorageLocation,
			ForceDestroy:              cfg.LogExportStorageForceDestroy,
			Versioning:                cfg.LogExportStorageVersioning,
			RetentionPolicyEnabled:    retentionEnabled,
			RetentionPolicyIsLocked:   retentionLocked,
			RetentionPolicyPeriodDays: retentionDays,
		},

		PubSubOptions: &logging.PubSubOptions{
			LoggingSinkName:   "sk-c-logging-pub",
			LoggingSinkFilter: logFilter,
			CreateSubscriber:  true,
		},

		ProjectOptions: &logging.ProjectOptions{
			LoggingSinkName:          "sk-c-logging-prj",
			LoggingSinkFilter:        logFilter,
			LogBucketID:              "AggregatedLogs",
			LogBucketDescription:     "Project destination log bucket for aggregated logs",
			Location:                 cfg.DefaultRegion,
			EnableAnalytics:          true,
			LinkedDatasetID:          "ds_c_prj_aggregated_logs_analytics",
			LinkedDatasetDescription: "Project destination BigQuery Dataset for Logbucket analytics",
		},
	})
	if err != nil {
		return nil, err
	}

	// ====================================================================
	// Billing Export BigQuery Dataset
	// This lives outside the centralized-logging module in the upstream TF
	// (log_sinks.tf google_bigquery_dataset.billing_dataset). The actual
	// billing export must be configured manually in the Cloud Console.
	// ====================================================================
	if _, err := bigquery.NewDataset(ctx, "billing-dataset", &bigquery.DatasetArgs{
		Project:      billingExportProjectID,
		DatasetId:    pulumi.String("billing_data"),
		FriendlyName: pulumi.String("GCP Billing Data"),
		Location:     pulumi.String(cfg.BillingExportDatasetLocation),
	}); err != nil {
		return nil, err
	}

	return &LoggingOutputs{
		StorageBucketName: cl.StorageBucketName,
		PubSubTopicName:   cl.PubSubTopicName,
		LogBucketName:     cl.ProjectLogBucketName,
		LinkedDatasetName: cl.LinkedDatasetName,
		LastResource:      cl.LastResource,
	}, nil
}
