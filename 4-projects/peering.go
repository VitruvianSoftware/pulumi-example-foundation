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

	libnet "github.com/VitruvianSoftware/pulumi-library/pkg/networking"
	libproject "github.com/VitruvianSoftware/pulumi-library/pkg/project"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/dns"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/tags"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// deployPeeringNetwork creates the full peering network infrastructure for the
// peering project, matching upstream's example_peering_project.tf (305 lines).
//
// Creates:
//   - VPC with a single subnet (flow logs enabled)
//   - DNS policy with inbound forwarding + logging
//   - Bi-directional VPC peering to the shared VPC host
//   - Network firewall policy with mandatory + optional rules
//   - Secure tags for IAP SSH and RDP access
func deployPeeringNetwork(
	ctx *pulumi.Context,
	cfg *ProjectsConfig,
	peeringProject *libproject.Project,
	networkProjectID pulumi.StringOutput,
) (pulumi.StringOutput, error) {
	projectID := peeringProject.Project.ProjectId
	vpcName := fmt.Sprintf("vpc-%s-peering-base", cfg.EnvCode)

	// 1. VPC + Subnet
	peeringVpc, err := libnet.NewNetworking(ctx, "peering-vpc", &libnet.NetworkingArgs{
		ProjectID: projectID,
		VPCName:   pulumi.String(vpcName),
		Subnets: []libnet.SubnetArgs{
			{
				Name:     fmt.Sprintf("sb-%s-%s-peered-%s", cfg.EnvCode, cfg.BusinessCode, cfg.SubnetRegion),
				Region:   cfg.SubnetRegion,
				CIDR:     cfg.SubnetIPRange,
				FlowLogs: true,
			},
		},
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	// 2. DNS Policy — inbound forwarding + logging
	_, err = dns.NewPolicy(ctx, "peering-dns-policy", &dns.PolicyArgs{
		Project:                 projectID,
		Name:                    pulumi.String(fmt.Sprintf("dp-%s-peering-base-default-policy", cfg.EnvCode)),
		EnableInboundForwarding: pulumi.Bool(true),
		EnableLogging:           pulumi.Bool(true),
		Networks: dns.PolicyNetworkArray{
			&dns.PolicyNetworkArgs{
				NetworkUrl: peeringVpc.VPC.SelfLink,
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{peeringVpc.VPC}))
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	// 3. Bi-directional VPC Peering (peering project <-> shared VPC host)
	hostVpcRef := pulumi.Sprintf("projects/%s/global/networks/vpc-%s-svpc-base", networkProjectID, cfg.EnvCode)

	peeringToHost, err := compute.NewNetworkPeering(ctx, "peering-to-host", &compute.NetworkPeeringArgs{
		Network:            peeringVpc.VPC.SelfLink,
		PeerNetwork:        hostVpcRef,
		Name:               pulumi.String(fmt.Sprintf("%s-%s-peering-base-to-svpc", cfg.BusinessCode, cfg.EnvCode)),
		ExportCustomRoutes: pulumi.Bool(false),
		ImportCustomRoutes: pulumi.Bool(true),
	})
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	_, err = compute.NewNetworkPeering(ctx, "host-to-peering", &compute.NetworkPeeringArgs{
		Network:            hostVpcRef,
		PeerNetwork:        peeringVpc.VPC.SelfLink,
		Name:               pulumi.String(fmt.Sprintf("svpc-to-%s-%s-peering-base", cfg.BusinessCode, cfg.EnvCode)),
		ExportCustomRoutes: pulumi.Bool(true),
		ImportCustomRoutes: pulumi.Bool(false),
	}, pulumi.DependsOn([]pulumi.Resource{peeringToHost}))
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	// 4. Secure Tags for IAP-based SSH and RDP access
	if cfg.PeeringIAPFWEnabled {
		// SSH tag key + value
		sshTagKey, err := tags.NewTagKey(ctx, "peering-ssh-tag-key", &tags.TagKeyArgs{
			ShortName: pulumi.String("ssh-iap-access"),
			Parent:    pulumi.Sprintf("projects/%s", projectID),
			Purpose:   pulumi.String("GCE_FIREWALL"),
			PurposeData: pulumi.StringMap{
				"network": pulumi.Sprintf("%s/%s", projectID, vpcName),
			},
		}, pulumi.DependsOn([]pulumi.Resource{peeringVpc.VPC}))
		if err != nil {
			return pulumi.StringOutput{}, err
		}

		_, err = tags.NewTagValue(ctx, "peering-ssh-tag-value", &tags.TagValueArgs{
			ShortName: pulumi.String("allow"),
			Parent:    pulumi.Sprintf("tagKeys/%s", sshTagKey.Name),
		})
		if err != nil {
			return pulumi.StringOutput{}, err
		}

		// RDP tag key + value
		rdpTagKey, err := tags.NewTagKey(ctx, "peering-rdp-tag-key", &tags.TagKeyArgs{
			ShortName: pulumi.String("rdp-iap-access"),
			Parent:    pulumi.Sprintf("projects/%s", projectID),
			Purpose:   pulumi.String("GCE_FIREWALL"),
			PurposeData: pulumi.StringMap{
				"network": pulumi.Sprintf("%s/%s", projectID, vpcName),
			},
		}, pulumi.DependsOn([]pulumi.Resource{peeringVpc.VPC}))
		if err != nil {
			return pulumi.StringOutput{}, err
		}

		_, err = tags.NewTagValue(ctx, "peering-rdp-tag-value", &tags.TagValueArgs{
			ShortName: pulumi.String("allow"),
			Parent:    pulumi.Sprintf("tagKeys/%s", rdpTagKey.Name),
		})
		if err != nil {
			return pulumi.StringOutput{}, err
		}
	}

	// 5. Firewall Policy — matching upstream's firewall_rules module
	fwRules := buildPeeringFirewallRules(cfg)

	_, err = libnet.NewNetworkFirewallPolicy(ctx, "peering-fw", &libnet.NetworkFirewallPolicyArgs{
		ProjectID:  projectID,
		PolicyName: fmt.Sprintf("fp-%s-peering-project-firewalls", cfg.EnvCode),
		Description: fmt.Sprintf("Firewall rules for Peering Network: %s.", vpcName),
		TargetVPCs: []pulumi.StringInput{
			pulumi.Sprintf("projects/%s/global/networks/%s", projectID, peeringVpc.VPC.Name),
		},
		Rules: fwRules,
	}, pulumi.DependsOn([]pulumi.Resource{peeringVpc.VPC}))
	if err != nil {
		return pulumi.StringOutput{}, err
	}

	return peeringVpc.VPC.SelfLink, nil
}

// buildPeeringFirewallRules constructs the peering project's firewall rules
// using the library's FirewallRule struct with FirewallRuleMatch, matching
// upstream's example_peering_project.tf rule set.
func buildPeeringFirewallRules(cfg *ProjectsConfig) []libnet.FirewallRule {
	rules := []libnet.FirewallRule{
		// Priority 65530: Deny all egress TCP/UDP
		{
			Priority:      65530,
			Direction:     "EGRESS",
			Action:        "deny",
			RuleName:      fmt.Sprintf("fw-%s-peering-base-65530-e-d-all-all-tcp-udp", cfg.EnvCode),
			Description:   "Lower priority rule to deny all egress traffic.",
			EnableLogging: cfg.FirewallEnableLogging,
			Match: libnet.FirewallRuleMatch{
				DestIpRanges: []string{"0.0.0.0/0"},
				Layer4Configs: []libnet.FirewallLayer4Config{
					{IpProtocol: "tcp"},
					{IpProtocol: "udp"},
				},
			},
		},
		// Priority 10000: Allow Google Private APIs egress (199.36.153.8/30)
		{
			Priority:      10000,
			Direction:     "EGRESS",
			Action:        "allow",
			RuleName:      fmt.Sprintf("fw-%s-peering-base-10000-e-a-allow-google-apis-all-tcp-443", cfg.EnvCode),
			Description:   "Lower priority rule to allow private google apis on TCP port 443.",
			EnableLogging: cfg.FirewallEnableLogging,
			Match: libnet.FirewallRuleMatch{
				DestIpRanges: []string{"199.36.153.8/30"},
				Layer4Configs: []libnet.FirewallLayer4Config{
					{IpProtocol: "tcp", Ports: []string{"443"}},
				},
			},
		},
	}

	// IAP SSH rule (priority 1000)
	if cfg.PeeringIAPFWEnabled {
		rules = append(rules, libnet.FirewallRule{
			Priority:      1000,
			Direction:     "INGRESS",
			Action:        "allow",
			RuleName:      fmt.Sprintf("fw-%s-peering-base-1000-i-a-all-allow-iap-ssh-tcp-22", cfg.EnvCode),
			Description:   "Allow SSH via IAP for tagged instances.",
			EnableLogging: true,
			Match: libnet.FirewallRuleMatch{
				SrcIpRanges: []string{"35.235.240.0/20"}, // IAP forwarders
				Layer4Configs: []libnet.FirewallLayer4Config{
					{IpProtocol: "tcp", Ports: []string{"22"}},
				},
			},
		})

		// IAP RDP rule (priority 1001)
		rules = append(rules, libnet.FirewallRule{
			Priority:      1001,
			Direction:     "INGRESS",
			Action:        "allow",
			RuleName:      fmt.Sprintf("fw-%s-peering-base-1001-i-a-all-allow-iap-rdp-tcp-3389", cfg.EnvCode),
			Description:   "Allow RDP via IAP for tagged instances.",
			EnableLogging: true,
			Match: libnet.FirewallRuleMatch{
				SrcIpRanges: []string{"35.235.240.0/20"}, // IAP forwarders
				Layer4Configs: []libnet.FirewallLayer4Config{
					{IpProtocol: "tcp", Ports: []string{"3389"}},
				},
			},
		})
	}

	// Optional: Windows KMS activation (priority 0)
	if cfg.WindowsActivation {
		rules = append(rules, libnet.FirewallRule{
			Priority:      0,
			Direction:     "EGRESS",
			Action:        "allow",
			RuleName:      fmt.Sprintf("fw-%s-peering-base-0-e-a-allow-win-activation-all-tcp-1688", cfg.EnvCode),
			Description:   "Allow access to kms.windows.googlecloud.com for Windows license activation.",
			EnableLogging: cfg.FirewallEnableLogging,
			Match: libnet.FirewallRuleMatch{
				DestIpRanges: []string{"35.190.247.13/32"},
				Layer4Configs: []libnet.FirewallLayer4Config{
					{IpProtocol: "tcp", Ports: []string{"1688"}},
				},
			},
		})
	}

	// Optional: Load balancer health checks (priority 1000)
	if cfg.OptionalFWRulesEnabled {
		rules = append(rules, libnet.FirewallRule{
			Priority:      1002, // Offset from IAP rules which use 1000/1001
			Direction:     "INGRESS",
			Action:        "allow",
			RuleName:      fmt.Sprintf("fw-%s-peering-base-1002-i-a-all-allow-lb-tcp-80-8080-443", cfg.EnvCode),
			Description:   "Allow traffic for Internal & Global load balancing health check and load balancing IP ranges.",
			EnableLogging: cfg.FirewallEnableLogging,
			Match: libnet.FirewallRuleMatch{
				SrcIpRanges: []string{
					"35.191.0.0/16",
					"130.211.0.0/22",
					"209.85.152.0/22",
					"209.85.204.0/22",
				},
				Layer4Configs: []libnet.FirewallLayer4Config{
					{IpProtocol: "tcp", Ports: []string{"80", "8080", "443"}},
				},
			},
		})
	}

	return rules
}
