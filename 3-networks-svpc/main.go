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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/accesscontextmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/VitruvianSoftware/pulumi-library/pkg/networking"
	"github.com/VitruvianSoftware/pulumi-library/pkg/vpc_sc"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadNetConfig(ctx)

		// ====================================================================
		// SHARED ENVIRONMENT (Deployed Once)
		// ====================================================================
		if cfg.Env == "shared" {
			// 1. Hierarchical Firewall Policy (org/folder level)
			fwPolicy, err := networking.NewHierarchicalFirewallPolicy(ctx, "hierarchical-fw", &networking.HierarchicalFirewallPolicyArgs{
				ParentID:      pulumi.String(cfg.ParentID),
				ShortName:     fmt.Sprintf("fw-%s-svpc-hierarchical", cfg.Env),
				Description:   "Hierarchical firewall rules",
				Associations:  cfg.FirewallAssociations,
				EnableLogging: cfg.FirewallPoliciesEnableLogging,
			})
			if err != nil {
				return err
			}

			ctx.Export("hierarchical_fw", fwPolicy.Policy.ID())
			return nil
		}

		// ====================================================================
		// PER-ENVIRONMENT (development, nonproduction, production)
		// ====================================================================

		// Compute environment-specific advertised IP ranges
		// Production advertises the Google DNS forwarding source range + PSC endpoint
		// Other environments only advertise the PSC endpoint
		advertisedRanges := []networking.AdvertisedIPRange{
			{Range: cfg.PscIP + "/32", Description: "PSC Endpoint"},
		}
		if cfg.Env == "production" {
			advertisedRanges = append([]networking.AdvertisedIPRange{
				{Range: "35.199.192.0/19", Description: "Google DNS Forwarding Source"},
			}, advertisedRanges...)
		}

		// 1. Shared VPC Host
		if _, err := compute.NewSharedVPCHostProject(ctx, "svpc-host", &compute.SharedVPCHostProjectArgs{
			Project: pulumi.String(cfg.ProjectID),
		}); err != nil {
			return err
		}

		// 2. VPC & Subnets (delete_default_routes_on_create = true)
		netName := fmt.Sprintf("vpc-%s-svpc", cfg.EnvCode)
		netOpts := &networking.NetworkingArgs{
			ProjectID: pulumi.String(cfg.ProjectID),
			VPCName:   pulumi.String(netName),
			EnablePSA: true,
			Subnets: []networking.SubnetArgs{
				{
					Name:   fmt.Sprintf("sb-%s-svpc-%s", cfg.EnvCode, cfg.Region1),
					Region: cfg.Region1,
					CIDR:   "10.8.64.0/18",
					SecondaryRanges: []networking.SecondaryRangeArgs{
						{RangeName: fmt.Sprintf("rn-%s-svpc-%s-gke-pod", cfg.EnvCode, cfg.Region1), CIDR: "100.72.64.0/18"},
						{RangeName: fmt.Sprintf("rn-%s-svpc-%s-gke-svc", cfg.EnvCode, cfg.Region1), CIDR: "100.73.64.0/18"},
					},
					FlowLogs: true,
				},
				{
					Name:   fmt.Sprintf("sb-%s-svpc-%s", cfg.EnvCode, cfg.Region2),
					Region: cfg.Region2,
					CIDR:   "10.9.64.0/18",
					SecondaryRanges: []networking.SecondaryRangeArgs{
						{RangeName: fmt.Sprintf("rn-%s-svpc-%s-gke-pod", cfg.EnvCode, cfg.Region2), CIDR: "100.74.64.0/18"},
						{RangeName: fmt.Sprintf("rn-%s-svpc-%s-gke-svc", cfg.EnvCode, cfg.Region2), CIDR: "100.75.64.0/18"},
					},
					FlowLogs: true,
				},
				{ // Proxy-only subnets for ILB
					Name:    fmt.Sprintf("sb-%s-svpc-%s-proxy", cfg.EnvCode, cfg.Region1),
					Region:  cfg.Region1,
					CIDR:    "10.26.2.0/23",
					Role:    "ACTIVE",
					Purpose: "REGIONAL_MANAGED_PROXY",
				},
				{
					Name:    fmt.Sprintf("sb-%s-svpc-%s-proxy", cfg.EnvCode, cfg.Region2),
					Region:  cfg.Region2,
					CIDR:    "10.27.2.0/23",
					Role:    "ACTIVE",
					Purpose: "REGIONAL_MANAGED_PROXY",
				},
			},
		}

		vpcModule, err := networking.NewNetworking(ctx, "svpc", netOpts)
		if err != nil {
			return err
		}

		// 3. VPC-Level Firewall Policy (Default Deny Egress) — data-driven rules
		_, err = networking.NewNetworkFirewallPolicy(ctx, "vpc-fw", &networking.NetworkFirewallPolicyArgs{
			ProjectID:  pulumi.String(cfg.ProjectID),
			PolicyName: fmt.Sprintf("fp-%s-svpc-firewalls", cfg.EnvCode),
			TargetVPCs: []pulumi.StringInput{
				pulumi.Sprintf("projects/%s/global/networks/%s", cfg.ProjectID, vpcModule.VPC.Name),
			},
			Rules: networking.BuildFoundationRules(cfg.EnvCode, true, cfg.PscIP+"/32", []string{"10.8.64.0/18", "10.9.64.0/18"}, true),
		}, pulumi.DependsOn([]pulumi.Resource{vpcModule.VPC}))
		if err != nil {
			return err
		}

		// 4. Private Service Connect (PSC) — googleapis + gcr.io + pkg.dev DNS
		_, err = networking.NewPrivateServiceConnect(ctx, "psc", &networking.PrivateServiceConnectArgs{
			ProjectID:            pulumi.String(cfg.ProjectID),
			NetworkSelfLink:      vpcModule.VPC.SelfLink,
			DnsCode:              fmt.Sprintf("dz-%s-svpc", cfg.EnvCode),
			IPAddress:            cfg.PscIP,
			ForwardingRuleTarget: "vpc-sc",
		}, pulumi.DependsOn([]pulumi.Resource{vpcModule.VPC}))
		if err != nil {
			return err
		}

		// 5. DNS Policy (inbound forwarding + logging)
		_, err = dns.NewPolicy(ctx, "dns-default-policy", &dns.PolicyArgs{
			Project:                 pulumi.String(cfg.ProjectID),
			Name:                    pulumi.String(fmt.Sprintf("dp-%s-svpc-default-policy", cfg.EnvCode)),
			EnableInboundForwarding: pulumi.Bool(true),
			EnableLogging:           pulumi.Bool(cfg.DnsEnableLogging),
			Networks: dns.PolicyNetworkArray{
				&dns.PolicyNetworkArgs{
					NetworkUrl: vpcModule.VPC.SelfLink,
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{vpcModule.VPC}))
		if err != nil {
			return err
		}

		// 6. Egress internet route (tag-based, only when NAT is enabled)
		_, err = compute.NewRoute(ctx, "egress-internet", &compute.RouteArgs{
			Project:        pulumi.String(cfg.ProjectID),
			Name:           pulumi.String(fmt.Sprintf("rt-%s-svpc-1000-egress-internet-default", cfg.EnvCode)),
			Network:        vpcModule.VPC.ID(),
			DestRange:      pulumi.String("0.0.0.0/0"),
			NextHopGateway: pulumi.String("default-internet-gateway"),
			Priority:       pulumi.Int(1000),
			Tags:           pulumi.StringArray{pulumi.String("egress-internet")},
		}, pulumi.DependsOn([]pulumi.Resource{vpcModule.VPC}))
		if err != nil {
			return err
		}

		// 7. DNS Peering / Forwarding Zones
		if cfg.EnvCode == "p" {
			_, err = networking.NewDnsZone(ctx, "dns-forwarding", &networking.DnsZoneArgs{
				ProjectID:                 pulumi.String(cfg.ProjectID),
				Name:                      "fz-dns-hub",
				Domain:                    cfg.Domain,
				Type:                      "forwarding",
				NetworkSelfLink:           vpcModule.VPC.SelfLink,
				TargetNameServerAddresses: cfg.TargetNameServers,
			})
			if err != nil {
				return err
			}
		} else {
			_, err = networking.NewDnsZone(ctx, "dns-peering", &networking.DnsZoneArgs{
				ProjectID:             pulumi.String(cfg.ProjectID),
				Name:                  fmt.Sprintf("dz-%s-svpc-to-dns-hub", cfg.EnvCode),
				Domain:                cfg.Domain,
				Type:                  "peering",
				NetworkSelfLink:       vpcModule.VPC.SelfLink,
				TargetNetworkSelfLink: pulumi.String(fmt.Sprintf("projects/%s/global/networks/vpc-p-svpc", cfg.DNSProjectID)),
			})
			if err != nil {
				return err
			}
		}

		// 8. BGP Cloud Routers — 4 total (2 per region), matching upstream
		for _, reg := range []string{cfg.Region1, cfg.Region2} {
			for _, crIdx := range []string{"5", "6"} {
				_, err = networking.NewCloudRouter(ctx, fmt.Sprintf("cr-%s-cr%s", reg, crIdx), &networking.RouterArgs{
					ProjectID:          pulumi.String(cfg.ProjectID),
					Region:             reg,
					Network:            vpcModule.VPC.SelfLink,
					BgpAsn:             cfg.BgpAsn,
					AdvertisedGroups:   []string{"ALL_SUBNETS"},
					AdvertisedIpRanges: advertisedRanges,
					EnableNat:          false, // BGP routers don't have NAT
				}, pulumi.DependsOn([]pulumi.Resource{vpcModule.VPC}))
				if err != nil {
					return err
				}
			}
		}

		// 9. Separate NAT Routers — 1 per region with static IPs (matches upstream nat.tf)
		for _, reg := range []string{cfg.Region1, cfg.Region2} {
			_, err = networking.NewCloudRouter(ctx, fmt.Sprintf("nat-router-%s", reg), &networking.RouterArgs{
				ProjectID:       pulumi.String(cfg.ProjectID),
				Region:          reg,
				Network:         vpcModule.VPC.SelfLink,
				BgpAsn:          cfg.NatBgpAsn,
				EnableNat:       true,
				NatNumAddresses: cfg.NatNumAddresses,
			}, pulumi.DependsOn([]pulumi.Resource{vpcModule.VPC}))
			if err != nil {
				return err
			}
		}

		// 10. VPC Service Controls Perimeter
		if cfg.PolicyID != "" {
			_, err = vpc_sc.NewVpcServiceControls(ctx, "vpc-sc-perimeter", &vpc_sc.VpcServiceControlsArgs{
				PolicyID:              pulumi.String(cfg.PolicyID),
				Prefix:                fmt.Sprintf("%s_svpc", cfg.EnvCode),
				Members:               cfg.VpcScMembers,
				MembersDryRun:         cfg.VpcScMembers,
				ProjectNumbers:        cfg.VpcScProjects,
				RestrictedServices:    cfg.VpcScRestrictedServices,
				Enforce:               cfg.EnforceVpcSc,
				IngressPolicies:       cfg.VpcScIngressPolicies,
				EgressPolicies:        cfg.VpcScEgressPolicies,
				IngressPoliciesDryRun: cfg.VpcScIngressPoliciesDryRun,
				EgressPoliciesDryRun:  cfg.VpcScEgressPoliciesDryRun,
			})
			if err != nil {
				return err
			}
		}

		ctx.Export("network_id", vpcModule.VPC.ID())
		ctx.Export("network_name", vpcModule.VPC.Name)
		return nil
	})
}

