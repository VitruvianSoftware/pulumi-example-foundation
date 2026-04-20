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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Folders struct {
	Common  *organizations.Folder
	Network *organizations.Folder
}

// deployFolders creates the top-level organization folders (Common and Network).
// This mirrors the Terraform foundation's 1-org/envs/shared/folders.tf exactly.
//
// Environment folders (development, nonproduction, production) are created in
// the 2-environments stage, matching the upstream architecture where each stage
// owns its specific resources.
func deployFolders(ctx *pulumi.Context, cfg *OrgConfig) (*Folders, error) {
	// Folder resource options — apply deletion protection when enabled (D11)
	var folderOpts []pulumi.ResourceOption
	if cfg.FolderDeletionProtection {
		folderOpts = append(folderOpts, pulumi.Protect(true))
	}

	// Common Folder
	common, err := organizations.NewFolder(ctx, "common-folder", &organizations.FolderArgs{
		DisplayName:        pulumi.String(fmt.Sprintf("%s-common", cfg.FolderPrefix)),
		Parent:             pulumi.String(cfg.Parent),
		DeletionProtection: pulumi.Bool(cfg.FolderDeletionProtection),
	}, folderOpts...)
	if err != nil {
		return nil, err
	}

	// Network Folder
	network, err := organizations.NewFolder(ctx, "network-folder", &organizations.FolderArgs{
		DisplayName:        pulumi.String(fmt.Sprintf("%s-network", cfg.FolderPrefix)),
		Parent:             pulumi.String(cfg.Parent),
		DeletionProtection: pulumi.Bool(cfg.FolderDeletionProtection),
	}, folderOpts...)
	if err != nil {
		return nil, err
	}

	return &Folders{
		Common:  common,
		Network: network,
	}, nil
}

