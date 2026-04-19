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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/tags"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployTags creates org-level tag keys and values for environment
// classification. Tags enable fine-grained IAM conditions and resource
// organization across the foundation hierarchy.
// This mirrors the Terraform foundation's tags.tf.
func deployTags(ctx *pulumi.Context, cfg *OrgConfig) error {
	parent := "organizations/" + cfg.OrgID
	if cfg.ParentFolder != "" {
		parent = "folders/" + cfg.ParentFolder
	}

	// Environment tag key
	envTagKey, err := tags.NewTagKey(ctx, "environment-tag", &tags.TagKeyArgs{
		Parent:      pulumi.String(parent),
		ShortName:   pulumi.String("environment"),
		Description: pulumi.String("Environment classification for foundation resources"),
	})
	if err != nil {
		return err
	}

	// Tag values for each lifecycle stage
	envValues := []string{"bootstrap", "common", "development", "nonproduction", "production"}
	for _, env := range envValues {
		if _, err := tags.NewTagValue(ctx, "tag-value-"+env, &tags.TagValueArgs{
			Parent:      envTagKey.ID(),
			ShortName:   pulumi.String(env),
			Description: pulumi.String(env + " environment"),
		}); err != nil {
			return err
		}
	}

	return nil
}
