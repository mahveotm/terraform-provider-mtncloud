# Discover the actual type code and its options using mtncloud_router_types.
# This variable must identify a creatable type without a network server.
variable "router_type" {
  type        = string
  description = "Router type code available in your account."
}

resource "mtncloud_router" "app" {
  name  = "app-router"
  group = "MTNNG_CLOUD_AZ_1"
  type  = var.router_type

  # Set cloud_id if the group has more than one cloud.
  # Supply any required type-specific options using options_json = jsonencode({...}).
  # Use the API optionTypes metadata; option names vary by backend.
}
