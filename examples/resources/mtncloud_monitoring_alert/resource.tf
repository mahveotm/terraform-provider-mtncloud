# An alert that pages a contact when any check reaches critical severity.
resource "mtncloud_contact" "oncall" {
  name          = "on-call"
  email_address = "oncall@example.ng"
}

resource "mtncloud_monitoring_alert" "page_oncall" {
  name         = "page-on-call"
  all_checks   = true
  min_severity = "critical"
  contact_ids  = [tonumber(mtncloud_contact.oncall.id)]
}
