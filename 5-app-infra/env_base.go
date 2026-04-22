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
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// EnvBaseArgs configures a standard Compute Instance deployment,
// matching the upstream Terraform env_base module.
type EnvBaseArgs struct {
	Env                string
	BusinessUnit       string
	ProjectSuffix      string
	Hostname           string // upstream default: "example-app"
	ProjectID          pulumi.StringInput
	Region             string
	SubnetworkSelfLink pulumi.StringInput
	IAPFirewallTags    pulumi.StringMapInput // nil for non-peering projects
}

// EnvBaseResult holds outputs from the env_base deployment.
type EnvBaseResult struct {
	InstanceSelfLink pulumi.StringOutput
}

// deployEnvBase creates a service account and Compute Instance in the target
// project, matching the upstream Terraform foundation's env_base module.
//
// Upstream uses terraform-google-modules/vm which creates an instance template
// then a compute instance from that template. Here we create the instance
// directly since we're not using the TF module wrapper.
func deployEnvBase(ctx *pulumi.Context, name string, args *EnvBaseArgs) (*EnvBaseResult, error) {
	hostname := args.Hostname
	if hostname == "" {
		hostname = "example-app"
	}

	// 1. Service Account — matching upstream's google_service_account.compute_engine_service_account
	sa, err := serviceaccount.NewAccount(ctx, name+"-sa", &serviceaccount.AccountArgs{
		AccountId:   pulumi.String("sa-example-app"),
		DisplayName: pulumi.String("Example app service Account"),
		Project:     args.ProjectID,
	})
	if err != nil {
		return nil, err
	}

	// 2. Compute Instance
	instanceArgs := &compute.InstanceArgs{
		Project:     args.ProjectID,
		Name:        pulumi.String(fmt.Sprintf("%s-%s", hostname, args.ProjectSuffix)),
		MachineType: pulumi.String("f1-micro"),
		Zone:        pulumi.String(args.Region + "-a"),
		BootDisk: &compute.InstanceBootDiskArgs{
			InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
				Image: pulumi.String("debian-cloud/debian-11"),
			},
		},
		NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
			&compute.InstanceNetworkInterfaceArgs{
				Subnetwork: args.SubnetworkSelfLink,
			},
		},
		ServiceAccount: &compute.InstanceServiceAccountArgs{
			Email: sa.Email,
			// upstream: scopes = ["compute-rw"]
			Scopes: pulumi.StringArray{pulumi.String("https://www.googleapis.com/auth/compute")},
		},
		Metadata: pulumi.StringMap{
			"block-project-ssh-keys": pulumi.String("true"),
		},
	}

	// IAP firewall tags — applied via params.resource_manager_tags, matching
	// the upstream TF vm module's resource_manager_tags variable which maps
	// to the same google_compute_instance params block.
	if args.IAPFirewallTags != nil {
		instanceArgs.Params = &compute.InstanceParamsArgs{
			ResourceManagerTags: args.IAPFirewallTags,
		}
	}

	inst, err := compute.NewInstance(ctx, name+"-inst", instanceArgs)
	if err != nil {
		return nil, err
	}

	return &EnvBaseResult{
		InstanceSelfLink: inst.SelfLink,
	}, nil
}
