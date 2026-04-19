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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/tags"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployTags creates org-level tag keys and values for environment
// classification, and binds them to the common, network, and bootstrap folders.
// Tags enable fine-grained IAM conditions and resource organization across
// the foundation hierarchy.
// This mirrors the Terraform foundation's tags.tf.
func deployTags(ctx *pulumi.Context, cfg *OrgConfig, folders *Folders) (pulumi.MapOutput, error) {
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
		return pulumi.MapOutput{}, err
	}

	// Tag values for each lifecycle stage
	envValues := []string{"bootstrap", "common", "development", "nonproduction", "production"}
	tagValueMap := make(map[string]*tags.TagValue)
	tagOutputMap := make(pulumi.Map)

	for _, env := range envValues {
		tv, err := tags.NewTagValue(ctx, "tag-value-"+env, &tags.TagValueArgs{
			Parent:      envTagKey.ID(),
			ShortName:   pulumi.String(env),
			Description: pulumi.String(env + " environment"),
		})
		if err != nil {
			return pulumi.MapOutput{}, err
		}
		tagValueMap[env] = tv
		tagOutputMap[fmt.Sprintf("environment_%s", env)] = tv.ID()
	}

	// ========================================================================
	// Folder Tag Bindings (D13)
	// Bind environment tags to foundation folders, mirroring TF tags.tf.
	// ========================================================================

	// Common folder → production tag (shared infra is production-grade)
	if _, err := tags.NewTagBinding(ctx, "tag-binding-common", &tags.TagBindingArgs{
		Parent: folders.Common.Name.ApplyT(func(name string) string {
			return fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s", name)
		}).(pulumi.StringOutput),
		TagValue: tagValueMap["production"].ID(),
	}); err != nil {
		return pulumi.MapOutput{}, err
	}

	// Network folder → production tag
	if _, err := tags.NewTagBinding(ctx, "tag-binding-network", &tags.TagBindingArgs{
		Parent: folders.Network.Name.ApplyT(func(name string) string {
			return fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s", name)
		}).(pulumi.StringOutput),
		TagValue: tagValueMap["production"].ID(),
	}); err != nil {
		return pulumi.MapOutput{}, err
	}

	// Bootstrap folder → bootstrap tag (when bootstrap_folder_name is provided)
	if cfg.BootstrapFolderName != "" {
		if _, err := tags.NewTagBinding(ctx, "tag-binding-bootstrap", &tags.TagBindingArgs{
			Parent:   pulumi.Sprintf("//cloudresourcemanager.googleapis.com/%s", cfg.BootstrapFolderName),
			TagValue: tagValueMap["bootstrap"].ID(),
		}); err != nil {
			return pulumi.MapOutput{}, err
		}
	}

	return tagOutputMap.ToMapOutput(), nil
}
