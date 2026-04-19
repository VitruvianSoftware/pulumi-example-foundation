# Troubleshooting

## Terminology

See [GLOSSARY.md](./GLOSSARY.md).

- - -

## Problems

- [Common issues](#common-issues)
- [Caller does not have permission in the Organization](#caller-does-not-have-permission-in-the-organization)
- [Billing quota exceeded](#billing-quota-exceeded)
- [Pulumi state lock errors](#pulumi-state-lock-errors)

- - -

## Common issues

- [Project quota exceeded](#project-quota-exceeded)
- [Stack reference not found](#stack-reference-not-found)
- [Application authenticated using end user credentials](#application-authenticated-using-end-user-credentials)
- [Cannot assign requested address error in Cloud Shell](#cannot-assign-requested-address-error-in-cloud-shell)
- [VPC peering rate limit exceeded](#vpc-peering-rate-limit-exceeded)

- - -

### Project quota exceeded

**Error message:**

```text
Error code 8, message: The project cannot be created because you have exceeded your allotted project quota
```

**Cause:**

This message means you have reached your [project creation quota](https://support.google.com/cloud/answer/6330231).

**Solution:**

Use the [Request Project Quota Increase](https://support.google.com/code/contact/project_quota_increase) form to request a quota increase.

In the support form, for the field **Email addresses that will be used to create projects**, use the email address of the `proj_sa_email` service account created by the 0-bootstrap step. You can retrieve it with:

```bash
cd 0-bootstrap
pulumi stack output proj_sa_email
```

**Notes:**

- If you see other quota errors, see the [Quota documentation](https://cloud.google.com/docs/quota).

### Stack reference not found

**Error message:**

```text
error: stack reference "organization/vitruvian/0-bootstrap/production" not found
```

or

```text
failed to load checkpoint: blob (key ".pulumi/stacks/...") (code=Unknown)
```

**Cause:**

The stages after `0-bootstrap` use [Pulumi Stack References](./GLOSSARY.md#pulumi-stack-reference) to read common configuration like the organization ID from the output of the `0-bootstrap` stage. The error means that either:
- The referenced stack has not been deployed yet
- The stack name is incorrect
- You are using a different backend than the one the referenced stack was deployed to

**Solution:**

1. Verify the referenced stack exists:
   ```bash
   pulumi stack ls
   ```
2. Ensure the stack name in your config matches the fully qualified name:
   ```bash
   pulumi config get bootstrap_stack_name
   ```
3. Ensure you are logged in to the same Pulumi backend where the referenced stack was deployed.

### Application authenticated using end user credentials

**Error message:**

```text
Error 403: Your application has authenticated using end user credentials from the Google Cloud SDK or Google Cloud Shell which are not supported by the X.googleapis.com.
```

**Cause:**

When using application default credentials in Cloud Shell, a billing project is not available for APIs like `securitycenter.googleapis.com` or `accesscontextmanager.googleapis.com`.

**Solution:**

Re-run the command using impersonation or a billing project:

```bash
# Provide impersonation
gcloud ... --impersonate-service-account=sa-terraform-org@<SEED_PROJECT_ID>.iam.gserviceaccount.com

# Or provide a billing project
gcloud ... --billing-project=<A-VALID-PROJECT-ID>
```

### Cannot assign requested address error in Cloud Shell

**Error message:**

```text
dial tcp [2607:f8b0:400c:c15::5f]:443: connect: cannot assign requested address
```

**Cause:**

This is a [known issue](https://github.com/hashicorp/terraform-provider-google/issues/6782) regarding IPv6 in Cloud Shell that also affects Pulumi's Google Cloud provider.

**Solution:**

1. Use a [workaround](https://stackoverflow.com/a/62827358) to force API calls to use the `private.googleapis.com` range (199.36.153.8/30), or
2. Deploy from a local machine that supports IPv6.

### VPC peering rate limit exceeded

**Error message:**

```text
Error 403: Rate Limit Exceeded ... CONCURRENT_OPERATIONS_QUOTA_EXCEEDED
```

**Cause:**

In the Hub and Spoke network mode, too many peering operations may occur simultaneously.

**Solution:**

This is a transient error. Wait at least one minute and retry the deploy:

```bash
pulumi up --refresh
```

- - -

### Caller does not have permission in the Organization

**Error message:**

```text
Error: Error when reading or editing Organization Not Found : <organization-id>: googleapi: Error 403: The caller does not have permission, forbidden
```

**Cause:**

User running Pulumi is missing the [Organization Administrator](https://cloud.google.com/iam/docs/understanding-roles#resource-manager-roles) role.

**Solution:**

Ensure the user has the required role, then re-authenticate:

```bash
gcloud auth application-default login
gcloud auth list  # confirm correct account is active
```

### Billing quota exceeded

**Error message:**

```text
Error: Error setting billing account "XXXXXX-XXXXXX-XXXXXX" for project: Precondition check failed., failedPrecondition
```

**Cause:**

Most likely a billing quota issue.

**Solution:**

```bash
gcloud alpha billing projects link projects/some-project --billing-account XXXXXX-XXXXXX-XXXXXX
```

If output states `Cloud billing quota exceeded`, use the [Request Billing Quota Increase](https://support.google.com/code/contact/billing_quota_increase) form.

### Pulumi state lock errors

**Error message:**

```text
error: the stack is currently locked by 1 lock(s). Either wait for the other process(es) to end or delete the lock.
```

**Cause:**

A previous Pulumi operation was interrupted (timeout, killed process), leaving the state locked.

**Solution:**

```bash
# Cancel the stuck update
pulumi cancel

# If that doesn't work, force unlock (use with caution)
pulumi stack export | pulumi stack import
```

**Warning:** Before force-unlocking, ensure no other Pulumi operations are actually running. Review the [Pulumi state documentation](https://www.pulumi.com/docs/concepts/state/) for more details.
