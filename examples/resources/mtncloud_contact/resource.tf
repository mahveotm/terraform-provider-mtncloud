# A monitoring contact: a notification target alerts can page.
resource "mtncloud_contact" "oncall" {
  name          = "on-call"
  email_address = "oncall@example.ng"
  sms_address   = "+2348000000000"
}
