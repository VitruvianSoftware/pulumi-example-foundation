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
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type BUProjects struct {
	AppProjectID  pulumi.StringOutput
	DataProjectID pulumi.StringOutput
}

func deployBusinessUnitProjects(ctx *pulumi.Context, cfg *ProjectsConfig, folderID pulumi.StringOutput) (*BUProjects, error) {
	if cfg.ProjectPrefix == "" {
		cfg.ProjectPrefix = "prj"
	}

	// 1. App Project using Project Component
	app, err := project.NewProject(ctx, "app-project", &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-%s-%s-app", cfg.ProjectPrefix, cfg.Env, cfg.BusinessCode)),
		Name:           pulumi.String(fmt.Sprintf("%s-%s-%s-app", cfg.ProjectPrefix, cfg.Env, cfg.BusinessCode)),
		FolderID:       folderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		ActivateApis: pulumi.StringArray{
			pulumi.String("run.googleapis.com"),
			pulumi.String("artifactregistry.googleapis.com"),
		},
	})
	if err != nil {
		return nil, err
	}

	// 2. Data Project using Project Component
	data, err := project.NewProject(ctx, "data-project", &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-%s-%s-data", cfg.ProjectPrefix, cfg.Env, cfg.BusinessCode)),
		Name:           pulumi.String(fmt.Sprintf("%s-%s-%s-data", cfg.ProjectPrefix, cfg.Env, cfg.BusinessCode)),
		FolderID:       folderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		ActivateApis: pulumi.StringArray{
			pulumi.String("bigquery.googleapis.com"),
		},
	})
	if err != nil {
		return nil, err
	}

	return &BUProjects{
		AppProjectID:  app.Project.ProjectId,
		DataProjectID: data.Project.ProjectId,
	}, nil
}
