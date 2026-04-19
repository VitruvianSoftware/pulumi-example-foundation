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

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadNetConfig(ctx)

		// ====================================================================
		// 1. Hub VPC — central routing point for the hub-and-spoke topology
		// ====================================================================
		hubVPC, err := compute.NewNetwork(ctx, "hub-vpc", &compute.NetworkArgs{
			Project:                       pulumi.String(cfg.HubProjectID),
			Name:                          pulumi.String(fmt.Sprintf("vpc-%s-hub", cfg.Env)),
			AutoCreateSubnetworks:         pulumi.Bool(false),
			RoutingMode:                   pulumi.String("GLOBAL"),
			DeleteDefaultRoutesOnCreate: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Hub subnets
		if _, err := compute.NewSubnetwork(ctx, "hub-subnet", &compute.SubnetworkArgs{
			Project:               pulumi.String(cfg.HubProjectID),
			Name:                  pulumi.String(fmt.Sprintf("sb-%s-hub-%s", cfg.Env, cfg.Region1)),
			Network:               hubVPC.ID(),
			Region:                pulumi.String(cfg.Region1),
			IpCidrRange:           pulumi.String("10.0.0.0/18"),
			PrivateIpGoogleAccess: pulumi.Bool(true),
			LogConfig: &compute.SubnetworkLogConfigArgs{
				AggregationInterval: pulumi.String("INTERVAL_5_SEC"),
				FlowSampling:        pulumi.Float64(0.5),
				Metadata:            pulumi.String("INCLUDE_ALL_METADATA"),
			},
		}); err != nil {
			return err
		}

		// ====================================================================
		// 2. Spoke VPC — workload VPC peered to the hub
		// ====================================================================
		spokeVPC, err := compute.NewNetwork(ctx, "spoke-vpc", &compute.NetworkArgs{
			Project:                       pulumi.String(cfg.SpokeProjectID),
			Name:                          pulumi.String(fmt.Sprintf("vpc-%s-spoke", cfg.Env)),
			AutoCreateSubnetworks:         pulumi.Bool(false),
			RoutingMode:                   pulumi.String("GLOBAL"),
			DeleteDefaultRoutesOnCreate: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Spoke subnets with GKE secondary ranges
		if _, err := compute.NewSubnetwork(ctx, "spoke-subnet", &compute.SubnetworkArgs{
			Project:               pulumi.String(cfg.SpokeProjectID),
			Name:                  pulumi.String(fmt.Sprintf("sb-%s-spoke-%s", cfg.Env, cfg.Region1)),
			Network:               spokeVPC.ID(),
			Region:                pulumi.String(cfg.Region1),
			IpCidrRange:           pulumi.String("10.1.0.0/18"),
			PrivateIpGoogleAccess: pulumi.Bool(true),
			LogConfig: &compute.SubnetworkLogConfigArgs{
				AggregationInterval: pulumi.String("INTERVAL_5_SEC"),
				FlowSampling:        pulumi.Float64(0.5),
				Metadata:            pulumi.String("INCLUDE_ALL_METADATA"),
			},
			SecondaryIpRanges: compute.SubnetworkSecondaryIpRangeArray{
				&compute.SubnetworkSecondaryIpRangeArgs{
					RangeName:   pulumi.String(fmt.Sprintf("rn-%s-spoke-%s-gke-pod", cfg.Env, cfg.Region1)),
					IpCidrRange: pulumi.String("100.66.0.0/16"),
				},
				&compute.SubnetworkSecondaryIpRangeArgs{
					RangeName:   pulumi.String(fmt.Sprintf("rn-%s-spoke-%s-gke-svc", cfg.Env, cfg.Region1)),
					IpCidrRange: pulumi.String("100.67.0.0/16"),
				},
			},
		}); err != nil {
			return err
		}

		// ====================================================================
		// 3. VPC Peering — hub ↔ spoke bidirectional peering
		// Export custom routes from hub so spokes can route through it.
		// ====================================================================
		if _, err := compute.NewNetworkPeering(ctx, "hub-to-spoke", &compute.NetworkPeeringArgs{
			Network:              hubVPC.SelfLink,
			PeerNetwork:          spokeVPC.SelfLink,
			Name:                 pulumi.String(fmt.Sprintf("peer-%s-hub-to-spoke", cfg.Env)),
			ExportCustomRoutes:   pulumi.Bool(true),
			ImportCustomRoutes:   pulumi.Bool(false),
			ExportSubnetRoutesWithPublicIp: pulumi.Bool(true),
		}); err != nil {
			return err
		}

		if _, err := compute.NewNetworkPeering(ctx, "spoke-to-hub", &compute.NetworkPeeringArgs{
			Network:              spokeVPC.SelfLink,
			PeerNetwork:          hubVPC.SelfLink,
			Name:                 pulumi.String(fmt.Sprintf("peer-%s-spoke-to-hub", cfg.Env)),
			ExportCustomRoutes:   pulumi.Bool(false),
			ImportCustomRoutes:   pulumi.Bool(true),
			ExportSubnetRoutesWithPublicIp: pulumi.Bool(true),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 4. Hierarchical Firewall Policies
		// ====================================================================
		fwPolicy, err := compute.NewFirewallPolicy(ctx, "fw-policy", &compute.FirewallPolicyArgs{
			Parent:    pulumi.String(cfg.ParentID),
			ShortName: pulumi.String(fmt.Sprintf("fw-%s-hub-spoke", cfg.Env)),
		})
		if err != nil {
			return err
		}

		// Allow IAP TCP forwarding
		if _, err := compute.NewFirewallPolicyRule(ctx, "fw-allow-iap", &compute.FirewallPolicyRuleArgs{
			FirewallPolicy: fwPolicy.ID(),
			Priority:       pulumi.Int(100),
			Direction:      pulumi.String("INGRESS"),
			Action:         pulumi.String("allow"),
			Description:    pulumi.String("Allow IAP TCP forwarding for SSH and RDP"),
			Match: &compute.FirewallPolicyRuleMatchArgs{
				SrcIpRanges: pulumi.StringArray{pulumi.String("35.235.240.0/20")},
				Layer4Configs: compute.FirewallPolicyRuleMatchLayer4ConfigArray{
					&compute.FirewallPolicyRuleMatchLayer4ConfigArgs{
						IpProtocol: pulumi.String("tcp"),
						Ports:      pulumi.StringArray{pulumi.String("22"), pulumi.String("3389")},
					},
				},
			},
		}); err != nil {
			return err
		}

		// Allow health checks
		if _, err := compute.NewFirewallPolicyRule(ctx, "fw-allow-health-checks", &compute.FirewallPolicyRuleArgs{
			FirewallPolicy: fwPolicy.ID(),
			Priority:       pulumi.Int(200),
			Direction:      pulumi.String("INGRESS"),
			Action:         pulumi.String("allow"),
			Description:    pulumi.String("Allow health check probes from GCP load balancers"),
			Match: &compute.FirewallPolicyRuleMatchArgs{
				SrcIpRanges: pulumi.StringArray{
					pulumi.String("130.211.0.0/22"),
					pulumi.String("35.191.0.0/16"),
				},
				Layer4Configs: compute.FirewallPolicyRuleMatchLayer4ConfigArray{
					&compute.FirewallPolicyRuleMatchLayer4ConfigArgs{
						IpProtocol: pulumi.String("tcp"),
					},
				},
			},
		}); err != nil {
			return err
		}

		// Associate firewall policy
		if _, err := compute.NewFirewallPolicyAssociation(ctx, "fw-association", &compute.FirewallPolicyAssociationArgs{
			FirewallPolicy:   fwPolicy.ID(),
			AttachmentTarget: pulumi.String(cfg.ParentID),
			Name:             pulumi.String(fmt.Sprintf("fw-assoc-%s-hub-spoke", cfg.Env)),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 5. DNS Policy — logging and inbound forwarding on hub VPC
		// ====================================================================
		if _, err := dns.NewPolicy(ctx, "dns-policy", &dns.PolicyArgs{
			Project:                 pulumi.String(cfg.HubProjectID),
			Name:                    pulumi.String(fmt.Sprintf("dp-%s-hub", cfg.Env)),
			EnableInboundForwarding: pulumi.Bool(true),
			EnableLogging:           pulumi.Bool(true),
			Networks: dns.PolicyNetworkArray{
				&dns.PolicyNetworkArgs{
					NetworkUrl: hubVPC.SelfLink,
				},
			},
		}); err != nil {
			return err
		}

		// ====================================================================
		// 6. Cloud NAT — on both hub and spoke VPCs
		// ====================================================================
		for _, vpcEntry := range []struct {
			name, project string
			vpc           *compute.Network
		}{
			{"hub", cfg.HubProjectID, hubVPC},
			{"spoke", cfg.SpokeProjectID, spokeVPC},
		} {
			router, err := compute.NewRouter(ctx, fmt.Sprintf("router-%s", vpcEntry.name), &compute.RouterArgs{
				Project: pulumi.String(vpcEntry.project),
				Name:    pulumi.String(fmt.Sprintf("cr-%s-%s-%s-router", cfg.Env, vpcEntry.name, cfg.Region1)),
				Region:  pulumi.String(cfg.Region1),
				Network: vpcEntry.vpc.SelfLink,
			})
			if err != nil {
				return err
			}

			if _, err := compute.NewRouterNat(ctx, fmt.Sprintf("nat-%s", vpcEntry.name), &compute.RouterNatArgs{
				Project:                      pulumi.String(vpcEntry.project),
				Router:                       router.Name,
				Region:                       pulumi.String(cfg.Region1),
				Name:                         pulumi.String(fmt.Sprintf("rn-%s-%s-%s", cfg.Env, vpcEntry.name, cfg.Region1)),
				NatIpAllocateOption:          pulumi.String("AUTO_ONLY"),
				SourceSubnetworkIpRangesToNat: pulumi.String("ALL_SUBNETWORKS_ALL_IP_RANGES"),
				LogConfig: &compute.RouterNatLogConfigArgs{
					Enable: pulumi.Bool(true),
					Filter: pulumi.String("ERRORS_ONLY"),
				},
			}); err != nil {
				return err
			}
		}

		// ====================================================================
		// 7. Restricted Google APIs route on hub VPC
		// ====================================================================
		if _, err := compute.NewRoute(ctx, "restricted-apis", &compute.RouteArgs{
			Project:        pulumi.String(cfg.HubProjectID),
			Name:           pulumi.String(fmt.Sprintf("rt-%s-hub-restricted-apis", cfg.Env)),
			DestRange:      pulumi.String("199.36.153.4/30"),
			Network:        hubVPC.SelfLink,
			NextHopGateway: pulumi.String("default-internet-gateway"),
			Priority:       pulumi.Int(1000),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 8. Exports
		// ====================================================================
		ctx.Export("hub_vpc_id", hubVPC.ID())
		ctx.Export("hub_vpc_self_link", hubVPC.SelfLink)
		ctx.Export("spoke_vpc_id", spokeVPC.ID())
		ctx.Export("spoke_vpc_self_link", spokeVPC.SelfLink)

		return nil
	})
}

// NetConfig holds configuration for the hub-and-spoke networks stage.
type NetConfig struct {
	Env            string
	HubProjectID   string
	SpokeProjectID string
	Region1        string
	Region2        string
	ParentID       string // "organizations/123" or "folders/456"
}

func loadNetConfig(ctx *pulumi.Context) *NetConfig {
	conf := config.New(ctx, "")
	c := &NetConfig{
		Env:            conf.Require("env"),
		HubProjectID:   conf.Require("hub_project_id"),
		SpokeProjectID: conf.Require("spoke_project_id"),
		Region1:        conf.Get("region1"),
		Region2:        conf.Get("region2"),
		ParentID:       conf.Require("parent_id"),
	}
	if c.Region1 == "" {
		c.Region1 = "us-central1"
	}
	if c.Region2 == "" {
		c.Region2 = "us-west1"
	}
	return c
}
