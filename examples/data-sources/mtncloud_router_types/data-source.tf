data "mtncloud_router_types" "available" {}

output "router_types" {
  value = jsondecode(data.mtncloud_router_types.available.types_json)
}
