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

	libproject "github.com/VitruvianSoftware/pulumi-library/go/pkg/project"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/kms"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/storage"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// CMEKResult holds outputs from the CMEK storage deployment.
type CMEKResult struct {
	BucketName  pulumi.StringOutput
	KeyringName pulumi.StringOutput
}

// deployCMEKStorage creates a KMS keyring with a crypto key and a CMEK-encrypted
// GCS bucket on the SVPC project, matching upstream's example_storage_cmek.tf.
//
// Upstream creates:
//   - KMS keyring (in a separate KMS project, but we use the SVPC project)
//   - Crypto key with rotation period
//   - IAM binding for the GCS service account as encrypter/decrypter
//   - CMEK-encrypted GCS bucket
func deployCMEKStorage(
	ctx *pulumi.Context,
	cfg *ProjectsConfig,
	svpcProject *libproject.Project,
	kmsProjectID pulumi.StringOutput,
) (*CMEKResult, error) {
	projectID := svpcProject.Project.ProjectId

	// 1. KMS Keyring
	keyring, err := kms.NewKeyRing(ctx, "cmek-keyring", &kms.KeyRingArgs{
		Name:     pulumi.String(cfg.KeyringName),
		Location: pulumi.String(cfg.KMSLocation),
		Project:  kmsProjectID,
	})
	if err != nil {
		return nil, err
	}

	// 2. Crypto Key with rotation
	cryptoKey, err := kms.NewCryptoKey(ctx, "cmek-crypto-key", &kms.CryptoKeyArgs{
		Name:            pulumi.String(cfg.KeyName),
		KeyRing:         keyring.ID(),
		RotationPeriod:  pulumi.String(cfg.KeyRotationPeriod),
		Purpose:         pulumi.String("ENCRYPT_DECRYPT"),
	}, pulumi.Parent(keyring))
	if err != nil {
		return nil, err
	}

	// 3. Grant the GCS service account permission to use the crypto key
	// The GCS service account is service-{project_number}@gs-project-accounts.iam.gserviceaccount.com
	gcsServiceAccount := svpcProject.Project.Number.ApplyT(func(n string) string {
		return fmt.Sprintf("serviceAccount:service-%s@gs-project-accounts.iam.gserviceaccount.com", n)
	}).(pulumi.StringOutput)

	_, err = kms.NewCryptoKeyIAMMember(ctx, "cmek-gcs-encrypter", &kms.CryptoKeyIAMMemberArgs{
		CryptoKeyId: cryptoKey.ID(),
		Role:        pulumi.String("roles/cloudkms.cryptoKeyEncrypterDecrypter"),
		Member:      gcsServiceAccount,
	}, pulumi.Parent(cryptoKey))
	if err != nil {
		return nil, err
	}

	// 4. Random suffix for bucket name uniqueness (matches upstream random_string)
	bucketSuffix, err := random.NewRandomString(ctx, "cmek-bucket-suffix", &random.RandomStringArgs{
		Length:  pulumi.Int(5),
		Upper:   pulumi.Bool(false),
		Special: pulumi.Bool(false),
		Numeric: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	// 5. CMEK-encrypted GCS bucket
	bucketName := pulumi.All(projectID, bucketSuffix.Result).ApplyT(func(vals []interface{}) string {
		return fmt.Sprintf("%s-%s-cmek-encrypted-%s", cfg.GCSBucketPrefix, vals[0], vals[1])
	}).(pulumi.StringOutput)

	bucketArgs := &storage.BucketArgs{
		Project:                  projectID,
		Name:                     bucketName,
		Location:                 pulumi.String(cfg.GCSLocation),
		UniformBucketLevelAccess: pulumi.Bool(true),
		Encryption: &storage.BucketEncryptionArgs{
			DefaultKmsKeyName: cryptoKey.ID(),
		},
	}

	// Custom placement config for dual-region buckets (matches upstream)
	if len(cfg.GCSPlacementRegions) == 2 {
		bucketArgs.CustomPlacementConfig = &storage.BucketCustomPlacementConfigArgs{
			DataLocations: pulumi.ToStringArray(cfg.GCSPlacementRegions),
		}
	}

	bucket, err := storage.NewBucket(ctx, "cmek-bucket", bucketArgs,
		pulumi.DependsOn([]pulumi.Resource{cryptoKey}))
	if err != nil {
		return nil, err
	}

	return &CMEKResult{
		BucketName:  bucket.Name,
		KeyringName: keyring.Name,
	}, nil
}
