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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/kms"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SeedProject holds outputs from the seed project deployment.
type SeedProject struct {
	ProjectID       pulumi.StringOutput
	StateBucketName pulumi.StringOutput
	KMSKeyID        pulumi.StringOutput
}

// CICDProject holds outputs from the CI/CD project deployment.
type CICDProject struct {
	ProjectID pulumi.StringOutput
}

// deploySeedProject creates the seed project that hosts Terraform/Pulumi state
// and the service accounts used by the foundation pipeline. This is the
// equivalent of prj-b-seed in the Terraform foundation.
func deploySeedProject(ctx *pulumi.Context, cfg *Config, folderID pulumi.StringOutput) (*SeedProject, error) {
	// The seed project activates all APIs required for foundation management.
	// This full list matches the Terraform foundation's bootstrap module.
	seed, err := project.NewProject(ctx, "seed-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-b-seed", cfg.ProjectPrefix)),
		Name:            pulumi.String(fmt.Sprintf("%s-b-seed", cfg.ProjectPrefix)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
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
	})
	if err != nil {
		return nil, err
	}

	// KMS Key Ring and Crypto Key for state bucket encryption.
	// Rotation period is 90 days, matching GCP security best practices.
	keyRing, err := kms.NewKeyRing(ctx, "state-bucket-keyring", &kms.KeyRingArgs{
		Project:  seed.Project.ProjectId,
		Name:     pulumi.String(fmt.Sprintf("%s-keyring", cfg.ProjectPrefix)),
		Location: pulumi.String(cfg.DefaultRegion),
	})
	if err != nil {
		return nil, err
	}

	cryptoKey, err := kms.NewCryptoKey(ctx, "state-bucket-key", &kms.CryptoKeyArgs{
		Name:           pulumi.String(fmt.Sprintf("%s-key", cfg.ProjectPrefix)),
		KeyRing:        keyRing.ID(),
		RotationPeriod: pulumi.String("7776000s"), // 90 days
	})
	if err != nil {
		return nil, err
	}

	// When RandomSuffix is enabled, append a random hex suffix to the bucket
	// name. This matches the upstream bootstrap module which uses a separate
	// random_id resource (byte_length=2) for the GCS bucket name.
	var stateBucketName pulumi.StringInput
	if cfg.RandomSuffix {
		bucketSuffix, err := random.NewRandomId(ctx, "state-bucket-suffix", &random.RandomIdArgs{
			ByteLength: pulumi.Int(2),
		})
		if err != nil {
			return nil, err
		}
		stateBucketName = pulumi.Sprintf("%s-%s-b-seed-tfstate-%s", cfg.BucketPrefix, cfg.ProjectPrefix, bucketSuffix.Hex)
	} else {
		stateBucketName = pulumi.String(fmt.Sprintf("%s-%s-b-seed-tfstate", cfg.BucketPrefix, cfg.ProjectPrefix))
	}

	// State bucket with KMS encryption, versioning, and uniform bucket-level access.
	stateBucket, err := storage.NewBucket(ctx, "tf-state-bucket", &storage.BucketArgs{
		Project:                  seed.Project.ProjectId,
		Name:                     stateBucketName,
		Location:                 pulumi.String(cfg.DefaultRegionGCS),
		UniformBucketLevelAccess: pulumi.Bool(true),
		ForceDestroy:             pulumi.Bool(cfg.BucketForceDestroy),
		Versioning: &storage.BucketVersioningArgs{
			Enabled: pulumi.Bool(true),
		},
		Encryption: &storage.BucketEncryptionArgs{
			DefaultKmsKeyName: cryptoKey.ID(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &SeedProject{
		ProjectID:       seed.Project.ProjectId,
		StateBucketName: stateBucket.Name,
		KMSKeyID:        cryptoKey.ID(),
	}, nil
}

// deployCICDProject creates the CI/CD project that hosts the pipeline
// infrastructure (Artifact Registry, Cloud Build, Workload Identity, etc.).
// This is the equivalent of prj-b-cicd in the Terraform foundation.
func deployCICDProject(ctx *pulumi.Context, cfg *Config, folderID pulumi.StringOutput) (*CICDProject, error) {
	cicd, err := project.NewProject(ctx, "cicd-project", &project.ProjectArgs{
		ProjectID:       pulumi.String(fmt.Sprintf("%s-b-cicd", cfg.ProjectPrefix)),
		Name:            pulumi.String(fmt.Sprintf("%s-b-cicd", cfg.ProjectPrefix)),
		FolderID:        folderID,
		BillingAccount:  pulumi.String(cfg.BillingAccount),
		RandomProjectID: cfg.RandomSuffix,
		ActivateApis: []string{
			"serviceusage.googleapis.com",
			"servicenetworking.googleapis.com",
			"compute.googleapis.com",
			"logging.googleapis.com",
			"iam.googleapis.com",
			"admin.googleapis.com",
			"artifactregistry.googleapis.com",
			"cloudbuild.googleapis.com",
			"cloudresourcemanager.googleapis.com",
			"cloudbilling.googleapis.com",
			"appengine.googleapis.com",
			"storage-api.googleapis.com",
			"billingbudgets.googleapis.com",
			"dns.googleapis.com",
			"workflows.googleapis.com",
			"cloudscheduler.googleapis.com",
		},
	})
	if err != nil {
		return nil, err
	}

	return &CICDProject{
		ProjectID: cicd.Project.ProjectId,
	}, nil
}
