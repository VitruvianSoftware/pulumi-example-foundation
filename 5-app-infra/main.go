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

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadAppInfraConfig(ctx)

		// 1. Resolve Project IDs from the Stage 4 stack.
		projStack, err := pulumi.NewStackReference(ctx, "projects", &pulumi.StackReferenceArgs{
			Name: pulumi.String(cfg.ProjectsStackName),
		})
		if err != nil {
			return err
		}

		// 1b. Resolve bootstrap stack
		bootstrapStack, err := pulumi.NewStackReference(ctx, "bootstrap", &pulumi.StackReferenceArgs{
			Name: pulumi.String(fmt.Sprintf("VitruvianSoftware/foundation-0-bootstrap/%s", cfg.Env)),
		})
		if err != nil {
			return err
		}

		appProjectID := projStack.GetStringOutput(pulumi.String("svpc_project_id"))
		peeringProjectID := projStack.GetStringOutput(pulumi.String("peering_project_id"))
		confSpaceProjectID := projStack.GetStringOutput(pulumi.String("confidential_space_project_id"))
		confSpaceWorkloadSA := projStack.GetStringOutput(pulumi.String("confidential_space_workload_sa"))
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
		peeringSubnetSelfLink := projStack.GetStringOutput(pulumi.String("peering_subnetwork_self_link"))
		networkProjectID := projStack.GetStringOutput(pulumi.String("network_project_id"))
		cicdProjectID := bootstrapStack.GetStringOutput(pulumi.String("cicd_project_id"))

		// Reconstruct SVPC subnet self link
		svpcSubnetSelfLink := pulumi.Sprintf("projects/%s/regions/%s/subnetworks/sb-%s-svpc-%s", networkProjectID, cfg.Region, cfg.EnvCode, cfg.Region)

		// 2. Deploy SVPC Instance
		err = deployEnvBase(ctx, "sample-svpc", &EnvBaseArgs{
			Env:                cfg.Env,
			BusinessUnit:       cfg.BusinessCode,
			ProjectSuffix:      "sample-svpc",
			ProjectID:          appProjectID,
			Region:             cfg.Region,
			SubnetworkSelfLink: svpcSubnetSelfLink,
			IAPFirewallTags:    nil, // No tags for SVPC
		})
		if err != nil {
			return err
		}

		// 3. Deploy Peering Instance
		err = deployEnvBase(ctx, "sample-peering", &EnvBaseArgs{
			Env:                cfg.Env,
			BusinessUnit:       cfg.BusinessCode,
			ProjectSuffix:      "sample-peering",
			ProjectID:          peeringProjectID,
			Region:             cfg.Region,
			SubnetworkSelfLink: peeringSubnetSelfLink,
			IAPFirewallTags:    iapFirewallTags,
		})
		if err != nil {
			return err
		}

		// 4. Deploy Confidential Space
		if cfg.ConfidentialImageDigest != "" {
			err = deployConfidentialSpace(ctx, "conf-space", &ConfidentialSpaceArgs{
				Env:                      cfg.Env,
				BusinessUnit:             cfg.BusinessCode,
				ProjectID:                confSpaceProjectID,
				Region:                   cfg.Region,
				SubnetworkSelfLink:       svpcSubnetSelfLink,
				WorkloadSAEmail:          confSpaceWorkloadSA,
				ConfidentialImageDigest:  cfg.ConfidentialImageDigest,
				ConfidentialMachineType:  "n2d-standard-2",
				ConfidentialInstanceType: "SEV",
				CpuPlatform:              "AMD Milan",
				CloudBuildProjectID:      cicdProjectID,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

type AppInfraConfig struct {
	Env                     string
	EnvCode                 string
	BusinessCode            string
	Region                  string
	ProjectsStackName       string
	NetworkStackName        string
	ConfidentialImageDigest string
}

func loadAppInfraConfig(ctx *pulumi.Context) *AppInfraConfig {
	conf := config.New(ctx, "")
	c := &AppInfraConfig{
		Env:                     conf.Require("env"),
		BusinessCode:            conf.Get("business_code"),
		Region:                  conf.Get("region"),
		ProjectsStackName:       conf.Get("projects_stack_name"),
		NetworkStackName:        conf.Get("network_stack_name"),
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
	if c.NetworkStackName == "" {
		c.NetworkStackName = fmt.Sprintf("VitruvianSoftware/foundation-3-networks-svpc/%s", c.Env)
	}
	envCodes := map[string]string{"development": "d", "nonproduction": "n", "production": "p"}
	c.EnvCode = envCodes[c.Env]
	if c.EnvCode == "" {
		c.EnvCode = c.Env[:1]
	}
	return c
}
