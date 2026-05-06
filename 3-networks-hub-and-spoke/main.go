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

	"github.com/VitruvianSoftware/pulumi-library/go/pkg/networking"
	"github.com/VitruvianSoftware/pulumi-library/go/pkg/vpc_sc"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadNetConfig(ctx)

		// Compute environment-specific advertised IP ranges
		advertisedRanges := []networking.AdvertisedIPRange{
			{Range: cfg.PscIP + "/32", Description: "PSC Endpoint"},
		}
		// Hub (env_code "c") also advertises the DNS forwarding source range
		if cfg.EnvCode == "c" {
			advertisedRanges = append([]networking.AdvertisedIPRange{
				{Range: "35.199.192.0/19", Description: "Google DNS Forwarding Source"},
			}, advertisedRanges...)
		}

		// ====================================================================
		// HUB ENVIRONMENT (Deployed in "shared" / env_code "c")
		// ====================================================================
		if cfg.Env == "shared" {
			// 1. Hierarchical Firewall Policy (org/folder level)
			_, err := networking.NewHierarchicalFirewallPolicy(ctx, "hierarchical-fw", &networking.HierarchicalFirewallPolicyArgs{
				ParentID:      pulumi.String(cfg.ParentID),
				ShortName:     fmt.Sprintf("fw-%s-svpc-hierarchical", cfg.Env),
				Description:   "Hierarchical firewall rules",
				Associations:  cfg.FirewallAssociations,
				EnableLogging: cfg.FirewallPoliciesEnableLogging,
			})
			if err != nil {
				return err
			}

			// 2. Hub Shared VPC Host
			if _, err := compute.NewSharedVPCHostProject(ctx, "hub-svpc-host", &compute.SharedVPCHostProjectArgs{
				Project: pulumi.String(cfg.HubProjectID),
			}); err != nil {
				return err
			}

			// 3. Hub VPC & Subnets
			hubVpcName := fmt.Sprintf("vpc-%s-svpc-hub", cfg.EnvCode)
			hubNetOpts := &networking.NetworkingArgs{
				ProjectID: pulumi.String(cfg.HubProjectID),
				VPCName:   pulumi.String(hubVpcName),
				EnablePSA: true,
				Subnets: []networking.SubnetArgs{
					{
						Name:   fmt.Sprintf("sb-%s-svpc-hub-%s", cfg.EnvCode, cfg.Region1),
						Region: cfg.Region1,
						CIDR:   "10.0.64.0/18",
						SecondaryRanges: []networking.SecondaryRangeArgs{
							{RangeName: fmt.Sprintf("rn-%s-hub-%s-gke-pod", cfg.EnvCode, cfg.Region1), CIDR: "100.64.64.0/18"},
							{RangeName: fmt.Sprintf("rn-%s-hub-%s-gke-svc", cfg.EnvCode, cfg.Region1), CIDR: "100.65.64.0/18"},
						},
						FlowLogs: true,
					},
					{
						Name:   fmt.Sprintf("sb-%s-svpc-hub-%s", cfg.EnvCode, cfg.Region2),
						Region: cfg.Region2,
						CIDR:   "10.1.64.0/18",
						SecondaryRanges: []networking.SecondaryRangeArgs{
							{RangeName: fmt.Sprintf("rn-%s-hub-%s-gke-pod", cfg.EnvCode, cfg.Region2), CIDR: "100.66.64.0/18"},
							{RangeName: fmt.Sprintf("rn-%s-hub-%s-gke-svc", cfg.EnvCode, cfg.Region2), CIDR: "100.67.64.0/18"},
						},
						FlowLogs: true,
					},
				},
			}

			hubVpc, err := networking.NewNetworking(ctx, "hub", hubNetOpts)
			if err != nil {
				return err
			}

			// 4. Hub VPC-Level Firewall — data-driven rules
			hubFw, err := networking.NewNetworkFirewallPolicy(ctx, "hub-vpc-fw", &networking.NetworkFirewallPolicyArgs{
				ProjectID:  pulumi.String(cfg.HubProjectID),
				PolicyName: fmt.Sprintf("fp-%s-hub-firewalls", cfg.EnvCode),
				TargetVPCs: []pulumi.StringInput{
					pulumi.Sprintf("projects/%s/global/networks/%s", cfg.HubProjectID, hubVpc.VPC.Name),
				},
				Rules: networking.BuildFoundationRules(cfg.EnvCode, true, cfg.PscIP+"/32", []string{"10.0.64.0/18", "10.1.64.0/18"}, true),
			}, pulumi.DependsOn([]pulumi.Resource{hubVpc.VPC}))
			if err != nil {
				return err
			}

			// 5. PSC on hub
			_, err = networking.NewPrivateServiceConnect(ctx, "hub-psc", &networking.PrivateServiceConnectArgs{
				ProjectID:            pulumi.String(cfg.HubProjectID),
				NetworkSelfLink:      hubVpc.VPC.SelfLink,
				DnsCode:              fmt.Sprintf("dz-%s-hub", cfg.EnvCode),
				IPAddress:            cfg.PscIP,
				ForwardingRuleTarget: "vpc-sc",
			}, pulumi.DependsOn([]pulumi.Resource{hubVpc.VPC}))
			if err != nil {
				return err
			}

			// 6. DNS Policy on hub
			hubDnsPolicy, err := dns.NewPolicy(ctx, "hub-dns-policy", &dns.PolicyArgs{
				Project:                 pulumi.String(cfg.HubProjectID),
				Name:                    pulumi.String(fmt.Sprintf("dp-%s-hub-default-policy", cfg.EnvCode)),
				EnableInboundForwarding: pulumi.Bool(true),
				EnableLogging:           pulumi.Bool(cfg.DnsEnableLogging),
				Networks: dns.PolicyNetworkArray{
					&dns.PolicyNetworkArgs{
						NetworkUrl: hubVpc.VPC.SelfLink,
					},
				},
			}, pulumi.DependsOn([]pulumi.Resource{hubVpc.VPC}))
			if err != nil {
				return err
			}

			// 7. DNS forwarding zone on hub
			_, err = networking.NewDnsZone(ctx, "dns-forwarding", &networking.DnsZoneArgs{
				ProjectID:                 pulumi.String(cfg.HubProjectID),
				Name:                      "fz-dns-hub",
				Domain:                    cfg.Domain,
				Type:                      "forwarding",
				NetworkSelfLink:           hubVpc.VPC.SelfLink,
				TargetNameServerAddresses: cfg.TargetNameServers,
			})
			if err != nil {
				return err
			}

			// 8. Transitivity Appliance — MIG+ILB per region
			_, err = networking.NewTransitivityAppliance(ctx, "transitivity", &networking.TransitivityApplianceArgs{
				ProjectID:   pulumi.String(cfg.HubProjectID),
				Regions:     []string{cfg.Region1, cfg.Region2},
				Network:     hubVpc.VPC.SelfLink,
				NetworkName: hubVpcName,
				Subnetworks: map[string]pulumi.StringInput{
					cfg.Region1: hubVpc.Subnets[fmt.Sprintf("sb-%s-svpc-hub-%s", cfg.EnvCode, cfg.Region1)].SelfLink,
					cfg.Region2: hubVpc.Subnets[fmt.Sprintf("sb-%s-svpc-hub-%s", cfg.EnvCode, cfg.Region2)].SelfLink,
				},
				RegionalAggregates: map[string][]string{
					cfg.Region1: {"10.0.0.0/16", "10.8.0.0/16", "100.64.0.0/18"},
					cfg.Region2: {"10.1.0.0/16", "10.9.0.0/16", "100.66.0.0/18"},
				},
				FirewallPolicy: hubFw.Policy.Name,
			}, pulumi.DependsOn([]pulumi.Resource{hubVpc.VPC}))
			if err != nil {
				return err
			}

			// 9. Hub BGP Routers — 4 total (2 per region), hub only (not on spokes)
			for _, reg := range []string{cfg.Region1, cfg.Region2} {
				for _, crIdx := range []string{"5", "6"} {
					_, err = networking.NewCloudRouter(ctx, fmt.Sprintf("hub-cr-%s-cr%s", reg, crIdx), &networking.RouterArgs{
						ProjectID:          pulumi.String(cfg.HubProjectID),
						Region:             reg,
						Network:            hubVpc.VPC.SelfLink,
						BgpAsn:             cfg.BgpAsn,
						AdvertisedGroups:   []string{"ALL_SUBNETS"},
						AdvertisedIpRanges: advertisedRanges,
						EnableNat:          false,
					}, pulumi.DependsOn([]pulumi.Resource{hubVpc.VPC}))
					if err != nil {
						return err
					}
				}
			}

			// 10. Separate NAT routers on hub
			for _, reg := range []string{cfg.Region1, cfg.Region2} {
				_, err = networking.NewCloudRouter(ctx, fmt.Sprintf("hub-nat-%s", reg), &networking.RouterArgs{
					ProjectID:       pulumi.String(cfg.HubProjectID),
					Region:          reg,
					Network:         hubVpc.VPC.SelfLink,
					BgpAsn:          cfg.NatBgpAsn,
					EnableNat:       true,
					NatNumAddresses: cfg.NatNumAddresses,
				}, pulumi.DependsOn([]pulumi.Resource{hubVpc.VPC}))
				if err != nil {
					return err
				}
			}

			// Exports — matching TF 3-networks-hub-and-spoke/envs/shared/outputs.tf
			ctx.Export("shared_vpc_host_project_id", pulumi.String(cfg.HubProjectID))
			ctx.Export("network_name", hubVpc.VPC.Name)
			ctx.Export("dns_policy", hubDnsPolicy.ID()) // DNS policy ID
			return nil
		}

		// ====================================================================
		// SPOKE ENVIRONMENT (dev, nonprod, prod)
		// ====================================================================

		if _, err := compute.NewSharedVPCHostProject(ctx, "spoke-svpc-host", &compute.SharedVPCHostProjectArgs{
			Project: pulumi.String(cfg.SpokeProjectID),
		}); err != nil {
			return err
		}

		spokeVpcName := fmt.Sprintf("vpc-%s-svpc-spoke", cfg.EnvCode)
		spokeNetOpts := &networking.NetworkingArgs{
			ProjectID: pulumi.String(cfg.SpokeProjectID),
			VPCName:   pulumi.String(spokeVpcName),
			EnablePSA: true,
			Subnets: []networking.SubnetArgs{
				{
					Name:   fmt.Sprintf("sb-%s-svpc-spoke-%s", cfg.EnvCode, cfg.Region1),
					Region: cfg.Region1,
					CIDR:   "10.8.64.0/18",
					SecondaryRanges: []networking.SecondaryRangeArgs{
						{RangeName: fmt.Sprintf("rn-%s-spoke-%s-gke-pod", cfg.EnvCode, cfg.Region1), CIDR: "100.72.64.0/18"},
						{RangeName: fmt.Sprintf("rn-%s-spoke-%s-gke-svc", cfg.EnvCode, cfg.Region1), CIDR: "100.73.64.0/18"},
					},
					FlowLogs: true,
				},
				{
					Name:   fmt.Sprintf("sb-%s-svpc-spoke-%s", cfg.EnvCode, cfg.Region2),
					Region: cfg.Region2,
					CIDR:   "10.9.64.0/18",
					SecondaryRanges: []networking.SecondaryRangeArgs{
						{RangeName: fmt.Sprintf("rn-%s-spoke-%s-gke-pod", cfg.EnvCode, cfg.Region2), CIDR: "100.74.64.0/18"},
						{RangeName: fmt.Sprintf("rn-%s-spoke-%s-gke-svc", cfg.EnvCode, cfg.Region2), CIDR: "100.75.64.0/18"},
					},
					FlowLogs: true,
				},
				{
					Name:    fmt.Sprintf("sb-%s-svpc-spoke-%s-proxy", cfg.EnvCode, cfg.Region1),
					Region:  cfg.Region1,
					CIDR:    "10.26.2.0/23",
					Role:    "ACTIVE",
					Purpose: "REGIONAL_MANAGED_PROXY",
				},
				{
					Name:    fmt.Sprintf("sb-%s-svpc-spoke-%s-proxy", cfg.EnvCode, cfg.Region2),
					Region:  cfg.Region2,
					CIDR:    "10.27.2.0/23",
					Role:    "ACTIVE",
					Purpose: "REGIONAL_MANAGED_PROXY",
				},
			},
		}

		spokeVpc, err := networking.NewNetworking(ctx, "spoke", spokeNetOpts)
		if err != nil {
			return err
		}

		// Bi-Directional VPC Peering (Spoke <-> Hub)
		// Order matters: local peering first, then peer peering depends on it
		hubVpcRef := fmt.Sprintf("projects/%s/global/networks/vpc-c-svpc-hub", cfg.HubProjectID)

		spokeToHub, err := compute.NewNetworkPeering(ctx, "spoke-to-hub", &compute.NetworkPeeringArgs{
			Network:            spokeVpc.VPC.SelfLink,
			PeerNetwork:        pulumi.String(hubVpcRef),
			Name:               pulumi.String(fmt.Sprintf("np-%s-svpc-spoke-vpc-c-svpc-hub", cfg.EnvCode)),
			ExportCustomRoutes: pulumi.Bool(false),
			ImportCustomRoutes: pulumi.Bool(true), // Import hub's custom routes
		})
		if err != nil {
			return err
		}

		_, err = compute.NewNetworkPeering(ctx, "hub-to-spoke", &compute.NetworkPeeringArgs{
			Network:            pulumi.String(hubVpcRef),
			PeerNetwork:        spokeVpc.VPC.SelfLink,
			Name:               pulumi.String(fmt.Sprintf("np-vpc-c-svpc-hub-%s-svpc-spoke", cfg.EnvCode)),
			ExportCustomRoutes: pulumi.Bool(true), // Export hub's custom routes to spoke
			ImportCustomRoutes: pulumi.Bool(false),
		}, pulumi.DependsOn([]pulumi.Resource{spokeToHub})) // Must create after spoke-to-hub
		if err != nil {
			return err
		}

		// Spoke VPC-Level Firewall — data-driven rules
		_, err = networking.NewNetworkFirewallPolicy(ctx, "spoke-vpc-fw", &networking.NetworkFirewallPolicyArgs{
			ProjectID:  pulumi.String(cfg.SpokeProjectID),
			PolicyName: fmt.Sprintf("fp-%s-spoke-firewalls", cfg.EnvCode),
			TargetVPCs: []pulumi.StringInput{
				pulumi.Sprintf("projects/%s/global/networks/%s", cfg.SpokeProjectID, spokeVpc.VPC.Name),
			},
			Rules: networking.BuildFoundationRules(cfg.EnvCode, true, cfg.PscIP+"/32", []string{"10.8.64.0/18", "10.9.64.0/18"}, true),
		}, pulumi.DependsOn([]pulumi.Resource{spokeVpc.VPC}))
		if err != nil {
			return err
		}

		// PSC on spoke
		_, err = networking.NewPrivateServiceConnect(ctx, "spoke-psc", &networking.PrivateServiceConnectArgs{
			ProjectID:            pulumi.String(cfg.SpokeProjectID),
			NetworkSelfLink:      spokeVpc.VPC.SelfLink,
			DnsCode:              fmt.Sprintf("dz-%s-spoke", cfg.EnvCode),
			IPAddress:            cfg.PscIP,
			ForwardingRuleTarget: "vpc-sc",
		}, pulumi.DependsOn([]pulumi.Resource{spokeVpc.VPC}))
		if err != nil {
			return err
		}

		// DNS Policy on spoke
		_, err = dns.NewPolicy(ctx, "spoke-dns-policy", &dns.PolicyArgs{
			Project:                 pulumi.String(cfg.SpokeProjectID),
			Name:                    pulumi.String(fmt.Sprintf("dp-%s-spoke-default-policy", cfg.EnvCode)),
			EnableInboundForwarding: pulumi.Bool(true),
			EnableLogging:           pulumi.Bool(cfg.DnsEnableLogging),
			Networks: dns.PolicyNetworkArray{
				&dns.PolicyNetworkArgs{
					NetworkUrl: spokeVpc.VPC.SelfLink,
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{spokeVpc.VPC}))
		if err != nil {
			return err
		}

		// DNS peering from spoke to hub
		_, err = networking.NewDnsZone(ctx, "dns-peering", &networking.DnsZoneArgs{
			ProjectID:             pulumi.String(cfg.SpokeProjectID),
			Name:                  fmt.Sprintf("dz-%s-svpc-spoke-to-dns-hub", cfg.EnvCode),
			Domain:                cfg.Domain,
			Type:                  "peering",
			NetworkSelfLink:       spokeVpc.VPC.SelfLink,
			TargetNetworkSelfLink: pulumi.String(hubVpcRef),
		})
		if err != nil {
			return err
		}

		// NAT routers on spoke (spokes don't get BGP routers in hub-and-spoke)
		for _, reg := range []string{cfg.Region1, cfg.Region2} {
			_, err = networking.NewCloudRouter(ctx, fmt.Sprintf("spoke-nat-%s", reg), &networking.RouterArgs{
				ProjectID:       pulumi.String(cfg.SpokeProjectID),
				Region:          reg,
				Network:         spokeVpc.VPC.SelfLink,
				BgpAsn:          cfg.NatBgpAsn,
				EnableNat:       true,
				NatNumAddresses: cfg.NatNumAddresses,
			}, pulumi.DependsOn([]pulumi.Resource{spokeVpc.VPC}))
			if err != nil {
				return err
			}
		}

		// VPC-SC on spoke
		var perimeterName pulumi.StringOutput
		var accessLevelName pulumi.StringOutput
		var accessLevelDryRunName pulumi.StringOutput
		if cfg.PolicyID != "" {
			perimeter, err := vpc_sc.NewVpcServiceControls(ctx, "vpc-sc-perimeter", &vpc_sc.VpcServiceControlsArgs{
				PolicyID:              pulumi.String(cfg.PolicyID),
				Prefix:                fmt.Sprintf("%s_spoke", cfg.EnvCode),
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
			perimeterName = perimeter.Perimeter.Name
			accessLevelName = perimeter.AccessLevel.Name
			accessLevelDryRunName = perimeter.AccessLevelDryRun.Name
		} else {
			perimeterName = pulumi.String("").ToStringOutput()
			accessLevelName = pulumi.String("").ToStringOutput()
			accessLevelDryRunName = pulumi.String("").ToStringOutput()
		}

		var acmPolicyID pulumi.StringOutput
		if cfg.OrgStackName != "" {
			orgStack, err := pulumi.NewStackReference(ctx, "org", &pulumi.StackReferenceArgs{
				Name: pulumi.String(cfg.OrgStackName),
			})
			if err != nil {
				return err
			}
			acmPolicyID = orgStack.GetStringOutput(pulumi.String("access_context_manager_policy_id"))
		} else {
			acmPolicyID = pulumi.String("").ToStringOutput()
		}

		// Exports — matching TF 3-networks-hub-and-spoke/envs/{env}/outputs.tf
		ctx.Export("access_context_manager_policy_id", acmPolicyID)
		ctx.Export("shared_vpc_host_project_id", pulumi.String(cfg.SpokeProjectID))
		ctx.Export("network_name", spokeVpc.VPC.Name)
		ctx.Export("network_self_link", spokeVpc.VPC.SelfLink)

		// Subnet exports as arrays (matching TF subnets_names/ips/self_links/secondary_ranges)
		var subnetNames, subnetIPs, subnetSelfLinks pulumi.StringArray
		for _, subnet := range spokeVpc.Subnets {
			subnetNames = append(subnetNames, subnet.Name)
			subnetIPs = append(subnetIPs, subnet.IpCidrRange)
			subnetSelfLinks = append(subnetSelfLinks, subnet.SelfLink)
		}
		ctx.Export("subnets_names", subnetNames)
		ctx.Export("subnets_ips", subnetIPs)
		ctx.Export("subnets_self_links", subnetSelfLinks)
		ctx.Export("subnets_secondary_ranges", pulumi.ToStringArray([]string{}))
		ctx.Export("enforce_vpcsc", pulumi.Bool(cfg.EnforceVpcSc))
		ctx.Export("service_perimeter_name", perimeterName)
		ctx.Export("access_level_name", accessLevelName)
		ctx.Export("access_level_name_dry_run", accessLevelDryRunName)

		return nil
	})
}

type NetConfig struct {
	Env                           string
	EnvCode                       string
	HubProjectID                  string
	SpokeProjectID                string
	Region1                       string
	Region2                       string
	ParentID                      string
	Domain                        string
	PolicyID                      string
	OrgStackName                  string
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
		Env:            conf.Require("env"),
		EnvCode:        conf.Require("env_code"),
		HubProjectID:   conf.Require("hub_project_id"),
		SpokeProjectID: conf.Get("spoke_project_id"),
		Region1:        conf.Get("region1"),
		Region2:        conf.Get("region2"),
		ParentID:       conf.Get("parent_id"),
		Domain:         conf.Get("domain"),
		PolicyID:       conf.Get("policy_id"),
		OrgStackName:   conf.Get("org_stack_name"),
		PscIP:          conf.Get("psc_ip"),
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
	if c.OrgStackName == "" {
		c.OrgStackName = "org"
	}
	if c.PscIP == "" {
		c.PscIP = "10.17.0.6"
	}
	if c.ParentID == "" {
		c.ParentID = "organizations/123456789012"
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
