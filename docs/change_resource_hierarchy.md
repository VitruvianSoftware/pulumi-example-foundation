# Resource Hierarchy Customizations

This document contains guidance for customizing the Cloud Resource Manager
resource hierarchy (folders and projects) in the Pulumi Foundation. It is the
Pulumi equivalent of the upstream Terraform foundation's
[change_resource_hierarchy.md](https://github.com/terraform-google-modules/terraform-example-foundation/blob/master/docs/change_resource_hierarchy.md).

## Default Hierarchy

The default deployment creates a flat hierarchy with environment folders at the
same level and three special folders:

| Folder | Description |
|--------|-------------|
| `fldr-bootstrap` | Contains the seed and CI/CD projects |
| `fldr-common` | Contains shared resources (logging, SCC, KMS, Secrets) |
| `fldr-network` | Contains network resources (DNS Hub, Interconnect, SVPCs) |
| `fldr-production` | Environment folder for production workloads |
| `fldr-nonproduction` | Environment folder for pre-production testing |
| `fldr-development` | Environment folder for development and sandboxing |

```
example-organization/
├── fldr-bootstrap
├── fldr-common
├── fldr-network
├── fldr-development
├── fldr-nonproduction
└── fldr-production
```

## Custom Hierarchy: Business Unit Sub-Folders

To create a deeper hierarchy with business unit folders under each environment:

```
example-organization/
├── fldr-bootstrap
├── fldr-common
├── fldr-network
├── fldr-development
│   ├── finance
│   └── retail
├── fldr-nonproduction
│   ├── finance
│   └── retail
└── fldr-production
    ├── finance
    └── retail
```

### Step 1: Modify Stage 2 (Environments)

In Stage 2, add sub-folder creation within each environment's deployment.

#### TypeScript

In `2-environments/envs/<env>/index.ts`, add folder resources after the
environment folder is created:

```typescript
import * as gcp from "@pulumi/gcp";

// The environment folder already exists from Stage 1
const envFolderId = orgStackRef.getOutput("development_folder_name");

// Create business unit sub-folders
const financeFolder = new gcp.organizations.Folder("finance", {
    displayName: "finance",
    parent: envFolderId,
});

const retailFolder = new gcp.organizations.Folder("retail", {
    displayName: "retail",
    parent: envFolderId,
});

// Export the folder hierarchy for downstream stages
export const folderHierarchy = {
    finance: financeFolder.name,
    retail: retailFolder.name,
};
```

#### Go

In `2-environments/main.go`, add folder creation within the Pulumi run function:

```go
financeFolder, err := organizations.NewFolder(ctx, "finance", &organizations.FolderArgs{
    DisplayName: pulumi.String("finance"),
    Parent:      envFolderID,
})
if err != nil {
    return err
}

retailFolder, err := organizations.NewFolder(ctx, "retail", &organizations.FolderArgs{
    DisplayName: pulumi.String("retail"),
    Parent:      envFolderID,
})
if err != nil {
    return err
}

ctx.Export("folder_hierarchy", pulumi.Map{
    "finance": financeFolder.Name,
    "retail":  retailFolder.Name,
})
```

### Step 2: Modify Stage 4 (Projects)

In Stage 4, read the folder hierarchy from the environments stage and create
projects under the appropriate business unit folder.

#### TypeScript

```typescript
// Read the folder hierarchy from Stage 2
const envStackRef = new pulumi.StackReference(envStackName);
const folderHierarchy = envStackRef.getOutput("folderHierarchy");

// Use the business unit folder instead of the environment folder
const buFolderID = folderHierarchy.apply(h => h["finance"]);

// Create projects under the BU folder
const project = new ProjectFactory("finance-app", {
    projectId: pulumi.interpolate`prj-${envCode}-fin-app-${randomSuffix}`,
    name: "Finance App",
    folderID: buFolderID,
    billingAccount: billingAccount,
    activateApis: ["compute.googleapis.com"],
});
```

#### Go

```go
folderHierarchy := envStackRef.GetOutput(pulumi.String("folder_hierarchy"))
buFolderID := folderHierarchy.ApplyT(func(v interface{}) string {
    m := v.(map[string]interface{})
    return m["finance"].(string)
}).(pulumi.StringOutput)

p, err := project.NewProject(ctx, "finance-app", &project.ProjectArgs{
    ProjectID:      pulumi.Sprintf("prj-%s-fin-app-%s", envCode, randomSuffix),
    Name:           pulumi.String("Finance App"),
    FolderID:       buFolderID,
    BillingAccount: pulumi.String(billingAccount),
    ActivateApis:   []string{"compute.googleapis.com"},
})
```

### Step 3: Adding New Business Units

To add a new business unit:

1. **Stage 2**: Add a new folder resource and export it in the `folderHierarchy` map
2. **Stage 4**: Duplicate the `business_unit_1` directory, update the `business_code` config value, and point to the new folder in the hierarchy
3. **Stage 5**: Duplicate the `business_unit_1` directory under `5-app-infra` with the new business code

### Key Differences from Terraform

| Aspect | Terraform | Pulumi |
|--------|-----------|--------|
| Hierarchy traversal | `tf-wrapper.sh` scans for matching directories by `max_depth` | Branch-to-stack mapping is native; no wrapper needed |
| Backend configuration | Each BU/env combo needs `backend.tf` with unique prefix | Pulumi stacks handle state isolation automatically |
| Cross-stage references | `terraform_remote_state` data source | Pulumi Stack References |
| BU code propagation | `sed` script to rename across `.tf` files | Change `business_code` config value per stack |

## Pulumi Advantages

1. **No wrapper script changes**: Pulumi's native branch-to-stack mapping means
   no `tf-wrapper.sh` depth configuration is needed
2. **Stack-level isolation**: Each business unit + environment combination can be
   a separate Pulumi stack, with automatic state isolation
3. **Type-safe config**: Business unit codes and folder references are validated
   at compile time (TypeScript) or build time (Go)
4. **Simplified duplication**: Copy a `business_unit_1` directory, change the
   `business_code` config, and deploy — no `sed` scripts needed
