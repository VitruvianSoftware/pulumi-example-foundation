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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// LoggingOutputs holds resource references for downstream exports.
type LoggingOutputs struct {
	StorageBucketName pulumi.StringOutput
	PubSubTopicName   pulumi.StringOutput
}

// deployCentralizedLogging creates the centralized logging infrastructure:
// org-level sinks that export audit logs to Storage, Pub/Sub, and a Logging
// project bucket with a linked BigQuery dataset for analytics.
// This mirrors the Terraform foundation's log_sinks.tf and centralized-logging module.
//
// Critical fix (D15): Grants sink writer identities IAM on destinations so
// logs are actually delivered instead of failing with 403.
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
	if _, err := logging.NewLinkedDataset(ctx, "aggregated-logs-dataset", &logging.LinkedDatasetArgs{
		Parent:      auditProjectID.ApplyT(func(id string) string { return "projects/" + id }).(pulumi.StringOutput),
		Bucket:      logProjectBucket.ID(),
		LinkId:      pulumi.String("ds_c_prj_aggregated_logs_analytics"),
		Location:    pulumi.String(cfg.DefaultRegion),
		Description: pulumi.String("Project destination BigQuery Dataset for Logbucket analytics"),
	}); err != nil {
		return nil, err
	}

	// 1b. Organization Sink → Logging Project Bucket
	_, err = logging.NewOrganizationSink(ctx, "org-sink-project", &logging.OrganizationSinkArgs{
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


	// ========================================================================
	// 2. Organization Sink → Cloud Storage (long-term retention)
	// ========================================================================
	logBucket, err := storage.NewBucket(ctx, "org-log-storage", &storage.BucketArgs{
		Project: auditProjectID,
		Name: auditProjectID.ApplyT(func(id string) string {
			return fmt.Sprintf("bkt-%s-org-logs", id)
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
		Name:    pulumi.String("tp-org-logs"),
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
	if _, err := pubsub.NewTopicIAMMember(ctx, "pubsub-sink-writer", &pubsub.TopicIAMMemberArgs{
		Project: auditProjectID,
		Topic:   logTopic.Name,
		Role:    pulumi.String("roles/pubsub.publisher"),
		Member:  pubsubSink.WriterIdentity,
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 4. Billing Export BigQuery Dataset
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
