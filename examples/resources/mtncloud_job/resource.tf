# Run a workflow on a cron schedule.
resource "mtncloud_job" "nightly_maintenance" {
  name                = "nightly-maintenance"
  workflow_id         = mtncloud_workflow.maintenance.id
  schedule_mode       = "schedule"
  execute_schedule_id = mtncloud_execute_schedule.nightly.id
  target_type         = "instance"
  targets             = [123]
}

# Run a single task manually (triggered on demand).
resource "mtncloud_job" "adhoc_restart" {
  name           = "adhoc-restart"
  task_id        = mtncloud_task.deploy.id
  schedule_mode  = "manual"
  target_type    = "instance-label"
  instance_label = "web"
}
