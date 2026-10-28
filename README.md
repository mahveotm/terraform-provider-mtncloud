# Terraform Provider for MTN Cloud

Community Terraform provider for [MTN Cloud](https://console.cloud.mtn.ng).

> Unofficial community project. Not affiliated with MTN Nigeria.

## Status

This provider covers the highest-value MTN Cloud workflows:

- Discover groups, instance types, virtual images, networks, resource pools,
  security groups, and service plans (data sources).
- Provision compute instances using human-friendly names.
- Manage networks (create/update/delete) using human-friendly group/zone/type/pool names.
- Manage security groups and security group rules.
- Manage MTN routers with resource-pool, subnet, and external-network inputs.
- Manage S3-compatible storage buckets and archive buckets.

## Example

```hcl
terraform {
  required_providers {
    mtncloud = {
      source  = "mahveotm/mtncloud"
      version = "~> 0.1"
    }
  }
}

provider "mtncloud" {
  token = var.mtncloud_token
}

resource "mtncloud_security_group" "web" {
  name        = "web-servers"
  description = "SSH and HTTPS access"
}

resource "mtncloud_security_group_rule" "ssh" {
  security_group_id = mtncloud_security_group.web.id
  name              = "allow-ssh"
  direction         = "ingress"
  policy            = "accept"
  protocol          = "tcp"
  port_range        = "22"
  source_type       = "cidr"
  source            = "203.0.113.10/32"
  destination_type  = "instance"
  ethertype         = "IPv4"
  enabled           = true
}

resource "mtncloud_instance" "web" {
  name          = "web-01"
  group         = "MTNNG_CLOUD_AZ_1"
  type          = "MTN-CS10"
  plan          = "G2S4"
  resource_pool = "my-project"

  security_group = mtncloud_security_group.web.name
  labels         = ["terraform", "web"]
}
```

## Authentication

Token authentication:

```hcl
provider "mtncloud" {
  token = var.mtncloud_token
}
```

Username/password authentication:

```hcl
provider "mtncloud" {
  username = var.mtncloud_username
  password = var.mtncloud_password
}
```

Supported environment variables:

- `MTN_CLOUD_URL`
- `MTN_CLOUD_TOKEN`
- `MTN_CLOUD_USERNAME`
- `MTN_CLOUD_PASSWORD`
- `MTN_CLOUD_TIMEOUT`
- `MTN_CLOUD_VERIFY_SSL`

## Local Development

```bash
make test
make build
make install-local
```

Then use the local development source address:

```hcl
terraform {
  required_providers {
    mtncloud = {
      source  = "mahveotm/mtncloud"
      version = "0.1.0"
    }
  }
}
```

## Router gateways

External router gateways can be discovered by name with
`mtncloud_external_network`, or listed with `mtncloud_external_networks`. See the
[router example](examples/resources/mtncloud_router/README.md) for the complete
workflow using existing networks.

## Import

```bash
terraform import mtncloud_instance.web 123
terraform import mtncloud_network.app 99
terraform import mtncloud_security_group.web 42
terraform import mtncloud_security_group_rule.ssh 42:99
terraform import mtncloud_storage_bucket.archive 7
terraform import mtncloud_archive_bucket.vault 12
```

## License

Licensed under the [Mozilla Public License 2.0](LICENSE).
