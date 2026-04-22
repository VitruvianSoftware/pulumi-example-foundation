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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/accesscontextmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	gcpproject "github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ConfidentialSpaceResult holds outputs from the Confidential Space project.
type ConfidentialSpaceResult struct {
	ProjectID       pulumi.StringOutput
	ProjectNumber   pulumi.StringOutput
	WorkloadSAEmail pulumi.StringOutput
}

// deployConfidentialSpaceProject creates a Confidential Space project with a
// workload service account, matching upstream's example_confidential_space_project.tf.
//
// Unlike upstream, which gates this on enable_cloudbuild_deploy, we use a
// dedicated toggle (confidential_space_enabled) to keep the project independent
// of the CI/CD platform choice.
//
// Creates:
//   - Project attached to Shared VPC host with VPC-SC perimeter
//   - Workload Service Account for Confidential Space
//   - IAM role bindings for the workload SA
func deployConfidentialSpaceProject(
	ctx *pulumi.Context,
	cfg *ProjectsConfig,
	folderID pulumi.StringOutput,
	networkProjectID pulumi.StringOutput,
	perimeterName pulumi.StringOutput,
) (*ConfidentialSpaceResult, error) {

	// 1. Create the Confidential Space project
	confProject, err := project.NewProject(ctx, "bu-conf-space-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-%s-%s-conf-space", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		Name:            pulumi.String(fmt.Sprintf("%s-%s-%s-conf-space", cfg.ProjectPrefix, cfg.EnvCode, cfg.BusinessCode)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
		Labels:          projectLabels(cfg, "sample-instance", "svpc"),
		Budget:          budgetConfig(cfg),
		ActivateApis: []string{
			"accesscontextmanager.googleapis.com",
			"artifactregistry.googleapis.com",
			"iamcredentials.googleapis.com",
			"compute.googleapis.com",
			"confidentialcomputing.googleapis.com",
			"cloudkms.googleapis.com",
			"billingbudgets.googleapis.com",
		},
	})
	if err != nil {
		return nil, err
	}

	// 2. Attach as a Shared VPC service project
	if _, err := compute.NewSharedVPCServiceProject(ctx, "conf-space-svpc-attachment", &compute.SharedVPCServiceProjectArgs{
		HostProject:    networkProjectID,
		ServiceProject: confProject.Project.ProjectId,
	}); err != nil {
		return nil, err
	}

	// 3. VPC-SC Perimeter attachment
	if cfg.EnforceVpcSc {
		_, err := accesscontextmanager.NewServicePerimeterResource(ctx, "conf-space-vpcsc-attach", &accesscontextmanager.ServicePerimeterResourceArgs{
			PerimeterName: perimeterName,
			Resource: confProject.Project.Number.ApplyT(func(n string) string {
				return fmt.Sprintf("projects/%s", n)
			}).(pulumi.StringOutput),
		})
		if err != nil {
			return nil, err
		}
	}

	// 4. Workload Service Account for Confidential Space
	workloadSA, err := serviceaccount.NewAccount(ctx, "conf-space-workload-sa", &serviceaccount.AccountArgs{
		AccountId:   pulumi.String("confidential-space-workload-sa"),
		DisplayName: pulumi.String("Workload Service Account for confidential space"),
		Project:     confProject.Project.ProjectId,
	})
	if err != nil {
		return nil, err
	}

	// 5. IAM role bindings for the workload SA (matching upstream iam_roles local)
	workloadRoles := []string{
		"roles/iam.serviceAccountUser",
		"roles/confidentialcomputing.workloadUser",
		"roles/iam.workloadIdentityPoolAdmin",
		"roles/storage.admin",
		"roles/logging.logWriter",
	}
	for _, role := range workloadRoles {
		_, err := gcpproject.NewIAMMember(ctx, fmt.Sprintf("conf-space-sa-%s", role), &gcpproject.IAMMemberArgs{
			Project: confProject.Project.ProjectId,
			Role:    pulumi.String(role),
			Member:  pulumi.Sprintf("serviceAccount:%s", workloadSA.Email),
		})
		if err != nil {
			return nil, err
		}
	}

	return &ConfidentialSpaceResult{
		ProjectID:       confProject.Project.ProjectId,
		ProjectNumber:   confProject.Project.Number,
		WorkloadSAEmail: workloadSA.Email,
	}, nil
}
