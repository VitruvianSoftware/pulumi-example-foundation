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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/bigquery"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/logging"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// LoggingOutputs holds resource references for downstream exports.
type LoggingOutputs struct {
	StorageBucketName    pulumi.StringOutput
	PubSubTopicName      pulumi.StringOutput
	LogBucketName        pulumi.StringOutput // upstream: logs_export_project_logbucket_name
	LinkedDatasetName    pulumi.StringOutput // upstream: logs_export_project_linked_dataset_name
	BillingSinkNames     []pulumi.StringOutput // upstream: billing_sink_names
	// LastResource is the last resource created by the logging deployment,
	// used for dependency ordering (e.g., policies must wait for sinks).
	LastResource pulumi.Resource
}

// deployCentralizedLogging creates the centralized logging infrastructure:
// org-level sinks that export audit logs to Storage, Pub/Sub, and a Logging
// project bucket with a linked BigQuery dataset for analytics.
// This mirrors the Terraform foundation's log_sinks.tf and centralized-logging module.
//
// Critical fix (D15): Grants sink writer identities IAM on destinations so
// logs are actually delivered instead of failing with 403.
//
// When EnableBillingAccountSink is true (default), billing account sinks are
// created alongside the org sinks, mirroring the upstream centralized-logging
// module's enable_billing_account_sink = true behavior.
func deployCentralizedLogging(ctx *pulumi.Context, cfg *OrgConfig, auditProjectID, billingExportProjectID pulumi.StringOutput) (*LoggingOutputs, error) {
	// Comprehensive log filter covering all audit and network logs
	logFilter := `logName: /logs/cloudaudit.googleapis.com%2Factivity OR
logName: /logs/cloudaudit.googleapis.com%2Fsystem_event OR
logName: /logs/cloudaudit.googleapis.com%2Fdata_access OR
logName: /logs/cloudaudit.googleapis.com%2Faccess_transparency OR
logName: /logs/cloudaudit.googleapis.com%2Fpolicy OR
logName: /logs/compute.googleapis.com%2Fvpc_flows OR
logName: /logs/compute.googleapis.com%2Ffirewall OR
logName: /logs/dns.googleapis.com%2Fdns_queries`

	// Track the last resource created for dependency ordering
	var lastResource pulumi.Resource

	// Random suffix for globally-unique resource names (matches upstream random_string.suffix)
	suffix, err := random.NewRandomString(ctx, "log-suffix", &random.RandomStringArgs{
		Length:  pulumi.Int(4),
		Upper:   pulumi.Bool(false),
		Special: pulumi.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// 1. Logging Project Bucket (G10)
	// Create the logging bucket that the org sink writes to. Without this,
	// the sink destination doesn't exist and logs are silently dropped.
	// Mirrors: centralized-logging module's project log_bucket_id config.
	// ========================================================================
	logProjectBucket, err := logging.NewProjectBucketConfig(ctx, "aggregated-logs-bucket", &logging.ProjectBucketConfigArgs{
		Project:         auditProjectID,
		Location:        pulumi.String(cfg.DefaultRegion),
		BucketId:        pulumi.String("AggregatedLogs"),
		Description:     pulumi.String("Project destination log bucket for aggregated logs"),
		EnableAnalytics: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// Linked BigQuery dataset for log analytics (query logs via SQL)
	// Mirrors: linked_dataset_id = "ds_c_prj_aggregated_logs_analytics"
	linkedDataset, err := logging.NewLinkedDataset(ctx, "aggregated-logs-dataset", &logging.LinkedDatasetArgs{
		Parent:      auditProjectID.ApplyT(func(id string) string { return "projects/" + id }).(pulumi.StringOutput),
		Bucket:      logProjectBucket.ID(),
		LinkId:      pulumi.String("ds_c_prj_aggregated_logs_analytics"),
		Location:    pulumi.String(cfg.DefaultRegion),
		Description: pulumi.String("Project destination BigQuery Dataset for Logbucket analytics"),
	})
	if err != nil {
		return nil, err
	}

	// 1b. Organization Sink → Logging Project Bucket
	orgSinkProject, err := logging.NewOrganizationSink(ctx, "org-sink-project", &logging.OrganizationSinkArgs{
		Name:  pulumi.String("sk-c-logging-prj"),
		OrgId: pulumi.String(cfg.OrgID),
		Destination: auditProjectID.ApplyT(func(id string) string {
			return fmt.Sprintf("logging.googleapis.com/projects/%s/locations/%s/buckets/AggregatedLogs", id, cfg.DefaultRegion)
		}).(pulumi.StringOutput),
		Filter:          pulumi.String(logFilter),
		IncludeChildren: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	lastResource = orgSinkProject

	// ========================================================================
	// 2. Organization Sink → Cloud Storage (long-term retention)
	// ========================================================================
	logBucket, err := storage.NewBucket(ctx, "org-log-storage", &storage.BucketArgs{
		Project: auditProjectID,
		Name: pulumi.All(auditProjectID, suffix.Result).ApplyT(func(args []interface{}) string {
			id := args[0].(string)
			s := args[1].(string)
			return fmt.Sprintf("bkt-%s-org-logs-%s", id, s)
		}).(pulumi.StringOutput),
		Location:                 pulumi.String(cfg.LogExportStorageLocation),
		UniformBucketLevelAccess: pulumi.Bool(true),
		ForceDestroy:             pulumi.Bool(cfg.LogExportStorageForceDestroy),
		Versioning: &storage.BucketVersioningArgs{
			Enabled: pulumi.Bool(cfg.LogExportStorageVersioning),
		},
		RetentionPolicy: logStorageRetentionPolicy(cfg),
	})
	if err != nil {
		return nil, err
	}

	storageSink, err := logging.NewOrganizationSink(ctx, "org-sink-storage", &logging.OrganizationSinkArgs{
		Name:  pulumi.String("sk-c-logging-bkt"),
		OrgId: pulumi.String(cfg.OrgID),
		Destination: logBucket.Name.ApplyT(func(name string) string {
			return fmt.Sprintf("storage.googleapis.com/%s", name)
		}).(pulumi.StringOutput),
		Filter:          pulumi.String(logFilter),
		IncludeChildren: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// Grant storage sink writer identity access to create objects in the bucket (D15)
	if _, err := storage.NewBucketIAMMember(ctx, "storage-sink-writer", &storage.BucketIAMMemberArgs{
		Bucket: logBucket.Name,
		Role:   pulumi.String("roles/storage.objectCreator"),
		Member: storageSink.WriterIdentity,
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 3. Organization Sink → Pub/Sub (real-time streaming / external export)
	// ========================================================================
	logTopic, err := pubsub.NewTopic(ctx, "org-log-topic", &pubsub.TopicArgs{
		Project: auditProjectID,
		Name: suffix.Result.ApplyT(func(s string) string {
			return fmt.Sprintf("tp-org-logs-%s", s)
		}).(pulumi.StringOutput),
	})
	if err != nil {
		return nil, err
	}

	if _, err := pubsub.NewSubscription(ctx, "org-log-subscription", &pubsub.SubscriptionArgs{
		Project: auditProjectID,
		Name:    pulumi.String("sub-org-logs"),
		Topic:   logTopic.Name,
	}); err != nil {
		return nil, err
	}

	pubsubSink, err := logging.NewOrganizationSink(ctx, "org-sink-pubsub", &logging.OrganizationSinkArgs{
		Name:  pulumi.String("sk-c-logging-pub"),
		OrgId: pulumi.String(cfg.OrgID),
		Destination: logTopic.ID().ApplyT(func(id string) string {
			return fmt.Sprintf("pubsub.googleapis.com/%s", id)
		}).(pulumi.StringOutput),
		Filter:          pulumi.String(logFilter),
		IncludeChildren: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// Grant Pub/Sub sink writer identity access to publish to the topic (D15)
	pubsubSinkWriter, err := pubsub.NewTopicIAMMember(ctx, "pubsub-sink-writer", &pubsub.TopicIAMMemberArgs{
		Project: auditProjectID,
		Topic:   logTopic.Name,
		Role:    pulumi.String("roles/pubsub.publisher"),
		Member:  pubsubSink.WriterIdentity,
	})
	if err != nil {
		return nil, err
	}
	lastResource = pubsubSinkWriter

	// ========================================================================
	// 4a. Billing Account Sinks (Gap 1 — upstream enable_billing_account_sink)
	// Creates billing account sinks to the same three destinations so that
	// billing account audit logs are captured alongside org-level logs.
	// ========================================================================
	var billingSinkNames []pulumi.StringOutput
	if cfg.EnableBillingAccountSink {
		// Billing Account Sink → Cloud Storage
		billingSinkStorage, err := logging.NewBillingAccountSink(ctx, "billing-sink-storage", &logging.BillingAccountSinkArgs{
			Name:           pulumi.String("sk-c-logging-bkt-billing"),
			BillingAccount: pulumi.String(cfg.BillingAccount),
			Destination: logBucket.Name.ApplyT(func(name string) string {
				return fmt.Sprintf("storage.googleapis.com/%s", name)
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return nil, err
		}
		billingSinkNames = append(billingSinkNames, billingSinkStorage.Name)
		if _, err := storage.NewBucketIAMMember(ctx, "storage-sink-writer-billing", &storage.BucketIAMMemberArgs{
			Bucket: logBucket.Name,
			Role:   pulumi.String("roles/storage.objectCreator"),
			Member: billingSinkStorage.WriterIdentity,
		}); err != nil {
			return nil, err
		}

		// Billing Account Sink → Pub/Sub
		billingSinkPubsub, err := logging.NewBillingAccountSink(ctx, "billing-sink-pubsub", &logging.BillingAccountSinkArgs{
			Name:           pulumi.String("sk-c-logging-pub-billing"),
			BillingAccount: pulumi.String(cfg.BillingAccount),
			Destination: logTopic.ID().ApplyT(func(id string) string {
				return fmt.Sprintf("pubsub.googleapis.com/%s", id)
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return nil, err
		}
		billingSinkNames = append(billingSinkNames, billingSinkPubsub.Name)
		if _, err := pubsub.NewTopicIAMMember(ctx, "pubsub-sink-writer-billing", &pubsub.TopicIAMMemberArgs{
			Project: auditProjectID,
			Topic:   logTopic.Name,
			Role:    pulumi.String("roles/pubsub.publisher"),
			Member:  billingSinkPubsub.WriterIdentity,
		}); err != nil {
			return nil, err
		}

		// Billing Account Sink → Logging Project Bucket
		billingSinkProject, err := logging.NewBillingAccountSink(ctx, "billing-sink-project", &logging.BillingAccountSinkArgs{
			Name:           pulumi.String("sk-c-logging-prj-billing"),
			BillingAccount: pulumi.String(cfg.BillingAccount),
			Destination: auditProjectID.ApplyT(func(id string) string {
				return fmt.Sprintf("logging.googleapis.com/projects/%s/locations/%s/buckets/AggregatedLogs", id, cfg.DefaultRegion)
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return nil, err
		}
		billingSinkNames = append(billingSinkNames, billingSinkProject.Name)
		// Grant billing sink writer identity logWriter on the audit project
		billingProjectWriter, err := projects.NewIAMMember(ctx, "project-sink-writer-billing", &projects.IAMMemberArgs{
			Project: auditProjectID,
			Role:    pulumi.String("roles/logging.logWriter"),
			Member:  billingSinkProject.WriterIdentity,
		})
		if err != nil {
			return nil, err
		}
		lastResource = billingProjectWriter
	}

	// ========================================================================
	// 4b. Billing Export BigQuery Dataset
	// Note: The actual billing export must be configured manually in the
	// Cloud Console, as there is no API to automate this currently.
	// ========================================================================
	if _, err := bigquery.NewDataset(ctx, "billing-dataset", &bigquery.DatasetArgs{
		Project:      billingExportProjectID,
		DatasetId:    pulumi.String("billing_data"),
		FriendlyName: pulumi.String("GCP Billing Data"),
		Location:     pulumi.String(cfg.BillingExportDatasetLocation),
	}); err != nil {
		return nil, err
	}

	return &LoggingOutputs{
		StorageBucketName: logBucket.Name,
		PubSubTopicName:   logTopic.Name,
		LogBucketName:     logProjectBucket.ID().ApplyT(func(id string) string { return id }).(pulumi.StringOutput),
		LinkedDatasetName: linkedDataset.Name,
		BillingSinkNames:  billingSinkNames,
		LastResource:      lastResource,
	}, nil
}

// logStorageRetentionPolicy builds the retention policy args from config.
// Returns nil when no retention policy is configured (default).
func logStorageRetentionPolicy(cfg *OrgConfig) *storage.BucketRetentionPolicyArgs {
	if cfg.LogExportStorageRetentionPolicy == nil {
		return nil
	}
	retentionSeconds := cfg.LogExportStorageRetentionPolicy.RetentionPeriodDays * 86400
	return &storage.BucketRetentionPolicyArgs{
		IsLocked:        pulumi.Bool(cfg.LogExportStorageRetentionPolicy.IsLocked),
		RetentionPeriod: pulumi.String(fmt.Sprintf("%d", retentionSeconds)),
	}
}
