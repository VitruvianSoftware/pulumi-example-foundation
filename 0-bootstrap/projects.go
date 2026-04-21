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

	"github.com/VitruvianSoftware/pulumi-library/pkg/bootstrap"
	"github.com/VitruvianSoftware/pulumi-library/pkg/project"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SeedProject holds outputs from the seed project deployment.
type SeedProject struct {
	ProjectID              pulumi.StringOutput
	StateBucketName        pulumi.StringOutput
	ProjectsStateBucketName pulumi.StringOutput // Separate bucket for 4-projects stage
	KMSKeyID               pulumi.StringOutput
}

// CICDProject holds outputs from the CI/CD project deployment.
type CICDProject struct {
	ProjectID pulumi.StringOutput
}

// deploySeedProject creates the seed project that hosts Terraform/Pulumi state
// and the service accounts used by the foundation pipeline. This uses the
// Bootstrap component from the Vitruvian Pulumi Library, which mirrors the
// upstream terraform-google-modules/terraform-google-bootstrap module.
//
// The Bootstrap component handles:
//   - Seed project creation (with lien + default SA management)
//   - KMS key ring and crypto key for state encryption
//   - GCS state bucket with KMS encryption and versioning
//   - Org policy for cross-project SA usage
//   - State bucket IAM grants
func deploySeedProject(ctx *pulumi.Context, cfg *Config, folderID pulumi.StringOutput, bucketIAMMembers []pulumi.StringInput) (*SeedProject, error) {
	kmsPrevent := !cfg.BucketTFStateKMSForceDestroy
	b, err := bootstrap.NewBootstrap(ctx, "seed-bootstrap", &bootstrap.BootstrapArgs{
		OrgID:          cfg.OrgID,
		FolderID:       folderID,
		BillingAccount: cfg.BillingAccount,
		ProjectPrefix:  cfg.ProjectPrefix,
		DefaultRegion:  cfg.DefaultRegion,
		RandomSuffix:   cfg.RandomSuffix,
		ProjectLabels: pulumi.StringMap{
			"environment":       pulumi.String("bootstrap"),
			"application_name":  pulumi.String("seed-bootstrap"),
			"billing_code":      pulumi.String("1234"),
			"primary_contact":   pulumi.String("example1"),
			"secondary_contact": pulumi.String("example2"),
			"business_code":     pulumi.String("shared"),
			"env_code":          pulumi.String("b"),
			"vpc":               pulumi.String("none"),
		},
		ActivateApis: []string{
			"serviceusage.googleapis.com",
			"servicenetworking.googleapis.com",
			"cloudkms.googleapis.com",
			"compute.googleapis.com",
			"logging.googleapis.com",
			"bigquery.googleapis.com",
			"cloudresourcemanager.googleapis.com",
			"cloudbilling.googleapis.com",
			"cloudbuild.googleapis.com",
			"iam.googleapis.com",
			"admin.googleapis.com",
			"appengine.googleapis.com",
			"storage-api.googleapis.com",
			"monitoring.googleapis.com",
			"pubsub.googleapis.com",
			"securitycenter.googleapis.com",
			"accesscontextmanager.googleapis.com",
			"billingbudgets.googleapis.com",
			"essentialcontacts.googleapis.com",
			"assuredworkloads.googleapis.com",
			"cloudasset.googleapis.com",
		},

		// State Bucket
		BucketPrefix:       cfg.BucketPrefix,
		BucketForceDestroy: cfg.BucketForceDestroy,

		// KMS — matches TF foundation defaults
		KeyProtectionLevel: cfg.KMSKeyProtectionLevel,
		KMSPreventDestroy:  &kmsPrevent,

		// State bucket IAM — grant access to all pipeline SAs and org admins
		StateBucketIAMMembers: bucketIAMMembers,
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	// Create a separate state bucket for the 4-projects stage.
	// This isolates projects state from the main foundation state bucket.
	// Mirrors: module "gcp_projects_state_bucket" in build_github.tf.example
	projectsBucketName := b.SeedProjectID.ApplyT(func(id string) string {
		return fmt.Sprintf("%s-%s-gcp-projects-tfstate", cfg.BucketPrefix, id)
	}).(pulumi.StringOutput)

	projectsStateBucket, err := storage.NewBucket(ctx, "projects-state-bucket", &storage.BucketArgs{
		Name:                     projectsBucketName,
		Project:                  b.SeedProjectID,
		Location:                 pulumi.String(cfg.DefaultRegion),
		ForceDestroy:             pulumi.Bool(cfg.BucketForceDestroy),
		UniformBucketLevelAccess: pulumi.Bool(true),
		Versioning: &storage.BucketVersioningArgs{
			Enabled: pulumi.Bool(true),
		},
		Encryption: &storage.BucketEncryptionArgs{
			DefaultKmsKeyName: b.KMSKeyID,
		},
	}, pulumi.DependsOn([]pulumi.Resource{b}))
	if err != nil {
		return nil, err
	}

	return &SeedProject{
		ProjectID:               b.SeedProjectID,
		StateBucketName:         b.StateBucketName,
		ProjectsStateBucketName: projectsStateBucket.Name,
		KMSKeyID:                b.KMSKeyID,
	}, nil
}

// deployCICDProject creates the CI/CD project that hosts the pipeline
// infrastructure (Artifact Registry, Cloud Build, Workload Identity, etc.).
// This is the equivalent of prj-b-cicd in the Terraform foundation.
// The CI/CD project uses pkg/project directly (not pkg/bootstrap) because
// it doesn't need state bucket or KMS — those live in the seed project.
func deployCICDProject(ctx *pulumi.Context, cfg *Config, folderID pulumi.StringOutput) (*CICDProject, error) {
	cicd, err := project.NewProject(ctx, "cicd-project", &project.ProjectArgs{
		ProjectID:             pulumi.String(fmt.Sprintf("%s-b-cicd", cfg.ProjectPrefix)),
		Name:                  pulumi.String(fmt.Sprintf("%s-b-cicd", cfg.ProjectPrefix)),
		FolderID:              folderID,
		BillingAccount:        pulumi.String(cfg.BillingAccount),
		RandomProjectID:       cfg.RandomSuffix,
		DeletionPolicy:        pulumi.String(cfg.ProjectDeletionPolicy),
		DefaultServiceAccount: "disable",
		Labels: pulumi.StringMap{
			"environment":       pulumi.String("bootstrap"),
			"application_name":  pulumi.String("cloudbuild-bootstrap"),
			"billing_code":      pulumi.String("1234"),
			"primary_contact":   pulumi.String("example1"),
			"secondary_contact": pulumi.String("example2"),
			"business_code":     pulumi.String("shared"),
			"env_code":          pulumi.String("b"),
			"vpc":               pulumi.String("none"),
		},
		ActivateApis: []string{
			"serviceusage.googleapis.com",
			"servicenetworking.googleapis.com",
			"compute.googleapis.com",
			"logging.googleapis.com",
			"iam.googleapis.com",
			"iamcredentials.googleapis.com",
			"admin.googleapis.com",
			"artifactregistry.googleapis.com",
			"cloudbuild.googleapis.com",
			"cloudresourcemanager.googleapis.com",
			"cloudbilling.googleapis.com",
			"appengine.googleapis.com",
			"storage-api.googleapis.com",
			"billingbudgets.googleapis.com",
			"bigquery.googleapis.com",
			"dns.googleapis.com",
			"workflows.googleapis.com",
			"cloudscheduler.googleapis.com",
		},
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	return &CICDProject{
		ProjectID: cicd.Project.ProjectId,
	}, nil
}
