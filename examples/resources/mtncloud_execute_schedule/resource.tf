resource "mtncloud_execute_schedule" "nightly" {
  name     = "nightly-2am"
  cron     = "0 2 * * *"
  timezone = "Africa/Lagos"
}
