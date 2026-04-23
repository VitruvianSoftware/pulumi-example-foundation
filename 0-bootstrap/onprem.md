# CI/CD Access to On-Prem

The infrastructure created for CI/CD allows access to on-prem resources through several connectivity options. This applies whether you use GitHub Actions with self-hosted runners, Cloud Build with [Private Pools](https://cloud.google.com/build/docs/private-pools/private-pools-overview), or GitLab CI/CD.

## Connectivity Options

| Option | Description | Documentation |
|--------|-------------|---------------|
| **HA VPN** | High Availability VPN with 99.99% SLA | [Overview](https://cloud.google.com/network-connectivity/docs/vpn/concepts/topologies#overview) |
| **Dedicated Interconnect** | Physical cross-connect at colocation facility | [Setup Guide](https://cloud.google.com/network-connectivity/docs/interconnect/tutorials/dedicated-creating-9999-availability) |
| **Partner Interconnect** | Connection via a supported service provider | [Setup Guide](https://cloud.google.com/network-connectivity/docs/interconnect/tutorials/partner-creating-9999-availability) |

## Architecture

For CI/CD workloads to reach on-prem resources, the following must be configured:

1. **Cloud Router** — Configure [Custom route advertisement mode](https://cloud.google.com/network-connectivity/docs/router/concepts/overview#route-advertisement-custom) so that CI/CD runners (GitHub Actions self-hosted runners, Cloud Build Private Pool instances) can reach on-prem networks.

2. **Network mode** — Connectivity can be configured in either network mode:
   - [Dual Shared VPC](https://cloud.google.com/architecture/security-foundations/networking#vpcsharedvpc-id7-1-shared-vpc-) (stage `3-networks-svpc`)
   - [Hub and Spoke](https://cloud.google.com/architecture/security-foundations/networking#hub-and-spoke) (stage `3-networks-hub-and-spoke`)

3. **Route exchange** — [Import and export custom routes](https://cloud.google.com/vpc/docs/vpc-peering#importing-exporting-routes) must be configured in any VPC peering setup.

## Enabling HA VPN (Optional)

The `3-networks-*` stages include optional VPN example files that can be activated:

1. Create a secret for the VPN pre-shared key and grant access to the CI/CD service account:

   ```bash
   export project_id=<ENV_SECRETS_PROJECT>
   export secret_name=<VPN_PSK_SECRET_NAME>
   export member="serviceAccount:<CI_CD_SERVICE_ACCOUNT>"

   echo '<YOUR-PRESHARED-KEY-SECRET>' | gcloud secrets create "${secret_name}" \
     --project "${project_id}" --replication-policy=automatic --data-file=-

   gcloud secrets add-iam-policy-binding "${secret_name}" \
     --member="${member}" --role='roles/secretmanager.viewer' \
     --project "${project_id}"

   gcloud secrets add-iam-policy-binding "${secret_name}" \
     --member="${member}" --role='roles/secretmanager.secretAccessor' \
     --project "${project_id}"
   ```

2. Activate the VPN example in `3-networks-svpc/` or `3-networks-hub-and-spoke/`:
   - Rename `vpn.ts.example` → `vpn.ts` (TypeScript) or `vpn.go.example` → `vpn.go` (Go)
   - Update the configuration values for your on-prem environment
   - Run `pulumi up` to provision the VPN tunnels

## Enabling Interconnect (Optional)

For dedicated or partner interconnect:

1. Activate the interconnect example:
   - Rename `interconnect.ts.example` → `interconnect.ts` (or `.go`)
   - Update interconnect location, VLAN tags, and candidate subnets
   - Run `pulumi up`

See the [Interconnect examples](../3-networks-svpc/modules/) in the network stages for details.
