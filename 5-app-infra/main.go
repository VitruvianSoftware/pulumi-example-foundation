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
		bootstrapStack, err := pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.BootstrapStackName),
		})
		if err != nil {
			return err
		}

		// --- Resolve outputs from 4-projects ---
		appProjectID := projStack.GetStringOutput(pulumi.String("svpc_project_id"))
		peeringProjectID := projStack.GetStringOutput(pulumi.String("peering_project_id"))
		confSpaceProjectID := projStack.GetStringOutput(pulumi.String("confidential_space_project_id"))
		confSpaceProjectNumber := projStack.GetStringOutput(pulumi.String("confidential_space_project_number"))
		confSpaceWorkloadSA := projStack.GetStringOutput(pulumi.String("confidential_space_workload_sa"))
		peeringSubnetSelfLink := projStack.GetStringOutput(pulumi.String("peering_subnetwork_self_link"))
		networkProjectID := projStack.GetStringOutput(pulumi.String("network_project_id"))

		// IAP firewall tags come as a map[string]interface{} from stack references;
		// convert to map[string]string for the Compute Instance params.
		iapFirewallTags := projStack.GetOutput(pulumi.String("iap_firewall_tags")).ApplyT(func(v interface{}) map[string]string {
			m := make(map[string]string)
			if v == nil {
				return m
			}
			if vm, ok := v.(map[string]interface{}); ok {
				for key, val := range vm {
					m[key] = fmt.Sprintf("%v", val)
				}
			}
			return m
		}).(pulumi.StringMapOutput)

		// --- Resolve outputs from 0-bootstrap ---
		cicdProjectID := bootstrapStack.GetStringOutput(pulumi.String("cicd_project_id"))

		// Reconstruct SVPC subnet self link from deterministic naming convention
		svpcSubnetSelfLink := pulumi.Sprintf(
			"projects/%s/regions/%s/subnetworks/sb-%s-svpc-%s",
			networkProjectID, cfg.Region, cfg.EnvCode, cfg.Region,
		)

		// 3. Deploy SVPC Instance (upstream: module "gce_instance" with project_suffix = "sample-svpc")
		svpcResult, err := deployEnvBase(ctx, "sample-svpc", &EnvBaseArgs{
			Env:                cfg.Env,
			BusinessUnit:       cfg.BusinessCode,
			ProjectSuffix:      "sample-svpc",
			Hostname:           "example-app",
			ProjectID:          appProjectID,
			Region:             cfg.Region,
			SubnetworkSelfLink: svpcSubnetSelfLink,
			IAPFirewallTags:    nil, // No tags for SVPC (upstream: null)
		})
		if err != nil {
			return err
		}

		// 4. Deploy Peering Instance (upstream: module "peering_gce_instance" with project_suffix = "sample-peering")
		peeringResult, err := deployEnvBase(ctx, "sample-peering", &EnvBaseArgs{
			Env:                cfg.Env,
			BusinessUnit:       cfg.BusinessCode,
			ProjectSuffix:      "sample-peering",
			Hostname:           "example-app",
			ProjectID:          peeringProjectID,
			Region:             cfg.Region,
			SubnetworkSelfLink: peeringSubnetSelfLink,
			IAPFirewallTags:    iapFirewallTags,
		})
		if err != nil {
			return err
		}

		// 5. Deploy Confidential Space (upstream: module "confidential_space")
		var confResult *ConfidentialSpaceResult
		if cfg.ConfidentialImageDigest != "" {
			confResult, err = deployConfidentialSpace(ctx, "conf-space", &ConfidentialSpaceArgs{
				Env:                      cfg.Env,
				BusinessUnit:             cfg.BusinessCode,
				ProjectID:                confSpaceProjectID,
				ProjectNumber:            confSpaceProjectNumber,
				Region:                   cfg.Region,
				SubnetworkSelfLink:       svpcSubnetSelfLink,
				WorkloadSAEmail:          confSpaceWorkloadSA,
				ConfidentialImageDigest:  cfg.ConfidentialImageDigest,
				ConfidentialMachineType:  "n2d-standard-2",
				ConfidentialInstanceType: "SEV",
				CpuPlatform:             "AMD Milan",
				CloudBuildProjectID:      cicdProjectID,
			})
			if err != nil {
				return err
			}
		}

		// 6. Exports — matching upstream outputs.tf
		ctx.Export("project_id", appProjectID)
		ctx.Export("region", pulumi.String(cfg.Region))
		ctx.Export("instances_self_links", svpcResult.InstanceSelfLink)
		ctx.Export("instances_names", svpcResult.InstanceName)
		ctx.Export("instances_zones", svpcResult.InstanceZone)
		ctx.Export("peering_instances_self_links", peeringResult.InstanceSelfLink)
		ctx.Export("peering_instances_names", peeringResult.InstanceName)
		ctx.Export("peering_instances_zones", peeringResult.InstanceZone)

		if confResult != nil {
			ctx.Export("confidential_space_project_id", confSpaceProjectID)
			ctx.Export("confidential_space_project_number", confSpaceProjectNumber)
			ctx.Export("workload_identity_pool_id", confResult.WorkloadPoolID)
			ctx.Export("workload_pool_provider_id", confResult.WorkloadPoolProviderID)
			ctx.Export("confidential_instances_self_links", confResult.InstanceSelfLink)
			ctx.Export("confidential_instances_names", confResult.InstanceName)
			ctx.Export("confidential_instances_zones", confResult.InstanceZone)
		}

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
