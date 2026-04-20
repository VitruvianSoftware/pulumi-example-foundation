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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudasset"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudfunctionsv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/securitycenter"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// caiRolesToMonitor defines the IAM roles that trigger SCC findings when
// granted to new members. These are high-privilege roles that should be
// closely monitored. Matches the upstream default roles_to_monitor variable.
var caiRolesToMonitor = []string{
	"roles/owner",
	"roles/editor",
	"roles/resourcemanager.organizationAdmin",
	"roles/iam.serviceAccountTokenCreator",
}

// deployCAIMonitoring deploys the Cloud Asset Inventory monitoring
// infrastructure. This mirrors the upstream Terraform foundation's
// 1-org/modules/cai-monitoring module.
//
// The CAI monitoring pipeline works as follows:
//  1. A Cloud Asset Organization Feed watches for IAM policy changes
//  2. Changes are published to a Pub/Sub topic
//  3. A Cloud Function (v2) is triggered by the Pub/Sub messages
//  4. The function checks for IAM bindings with monitored roles
//  5. Violations are reported as SCC findings via the SCC Source
//
// Resources created:
//   - Cloud Function service account with org-level SCC findings editor
//   - Artifact Registry repository for the function container image
//   - Cloud Storage bucket for function source code
//   - Pub/Sub topic and Cloud Asset Organization Feed
//   - SCC v2 Organization Source for findings
//   - Cloud Function v2 triggered by Pub/Sub
func deployCAIMonitoring(ctx *pulumi.Context, cfg *OrgConfig, sccProjectID pulumi.StringOutput) error {
	// ========================================================================
	// 1. Cloud Function Service Account
	// This SA runs the CAI monitoring function and needs:
	//   - roles/securitycenter.findingsEditor at org level
	//   - roles/pubsub.publisher, roles/eventarc.eventReceiver,
	//     roles/run.invoker at project level
	// ========================================================================
	caiSA, err := serviceaccount.NewAccount(ctx, "cai-monitoring-sa", &serviceaccount.AccountArgs{
		Project:     sccProjectID,
		AccountId:   pulumi.String("cai-monitoring"),
		Description: pulumi.String("Service account for CAI monitoring Cloud Function"),
	})
	if err != nil {
		return err
	}

	// Org-level: SCC findings editor so the function can create findings
	caiSAMember := caiSA.Email.ApplyT(func(email string) string {
		return fmt.Sprintf("serviceAccount:%s", email)
	}).(pulumi.StringOutput)

	findingsEditorIAM, err := organizations.NewIAMMember(ctx, "cai-sa-findings-editor", &organizations.IAMMemberArgs{
		OrgId:  pulumi.String(cfg.OrgID),
		Role:   pulumi.String("roles/securitycenter.findingsEditor"),
		Member: caiSAMember,
	})
	if err != nil {
		return err
	}

	// Project-level: roles for Pub/Sub, Eventarc, and Cloud Run
	var cfIAMResources []pulumi.Resource
	cfRoles := []struct{ name, role string }{
		{"cai-sa-pubsub-publisher", "roles/pubsub.publisher"},
		{"cai-sa-eventarc-receiver", "roles/eventarc.eventReceiver"},
		{"cai-sa-run-invoker", "roles/run.invoker"},
	}
	for _, r := range cfRoles {
		iam, err := projects.NewIAMMember(ctx, r.name, &projects.IAMMemberArgs{
			Project: sccProjectID,
			Role:    pulumi.String(r.role),
			Member:  caiSAMember,
		})
		if err != nil {
			return err
		}
		cfIAMResources = append(cfIAMResources, iam)
	}
	cfIAMResources = append(cfIAMResources, findingsEditorIAM)

	// ========================================================================
	// 2. Service Identities for dependent services
	// Ensures the service agent SAs exist before resources reference them.
	// Mirrors: google_project_service_identity in cai-monitoring/iam.tf
	// ========================================================================
	caiServices := []string{
		"cloudfunctions.googleapis.com",
		"artifactregistry.googleapis.com",
		"pubsub.googleapis.com",
	}
	for _, svc := range caiServices {
		if _, err := projects.NewServiceIdentity(ctx, fmt.Sprintf("cai-identity-%s", svc), &projects.ServiceIdentityArgs{
			Project: sccProjectID,
			Service: pulumi.String(svc),
		}); err != nil {
			return err
		}
	}

	// ========================================================================
	// 3. Artifact Registry Repository
	// Stores the Cloud Function container image.
	// Mirrors: google_artifact_registry_repository "cloudfunction"
	// ========================================================================
	arRepo, err := artifactregistry.NewRepository(ctx, "cai-monitoring-ar", &artifactregistry.RepositoryArgs{
		Project:      sccProjectID,
		Location:     pulumi.String(cfg.DefaultRegion),
		RepositoryId: pulumi.String("ar-cai-monitoring"),
		Description:  pulumi.String("Container images for the CAI monitoring Cloud Function"),
		Format:       pulumi.String("DOCKER"),
	})
	if err != nil {
		return err
	}

	// ========================================================================
	// 4. Cloud Storage Bucket for Function Source Code
	// The function source is zipped and uploaded to this bucket, then
	// referenced by the Cloud Function v2 deployment.
	// Mirrors: module "cloudfunction_source_bucket"
	// ========================================================================
	sourceBucket, err := storage.NewBucket(ctx, "cai-monitoring-source-bucket", &storage.BucketArgs{
		Project: sccProjectID,
		Name: sccProjectID.ApplyT(func(id string) string {
			return fmt.Sprintf("bkt-cai-monitoring-sources-%s", id)
		}).(pulumi.StringOutput),
		Location:                 pulumi.String(cfg.DefaultRegion),
		ForceDestroy:             pulumi.Bool(true),
		UniformBucketLevelAccess: pulumi.Bool(true),
	})
	if err != nil {
		return err
	}

	// Upload the function source code as a zip archive.
	// The source lives in 1-org/cai-monitoring-function/ and is archived
	// by Pulumi's FileArchive asset type.
	sourceObject, err := storage.NewBucketObject(ctx, "cai-monitoring-source-zip", &storage.BucketObjectArgs{
		Bucket: sourceBucket.Name,
		Name:   pulumi.String("cai-monitoring-function.zip"),
		Source: pulumi.NewFileArchive("./cai-monitoring-function"),
	})
	if err != nil {
		return err
	}

	// ========================================================================
	// 5. Pub/Sub Topic + Cloud Asset Organization Feed
	// The org feed watches all IAM_POLICY changes across the org and
	// publishes them to the Pub/Sub topic for the Cloud Function to consume.
	// Mirrors: google_cloud_asset_organization_feed "organization_feed"
	//          + module "pubsub_cai_feed"
	// ========================================================================
	caiTopic, err := pubsub.NewTopic(ctx, "cai-monitoring-topic", &pubsub.TopicArgs{
		Project: sccProjectID,
		Name:    pulumi.String("top-cai-monitoring-event"),
	})
	if err != nil {
		return err
	}

	if _, err := cloudasset.NewOrganizationFeed(ctx, "cai-org-feed", &cloudasset.OrganizationFeedArgs{
		FeedId:         pulumi.String("fd-cai-monitoring"),
		BillingProject: sccProjectID,
		OrgId:          pulumi.String(cfg.OrgID),
		ContentType:    pulumi.String("IAM_POLICY"),
		AssetTypes:     pulumi.StringArray{pulumi.String(".*")},
		FeedOutputConfig: &cloudasset.OrganizationFeedFeedOutputConfigArgs{
			PubsubDestination: &cloudasset.OrganizationFeedFeedOutputConfigPubsubDestinationArgs{
				Topic: caiTopic.ID(),
			},
		},
	}); err != nil {
		return err
	}

	// ========================================================================
	// 6. SCC v2 Organization Source
	// Creates a custom SCC source that the Cloud Function uses to report
	// findings about IAM policy violations.
	// Mirrors: google_scc_v2_organization_source "cai_monitoring"
	// ========================================================================
	sccSource, err := securitycenter.NewV2OrganizationSource(ctx, "cai-monitoring-source", &securitycenter.V2OrganizationSourceArgs{
		Organization: pulumi.String(cfg.OrgID),
		DisplayName:  pulumi.String("CAI Monitoring"),
		Description:  pulumi.String("SCC Finding Source for caiMonitoring Cloud Functions."),
	})
	if err != nil {
		return err
	}

	// ========================================================================
	// 7. Cloud Function v2
	// Deploys the caiMonitoring Node.js function that:
	//   - Receives IAM policy change events from Pub/Sub
	//   - Checks for grants of monitored privileged roles
	//   - Creates SCC findings when violations are detected
	// Mirrors: module "cloud_function" (GoogleCloudPlatform/cloud-functions)
	// ========================================================================

	// Build the ROLES environment variable from the monitored roles list
	rolesEnvVar := ""
	for i, role := range caiRolesToMonitor {
		if i > 0 {
			rolesEnvVar += ","
		}
		rolesEnvVar += role
	}

	// The builder SA (cai-monitoring-builder) was created in iam.go section 11.
	// It's used as the build_service_account for Cloud Build.
	builderSAEmail := sccProjectID.ApplyT(func(id string) string {
		return fmt.Sprintf("projects/%s/serviceAccounts/cai-monitoring-builder@%s.iam.gserviceaccount.com", id, id)
	}).(pulumi.StringOutput)

	// Wait for all IAM bindings to propagate before deploying the function.
	// Mirrors: time_sleep "wait_kms_iam" in upstream (60s delay).
	// In Pulumi we use explicit DependsOn on the IAM resources.
	if _, err := cloudfunctionsv2.NewFunction(ctx, "cai-monitoring-function", &cloudfunctionsv2.FunctionArgs{
		Project:     sccProjectID,
		Location:    pulumi.String(cfg.DefaultRegion),
		Name:        pulumi.String("caiMonitoring"),
		Description: pulumi.String("Check on the Organization for members (users, groups and service accounts) that contains the IAM roles listed."),
		BuildConfig: &cloudfunctionsv2.FunctionBuildConfigArgs{
			Runtime:    pulumi.String("nodejs20"),
			EntryPoint: pulumi.String("caiMonitoring"),
			Source: &cloudfunctionsv2.FunctionBuildConfigSourceArgs{
				StorageSource: &cloudfunctionsv2.FunctionBuildConfigSourceStorageSourceArgs{
					Bucket: sourceBucket.Name,
					Object: sourceObject.Name,
				},
			},
			DockerRepository: arRepo.ID(),
			ServiceAccount:   builderSAEmail,
		},
		ServiceConfig: &cloudfunctionsv2.FunctionServiceConfigArgs{
			ServiceAccountEmail: caiSA.Email,
			EnvironmentVariables: pulumi.StringMap{
				"ROLES":     pulumi.String(rolesEnvVar),
				"SOURCE_ID": sccSource.Name,
			},
		},
		EventTrigger: &cloudfunctionsv2.FunctionEventTriggerArgs{
			TriggerRegion:       pulumi.String(cfg.DefaultRegion),
			EventType:           pulumi.String("google.cloud.pubsub.topic.v1.messagePublished"),
			PubsubTopic:         caiTopic.ID(),
			RetryPolicy:         pulumi.String("RETRY_POLICY_RETRY"),
			ServiceAccountEmail: caiSA.Email,
		},
	}, pulumi.DependsOn(cfIAMResources)); err != nil {
		return err
	}

	return nil
}
