/*
 * Copyright 2026 Vitruvian Software
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

		// 1. Deploy Hub VPC
		hub, err := networking.NewNetworking(ctx, "hub-network", &networking.NetworkingArgs{
			ProjectID: pulumi.String(cfg.ProjectID),
			VPCName:   pulumi.String("vpc-" + cfg.Env + "-hub"),
			Subnets: []networking.SubnetArgs{
				{
					Name:   "sb-" + cfg.Env + "-hub-" + cfg.Region1,
					Region: cfg.Region1,
					CIDR:   "10.0.0.0/18",
				},
			},
			EnablePSA: true,
		})
		if err != nil {
			return err
		}

		// 2. Deploy Spoke VPC
		_, err = networking.NewNetworking(ctx, "spoke-network", &networking.NetworkingArgs{
			ProjectID: pulumi.String(cfg.ProjectID),
			VPCName:   pulumi.String("vpc-" + cfg.Env + "-spoke"),
			Subnets: []networking.SubnetArgs{
				{
					Name:   "sb-" + cfg.Env + "-spoke-" + cfg.Region1,
					Region: cfg.Region1,
					CIDR:   "10.1.0.0/18",
				},
			},
			EnablePSA: false,
		})
		if err != nil {
			return err
		}

		// In a real hub-and-spoke, we would add peering or VPN here.
		// For the reference architecture, we demonstrate the multi-VPC intent.

		ctx.Export("hub_vpc_id", hub.VPC.ID())
		return nil
	})
}

type NetConfig struct {
	Env       string
	ProjectID string
	Region1   string
}

func loadNetConfig(ctx *pulumi.Context) *NetConfig {
	conf := config.New(ctx, "")
	c := &NetConfig{
		Env:       conf.Require("env"),
		ProjectID: conf.Require("project_id"),
		Region1:   conf.Get("region1"),
	}
	if c.Region1 == "" {
		c.Region1 = "us-central1"
	}
	return c
}
