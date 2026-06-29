# Changelog

All notable changes to this provider are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Monitoring resources and matching lookup data sources: `mtncloud_contact`,
  `mtncloud_monitoring_check`, `mtncloud_monitoring_group`, and
  `mtncloud_monitoring_alert`, with client support for `/monitoring/*`, payload
  unit tests, generated docs/examples, sweepers, and acceptance tests.

### Fixed

- `mtncloud_contact` notification fields can now be cleared on update by sending
  empty channel values instead of preserving prior API state.

## [0.2.12] - 2026-06-29

### Added

- Governance resources and matching lookup data sources: `mtncloud_role`
  (with a `permission_set` JSON document in the Morpheus role API shape),
  `mtncloud_user` (write-only passwords, role assignment via `role_ids`), and
  `mtncloud_user_group`. Client support for `/roles`, `/users`, and
  `/user-groups`, with payload unit tests and acceptance tests that assert
  renames update in place rather than recreate.

## [0.2.11] - 2026-06-29

### Added

- `mtncloud_task` now supports the `write_attributes` and `nested_workflow` task
  types (the remaining upstream task types the MTN token can actually create).
  Appliance-side scripting types (groovy, ruby, javascript) are intentionally
  omitted because the API denies creating them with this token.

## [0.2.10] - 2026-06-29

### Fixed

- Fixed `mtncloud_wiki_page.content` planning for heredocs with a trailing
  newline by treating the API-stored value as semantically equivalent instead of
  mutating the configured plan value.

## [0.2.9] - 2026-06-29

### Fixed

- Restored the standard `tfplugindocs` generation path and removed the
  unnecessary custom docs-generation wrapper.

## [0.2.8] - 2026-06-29

### Added

- Automation resources and matching lookup data sources: `mtncloud_task`
  (single type-discriminated resource covering shell, python, ansible, powershell,
  email, and restart tasks, with per-type config validation), `mtncloud_workflow`
  (operational and provisioning task-sets with ordered member tasks),
  `mtncloud_execute_schedule` (cron schedules), and `mtncloud_job` (run a workflow
  or task on a schedule against instance targets).
- Client support for `/tasks`, `/task-sets`, `/execute-schedules`, and `/jobs`,
  with unit tests covering payload shapes and the job `scheduleMode` encoding.
- Generated documentation, examples, sweepers, and acceptance tests (including a
  rename guard that asserts name changes update in place rather than recreate).

## [0.2.7] - 2026-06-29

### Added

- New resources and matching data sources for budgets, credentials, Cypher
  secrets, environments, IPv4 IP pools, key pairs, network domains, scale
  thresholds, and wiki pages.
- Client support for the new MTN Cloud API areas, including shared query helpers.
- Generated documentation and examples for the new resources and data sources.
- Architecture and contribution documentation for provider development.
- Provider, resource, client, sweep, and acceptance-test helpers covering the
  expanded provider surface.

### Changed

- Refactored provider configuration, resource/data-source wiring, framework value
  conversions, and diagnostics into shared helpers.
- Standardized computed resource IDs with `UseStateForUnknown` to avoid update
  plans losing known IDs.
- Improved security group rule state handling for API-defaulted or omitted fields
  to reduce apply churn.
- Updated CI/release tooling and dependency versions, including Terraform plugin
  framework packages, GoReleaser, GitHub Actions, golangci-lint, and gRPC.

### Fixed

- Release workflow reruns now replace release assets to avoid stale checksum
  artifacts.

## [0.1.0] - 2026-06-28

Initial release of the MTN Cloud Terraform provider.

### Added

- **Provider configuration** with OAuth (`username`/`password`) or `token`
  authentication, a configurable API `url`, request `timeout`, and `max_retries`.
- Provider-level defaults `group`, `resource_pool`, and `availability_zone` that
  resources inherit unless overridden (resource value wins, AWS-style).
- `default_labels` and `default_tags` merged into every resource via computed
  `labels_all` / `tags_all` so shared metadata applies without per-resource repetition.
- **Resources**
  - `mtncloud_instance` — provisions instances from human-friendly names (group,
    resource pool, instance type, service plan, image) resolved to IDs internally.
  - `mtncloud_network` — manages networks; group/zone/type/resource-pool given by name.
  - `mtncloud_security_group` and `mtncloud_security_group_rule`.
  - `mtncloud_storage_bucket` — S3-compatible bucket; `secret_key` is write-only
    (the API never returns it).
  - `mtncloud_archive_bucket` — archive bucket backed by a storage provider.
- **Data sources** — `mtncloud_group`, `mtncloud_resource_pool`,
  `mtncloud_service_plan`, `mtncloud_instance_type`, `mtncloud_virtual_image`,
  `mtncloud_network`, `mtncloud_security_group`.
- **Plan-time validation** — CIDR blocks, port ranges, protocol/direction/policy
  enums, visibility and retention-policy values, VLAN range, and positive
  day/timeout values.
- Automatic retry with exponential backoff and jitter (429 on any method; 5xx and
  network errors on GETs only) honoring `Retry-After`.
- Import support for all resources via `terraform import`.

[Unreleased]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.2.12...HEAD
[0.2.12]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.2.11...v0.2.12
[0.2.11]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.2.10...v0.2.11
[0.2.10]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.2.9...v0.2.10
[0.2.9]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.2.8...v0.2.9
[0.2.8]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.2.7...v0.2.8
[0.2.7]: https://github.com/mahveotm/terraform-provider-mtncloud/compare/v0.1.0...v0.2.7
[0.1.0]: https://github.com/mahveotm/terraform-provider-mtncloud/releases/tag/v0.1.0
