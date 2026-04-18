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

// deployCentralizedLogging creates the centralized logging infrastructure:
// org-level sinks that export audit logs to Storage, Pub/Sub, and a Logging
// project bucket with a linked BigQuery dataset for analytics.
// This mirrors the Terraform foundation's log_sinks.tf.
func deployCentralizedLogging(ctx *pulumi.Context, cfg *OrgConfig, auditProjectID, billingExportProjectID pulumi.StringOutput) error {
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
	// 1. Organization Sink → Logging Project (primary destination)
	// Logs are sent to a logging bucket with a linked BigQuery dataset
	// for ad-hoc investigations, querying, and reporting.
	// ========================================================================
	if _, err := logging.NewOrganizationSink(ctx, "org-sink-project", &logging.OrganizationSinkArgs{
		Name:  pulumi.String("sk-c-logging-prj"),
		OrgId: pulumi.String(cfg.OrgID),
		Destination: auditProjectID.ApplyT(func(id string) string {
			return fmt.Sprintf("logging.googleapis.com/projects/%s/locations/global/buckets/AggregatedLogs", id)
		}).(pulumi.StringOutput),
		Filter:          pulumi.String(logFilter),
		IncludeChildren: pulumi.Bool(true),
	}); err != nil {
		return err
	}

	// ========================================================================
	// 2. Organization Sink → Cloud Storage (long-term retention)
	// ========================================================================
	logBucket, err := storage.NewBucket(ctx, "org-log-storage", &storage.BucketArgs{
		Project: auditProjectID,
		Name: auditProjectID.ApplyT(func(id string) string {
			return fmt.Sprintf("bkt-%s-org-logs", id)
		}).(pulumi.StringOutput),
		Location:                 pulumi.String(cfg.DefaultRegion),
		UniformBucketLevelAccess: pulumi.Bool(true),
		Versioning: &storage.BucketVersioningArgs{
			Enabled: pulumi.Bool(true),
		},
	})
	if err != nil {
		return err
	}

	if _, err := logging.NewOrganizationSink(ctx, "org-sink-storage", &logging.OrganizationSinkArgs{
		Name:  pulumi.String("sk-c-logging-bkt"),
		OrgId: pulumi.String(cfg.OrgID),
		Destination: logBucket.Name.ApplyT(func(name string) string {
			return fmt.Sprintf("storage.googleapis.com/%s", name)
		}).(pulumi.StringOutput),
		Filter:          pulumi.String(logFilter),
		IncludeChildren: pulumi.Bool(true),
	}); err != nil {
		return err
	}

	// ========================================================================
	// 3. Organization Sink → Pub/Sub (real-time streaming / external export)
	// ========================================================================
	logTopic, err := pubsub.NewTopic(ctx, "org-log-topic", &pubsub.TopicArgs{
		Project: auditProjectID,
		Name:    pulumi.String("tp-org-logs"),
	})
	if err != nil {
		return err
	}

	if _, err := pubsub.NewSubscription(ctx, "org-log-subscription", &pubsub.SubscriptionArgs{
		Project: auditProjectID,
		Name:    pulumi.String("sub-org-logs"),
		Topic:   logTopic.Name,
	}); err != nil {
		return err
	}

	if _, err := logging.NewOrganizationSink(ctx, "org-sink-pubsub", &logging.OrganizationSinkArgs{
		Name:  pulumi.String("sk-c-logging-pub"),
		OrgId: pulumi.String(cfg.OrgID),
		Destination: logTopic.ID().ApplyT(func(id string) string {
			return fmt.Sprintf("pubsub.googleapis.com/%s", id)
		}).(pulumi.StringOutput),
		Filter:          pulumi.String(logFilter),
		IncludeChildren: pulumi.Bool(true),
	}); err != nil {
		return err
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
		Location:     pulumi.String(cfg.DefaultRegion),
	}); err != nil {
		return err
	}

	return nil
}
