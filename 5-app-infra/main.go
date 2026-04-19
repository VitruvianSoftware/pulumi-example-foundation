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
	"github.com/VitruvianSoftware/pulumi-library/pkg/app"
	"github.com/VitruvianSoftware/pulumi-library/pkg/data"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadAppInfraConfig(ctx)

		// 1. Resolve Project IDs from the Stage 4 stack.
		// Stage 4 exports: svpc_project_id, floating_project_id,
		// peering_project_id, infra_pipeline_project_id.
		// We use the SVPC project for the app and the floating project for data,
		// matching a typical pattern where apps run in the VPC-attached project.
		projStack, err := pulumi.NewStackReference(ctx, "projects", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.ProjectsStackName),
		})
		if err != nil {
			return err
		}

		// Use GetStringOutput (returns a typed StringOutput) instead of
		// GetOutput + unsafe type assertion which panics on nil.
		appProjectID := projStack.GetStringOutput(pulumi.String("svpc_project_id"))
		dataProjectID := projStack.GetStringOutput(pulumi.String("floating_project_id"))

		// 2. Deploy Data Platform using the reusable Data component
		_, err = data.NewDataPlatform(ctx, "airline-data", &data.DataPlatformArgs{
			ProjectID: dataProjectID,
			Location:  pulumi.String("US"),
		})
		if err != nil {
			return err
		}

		// 3. Deploy Web App using the reusable App component (Cloud Run v2)
		_, err = app.NewCloudRunApp(ctx, "chat-demo", &app.CloudRunAppArgs{
			ProjectID: appProjectID,
			Name:      pulumi.String("chat-demo"),
			Image:     pulumi.String("us-docker.pkg.dev/cloudrun/container/hello"),
			Region:    pulumi.String(cfg.Region),
		})
		if err != nil {
			return err
		}

		return nil
	})
}

type AppInfraConfig struct {
	Env               string
	Region            string
	ProjectsStackName string
}

func loadAppInfraConfig(ctx *pulumi.Context) *AppInfraConfig {
	conf := config.New(ctx, "")
	c := &AppInfraConfig{
		Env:               conf.Require("env"),
		Region:            conf.Get("region"),
		ProjectsStackName: conf.Require("projects_stack_name"),
	}
	if c.Region == "" {
		c.Region = "us-central1"
	}
	return c
}
