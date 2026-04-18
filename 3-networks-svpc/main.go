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
	"github.com/VitruvianSoftware/pulumi-library/pkg/networking"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadNetConfig(ctx)

		// 1. Deploy Networking using the reusable Networking Component
		net, err := networking.NewNetworking(ctx, "shared-network", &networking.NetworkingArgs{
			ProjectID: pulumi.String(cfg.ProjectID),
			VPCName:   pulumi.String("vpc-" + cfg.Env + "-shared"),
			Subnets: []networking.SubnetArgs{
				{
					Name:   "sb-" + cfg.Env + "-shared-" + cfg.Region1,
					Region: cfg.Region1,
					CIDR:   "10.8.64.0/18",
				},
			},
			EnablePSA: true,
		})
		if err != nil {
			return err
		}

		// 2. Exports
		ctx.Export("network_id", net.VPC.ID())
		return nil
	})
}

type NetConfig struct {
	Env       string
	ProjectID string
	Region1   string
	Region2   string
}

func loadNetConfig(ctx *pulumi.Context) *NetConfig {
	conf := config.New(ctx, "")
	c := &NetConfig{
		Env:       conf.Require("env"),
		ProjectID: conf.Require("project_id"),
		Region1:   conf.Get("region1"),
		Region2:   conf.Get("region2"),
	}
	if c.Region1 == "" {
		c.Region1 = "us-central1"
	}
	return c
}
