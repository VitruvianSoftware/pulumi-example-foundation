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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadEnvConfig(ctx)

		// 1. Stack Reference to Organization
		orgStack, err := pulumi.NewStackReference(ctx, "organization", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.OrgStackName),
		})
		if err != nil {
			return err
		}

		// 2. Deploy Environment-specific Folder (if needed, though Org stage creates them)
		// Usually 2-environments stage manages shared projects in those folders.
		// For simplicity, we just export/use the folder ID.
		
		ctx.Export("env", pulumi.String(cfg.Env))
		return nil
	})
}

type EnvConfig struct {
	Env          string
	OrgID        string
	OrgStackName string
	FolderPrefix string
}

func loadEnvConfig(ctx *pulumi.Context) *EnvConfig {
	conf := config.New(ctx, "")
	return &EnvConfig{
		Env:          conf.Require("env"),
		OrgID:        conf.Require("org_id"),
		OrgStackName: conf.Require("org_stack_name"),
		FolderPrefix: conf.Get("folder_prefix"),
	}
}
