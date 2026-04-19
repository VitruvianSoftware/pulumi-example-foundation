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

	"github.com/VitruvianSoftware/pulumi-library/pkg/project"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BUProjects holds outputs from business unit project deployment.
type BUProjects struct {
	SVPCProjectID     pulumi.StringOutput
	FloatingProjectID pulumi.StringOutput
	PeeringProjectID  pulumi.StringOutput
}

// deployBusinessUnitProjects creates three project types per BU/env, matching
// the Terraform foundation's project factory pattern:
//   - SVPC-attached: connected to the Shared VPC host project
//   - Floating: standalone project, not attached to any VPC
//   - Peering: project that would peer with the host VPC
func deployBusinessUnitProjects(ctx *pulumi.Context, cfg *ProjectsConfig, folderID, networkProjectID pulumi.StringOutput) (*BUProjects, error) {
	standardAPIs := []string{
		"compute.googleapis.com",
		"container.googleapis.com",
		"run.googleapis.com",
		"artifactregistry.googleapis.com",
		"billingbudgets.googleapis.com",
		"logging.googleapis.com",
	}

	// ========================================================================
	// 1. SVPC-attached Project
	// This project is attached as a service project to the environment's
	// Shared VPC host, enabling shared network resource access.
	// ========================================================================
	svpcProject, err := project.NewProject(ctx, "bu-svpc-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-sample-svpc", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-sample-svpc", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: true,
		ActivateApis:    standardAPIs,
	})
	if err != nil {
		return nil, err
	}

	// Attach as a Shared VPC service project
	if _, err := compute.NewSharedVPCServiceProject(ctx, "svpc-attachment", &compute.SharedVPCServiceProjectArgs{
		HostProject:    networkProjectID,
		ServiceProject: svpcProject.Project.ProjectId,
	}); err != nil {
		return nil, err
	}

	// ========================================================================
	// 2. Floating Project (not attached to any VPC)
	// ========================================================================
	floatingProject, err := project.NewProject(ctx, "bu-floating-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-sample-floating", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-sample-floating", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: true,
		ActivateApis:    standardAPIs,
	})
	if err != nil {
		return nil, err
	}

	// ========================================================================
	// 3. Peering Project
	// ========================================================================
	peeringProject, err := project.NewProject(ctx, "bu-peering-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-sample-peering", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-sample-peering", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: true,
		ActivateApis:    standardAPIs,
	})
	if err != nil {
		return nil, err
	}

	return &BUProjects{
		SVPCProjectID:     svpcProject.Project.ProjectId,
		FloatingProjectID: floatingProject.Project.ProjectId,
		PeeringProjectID:  peeringProject.Project.ProjectId,
	}, nil
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
		RandomProjectID: true,
		ActivateApis: []string{
			"cloudbuild.googleapis.com",
			"artifactregistry.googleapis.com",
			"iam.googleapis.com",
			"cloudresourcemanager.googleapis.com",
			"billingbudgets.googleapis.com",
		},
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return infraProject.Project.ProjectId, nil
}
