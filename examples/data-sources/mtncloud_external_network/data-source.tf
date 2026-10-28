# Group and resource_pool can be inherited from the provider.
data "mtncloud_external_network" "public" {
  name          = "public-network-01"
  group         = "MTNNG_CLOUD_AZ_1"
  resource_pool = "my-project"
}

output "external_network_id" {
  value = tonumber(data.mtncloud_external_network.public.id)
}