type NetConfig struct {
	Env                           string
	EnvCode                       string // single-char env code (d, n, p)
	ProjectID                     string
	Region1                       string
	Region2                       string
	ParentID                      string
	PolicyID                      string
	DNSProjectID                  string
	Domain                        string
	PscIP                         string
	BgpAsn                        int
	NatBgpAsn                     int
	NatNumAddresses               int
	TargetNameServers             []string
	VpcScMembers                  []string
	VpcScProjects                 []string
	VpcScRestrictedServices       []string
	VpcScIngressPolicies          accesscontextmanager.ServicePerimeterStatusIngressPolicyArray
	VpcScEgressPolicies           accesscontextmanager.ServicePerimeterStatusEgressPolicyArray
	VpcScIngressPoliciesDryRun    accesscontextmanager.ServicePerimeterSpecIngressPolicyArray
	VpcScEgressPoliciesDryRun     accesscontextmanager.ServicePerimeterSpecEgressPolicyArray
	FirewallAssociations          []string
	FirewallPoliciesEnableLogging bool
	DnsEnableLogging              bool
	EnforceVpcSc                  bool
}

func loadNetConfig(ctx *pulumi.Context) *NetConfig {
	conf := config.New(ctx, "")

	c := &NetConfig{
		Env:          conf.Require("env"),
		EnvCode:      conf.Require("env_code"),
		ProjectID:    conf.Require("project_id"),
		Region1:      conf.Get("region1"),
		Region2:      conf.Get("region2"),
		ParentID:     conf.Require("parent_id"),
		Domain:       conf.Get("domain"),
		PolicyID:     conf.Get("policy_id"),
		DNSProjectID: conf.Get("dns_project_id"),
		PscIP:        conf.Get("psc_ip"),
	}
	conf.GetObject("vpc_sc_members", &c.VpcScMembers)
	conf.GetObject("vpc_sc_projects", &c.VpcScProjects)
	conf.GetObject("vpc_sc_restricted_services", &c.VpcScRestrictedServices)
	conf.GetObject("target_name_servers", &c.TargetNameServers)
	conf.GetObject("firewall_associations", &c.FirewallAssociations)

	if val, err := conf.TryBool("firewall_policies_enable_logging"); err == nil {
		c.FirewallPoliciesEnableLogging = val
	} else {
		c.FirewallPoliciesEnableLogging = true // Default to true matching TF
	}

	if val, err := conf.TryBool("dns_enable_logging"); err == nil {
		c.DnsEnableLogging = val
	} else {
		c.DnsEnableLogging = true
	}

	if val, err := conf.TryBool("enforce_vpcsc"); err == nil {
		c.EnforceVpcSc = val
	} else {
		c.EnforceVpcSc = true
	}

	if c.Region1 == "" {
		c.Region1 = "us-central1"
	}
	if c.Region2 == "" {
		c.Region2 = "us-west1"
	}
	if c.Domain == "" {
		c.Domain = "example.com."
	}
	if c.PscIP == "" {
		c.PscIP = "10.17.0.6"
	}
	if len(c.VpcScRestrictedServices) == 0 {
		c.VpcScRestrictedServices = vpc_sc.GetDefaultRestrictedServices()
	}
	if len(c.FirewallAssociations) == 0 {
		c.FirewallAssociations = []string{c.ParentID} // Fallback to parent
	}
	if len(c.TargetNameServers) == 0 {
		c.TargetNameServers = []string{"10.0.0.1"}
	}

	c.BgpAsn = 64514
	c.NatBgpAsn = 64514
	c.NatNumAddresses = 2

	return c
}
