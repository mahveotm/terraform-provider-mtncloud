# MTN selects its neutronRouter backend automatically.
resource "mtncloud_router" "app" {
  name          = "app-router"
  group         = "MTNNG_CLOUD_AZ_1"
  resource_pool = "my-project"

  # Morpheus subnet IDs belonging to the selected MTN resource pool.
  internal_subnet_ids = [123]

  # Optional Morpheus external network ID for an external gateway.
  # external_network_id = 34
}
