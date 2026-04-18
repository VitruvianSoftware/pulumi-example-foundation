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

type OrgProjects struct {
	AuditLogsProjectID     pulumi.StringOutput
	BillingExportProjectID pulumi.StringOutput
}

func deployOrgProjects(ctx *pulumi.Context, cfg *OrgConfig, commonFolderID pulumi.StringOutput) (*OrgProjects, error) {
	// 1. Audit Logs Project
	audit, err := project.NewProject(ctx, "org-logging", &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-c-logging", cfg.ProjectPrefix)),
		Name:           pulumi.String(fmt.Sprintf("%s-c-logging", cfg.ProjectPrefix)),
		FolderID:       commonFolderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		ActivateApis: pulumi.StringArray{
			pulumi.String("logging.googleapis.com"),
			pulumi.String("bigquery.googleapis.com"),
		},
	})
	if err != nil {
		return nil, err
	}

	// 2. Billing Export Project
	billing, err := project.NewProject(ctx, "org-billing-export", &project.ProjectArgs{
		ProjectID:      pulumi.String(fmt.Sprintf("%s-c-billing-export", cfg.ProjectPrefix)),
		Name:           pulumi.String(fmt.Sprintf("%s-c-billing-export", cfg.ProjectPrefix)),
		FolderID:       commonFolderID,
		BillingAccount: pulumi.String(cfg.BillingAccount),
		ActivateApis: pulumi.StringArray{
			pulumi.String("bigquery.googleapis.com"),
		},
	})
	if err != nil {
		return nil, err
	}

	return &OrgProjects{
		AuditLogsProjectID:     audit.Project.ProjectId,
		BillingExportProjectID: billing.Project.ProjectId,
	}, nil
}
