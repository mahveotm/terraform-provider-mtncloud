terraform {
  required_providers {
    mtncloud = {
      source = "mahveotm/mtncloud"
    }
  }
}

variable "group" {
  type    = string
  default = "MTNNG_CLOUD_AZ_1"
}

variable "resource_pool" {
  type        = string
  description = "Existing resource pool name or code."
}

variable "internal_network_name" {
  type        = string
  description = "Existing internal network in the selected resource pool. All its subnets will be attached."
}

variable "external_network_name" {
  type    = string
  default = "public-network-01"
}

variable "router_name" {
  type    = string
  default = "app-router"
}

variable "cloud_id" {
  type        = number
  default     = null
  description = "Set explicitly if the group contains multiple clouds."
}

# Set MTN_CLOUD_TOKEN in your environment.
provider "mtncloud" {
  group         = var.group
  resource_pool = var.resource_pool
}

data "mtncloud_group" "selected" {
  name = var.group
}

locals {
  cloud_id = var.cloud_id != null ? var.cloud_id : one(data.mtncloud_group.selected.cloud_ids)
}

data "mtncloud_external_network" "public" {
  name     = var.external_network_name
  cloud_id = local.cloud_id
}

data "mtncloud_network" "internal" {
  name     = var.internal_network_name
  cloud_id = local.cloud_id
}

resource "mtncloud_router" "app" {
  name     = var.router_name
  cloud_id = local.cloud_id

  external_network_id = tonumber(data.mtncloud_external_network.public.id)
  internal_subnet_ids = data.mtncloud_network.internal.subnet_ids
}

output "router_id" {
  value = mtncloud_router.app.id
}

output "router_external_ip" {
  value = mtncloud_router.app.external_ip
}
