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

	"github.com/VitruvianSoftware/pulumi-library/go/pkg/project"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/accesscontextmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BUProjects holds outputs from business unit project deployment.
type BUProjects struct {
	SVPCProjectID                pulumi.StringOutput
	SVPCProjectNumber            pulumi.StringOutput
	FloatingProjectID            pulumi.StringOutput
	PeeringProjectID             pulumi.StringOutput
	PeeringNetworkSelfLink       pulumi.StringOutput
	PeeringSubnetSelfLink        pulumi.StringOutput
	IAPFirewallTags              pulumi.MapOutput
	CMEKBucket                   *pulumi.StringOutput
	CMEKKeyring                  *pulumi.StringOutput
	CMEKKeys                     *pulumi.StringArrayOutput
	ConfSpaceProjectID           *pulumi.StringOutput
	ConfSpaceProjectNumber       *pulumi.StringOutput
	ConfSpaceWorkloadSA          *pulumi.StringOutput
	SubnetsSelfLinks             pulumi.StringArrayOutput
	VPCSCPerimeterName           pulumi.StringOutput
	PeeringComplete              pulumi.BoolOutput
	AccessContextManagerPolicyID pulumi.StringOutput
	RestrictedEnabledApis        []string
}

// budgetConfig returns the standard budget configuration used for every
// project, matching the upstream TF project_budget variable.
func budgetConfig(cfg *ProjectsConfig) *project.BudgetConfig {
	return &project.BudgetConfig{
		Amount:             cfg.BudgetAmount,
		AlertSpentPercents: cfg.BudgetAlertPercents,
		AlertSpendBasis:    cfg.BudgetSpendBasis,
	}
}

