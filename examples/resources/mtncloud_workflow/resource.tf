# Operational workflow: runs the referenced tasks on demand.
resource "mtncloud_workflow" "maintenance" {
  name = "nightly-maintenance"
  type = "operation"

  task {
    task_id = mtncloud_task.deploy.id
    phase   = "operation"
  }
  task {
    task_id = mtncloud_task.notify.id
    phase   = "operation"
  }
}

# Provisioning workflow: tasks run at specific provisioning phases.
resource "mtncloud_workflow" "provision" {
  name     = "web-provision"
  type     = "provision"
  platform = "linux"

  task {
    task_id = mtncloud_task.configure.id
    phase   = "postProvision"
  }
}
