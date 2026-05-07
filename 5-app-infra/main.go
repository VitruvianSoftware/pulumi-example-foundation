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

// 5-app-infra is the scaffold for deploying application infrastructure.
//
// This file provides the project scaffold (stack references, config loading).
// Actual compute workloads (VMs, confidential space) are defined in the
// example_*.go files which are excluded from compilation by default.
//
// To enable example workloads, remove the //go:build example constraint
// from the example files, or build with: go build -tags=example
package main

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadAppInfraConfig(ctx)

		// 1. Stack Reference: 4-projects (per-environment)
		projStack, err := pulumi.NewStackReference(ctx, "projects", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.ProjectsStackName),
		})
		if err != nil {
			return err
		}

		// 2. Stack Reference: 0-bootstrap (shared / common — not per-environment)
		_, err = pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.BootstrapStackName),
		})
		if err != nil {
			return err
		}

		// --- Resolve outputs from 4-projects ---
		appProjectID := projStack.GetStringOutput(pulumi.String("shared_vpc_project"))

		// 3. Scaffold Exports — matching TF 5-app-infra outputs structure
		// The project scaffold exports are always available. Compute workload
		// exports are added by the example files when enabled.
		ctx.Export("project_id", appProjectID)
		ctx.Export("region", pulumi.String(cfg.Region))

		return nil
	})
}

type AppInfraConfig struct {
	Env                    string
	EnvCode                string
	BusinessCode           string
	Region                 string
	ProjectsStackName      string
	BootstrapStackName     string
	ConfidentialImageDigest string
}

func loadAppInfraConfig(ctx *pulumi.Context) *AppInfraConfig {
	conf := config.New(ctx, "")
	c := &AppInfraConfig{
		Env:                    conf.Require("env"),
		BusinessCode:           conf.Get("business_code"),
		Region:                 conf.Get("region"),
		ProjectsStackName:      conf.Get("projects_stack_name"),
		BootstrapStackName:     conf.Get("bootstrap_stack_name"),
		ConfidentialImageDigest: conf.Get("confidential_image_digest"),
	}
	if c.Region == "" {
		c.Region = "us-central1"
	}
	if c.BusinessCode == "" {
		c.BusinessCode = "bu1"
	}
	if c.ProjectsStackName == "" {
		c.ProjectsStackName = fmt.Sprintf("VitruvianSoftware/foundation-4-projects/%s", c.Env)
	}
	if c.BootstrapStackName == "" {
		// Bootstrap is a shared stage — use the org_stack_name pattern with
		// the same naming convention as other stages. Fall back to a
		// default derived from the projects stack name.
		c.BootstrapStackName = strings.Replace(c.ProjectsStackName, "foundation-4-projects/"+c.Env, "foundation-0-bootstrap/shared", 1)
	}
	envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}
	c.EnvCode = envCodes[c.Env]
	if c.EnvCode == "" {
		c.EnvCode = c.Env[:1]
	}
	return c
}
