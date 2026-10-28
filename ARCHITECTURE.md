# Architecture

This document defines how `terraform-provider-mtncloud` is structured and the
conventions every resource follows. It is the source of truth for "how we build
here" — read it before adding a resource.

## What this provider is

A [terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework)
provider for **MTN Cloud**, a [Morpheus](https://morpheus.cloud)-backed platform whose
REST API lives at `https://console.cloud.mtn.ng/api`. The provider is single-service,
hand-written (no code generation), and published to the Terraform Registry.

## Design principles

1. **Explicit over clever.** Each resource is a readable, self-contained file. We
   share *boilerplate*, not *behavior* — there is no generic CRUD engine to learn.
2. **Thin, uniform client.** One Go method per API operation, built on a small set
   of generic envelope helpers. The provider layer never builds HTTP requests.
3. **Behaviour lives at the edges, conventions at the center.** Repeated mechanics
   (Configure, ID parsing, drift handling, error formatting, JSON envelopes) are
   extracted once; per-resource code is just schema + field mapping.
4. **Deliberate, layered tests.** A schema test runs in CI with no credentials;
   acceptance tests exercise the real API behind `TF_ACC`; sweepers clean up.

## Layering

```
HCL ──► provider (internal/provider) ──► client (internal/client) ──► MTN Cloud API
        schemas, state, plan logic       typed Go methods, JSON       HTTP/JSON
```

The provider layer depends on the client layer; the client layer knows nothing
about Terraform. Keep it that way.

## Package map

```
internal/client/
  client.go         HTTP core: auth, retries, get/post/put/delete, decodeResponse
  errors.go         APIError, IsNotFound
  query.go          generic envelope helpers — getByID / firstByName / createObj /
                    updateObj / listObjects (decode the {"x":…}/{"xs":[…]} wrappers)
  <area>.go         per-API-area: struct(s) with json tags, payload builder(s),
                    and thin method wrappers that call the query.go generics

internal/provider/
  provider.go       provider struct, Schema (auth + defaults), Configure, Metadata,
                    and the Resources()/DataSources() registration lists
  configure.go      resourceBase / dataSourceBase mixins (Configure + client/defaults),
                    configuredProvider/configuredClient, env-resolution helpers
  conversions.go    framework<->Go helpers: *Ptr, optionalString, maybe*, mergeAPI*,
                    mergeLabels, mergeTags
  diagnostics.go    parseID, handleReadError, opError — standardized lifecycle diags
  validators.go     shared attribute validators (validCIDR, validPortRange, …)
  resource_<x>.go        one resource (embeds resourceBase)
  data_source_<x>.go     one data source (embeds dataSourceBase)
  *_acc_test.go          TF_ACC acceptance tests
  provider_schema_test.go  always-on schema/registration guardrail
  sweep_test.go / acc_helpers_test.go  generic sweeper + acc-test helpers
```

## The shared seams (use these — don't re-implement)

- **`resourceBase` / `dataSourceBase`** (`configure.go`): embed in every resource /
  data source. They implement `Configure` and expose `r.client` (and `r.defaults`
  for resources that inherit provider-level group/labels/tags). Do **not** add a
  per-resource `Configure`.
- **`parseID(state.ID, "Label", &resp.Diagnostics) (int64, ok)`** — turn the string
  state ID into the numeric API id.
- **`handleReadError(ctx, err, "Label", &resp.State, &resp.Diagnostics) bool`** — in
  Read, a 404 removes the resource (drift); any other error is surfaced. Returns
  `true` when you should `return`.
- **`opError(&resp.Diagnostics, "Create"|"Update"|"Delete", "Label", err)`** — one
  consistent diagnostic string per failed operation.
- **client `query.go` generics** — a new client method is one line:
  `return firstByName[Credential](c, ctx, "/credentials", "credentials", name)`.

## Conventions (every resource follows these)

- **One file per resource and per data source.** Split a resource into
  `resource_<x>_{create,read,update,delete}.go` only when a single file exceeds
  ~300 lines. (`resource_instance.go` and `resource_network.go` are the current
  candidates — split them when next touched.)
- **IDs** are API-numeric, stored as `types.String`; parse with `parseID`.
- **Optional+Computed** string/number/bool fields reconcile via `mergeAPI*` so an
  absent API value never nulls a configured one. Plain Required/Optional fields are
  set directly.
- **Write-only secrets** (passwords, private keys, cypher value, domain password)
  are `Sensitive`, are sent on create/update, and are **never read back** — keep the
  prior state value (the API returns only masked/hashed forms).
- **Immutable fields** use `stringplanmodifier.RequiresReplace()` and say
  "Changing it forces a new …" in the description.
- **Data sources look up by name** via `firstByName`; **resources read by id**.
- **Errors** go through `opError`; **drift** through `handleReadError`.
- **JSON envelope keys are validated, not assumed.** Single-object and list wrappers
  differ (e.g. `page` vs `pages`); confirm against a live response or openapi.yaml.
- The provider's published type prefix is `mtncloud_` (see `Metadata`).

## Testing pyramid

| Layer | Where | Runs | Asserts |
|-------|-------|------|---------|
| Client unit | `internal/client/*_test.go` (httptest) | always / CI | request payload shape, wrapper keys, mappings |
| Provider schema | `internal/provider/provider_schema_test.go` | always / CI | every resource+data-source schema is valid; unique `mtncloud_` names |
| Acceptance | `internal/provider/*_acc_test.go` (`TF_ACC=1`) | manual / nightly | real create→update→import against the API |
| Sweeper | `internal/provider/sweep_test.go` (`-sweep`) | manual | deletes leftover `tf-acc-*` / `tf-smoke-*` objects |

- `make test` — unit + schema (no credentials needed).
- `make testacc` — acceptance (needs `MTN_CLOUD_TOKEN`; tokens expire ~hourly).
- `go test ./internal/provider -sweep=mtn` — clean up after a failed acc run.

Acceptance tests name every object with `accName("kind")` → `tf-acc-…` so the
sweeper can find them. Add a sweeper for each new resource that has a `List*`
client method (see `sweep_test.go`).

## Adding a resource — checklist

1. **Client** (`internal/client/<area>.go`): add the `Struct` (json tags), an
   `Input` struct + payload builder, and thin `Create/Get/GetByName/Update/Delete`
   methods using the `query.go` generics. Confirm the envelope wrapper keys.
2. **Resource** (`internal/provider/resource_<x>.go`): embed `resourceBase`; write
   `Metadata`, `Schema`, `Create/Read/Update/Delete`, `ImportState`, and a
   `set<X>State` mapper. Use `parseID` / `handleReadError` / `opError`.
3. **Data source** (`internal/provider/data_source_<x>.go`): embed `dataSourceBase`;
   look up by name via the client.
4. **Register** both `New*` constructors in `provider.go`.
5. **Example** (`examples/resources/mtncloud_<x>/resource.tf` and the data-source
   equivalent) — these are embedded into the generated docs.
6. **Docs**: `make docs` (tfplugindocs regenerates `docs/` from schema + examples).
7. **Tests**: a client unit test for any non-trivial payload mapping; an
   `*_acc_test.go` following `resource_credential_acc_test.go`; a sweeper entry if a
   `List*` method exists.
8. `make check` (build + vet + lint + unit test + gofmt) must be green.

## Client layer migration note

The Tier-0/Tier-1 resource client files use the `query.go` generics. A few older,
heterogeneous files (`networks.go`, `instances.go`, `security_groups.go` with its
nested rules, the bucket files, `groups.go` with its `normalize()`) still inline the
envelope decode. They are correct and tested; migrate them to the generics
opportunistically when next edited — do not rewrite core provisioning paths purely
for uniformity.

## Build, CI & release

- **CI** (`.github/workflows/test.yml`): build, vet, gofmt, golangci-lint, unit
  tests, and a docs-drift check on every push/PR.
- **Docs** are generated by `tfplugindocs` from schema descriptions + `examples/`
  via `go generate ./...`; CI fails if `docs/` is stale.
- **Release** (`.github/workflows/release.yml`): pushing a `vX.Y.Z` tag runs
  GoReleaser, which builds signed archives in the Registry's required layout.
- **Local install** for manual testing: `make install-local`.

## Out of scope (deliberate non-choices)

No subproviders, no schema code generation, and no generic CRUD scaffold. They suit
a multi-service, 100+-resource provider (see the sibling `terraform-provider-hpe`),
not this one.
