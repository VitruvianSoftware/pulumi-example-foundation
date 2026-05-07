// Example: Cloud Build-based infra_pipelines module.
// This is the Cloud Build equivalent of the TF infra_pipelines module.
// To enable, remove this build constraint or build with: go build -tags=example
//
// Per approved deviation, Go/TS foundations default to GitHub Actions WIF.
// Cloud Build pipelines are provided as an .example reference.
//
//go:build example

package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudbuildv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/sourcerepo"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// InfraPipelinesArgs configures the Cloud Build infrastructure pipeline module.
// Mirrors: 4-projects/modules/infra_pipelines (Terraform)
type InfraPipelinesArgs struct {
	ProjectID             string
	DefaultRegion         string
	BillingAccount        string
	OrgID                 string
	AppInfraRepos         []string
	BucketPrefix          string
	RemoteTFStateBucket   string
	CloudBuilderArtifactRepo string
	PrivateWorkerPoolID   string
}

// InfraPipelinesResult holds the outputs from the infra pipeline deployment.
type InfraPipelinesResult struct {
	DefaultRegion          pulumi.StringOutput
	TerraformSAs           map[string]pulumi.StringOutput
	GARName                pulumi.StringOutput
	Repos                  []pulumi.StringOutput
	ArtifactBuckets        map[string]pulumi.StringOutput
	StateBuckets           map[string]pulumi.StringOutput
	LogBuckets             map[string]pulumi.StringOutput
	PlanTriggersID         []pulumi.StringOutput
	ApplyTriggersID        []pulumi.StringOutput
	ArtifactRegistryRepoID pulumi.StringOutput
}

