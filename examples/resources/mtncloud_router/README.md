# Router with existing networks

This example creates an MTN router using an existing resource pool and internal
network. It discovers the external gateway through `mtncloud_external_network`;
public gateway options are not necessarily visible to `mtncloud_network`.
Use a provider version containing the external-network data sources. These data
sources are not available in version 0.2.15.

Set `MTN_CLOUD_TOKEN` in your environment, then create `terraform.tfvars`:

```hcl
resource_pool         = "my-project"
internal_network_name = "app-net"
```

The internal network must belong to the selected resource pool. All its subnets
are attached; choose a network whose subnets are available for attachment.
The group defaults to `MTNNG_CLOUD_AZ_1`. If it has multiple clouds, set `cloud_id`
in `terraform.tfvars` explicitly. The example rejects an ambiguous cloud choice.

```shell
terraform init
terraform plan -out=router.tfplan
terraform apply router.tfplan
terraform plan -detailed-exitcode
```

Creation waits for backend readiness. An unchanged follow-up plan exits with code
0. To inspect all available gateways before choosing `external_network_name`, use
`mtncloud_external_networks` with the same group, cloud, and resource pool.
External IDs are Morpheus option values, not OpenStack UUIDs.

When the router is no longer needed:

```shell
terraform plan -destroy -out=destroy.tfplan
terraform apply destroy.tfplan
```

Only the router is managed by this example; the existing networks and pool remain.
