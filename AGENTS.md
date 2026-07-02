# AGENTS.md

## What this is

UpCloud OpenTofu provider — all resources migrated to Plugin Framework (v6). No SDKv2 or mux. Serves at `registry.opentofu.org/upcloudltd/upcloud-community`.

## Build & test

make                    # build + fmtcheck
make test               # unit tests only (excludes upcloud/ acceptance)
make testacc            # all acceptance tests (requires credentials)
make testacc-gateway    # single service acceptance group

Unit tests live in `internal/..._test.go`. Acceptance tests in `upcloud/<service>/..._test.go` require `TF_ACC=1`. The `init()` in `test_helpers.go` auto-detects `tofu` in PATH and sets `TF_ACC_TERRAFORM_PATH`, `TF_ACC_PROVIDER_NAMESPACE=hashicorp`, `TF_ACC_PROVIDER_HOST=registry.opentofu.org` — no manual env vars needed.

## Credentials

Acceptance tests need `UPCLOUD_USERNAME` + `UPCLOUD_PASSWORD` or `UPCLOUD_TOKEN`.

## CI matrix

`.github/acctest-path-mapping.json` maps source path prefixes to `make testacc-<group>` targets. When adding a new service, add both the `Makefile` target and the mapping entry.

## Code structure

| Path | Purpose |
|---|---|
| `main.go` | Entry point, serves Framework provider |
| `upcloud/provider.go` | Provider + all resource/data source registrations |
| `upcloud/test_helpers.go` | Shared test infrastructure |
| `internal/service/<name>/` | Resource/data source implementations (Framework) |
| `internal/utils/` | Shared helpers: ID parsing, labels, errors, waiting |
| `internal/config/` | Provider config & API client setup |

## Testing quirks

- Acceptance tests **create real UpCloud resources** and cost money. Can take 150+ minutes.
- End-to-end tests (`TestEndToEnd`) part of acceptance suite. Use `TF_VAR_basename` + `TF_VAR_zone`.
- Gateway parser functions (`parseTunnelID`, `parseConnectionID`, `migrateTunnelID`) accept interfaces (`connectionLookup`, `tunnelLookup`) for unit testability.
- `golangci-lint run ./internal/...` for linting; `go vet ./...` for vetting.
- File format: `gofumpt` (configured as golangci-lint formatter).

## Composite IDs

Resources with compound foreign keys use `utils.MarshalID(a, b, ...)` and `utils.UnmarshalID(id, &a, &b)`. Format: `"{uuid}/{uuid}"` or `"{uuid}/{uuid}/{uuid}"`.

## Notable conventions

- `validator.String` over `diag.Errorf` for input validation.
- `SingleNestedAttribute` for `MaxItems: 1` blocks (was `TypeList` in SDKv2).
- `utils.LabelsAttribute("resource name")` for label maps.
- Pointer fields in API types (`*string`, `*int`, `*time.Time`) need nil-guard helpers — see `stringPointerToString`, `intPointerToInt64`, `timePointerToString`.
- `context.Background()` should never appear in build/read functions; always use the request `ctx` for cancellation propagation.