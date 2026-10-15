# Implementation Plan: Plugin Settings on Plugin Info

**Branch**: `601-plugin-info-settings` | **Date**: 2026-10-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/601-plugin-info-settings/spec.md`

## Summary

Add an optional `settings` list to `GetPluginInfoResponse` so a plugin can describe each
setting it reads and the environment variable the host injects. The list is additive: empty
means the plugin advertises nothing, and no capability flag changes. One validator in
`sdk/go/testing` enforces the descriptor rules. `pluginsdk` calls it at startup for static
plugin info and again before either plugin-info path returns a non-empty list. The mock can
carry a list, conformance checks whatever list a plugin returned, and the TypeScript client
reads the generated field through the existing `getPluginInfo` call. Design decisions are in
[research.md](research.md).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc,
connectrpc.com/connect, buf v1.32.1; no new dependencies

**Storage**: N/A (one repeated message on GetPluginInfoResponse)

**Testing**: `go test` (table tests, bufconn serve tests, conformance harness), vitest for the
TypeScript client message round trip

**Target Platform**: Library SDKs and wire protocol

**Project Type**: Protocol specification with Go and TypeScript SDKs

**Performance Goals**: `ValidatePluginSettings(nil)` and a `PluginInfo` with no settings stay
at 0 allocs/op. A valid list that has no URL or duration default also stays at 0 allocs/op

**Constraints**: Additive only (`buf breaking` clean). Generated code is never hand-edited.
Validation errors never include `default_value`. The manifest `configuration` object stays

**Scale/Scope**: One proto message and enum, one response field, one validator, two
plugin-info call sites, one mock field, one conformance extension, one TS test, four docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | `PluginSetting` and field 7 land, then `make generate`, before SDK code |
| II. Multi-provider consistency | PASS | The descriptor is provider-neutral. No new billing mode |
| III. Spec consumes, does not calculate | PASS | The list describes inputs. No pricing math |
| IV. Separation of concerns | PASS | Host forms and env injection stay in finfocus core (#1765) |
| V. Test-first | PASS | Validator, serve, mock, conformance, and TS tests are written before the code they cover |
| VI. Backward compatibility | PASS | New field number, empty list is valid, `buf breaking` clean |
| VII. Documentation | PASS | Proto comments, developer guide, pluginsdk README, manifest description |
| VIII. Performance | PASS | Empty-list guard allocates nothing (R3) |
| IX. Observability and validation | PASS | Startup and both response paths use one validator. Logs omit defaults |
| X. Follow established patterns | PASS | Rules live in `testing`, `pluginsdk` delegates, as in 052 and 556 |
| XI. Copyright headers | PASS | New Go files carry the Apache 2.0 header |
| XII. Automatic capability declaration | PASS | No capability is added or inferred |
| XIII. SDK synchronization | PASS | Go and TS bindings regenerated. Existing TS `getPluginInfo` returns the field |
| XIV. Documentation integrity | PASS | README example for `WithSettings` is compiled by the package tests' pattern |

Post-design re-check: unchanged, all PASS. No complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/601-plugin-info-settings/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── proto.md
│   └── go-sdk.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto                 # PluginSetting, enum, field 7
sdk/go/proto/finfocus/v1/                          # regenerated
sdk/typescript/packages/client/src/generated/      # regenerated
sdk/go/testing/plugin_settings.go                  # ValidatePluginSettings
sdk/go/testing/plugin_settings_test.go
sdk/go/testing/mock_plugin.go                      # Settings field
sdk/go/testing/conformance_test.go                 # settings check on plugin info
sdk/go/pluginsdk/plugin_info.go                    # WithSettings, Validate call
sdk/go/pluginsdk/plugin_settings.go                # sentinel alias
sdk/go/pluginsdk/sdk.go                            # both GetPluginInfo paths
sdk/go/pluginsdk/plugin_settings_test.go           # serve: static, provider, bad list
sdk/typescript/packages/client/test/plugin-settings.test.ts
schemas/plugin_manifest.schema.json                # configuration description
PLUGIN_DEVELOPER_GUIDE.md
sdk/go/pluginsdk/README.md
CLAUDE.md / AGENTS.md                              # agent context script
```

**Structure Decision**: Existing layout. The harness registers the plugin directly, so
conformance sees the list the plugin returned and checks it there. `pluginsdk.Serve` is the
path that refuses to hand a bad list to a host.

## Complexity Tracking

None.
