data "mtncloud_network" "public" {
  name     = "public-network-01"
  cloud_id = 4
}

# Existing internal network in the router's resource pool.
data "mtncloud_network" "app" {
  name     = "app-net"
  cloud_id = 4
}

resource "mtncloud_router" "app" {
  name          = "app-router"
  group         = "MTNNG_CLOUD_AZ_1"
  cloud_id      = 4
  resource_pool = "my-project"

  external_network_id = tonumber(data.mtncloud_network.public.id)
  # Attach all subnets belonging to the internal network.
  internal_subnet_ids = data.mtncloud_network.app.subnet_ids
}
