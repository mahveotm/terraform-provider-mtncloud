# An HTTP monitoring check. check_type selects the monitor implementation and is
# immutable; per-type settings go in config as a JSON document.
resource "mtncloud_monitoring_check" "home" {
  name           = "home-page"
  check_type     = "webGetCheck"
  description     = "Front door health"
  severity       = "critical"
  check_interval = 120000 # milliseconds

  config = jsonencode({
    webUrl = "https://www.example.ng"
  })
}
