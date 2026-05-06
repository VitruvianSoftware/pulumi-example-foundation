package parentiamremoverole

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/folder"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ParentIamRemoveRoleArgs struct {
	ParentType string
	ParentId   pulumi.StringInput
	Roles      []string
}

type ParentIamRemoveRole struct {
	pulumi.ResourceState
}

func NewParentIamRemoveRole(ctx *pulumi.Context, name string, args *ParentIamRemoveRoleArgs, opts ...pulumi.ResourceOption) (*ParentIamRemoveRole, error) {
	var resource ParentIamRemoveRole
	err := ctx.RegisterComponentResource("modules:parent-iam-remove-role:ParentIamRemoveRole", name, &resource, opts...)
	if err != nil {
		return nil, err
	}

	for _, role := range args.Roles {
		roleID := strings.ReplaceAll(strings.TrimPrefix(role, "roles/"), ".", "-")
		
		if args.ParentType == "project" {
			_, err = projects.NewIAMBinding(ctx, fmt.Sprintf("%s-%s", name, roleID), &projects.IAMBindingArgs{
				Project: args.ParentId,
				Role:    pulumi.String(role),
				Members: pulumi.StringArray{},
			}, pulumi.Parent(&resource))
			if err != nil {
				return nil, err
			}
		} else if args.ParentType == "folder" {
			_, err = folder.NewIAMBinding(ctx, fmt.Sprintf("%s-%s", name, roleID), &folder.IAMBindingArgs{
				Folder:  args.ParentId,
				Role:    pulumi.String(role),
				Members: pulumi.StringArray{},
			}, pulumi.Parent(&resource))
			if err != nil {
				return nil, err
			}
		} else if args.ParentType == "organization" {
			_, err = organizations.NewIAMBinding(ctx, fmt.Sprintf("%s-%s", name, roleID), &organizations.IAMBindingArgs{
				OrgId:   args.ParentId,
				Role:    pulumi.String(role),
				Members: pulumi.StringArray{},
			}, pulumi.Parent(&resource))
			if err != nil {
				return nil, err
			}
		}
	}

	return &resource, nil
}
