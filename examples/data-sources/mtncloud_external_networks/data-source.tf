data "mtncloud_external_networks" "available" {
  group         = "MTNNG_CLOUD_AZ_1"
  resource_pool = "my-project"
}

output "external_networks" {
  value = data.mtncloud_external_networks.available.networks
}
