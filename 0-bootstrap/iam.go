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
	"github.com/VitruvianSoftware/pulumi-library/pkg/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/billing"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func deployIAM(ctx *pulumi.Context, cfg *Config, projectID pulumi.StringOutput) (map[string]*serviceaccount.Account, error) {
	granularSAs := map[string]string{
		"bootstrap": "Foundation Bootstrap SA. Managed by Pulumi.",
		"org":       "Foundation Organization SA. Managed by Pulumi.",
		"env":       "Foundation Environment SA. Managed by Pulumi.",
		"net":       "Foundation Network SA. Managed by Pulumi.",
		"proj":      "Foundation Projects SA. Managed by Pulumi.",
		"apps":      "Foundation Apps SA. Managed by Pulumi.",
	}

	sas := make(map[string]*serviceaccount.Account)
	for key, desc := range granularSAs {
		sa, err := serviceaccount.NewAccount(ctx, fmt.Sprintf("sa-terraform-%s", key), &serviceaccount.AccountArgs{
			Project:     projectID,
			AccountId:   pulumi.String(fmt.Sprintf("sa-terraform-%s", key)),
			DisplayName: pulumi.String(desc),
		})
		if err != nil {
			return nil, err
		}
		sas[key] = sa
	}

	// Org Level IAM Roles using granular IAMMember component
	commonRoles := []string{"roles/browser"}

	granularSAOrgRoles := map[string][]string{
		"bootstrap": append([]string{
			"roles/resourcemanager.organizationAdmin",
			"roles/accesscontextmanager.policyAdmin",
			"roles/serviceusage.serviceUsageConsumer",
		}, commonRoles...),
	}

	for key, roles := range granularSAOrgRoles {
		for _, role := range roles {
			_, err := iam.NewIAMMember(ctx, fmt.Sprintf("org-iam-%s-%s", key, role), &iam.IAMMemberArgs{
				ParentID:   pulumi.String(cfg.OrgID),
				ParentType: pulumi.String("organization"),
				Role:       pulumi.String(role),
				Member:     sas[key].Email.ApplyT(func(email string) string { return fmt.Sprintf("serviceAccount:%s", email) }).(pulumi.StringOutput),
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// Billing IAM
	for key := range granularSAs {
		_, err := billing.NewAccountIamMember(ctx, fmt.Sprintf("billing-user-%s", key), &billing.AccountIamMemberArgs{
			BillingAccountId: pulumi.String(cfg.BillingAccount),
			Role:             pulumi.String("roles/billing.user"),
			Member:           sas[key].Email.ApplyT(func(email string) string { return fmt.Sprintf("serviceAccount:%s", email) }).(pulumi.StringOutput),
		})
		if err != nil {
			return nil, err
		}
	}

	return sas, nil
}
