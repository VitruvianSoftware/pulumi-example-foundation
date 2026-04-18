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
		// 1. Enable Shared VPC Host Project
		// ====================================================================
		if _, err := compute.NewSharedVPCHostProject(ctx, "svpc-host", &compute.SharedVPCHostProjectArgs{
			Project: pulumi.String(cfg.ProjectID),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 2. Create VPC Network
		// Auto-create subnets disabled; default routes removed at creation
		// to enforce private networking.
		// ====================================================================
		vpc, err := compute.NewNetwork(ctx, "shared-vpc", &compute.NetworkArgs{
			Project:                       pulumi.String(cfg.ProjectID),
			Name:                          pulumi.String(fmt.Sprintf("vpc-%s-shared-base", cfg.Env)),
			AutoCreateSubnetworks:         pulumi.Bool(false),
			RoutingMode:                   pulumi.String("GLOBAL"),
			DeleteDefaultRoutesOnCreation: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// ====================================================================
		// 3. Create Subnets with GKE Secondary Ranges
		// Multi-region deployment with private Google access and flow logging.
		// ====================================================================
		type subnetDef struct {
			name, region, cidr, podCIDR, svcCIDR string
		}
		subnets := []subnetDef{
			{
				name:    fmt.Sprintf("sb-%s-shared-base-%s", cfg.Env, cfg.Region1),
				region:  cfg.Region1,
				cidr:    "10.0.64.0/21",
				podCIDR: "100.64.64.0/21",
				svcCIDR: "100.64.72.0/21",
			},
			{
				name:    fmt.Sprintf("sb-%s-shared-base-%s", cfg.Env, cfg.Region2),
				region:  cfg.Region2,
				cidr:    "10.1.64.0/21",
				podCIDR: "100.65.64.0/21",
				svcCIDR: "100.65.72.0/21",
			},
		}

		for _, s := range subnets {
			if _, err := compute.NewSubnetwork(ctx, s.name, &compute.SubnetworkArgs{
				Project:               pulumi.String(cfg.ProjectID),
				Name:                  pulumi.String(s.name),
				Network:               vpc.ID(),
				Region:                pulumi.String(s.region),
				IpCidrRange:           pulumi.String(s.cidr),
				PrivateIpGoogleAccess: pulumi.Bool(true),
				LogConfig: &compute.SubnetworkLogConfigArgs{
					AggregationInterval: pulumi.String("INTERVAL_5_SEC"),
					FlowSampling:        pulumi.Float64(0.5),
					Metadata:            pulumi.String("INCLUDE_ALL_METADATA"),
				},
				SecondaryIpRanges: compute.SubnetworkSecondaryIpRangeArray{
					&compute.SubnetworkSecondaryIpRangeArgs{
						RangeName:   pulumi.String(fmt.Sprintf("rn-%s-shared-%s-gke-pod", cfg.Env, s.region)),
						IpCidrRange: pulumi.String(s.podCIDR),
					},
					&compute.SubnetworkSecondaryIpRangeArgs{
						RangeName:   pulumi.String(fmt.Sprintf("rn-%s-shared-%s-gke-svc", cfg.Env, s.region)),
						IpCidrRange: pulumi.String(s.svcCIDR),
					},
				},
			}); err != nil {
				return err
			}
		}

		// ====================================================================
		// 4. Private Service Access (PSA) for Cloud SQL, Memorystore, etc.
		// ====================================================================
		if _, err := compute.NewGlobalAddress(ctx, "psa-range", &compute.GlobalAddressArgs{
			Project:      pulumi.String(cfg.ProjectID),
			Name:         pulumi.String(fmt.Sprintf("ga-%s-shared-base-vpc-peering", cfg.Env)),
			Purpose:      pulumi.String("VPC_PEERING"),
			AddressType:  pulumi.String("INTERNAL"),
			PrefixLength: pulumi.Int(16),
			Network:      vpc.ID(),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 5. Hierarchical Firewall Policies
		// Applied at the org/folder level to enforce network security baseline.
		// ====================================================================
		fwPolicy, err := compute.NewFirewallPolicy(ctx, "fw-policy", &compute.FirewallPolicyArgs{
			Parent:    pulumi.String(cfg.ParentID),
			ShortName: pulumi.String(fmt.Sprintf("fw-%s-shared-base", cfg.Env)),
		})
		if err != nil {
			return err
		}

		// Allow IAP TCP forwarding (SSH/RDP without public IPs)
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

		// Allow Load Balancer health checks
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

		// Allow Windows KMS activation
		if _, err := compute.NewFirewallPolicyRule(ctx, "fw-allow-windows-kms", &compute.FirewallPolicyRuleArgs{
			FirewallPolicy: fwPolicy.ID(),
			Priority:       pulumi.Int(300),
			Direction:      pulumi.String("EGRESS"),
			Action:         pulumi.String("allow"),
			Description:    pulumi.String("Allow Windows KMS activation"),
			Match: &compute.FirewallPolicyRuleMatchArgs{
				DestIpRanges: pulumi.StringArray{pulumi.String("35.190.247.13/32")},
				Layer4Configs: compute.FirewallPolicyRuleMatchLayer4ConfigArray{
					&compute.FirewallPolicyRuleMatchLayer4ConfigArgs{
						IpProtocol: pulumi.String("tcp"),
						Ports:      pulumi.StringArray{pulumi.String("1688")},
					},
				},
			},
		}); err != nil {
			return err
		}

		// Associate firewall policy with the parent org/folder
		if _, err := compute.NewFirewallPolicyAssociation(ctx, "fw-association", &compute.FirewallPolicyAssociationArgs{
			FirewallPolicy:   fwPolicy.ID(),
			AttachmentTarget: pulumi.String(cfg.ParentID),
			Name:             pulumi.String(fmt.Sprintf("fw-assoc-%s-shared-base", cfg.Env)),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 6. DNS Policy — logging and inbound forwarding
		// ====================================================================
		if _, err := dns.NewPolicy(ctx, "dns-policy", &dns.PolicyArgs{
			Project:                 pulumi.String(cfg.ProjectID),
			Name:                    pulumi.String(fmt.Sprintf("dp-%s-shared-base", cfg.Env)),
			EnableInboundForwarding: pulumi.Bool(true),
			EnableLogging:           pulumi.Bool(true),
			Networks: dns.PolicyNetworkArray{
				&dns.PolicyNetworkArgs{
					NetworkUrl: vpc.SelfLink,
				},
			},
		}); err != nil {
			return err
		}

		// ====================================================================
		// 7. Cloud NAT — outbound connectivity for private instances
		// ====================================================================
		for _, region := range []string{cfg.Region1, cfg.Region2} {
			router, err := compute.NewRouter(ctx, fmt.Sprintf("router-%s", region), &compute.RouterArgs{
				Project: pulumi.String(cfg.ProjectID),
				Name:    pulumi.String(fmt.Sprintf("cr-%s-shared-base-%s-router", cfg.Env, region)),
				Region:  pulumi.String(region),
				Network: vpc.SelfLink,
			})
			if err != nil {
				return err
			}

			if _, err := compute.NewRouterNat(ctx, fmt.Sprintf("nat-%s", region), &compute.RouterNatArgs{
				Project:                        pulumi.String(cfg.ProjectID),
				Router:                         router.Name,
				Region:                         pulumi.String(region),
				Name:                           pulumi.String(fmt.Sprintf("rn-%s-shared-base-%s", cfg.Env, region)),
				NatIpAllocateOption:             pulumi.String("AUTO_ONLY"),
				SourceSubnetworkIpRangesToNat:   pulumi.String("ALL_SUBNETWORKS_ALL_IP_RANGES"),
				LogConfig: &compute.RouterNatLogConfigArgs{
					Enable: pulumi.Bool(true),
					Filter: pulumi.String("ERRORS_ONLY"),
				},
			}); err != nil {
				return err
			}
		}

		// ====================================================================
		// 8. Route for Restricted Google APIs
		// Private Google Access via restricted.googleapis.com VIP.
		// This ensures API calls from VMs stay on Google's network.
		// ====================================================================
		if _, err := compute.NewRoute(ctx, "restricted-apis", &compute.RouteArgs{
			Project:        pulumi.String(cfg.ProjectID),
			Name:           pulumi.String(fmt.Sprintf("rt-%s-shared-base-restricted-apis", cfg.Env)),
			DestRange:      pulumi.String("199.36.153.4/30"),
			Network:        vpc.SelfLink,
			NextHopGateway: pulumi.String("default-internet-gateway"),
			Priority:       pulumi.Int(1000),
		}); err != nil {
			return err
		}

		// ====================================================================
		// 9. Exports
		// ====================================================================
		ctx.Export("network_id", vpc.ID())
		ctx.Export("network_name", vpc.Name)
		ctx.Export("network_self_link", vpc.SelfLink)

		return nil
	})
}

// NetConfig holds configuration for the networks stage.
type NetConfig struct {
	Env       string
	ProjectID string
	Region1   string
	Region2   string
	ParentID  string // "organizations/123" or "folders/456" for firewall policies
}

func loadNetConfig(ctx *pulumi.Context) *NetConfig {
	conf := config.New(ctx, "")
	c := &NetConfig{
		Env:       conf.Require("env"),
		ProjectID: conf.Require("project_id"),
		Region1:   conf.Get("region1"),
		Region2:   conf.Get("region2"),
		ParentID:  conf.Require("parent_id"),
	}
	if c.Region1 == "" {
		c.Region1 = "us-central1"
	}
	if c.Region2 == "" {
		c.Region2 = "us-west1"
	}
	return c
}
