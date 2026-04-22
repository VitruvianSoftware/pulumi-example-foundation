package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type EnvBaseArgs struct {
	Env                string
	BusinessUnit       string
	ProjectSuffix      string
	ProjectID          pulumi.StringInput
	Region             string
	SubnetworkSelfLink pulumi.StringInput
	IAPFirewallTags    pulumi.StringMapInput
}

func deployEnvBase(ctx *pulumi.Context, name string, args *EnvBaseArgs) error {
	// 1. Create Service Account
	sa, err := serviceaccount.NewAccount(ctx, name+"-sa", &serviceaccount.AccountArgs{
		AccountId:   pulumi.String("sa-example-app"),
		DisplayName: pulumi.String("Example app service Account"),
		Project:     args.ProjectID,
	})
	if err != nil {
		return err
	}

	// 2. Create Compute Instance
	instanceArgs := &compute.InstanceArgs{
		Project:     args.ProjectID,
		Name:        pulumi.String(fmt.Sprintf("example-app-%s", args.ProjectSuffix)),
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
			Email:  sa.Email,
			Scopes: pulumi.StringArray{pulumi.String("https://www.googleapis.com/auth/compute")},
		},
		Metadata: pulumi.StringMap{
			"block-project-ssh-keys": pulumi.String("true"),
		},
	}

	if args.IAPFirewallTags != nil {
		instanceArgs.Params = &compute.InstanceParamsArgs{
			ResourceManagerTags: args.IAPFirewallTags,
		}
	}

	_, err = compute.NewInstance(ctx, name+"-inst", instanceArgs)
	return err
}