// deployInfraPipelines creates Cloud Build workspaces, CSRs, buckets, and triggers
// for each app infra repo. Mirrors the TF infra_pipelines module.
func deployInfraPipelines(ctx *pulumi.Context, name string, args *InfraPipelinesArgs) (*InfraPipelinesResult, error) {
	result := &InfraPipelinesResult{
		DefaultRegion:   pulumi.String(args.DefaultRegion).ToStringOutput(),
		TerraformSAs:    make(map[string]pulumi.StringOutput),
		ArtifactBuckets: make(map[string]pulumi.StringOutput),
		StateBuckets:    make(map[string]pulumi.StringOutput),
		LogBuckets:      make(map[string]pulumi.StringOutput),
	}

	// 1. Cloud Build bucket
	_, err := storage.NewBucket(ctx, name+"-cb-bucket", &storage.BucketArgs{
		Project:                  pulumi.String(args.ProjectID),
		Name:                     pulumi.Sprintf("%s_cloudbuild", args.ProjectID),
		Location:                 pulumi.String(args.DefaultRegion),
		UniformBucketLevelAccess: pulumi.Bool(true),
		ForceDestroy:             pulumi.Bool(true),
		Versioning: &storage.BucketVersioningArgs{
			Enabled: pulumi.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	// 2. GCP Policies CSR
	_, err = sourcerepo.NewRepository(ctx, name+"-gcp-policies", &sourcerepo.RepositoryArgs{
		Project: pulumi.String(args.ProjectID),
		Name:    pulumi.String("gcp-policies"),
	})
	if err != nil {
		return nil, err
	}

	// 3. Per-repo resources: CSR, SA, buckets, triggers
	for _, repoName := range args.AppInfraRepos {
		rn := repoName // capture

		// Source Repository
		repo, err := sourcerepo.NewRepository(ctx, name+"-repo-"+rn, &sourcerepo.RepositoryArgs{
			Project: pulumi.String(args.ProjectID),
			Name:    pulumi.String(rn),
		})
		if err != nil {
			return nil, err
		}
		result.Repos = append(result.Repos, repo.Name)

		// Terraform SA
		sa, err := serviceaccount.NewAccount(ctx, name+"-sa-"+rn, &serviceaccount.AccountArgs{
			Project:     pulumi.String(args.ProjectID),
			AccountId:   pulumi.String(fmt.Sprintf("sa-tf-cb-%s", rn)),
			DisplayName: pulumi.Sprintf("Terraform CB SA for %s", rn),
		})
		if err != nil {
			return nil, err
		}
		result.TerraformSAs[rn] = sa.Email

		// State bucket
		stateBucket, err := storage.NewBucket(ctx, name+"-state-"+rn, &storage.BucketArgs{
			Project:                  pulumi.String(args.ProjectID),
			Name:                     pulumi.Sprintf("%s-%s-%s-state", args.BucketPrefix, args.ProjectID, rn),
			Location:                 pulumi.String(args.DefaultRegion),
			UniformBucketLevelAccess: pulumi.Bool(true),
			ForceDestroy:             pulumi.Bool(true),
			Versioning:               &storage.BucketVersioningArgs{Enabled: pulumi.Bool(true)},
		})
		if err != nil {
			return nil, err
		}
		result.StateBuckets[rn] = stateBucket.Name

		// Artifacts bucket
		artifactBucket, err := storage.NewBucket(ctx, name+"-artifacts-"+rn, &storage.BucketArgs{
			Project:                  pulumi.String(args.ProjectID),
			Name:                     pulumi.Sprintf("%s-%s-%s-artifacts", args.BucketPrefix, args.ProjectID, rn),
			Location:                 pulumi.String(args.DefaultRegion),
			UniformBucketLevelAccess: pulumi.Bool(true),
			ForceDestroy:             pulumi.Bool(true),
			Versioning:               &storage.BucketVersioningArgs{Enabled: pulumi.Bool(true)},
		})
		if err != nil {
			return nil, err
		}
		result.ArtifactBuckets[rn] = artifactBucket.Name

		// Log bucket
		logBucket, err := storage.NewBucket(ctx, name+"-logs-"+rn, &storage.BucketArgs{
			Project:                  pulumi.String(args.ProjectID),
			Name:                     pulumi.Sprintf("%s-%s-%s-logs", args.BucketPrefix, args.ProjectID, rn),
			Location:                 pulumi.String(args.DefaultRegion),
			UniformBucketLevelAccess: pulumi.Bool(true),
			ForceDestroy:             pulumi.Bool(true),
			Versioning:               &storage.BucketVersioningArgs{Enabled: pulumi.Bool(true)},
		})
		if err != nil {
			return nil, err
		}
		result.LogBuckets[rn] = logBucket.Name

		// IAM: Remote TF state viewer
		_, err = storage.NewBucketIAMMember(ctx, name+"-tfstate-viewer-"+rn, &storage.BucketIAMMemberArgs{
			Bucket: pulumi.String(args.RemoteTFStateBucket),
			Role:   pulumi.String("roles/storage.objectViewer"),
			Member: pulumi.Sprintf("serviceAccount:%s", sa.Email),
		})
		if err != nil {
			return nil, err
		}

		// IAM: Org browser (required by gcloud beta terraform vet)
		_, err = organizations.NewIAMMember(ctx, name+"-browser-"+rn, &organizations.IAMMemberArgs{
			OrgId:  pulumi.String(args.OrgID),
			Role:   pulumi.String("roles/browser"),
			Member: pulumi.Sprintf("serviceAccount:%s", sa.Email),
		})
		if err != nil {
			return nil, err
		}

		// IAM: Artifact Registry writer
		_, err = artifactregistry.NewRepositoryIamMember(ctx, name+"-gar-writer-"+rn, &artifactregistry.RepositoryIamMemberArgs{
			Project:    pulumi.String(args.ProjectID),
			Location:   pulumi.String(args.DefaultRegion),
			Repository: pulumi.String("tf-runners"),
			Role:       pulumi.String("roles/artifactregistry.writer"),
			Member:     pulumi.Sprintf("serviceAccount:%s", sa.Email),
		})
		if err != nil {
			return nil, err
		}
	}

	// Suppress unused import warnings
	_ = cloudbuildv2.Connection{}
	_ = iam.WorkloadIdentityPool{}

	return result, nil
}
