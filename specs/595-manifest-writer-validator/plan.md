# Implementation Plan: Plugin Manifest Writer and Validator Agree

**Branch**: `595-manifest-writer-validator` | **Date**: 2026-10-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/595-manifest-writer-validator/spec.md`

## Summary

`pluginsdk.SaveManifest` writes a shape that `registry.ValidatePluginManifest` and the manifest JSON schema
reject, and the schema and validator are narrower than the protocol. The SDK gains one canonical manifest
encoding (protojson with proto names, schema enum strings, re-encoded with sorted keys) used for JSON and,
through a `yaml.Node` tree, for YAML; the loader normalizes every previously written form against the
message descriptor. The validator gains `supported_resources` and `capabilities` checks; method and
capability lists are derived from the proto descriptors, and the static schemas are guarded by a drift
test. The resource type bound rises to 256. Item 1 of #611 already landed in 56ef520.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod)

**Primary Dependencies**: google.golang.org/protobuf (protojson, protoreflect), gopkg.in/yaml.v3,
github.com/santhosh-tekuri/jsonschema/v6 (tests); no new dependencies

**Storage**: Files (manifest JSON or YAML written by plugin build tooling)

**Testing**: `go test` (table-driven), JSON schema compiled in Go tests, ajv for npm schema checks

**Target Platform**: Go SDK consumers (plugin authors, hosts, registries)

**Project Type**: Library (protocol spec plus SDK)

**Performance Goals**: `IsValidPluginCapability` and `IsValidServiceMethod` stay 0 allocs/op; manifest
encoding is a build-time operation with no hot-path requirement

**Constraints**: No `.proto` message changes; existing exported signatures unchanged; schema changes only
widen; existing validator error messages unchanged

**Scale/Scope**: 2 Go packages (`pluginsdk`, `registry`), 2 schema files, 3 docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Proto first | Pass | No message changes. Method and capability lists are read from the proto descriptors, so the proto stays the source of truth. |
| II. Multi-provider | Pass | No billing modes added; provider list unchanged. |
| III. Spec consumes | Pass | No pricing math. |
| IV. Separation | Pass | SDK and schema only. |
| V. Test first | Pass | Round-trip, agreement, and drift tests are written before the writer and validator changes and fail first. |
| VI. Backward compatibility | Pass | Schemas widen only; loader reads old formats; `buf breaking` unaffected. `SaveManifest` output changes, which is the bug fix; the old output was never valid against the schema. |
| VII. Documentation | Pass | Registry spec doc, pluginsdk README, registry README updated in the same PR. |
| VIII. Performance | Pass | Derived lists built once at init; validation is a slice scan with 0 allocs; benchmarks added. |
| IX. Validation layers | Pass | Schema and SDK validator brought into agreement with a test. |
| X. Patterns | Pass | Package-level slice enum pattern; existing validator style. |
| XI. Headers | Pass | Every new Go file (`manifest_codec.go`, `service_methods.go`, `schema_test.go`, golden-file test helpers) carries the Apache 2.0 header used by recent `pluginsdk` files. Existing files without it are not touched for this. |
| XII. Capability declaration | Pass | `ManifestCapabilityName` lets a manifest reuse the plugin's `PluginCapability` values. |
| XIII. SDK sync | Pass | TypeScript has no manifest writer or validator; no proto change, so no TS change. |
| XIV. Docs integrity | Pass | New exported symbols get godoc; README examples compile. |

Post-design re-check: unchanged; no violations, so Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/595-manifest-writer-validator/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── go-sdk.md
│   └── schema.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
schemas/
├── plugin_manifest.schema.json     # resource_types bound, methods, capabilities
└── plugin_registry.schema.json     # capabilities

sdk/go/pluginsdk/
├── manifest.go                     # SaveManifest, LoadManifest, MarshalManifestJSON/YAML
├── manifest_codec.go               # canonical encode, descriptor-driven normalize (new)
├── manifest_test.go                # round trip, determinism, legacy inputs
└── testdata/manifest.golden.{json,yaml}   # byte-stability golden files (new)

sdk/go/registry/
├── domain.go                       # capability set, ManifestCapabilityName
├── billing_modes.go                # AllManifestBillingModes, IsValidManifestBillingMode (new)
├── service_methods.go              # IsValidServiceMethod, AllServiceMethods (new)
├── validate.go                     # supported_resources, capabilities, methods
├── domain_test.go
├── validate_test.go
└── schema_test.go                  # schema agreement, drift, example manifests (new)

docs/plugin-registry-specification.md
sdk/go/pluginsdk/README.md
sdk/go/registry/README.md
```

**Structure Decision**: Changes stay in the two existing packages. The codec goes in its own file so
`manifest.go` keeps the public entry points readable. `registry` imports the generated proto package for
descriptors (no cycle: generated code imports nothing from `registry`).

## Complexity Tracking

No violations.
