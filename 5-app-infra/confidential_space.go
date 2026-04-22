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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ConfidentialSpaceArgs configures a Confidential Space VM deployment,
// matching the upstream Terraform confidential_space module.
type ConfidentialSpaceArgs struct {
	Env                     string
	BusinessUnit            string
	ProjectID               pulumi.StringInput
	ProjectNumber           pulumi.StringInput // from 4-projects stack export
	Region                  string
	SubnetworkSelfLink      pulumi.StringInput
	WorkloadSAEmail         pulumi.StringInput
	ConfidentialImageDigest string
	ConfidentialMachineType string
	ConfidentialInstanceType string
	CpuPlatform             string
	CloudBuildProjectID     pulumi.StringInput
}

// ConfidentialSpaceResult holds outputs from the Confidential Space deployment.
type ConfidentialSpaceResult struct {
	InstanceSelfLink       pulumi.StringOutput
	WorkloadPoolID         pulumi.StringOutput
	WorkloadPoolProviderID pulumi.StringOutput
}

// deployConfidentialSpace creates a Workload Identity Pool, OIDC attestation
// provider, IAM bindings, and a Confidential VM, matching the upstream
// Terraform foundation's confidential_space module.
func deployConfidentialSpace(ctx *pulumi.Context, name string, args *ConfidentialSpaceArgs) (*ConfidentialSpaceResult, error) {
	// 1. Workload Identity Pool
	pool, err := iam.NewWorkloadIdentityPool(ctx, name+"-pool", &iam.WorkloadIdentityPoolArgs{
		WorkloadIdentityPoolId: pulumi.String("confidential-space-pool"),
		Disabled:               pulumi.Bool(false),
		Project:                args.ProjectID,
	})
	if err != nil {
		return nil, err
	}

	// 2. Workload Identity Pool Provider — OIDC attestation verifier
	// Attribute condition matches upstream's attribute_condition heredoc exactly.
	attributeCondition := args.WorkloadSAEmail.ToStringOutput().ApplyT(func(saEmail string) string {
		return fmt.Sprintf(
			`assertion.submods.container.image_digest == "%s" && "%s" in assertion.google_service_accounts && assertion.swname == "CONFIDENTIAL_SPACE" && "STABLE" in assertion.submods.confidential_space.support_attributes`,
			args.ConfidentialImageDigest, saEmail,
		)
	}).(pulumi.StringOutput)

	provider, err := iam.NewWorkloadIdentityPoolProvider(ctx, name+"-provider", &iam.WorkloadIdentityPoolProviderArgs{
		WorkloadIdentityPoolId:         pool.WorkloadIdentityPoolId,
		WorkloadIdentityPoolProviderId: pulumi.String("attestation-verifier"),
		DisplayName:                    pulumi.String("attestation-verifier"),
		Description:                    pulumi.String("OIDC provider for confidential computing attestation"),
		Project:                        args.ProjectID,
		Oidc: &iam.WorkloadIdentityPoolProviderOidcArgs{
			IssuerUri:        pulumi.String("https://confidentialcomputing.googleapis.com/"),
			AllowedAudiences: pulumi.StringArray{pulumi.String("https://sts.googleapis.com")},
		},
		AttributeMapping: pulumi.StringMap{
			"google.subject":         pulumi.String(`"gcpcs::" + assertion.submods.container.image_digest + "::" + assertion.submods.gce.project_number + "::" + assertion.submods.gce.instance_id`),
			"attribute.image_digest": pulumi.String(`assertion.submods.container.image_digest`),
		},
		AttributeCondition: attributeCondition,
	})
	if err != nil {
		return nil, err
	}

	// 3. IAM Binding for the Workload SA
	// Uses the project number from the 4-projects stack export — NOT a
	// runtime LookupProject call (which would be a Pulumi anti-pattern
	// breaking previews).
	member := args.ProjectNumber.ToStringOutput().ApplyT(func(num string) string {
		return fmt.Sprintf(
			"principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/confidential-space-pool/*",
			num,
		)
	}).(pulumi.StringOutput)

	serviceAccountID := pulumi.All(args.ProjectID, args.WorkloadSAEmail).ApplyT(func(a []interface{}) string {
		return fmt.Sprintf("projects/%s/serviceAccounts/%s", a[0].(string), a[1].(string))
	}).(pulumi.StringOutput)

	_, err = serviceaccount.NewIAMMember(ctx, name+"-iam", &serviceaccount.IAMMemberArgs{
		ServiceAccountId: serviceAccountID,
		Role:             pulumi.String("roles/iam.workloadIdentityUser"),
		Member:           member,
	}, pulumi.DependsOn([]pulumi.Resource{provider}))
	if err != nil {
		return nil, err
	}

	// 4. Confidential VM — TEE image reference from CI/CD project's Artifact Registry
	defaultTeeImageRef := args.CloudBuildProjectID.ToStringOutput().ApplyT(func(cbID string) string {
		return fmt.Sprintf("%s-docker.pkg.dev/%s/tf-runners/confidential_space_image:latest", args.Region, cbID)
	}).(pulumi.StringOutput)

	inst, err := compute.NewInstance(ctx, name+"-vm", &compute.InstanceArgs{
		Project:        args.ProjectID,
		Name:           pulumi.String("confidential-instance"),
		MachineType:    pulumi.String(args.ConfidentialMachineType),
		Zone:           pulumi.String(args.Region + "-a"),
		MinCpuPlatform: pulumi.String(args.CpuPlatform),
		ConfidentialInstanceConfig: &compute.InstanceConfidentialInstanceConfigArgs{
			EnableConfidentialCompute: pulumi.Bool(true),
			ConfidentialInstanceType:  pulumi.String(args.ConfidentialInstanceType),
		},
		ShieldedInstanceConfig: &compute.InstanceShieldedInstanceConfigArgs{
			EnableSecureBoot:          pulumi.Bool(true),
			EnableVtpm:                pulumi.Bool(true),
			EnableIntegrityMonitoring: pulumi.Bool(true),
		},
		BootDisk: &compute.InstanceBootDiskArgs{
			InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
				Image: pulumi.String("projects/confidential-space-images/global/images/family/confidential-space"),
			},
		},
		NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
			&compute.InstanceNetworkInterfaceArgs{
				Subnetwork: args.SubnetworkSelfLink,
			},
		},
		ServiceAccount: &compute.InstanceServiceAccountArgs{
			Email:  args.WorkloadSAEmail,
			Scopes: pulumi.StringArray{pulumi.String("https://www.googleapis.com/auth/cloud-platform")},
		},
		Metadata: pulumi.StringMap{
			"tee-image-reference": defaultTeeImageRef,
		},
	})
	if err != nil {
		return nil, err
	}

	return &ConfidentialSpaceResult{
		InstanceSelfLink:       inst.SelfLink,
		WorkloadPoolID:         pool.WorkloadIdentityPoolId,
		WorkloadPoolProviderID: provider.WorkloadIdentityPoolProviderId,
	}, nil
}
