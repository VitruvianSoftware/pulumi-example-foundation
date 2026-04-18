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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type SeedProject struct {
	ProjectID       pulumi.StringOutput
	StateBucketName pulumi.StringOutput
}

func deploySeedProject(ctx *pulumi.Context, cfg *Config, folderID pulumi.StringOutput) (*SeedProject, error) {
	// 1. Seed Project using the reusable Project Component
	seed, err := project.NewProject(ctx, "seed-project", &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-b-seed", cfg.ProjectPrefix)),
		Name:           pulumi.String(fmt.Sprintf("%s-b-seed", cfg.ProjectPrefix)),
		FolderID:       folderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		ActivateApis: pulumi.StringArray{
			pulumi.String("serviceusage.googleapis.com"),
			pulumi.String("cloudresourcemanager.googleapis.com"),
			pulumi.String("cloudbilling.googleapis.com"),
			pulumi.String("cloudbuild.googleapis.com"),
			pulumi.String("iam.googleapis.com"),
			pulumi.String("storage-api.googleapis.com"),
		},
	})
	if err != nil {
		return nil, err
	}

	// 2. State Bucket
	stateBucket, err := storage.NewBucket(ctx, "tf-state-bucket", &storage.BucketArgs{
		Project:  seed.Project.ProjectId,
		Name:     pulumi.String(fmt.Sprintf("%s-%s-b-seed-tfstate", cfg.BucketPrefix, cfg.ProjectPrefix)),
		Location: pulumi.String("US"),
		Versioning: &storage.BucketVersioningArgs{
			Enabled: pulumi.Bool(true),
		},
	}, pulumi.Parent(seed.Project))
	if err != nil {
		return nil, err
	}

	return &SeedProject{
		ProjectID:       seed.Project.ProjectId,
		StateBucketName: stateBucket.Name,
	}, nil
}
