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

	libsecurity "github.com/VitruvianSoftware/pulumi-library/pkg/security"
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

// CAIMonitoringOutputs holds resource names for downstream exports.
// Mirrors upstream outputs: cai_monitoring_artifact_registry, cai_monitoring_asset_feed,
// cai_monitoring_bucket, cai_monitoring_topic.
type CAIMonitoringOutputs struct {
	ArtifactRegistryName pulumi.StringOutput
	AssetFeedName        pulumi.StringOutput
	BucketName           pulumi.StringOutput
	TopicName            pulumi.StringOutput
}

// deployCAIMonitoring deploys the Cloud Asset Inventory monitoring
// infrastructure using the pkg/security library component.
//
// This replaces ~250 lines of inline resource orchestration with a single
// reusable component call. The component mirrors the upstream Terraform
// foundation's 1-org/modules/cai-monitoring module.
//
// The CAI monitoring pipeline works as follows:
//  1. A Cloud Asset Organization Feed watches for IAM policy changes
//  2. Changes are published to a Pub/Sub topic
//  3. A Cloud Function (v2) is triggered by the Pub/Sub messages
//  4. The function checks for IAM bindings with monitored roles
//  5. Violations are reported as SCC findings via the SCC Source
func deployCAIMonitoring(ctx *pulumi.Context, cfg *OrgConfig, sccProjectID pulumi.StringOutput) (*CAIMonitoringOutputs, error) {
	// The builder SA (cai-monitoring-builder) was created in iam.go section 11.
	// It's used as the build_service_account for Cloud Build.
	builderSAEmail := sccProjectID.ApplyT(func(id string) string {
		return fmt.Sprintf("projects/%s/serviceAccounts/cai-monitoring-builder@%s.iam.gserviceaccount.com", id, id)
	}).(pulumi.StringOutput)

	cai, err := libsecurity.NewCAIMonitoring(ctx, "cai-monitoring", &libsecurity.CAIMonitoringArgs{
		OrgID:               pulumi.String(cfg.OrgID),
		ProjectID:           sccProjectID,
		Location:            cfg.DefaultRegion,
		BuildServiceAccount: builderSAEmail,
		FunctionSourcePath:  "./cai-monitoring-function",
		RolesToMonitor:      caiRolesToMonitor,
	})
	if err != nil {
		return nil, err
	}

	return &CAIMonitoringOutputs{
		ArtifactRegistryName: cai.ArtifactRegistryName,
		AssetFeedName:        cai.AssetFeedName,
		BucketName:           cai.BucketName,
		TopicName:            cai.TopicName,
	}, nil
}
