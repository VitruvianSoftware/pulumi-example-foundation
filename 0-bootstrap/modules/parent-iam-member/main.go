package parentiammember

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/folder"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ParentIamMemberArgs struct {
	Member     pulumi.StringInput
	ParentType string
	ParentId   pulumi.StringInput
	Roles      []string
}

type ParentIamMember struct {
	pulumi.ResourceState
}

func NewParentIamMember(ctx *pulumi.Context, name string, args *ParentIamMemberArgs, opts ...pulumi.ResourceOption) (*ParentIamMember, error) {
	var resource ParentIamMember
	err := ctx.RegisterComponentResource("modules:parent-iam-member:ParentIamMember", name, &resource, opts...)
	if err != nil {
		return nil, err
	}

	for _, role := range args.Roles {
		roleID := strings.ReplaceAll(strings.TrimPrefix(role, "roles/"), ".", "-")
		
		if args.ParentType == "project" {
			_, err = projects.NewIAMMember(ctx, fmt.Sprintf("%s-%s", name, roleID), &projects.IAMMemberArgs{
				Project: args.ParentId,
				Role:    pulumi.String(role),
				Member:  args.Member,
			}, pulumi.Parent(&resource))
			if err != nil {
				return nil, err
			}
		} else if args.ParentType == "folder" {
			_, err = folder.NewIAMMember(ctx, fmt.Sprintf("%s-%s", name, roleID), &folder.IAMMemberArgs{
				Folder: args.ParentId,
				Role:   pulumi.String(role),
				Member: args.Member,
			}, pulumi.Parent(&resource))
			if err != nil {
				return nil, err
			}
		} else if args.ParentType == "organization" {
			_, err = organizations.NewIAMMember(ctx, fmt.Sprintf("%s-%s", name, roleID), &organizations.IAMMemberArgs{
				OrgId:  args.ParentId,
				Role:   pulumi.String(role),
				Member: args.Member,
			}, pulumi.Parent(&resource))
			if err != nil {
				return nil, err
			}
		}
	}

	return &resource, nil
}
