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
	"github.com/VitruvianSoftware/pulumi-library/pkg/policy"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/logging"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func deployOrgPoliciesAndSinks(ctx *pulumi.Context, cfg *OrgConfig, auditProjectID pulumi.StringOutput) error {
	// 1. Organization Sink
	_, err := logging.NewOrganizationSink(ctx, "org-sink", &logging.OrganizationSinkArgs{
		Name:        pulumi.String("sk-c-logging-prj"),
		OrgId:       pulumi.String(cfg.OrgID),
		Destination: auditProjectID.ApplyT(func(id string) string { return fmt.Sprintf("logging.googleapis.com/projects/%s/locations/global/buckets/AggregatedLogs", id) }).(pulumi.StringOutput),
		Filter:      pulumi.String("logName: /logs/cloudaudit.googleapis.com%2Factivity"),
	})
	if err != nil {
		return err
	}

	// 2. Example Organization Policy from the centralized library
	_, err = policy.NewOrgPolicy(ctx, "disable-serial-port", &policy.OrgPolicyArgs{
		ParentID:   pulumi.String("organizations/" + cfg.OrgID),
		Constraint: pulumi.String("constraints/compute.disableSerialPortAccess"),
		Boolean:    pulumi.Bool(true),
	})

	return err
}