// deployBusinessUnitProjects creates three project types per BU/env, matching
// the Terraform foundation's project factory pattern:
//   - SVPC-attached: connected to the Shared VPC host project w/ VPC-SC
//   - Floating: standalone project, not attached to any VPC
//   - Peering: project with its own VPC peered to the host network
func deployBusinessUnitProjects(ctx *pulumi.Context, cfg *ProjectsConfig, folderID, networkProjectID, perimeterName, kmsProjectID, acmPolicyID pulumi.StringOutput) (*BUProjects, error) {
	result := &BUProjects{}

	// ========================================================================
	// 1. SVPC-attached Project
	// This project is attached as a service project to the environment's
	// Shared VPC host, enabling shared network resource access.
	// ========================================================================
	svpcApis := []string{
		"compute.googleapis.com",
		"container.googleapis.com",
		"run.googleapis.com",
		"artifactregistry.googleapis.com",
		"billingbudgets.googleapis.com",
		"logging.googleapis.com",
		"accesscontextmanager.googleapis.com",
	}

	svpcProject, err := project.NewProject(ctx, "bu-svpc-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-sample-svpc", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-sample-svpc", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
		Labels:          projectLabels(cfg, "sample-application", "svpc"),
		Budget:          budgetConfig(cfg),
		ActivateApis:    svpcApis,
	})
	if err != nil {
		return nil, err
	}

	result.RestrictedEnabledApis = svpcApis

	// Attach as a Shared VPC service project
	if _, err := compute.NewSharedVPCServiceProject(ctx, "svpc-attachment", &compute.SharedVPCServiceProjectArgs{
		HostProject:    networkProjectID,
		ServiceProject: svpcProject.Project.ProjectId,
	}); err != nil {
		return nil, err
	}

	// VPC-SC Perimeter attachment — attach the SVPC project to the perimeter
	// matching upstream's vpc_service_control_attach_enabled behavior.
	if cfg.EnforceVpcSc {
		_, err := accesscontextmanager.NewServicePerimeterResource(ctx, "svpc-vpcsc-attach", &accesscontextmanager.ServicePerimeterResourceArgs{
			PerimeterName: perimeterName,
			Resource: svpcProject.Project.Number.ApplyT(func(n string) string {
				return fmt.Sprintf("projects/%s", n)
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return nil, err
		}
	}

	result.SVPCProjectID = svpcProject.Project.ProjectId
	result.SVPCProjectNumber = svpcProject.Project.Number

	// ========================================================================
	// 2. Floating Project (not attached to any VPC)
	// ========================================================================
	floatingProject, err := project.NewProject(ctx, "bu-floating-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-sample-floating", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-sample-floating", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
		Labels:          projectLabels(cfg, "sample-application", "none"),
		Budget:          budgetConfig(cfg),
		ActivateApis: []string{
			"compute.googleapis.com",
			"container.googleapis.com",
			"run.googleapis.com",
			"artifactregistry.googleapis.com",
			"billingbudgets.googleapis.com",
			"logging.googleapis.com",
		},
	})
	if err != nil {
		return nil, err
	}
	result.FloatingProjectID = floatingProject.Project.ProjectId

	// ========================================================================
	// 3. Peering Project — full VPC, subnet, DNS, peering, firewall
	// ========================================================================
	peeringProject, err := project.NewProject(ctx, "bu-peering-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-sample-peering", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-sample-peering", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
		Labels:          projectLabels(cfg, "sample-peering", "none"),
		Budget:          budgetConfig(cfg),
		ActivateApis: []string{
			"compute.googleapis.com",
			"dns.googleapis.com",
			"billingbudgets.googleapis.com",
			"logging.googleapis.com",
		},
	})
	if err != nil {
		return nil, err
	}
	result.PeeringProjectID = peeringProject.Project.ProjectId

	// Deploy peering network infrastructure (VPC, subnet, DNS, peering, firewall)
	if cfg.PeeringEnabled {
		peeringResult, err := deployPeeringNetwork(ctx, cfg, peeringProject, networkProjectID)
		if err != nil {
			return nil, err
		}
		result.PeeringNetworkSelfLink = peeringResult.NetworkSelfLink
		result.PeeringSubnetSelfLink = peeringResult.SubnetSelfLink
		result.IAPFirewallTags = peeringResult.IAPFirewallTags
	}

	// ========================================================================
	// 4. CMEK Storage — KMS keyring + encrypted GCS bucket on SVPC project
	// ========================================================================
	if cfg.CMEKEnabled {
		cmekResult, err := deployCMEKStorage(ctx, cfg, svpcProject, kmsProjectID)
		if err != nil {
			return nil, err
		}
		result.CMEKBucket = &cmekResult.BucketName
		result.CMEKKeyring = &cmekResult.KeyringName
	}

	// Populate TF-parity outputs
	if cfg.PeeringEnabled {
		result.SubnetsSelfLinks = pulumi.StringArray{result.PeeringSubnetSelfLink}.ToStringArrayOutput()
		result.PeeringComplete = pulumi.Bool(true).ToBoolOutput()
	} else {
		result.SubnetsSelfLinks = pulumi.ToStringArray([]string{}).ToStringArrayOutput()
		result.PeeringComplete = pulumi.Bool(false).ToBoolOutput()
	}
	result.VPCSCPerimeterName = perimeterName
	result.AccessContextManagerPolicyID = acmPolicyID

	return result, nil
}

// deployInfraPipelineProject creates the infrastructure pipeline project under
// the common folder. This project hosts the CI/CD pipeline for deploying
// application infrastructure (Stage 5).
func deployInfraPipelineProject(ctx *pulumi.Context, cfg *ProjectsConfig, commonFolderID pulumi.StringOutput) (pulumi.StringOutput, error) {
	infraProject, err := project.NewProject(ctx, "infra-pipeline-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-c-%s-infra-pipeline", cfg.ProjectPrefix, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-c-%s-infra-pipeline", cfg.ProjectPrefix, cfg.BusinessCode)),
		FolderID:        commonFolderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
		Labels:          projectLabels(cfg, "app-infra-pipelines", "none"),
		Budget:          budgetConfig(cfg),
		ActivateApis: []string{
			"cloudbuild.googleapis.com",
			"cloudkms.googleapis.com",
			"iam.googleapis.com",
			"artifactregistry.googleapis.com",
			"cloudresourcemanager.googleapis.com",
			"billingbudgets.googleapis.com",
			"confidentialcomputing.googleapis.com",
		},
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return infraProject.Project.ProjectId, nil
}
