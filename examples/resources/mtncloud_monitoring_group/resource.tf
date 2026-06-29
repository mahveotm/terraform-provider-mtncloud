# A check group: several checks rolled up to one health status.
resource "mtncloud_monitoring_check" "home" {
  name       = "home-page"
  check_type = "webGetCheck"
  config     = jsonencode({ webUrl = "https://www.example.ng" })
}

resource "mtncloud_monitoring_group" "frontends" {
  name      = "frontends"
  min_happy = 1
  severity  = "warning"
  check_ids = [tonumber(mtncloud_monitoring_check.home.id)]
}
