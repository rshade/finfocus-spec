---

description: "Task list for 595-manifest-writer-validator"
---

# Tasks: Plugin Manifest Writer and Validator Agree

**Input**: Design documents from `specs/595-manifest-writer-validator/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required. The constitution makes test-first mandatory (Principle V), and the spec's acceptance
criteria name a round-trip test, an agreement test, and a drift test. In each story, write the tests first
and see them fail before implementing.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1 to US4)

## Phase 1: Setup

**Purpose**: Confirm the baseline before changing anything.

- [X] T001 Record the baseline: run `go test ./sdk/go/pluginsdk/ ./sdk/go/registry/` and `npx ajv validate
      --spec=draft2020 --strict=false -c ajv-formats -s schemas/plugin_manifest.schema.json -d
      "examples/plugins/*.json"`; expect aws, gcp, kubecost, minimal valid and azure, greenops invalid (research R9)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared test helper that compiles the manifest and registry index schemas in Go, used by the
agreement, round-trip, drift, and example tests.

- [X] T002 Create `sdk/go/registry/schema_test.go` with the Apache 2.0 header (package `registry_test`) with helpers
      `compileSchema(t, name string) *jsonschema.Schema` (reads `../../../schemas/<name>` with `os.ReadFile`, compiles
      with `santhosh-tekuri/jsonschema/v6`, as `sdk/go/pricing/validate.go` does) and `schemaVerdict(t, s, doc []byte)
      error`
- [X] T003 Add the same schema-compiling helper to `sdk/go/pluginsdk/manifest_test.go` (path
      `../../../schemas/plugin_manifest.schema.json`), since `pluginsdk_test` cannot import `registry_test` helpers

**Checkpoint**: Both test packages can validate a JSON document against the manifest schema.

---

## Phase 3: User Story 1 - A manifest the SDK writes passes the spec's own checks (Priority: P1) 🎯 MVP

**Goal**: `SaveManifest` writes the canonical form; `LoadManifest` reads it and every previous form.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run Manifest -v` (quickstart §1).

### Tests for User Story 1 (write first, must fail)

- [X] T004 [US1] In `sdk/go/pluginsdk/manifest_test.go`, replace the `TestManifestSaveLoad` fixture with a fully
      populated manifest whose values the current schema accepts (methods from the original five, capabilities from the
      existing 14 strings, billing modes from the schema list, `InstallationMethod: INSTALLATION_METHOD_BINARY`,
      `Security.SecurityLevel: SECURITY_LEVEL_VERIFIED`, `CreatedAt`/`UpdatedAt` set, two clouds in
      `SupportedResources`); for `.json`, `.yaml`, `.yml`: save, load, assert `proto.Equal` via `protocmp`
- [X] T005 [US1] Add `TestSaveManifestPassesValidation` in `sdk/go/pluginsdk/manifest_test.go`: for `.json`, pass the
      file to `registry.ValidatePluginManifest` and the compiled schema; for `.yaml`/`.yml`, convert the YAML to JSON
      (decode with `yaml.v3` into `any`, encode with `encoding/json`) and check both; assert the JSON keys are
      snake_case and `installation.installation_method == "binary"`, `security.security_level == "verified"`
- [X] T006 [US1] Add `TestSaveManifestDeterministic` in `sdk/go/pluginsdk/manifest_test.go`: save the same manifest
      twice per format and compare bytes; assert output ends with `\n`, uses two-space indent, contains no `&` for a `&`
      in a URL, and keeps list order (`keywords` written in input order)
- [X] T006a [US1] Add `TestManifestGolden` in `sdk/go/pluginsdk/manifest_test.go`: marshal the T004 fixture with
      `MarshalManifestJSON`/`MarshalManifestYAML` and compare with `sdk/go/pluginsdk/testdata/manifest.golden.json` and
      `manifest.golden.yaml` (regenerate with `-update`); the golden files pin byte stability across processes and
      builds (SC-002) and show timestamps as RFC 3339 UTC and `port` as a number (FR-003)
- [X] T007 [US1] Add `TestMarshalManifestMatchesSave` in `sdk/go/pluginsdk/manifest_test.go`:
      `MarshalManifestJSON`/`MarshalManifestYAML` bytes equal the saved file bytes (FR-008)
- [X] T008 [US1] Add `TestLoadManifestLegacyFormats` in `sdk/go/pluginsdk/manifest_test.go` with inline fixtures written
      to `t.TempDir()`: camelCase JSON with `"INSTALLATION_METHOD_BINARY"`, JSON with numeric enum `1`, YAML in the old
      writer's shape (`specversion`, `supportedresources`, `installationmethod: 1`, `createdat: {seconds: ..., nanos:
      0}`), and canonical YAML; each loads to the expected manifest (FR-007)
- [X] T009 [US1] Add `TestLoadManifestInvalidEnum` in `sdk/go/pluginsdk/manifest_test.go`: `installation_method: zip`
      returns an error whose text names `installation_method` (FR-007a); and a YAML string-typing case:
      `metadata.version: "1.0"`-style values and a region `"no"` survive a YAML round trip as strings (FR-004)
- [X] T010 [US1] Run `go test ./sdk/go/pluginsdk/ -run TestManifest` and record that T004-T009 (and T006a) fail against
      the current writer

### Implementation for User Story 1

- [X] T011 [US1] Create `sdk/go/pluginsdk/manifest_codec.go` (Apache 2.0 header) with unexported `encodeManifestJSON(m)
      ([]byte, error)`: `protojson.MarshalOptions{UseProtoNames: true}`, decode into `map[string]any` (with
      `UseNumber`), rewrite `installation.installation_method` and `security.security_level` from the full enum name to
      the lowercase name without the type prefix (prefix derived from the enum descriptor's `UNSPECIFIED` value,
      research R2), re-encode with `json.Encoder`, `SetEscapeHTML(false)`, `SetIndent("", "  ")`
- [X] T012 [US1] In `sdk/go/pluginsdk/manifest_codec.go` add `encodeManifestYAML(m) ([]byte, error)`: unmarshal the
      canonical JSON into a `yaml.Node`, clear `Style` on every node so the encoder chooses plain or quoted scalars,
      encode with `yaml.NewEncoder` and `SetIndent(2)` (research R3)
- [X] T013 [US1] In `sdk/go/pluginsdk/manifest_codec.go` add the descriptor-driven normalizer `normalizeMessage(md
      protoreflect.MessageDescriptor, v map[string]any) (map[string]any, error)` (research R4): match keys by proto
      name, JSON name, or underscore-free case-insensitive name; enums accept short name, full name, or number and
      become the full name, else error naming the field path; `google.protobuf.Timestamp` given as an object with
      `seconds`/`nanos` becomes RFC 3339; maps normalize values against the map value message; lists normalize each
      element; unknown keys pass through (discarded later by `protojson`)
- [X] T014 [US1] In `sdk/go/pluginsdk/manifest_codec.go` add `decodeManifest(data []byte, ext string)
      (*pbc.PluginManifest, error)`: JSON via `encoding/json` (with `UseNumber`), YAML via `yaml.v3` into `any` (convert
      `map[string]any` keys, keep as is), then `normalizeMessage`, `json.Marshal`,
      `protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}`
- [X] T015 [US1] Rewrite `SaveManifest` and `LoadManifest` in `sdk/go/pluginsdk/manifest.go` to use the codec; add
      exported `MarshalManifestJSON` and `MarshalManifestYAML` with godoc per `contracts/go-sdk.md`; fix the
      `LoadManifest` doc comment (it does not validate; FR-006a) and remove the duplicated first line and the stale
      `//nolint:musttag` comments
- [X] T016 [US1] Run `go test ./sdk/go/pluginsdk/` until T004-T009 and T006a pass; create the golden files with
      `-update` and review them by eye

**Checkpoint**: A manifest saved by the SDK passes the SDK validator and the schema, byte-stably.

---

## Phase 4: User Story 2 - Full resource type tokens fit in a manifest (Priority: P2)

**Goal**: 256-character resource types; the entry form is documented.

**Independent Test**: `go test ./sdk/go/registry/ -run TestValidateSupportedResources -v` long-token cases
(quickstart §2).

### Tests for User Story 2

- [X] T017 [P] [US2] In `sdk/go/registry/validate_test.go` add a table `supportedResourcesCases` (shared with US3) with
      passing cases: a 60-character `azure-native:` token under `azure`,
      `azure:compute/linuxVirtualMachine:LinuxVirtualMachine` (53), a 256-character token, short names `ec2`/`vm`; and a
      failing case: a 257-character token; run each through `registry.ValidatePluginManifest` and the compiled schema
      (T002) and assert the expected verdict for both
- [X] T018 [US2] Run the test and record that the 53-, 60-, and 256-character cases fail against the schema

### Implementation for User Story 2

- [X] T019 [US2] In `schemas/plugin_manifest.schema.json` set `resource_types.items.maxLength` to 256 and replace its
      description with: the entry is the resource type string matched against `ResourceDescriptor.resource_type`; the
      full type token (for example `azure-native:compute:VirtualMachine`) is recommended; a native package's types are
      listed under their cloud (FR-009, FR-010)
- [X] T020 [US2] Add `MaxResourceTypeLength = 256` with godoc to the constants block in `sdk/go/registry/validate.go`

**Checkpoint**: Long tokens validate against the schema (the SDK validator checks them after US3).

---

## Phase 5: User Story 3 - The SDK validator checks resource support (Priority: P2)

**Goal**: `ValidatePluginManifest` validates `supported_resources` with the schema's rules.

**Independent Test**: `go test ./sdk/go/registry/ -run 'TestValidateSupportedResources|TestSchemaAgreement' -v`.

### Tests for User Story 3

- [X] T021 [US3] Extend `supportedResourcesCases` in `sdk/go/registry/validate_test.go` with failing cases and expected
      error substrings per `contracts/go-sdk.md`: key `azure-native`; entry not an object; extra key `skus`; missing
      `resource_types`; empty `resource_types`; duplicate type; empty-string type; billing mode `hourly`; duplicate
      billing mode; region `x` (1 char); region of 31 chars; duplicate region; and passing cases with valid
      `billing_modes` and `regions`
- [X] T022 [US3] Add `TestSchemaAgreement` in `sdk/go/registry/schema_test.go` that runs every `supportedResourcesCases`
      entry through both validators and asserts both accept or both reject (FR-013)
- [X] T023 [US3] Run the tests and record the failures (the SDK validator accepts every case today)

### Implementation for User Story 3

- [X] T024 [US3] In `sdk/go/registry/domain.go` add the package-level slice `allManifestBillingModes` (the 38 strings of
      the schema's `billing_modes` enum, in schema order), `AllManifestBillingModes() []string`, and
      `IsValidManifestBillingMode(string) bool` with godoc noting they differ from `pricing.BillingMode`, following the
      zero-allocation enum pattern
- [X] T025 [US3] In `sdk/go/registry/validate.go` add `validateSupportedResources(specification map[string]interface{})
      error`, called from `validateSpecification` after `validateSupportedProviders`; skip when absent; rules per FR-011
      ("each key is a valid provider; each entry is an object with only `resource_types`, `billing_modes`, and
      `regions`; `resource_types` is required, non-empty, unique, with entries of 1 to 256 characters; `billing_modes`
      entries are unique and in the schema's list; `regions` entries are unique and 2 to 30 characters"); iterate keys
      in sorted order so the first reported error is stable; split into helpers to stay under the `gocognit`/`funlen`
      limits; leave every existing message unchanged (FR-013a)
- [X] T026 [US3] Run `go test ./sdk/go/registry/` until T017, T021, T022 pass

**Checkpoint**: SDK validator and schema agree on `supported_resources`.

---

## Phase 6: User Story 4 - A manifest can declare every RPC and capability (Priority: P3)

**Goal**: Methods and capabilities derived from the proto; schemas widened; drift guarded.

**Independent Test**: `go test ./sdk/go/registry/ -run 'TestServiceMethods|Capabilit|BillingModes|TestSchemaDrift' -v`
(quickstart §3).

### Tests for User Story 4

- [X] T027 [P] [US4] In `sdk/go/registry/domain_test.go` add `TestServiceMethods`: `AllServiceMethods()` has the 13
      `CostSourceService` RPCs in declaration order; `IsValidServiceMethod` accepts each and rejects `Allocate`,
      `GetStats`, `HealthCheck`, `""`, `name`; update the `AllPluginCapabilities` length expectation (line ~420) from 14
      to 32
- [X] T028 [P] [US4] In `sdk/go/registry/domain_test.go` add `TestCapabilities`: `ManifestCapabilityName` maps every
      non-unspecified `pbc.PluginCapability` value to its lowercase name without prefix (`DRY_RUN` → `dry_run`,
      `RECOMMENDATION_SCORING` → `recommendation_scoring`), returns `""` for `UNSPECIFIED` and `PluginCapability(999)`;
      `IsValidPluginCapability` accepts the 14 existing strings and every mapped name, rejects `teleport` and `DRY_RUN`
- [X] T029 [US4] In `sdk/go/registry/validate_test.go` add cases: a manifest listing all 13 methods passes (SDK and
      schema); a manifest listing all 32 capabilities passes; method `Allocate` fails; capability `teleport` fails with
      `specification.capabilities[0]: 'teleport' is not a valid capability`; capability not an array fails
- [X] T030 [US4] Add `TestSchemaDrift` in `sdk/go/registry/schema_test.go` (FR-017): manifest `methods` enum equals
      `AllServiceMethods()`; manifest `capabilities` enum equals `AllPluginCapabilities()` and the registry index enum
      contains it; manifest `billing_modes` enum equals `AllManifestBillingModes()`; manifest
      `resource_types.items.maxLength` equals `MaxResourceTypeLength`; manifest `methods` has no `maxItems`; failures
      name the missing or extra value
- [X] T031 [US4] Add `BenchmarkIsValidServiceMethod` in `sdk/go/registry/domain_test.go` and confirm
      `BenchmarkIsValidPluginCapability` still exists
- [X] T032 [US4] Run the tests and record the failures

### Implementation for User Story 4

- [X] T033 [P] [US4] Create `sdk/go/registry/service_methods.go` (Apache 2.0 header): package-level `allServiceMethods
      []string` built once at init from
      `pbc.File_finfocus_v1_costsource_proto.Services().ByName("CostSourceService").Methods()`; `AllServiceMethods()`
      and `IsValidServiceMethod(name string) bool` (slice scan, 0 allocs) with godoc
- [X] T034 [US4] In `sdk/go/registry/domain.go` build `protoCapabilityNames` once at init from
      `pbc.PluginCapability(0).Descriptor().Values()` (skip 0, strip `PLUGIN_CAPABILITY_`, lowercase), append them to
      `allPluginCapabilities` in enum-number order, add `ManifestCapabilityName(pbc.PluginCapability) string`; update
      the godoc of `AllPluginCapabilities` and `IsValidPluginCapability`
- [X] T035 [US4] In `sdk/go/registry/validate.go` replace the hard-coded `validMethods` map with `IsValidServiceMethod`,
      and add `validateCapabilities` (optional array of unique strings, each `IsValidPluginCapability`) called from
      `validateSpecification`
- [X] T036 [US4] In `schemas/plugin_manifest.schema.json` set `methods.items.enum` to the 13 RPC names in declaration
      order, remove `methods.maxItems`, and append the 18 protocol capability names to `capabilities.items.enum`; update
      both descriptions
- [X] T037 [US4] In `schemas/plugin_registry.schema.json` append the same 18 capability names to the `capabilities` enum
      and update its description
- [X] T038 [US4] Run `go test ./sdk/go/registry/ ./sdk/go/pluginsdk/` and `go test
      -bench='IsValidPluginCapability|IsValidServiceMethod' -benchmem ./sdk/go/registry/` until all pass with 0
      allocs/op

**Checkpoint**: Every RPC and capability can be declared; drift is caught.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T039 [P] Add `TestExampleManifests` in `sdk/go/registry/schema_test.go`: `aws-cost-plugin.json`,
      `gcp-cost-plugin.json`, `kubecost-plugin.json`, `minimal-plugin.json` under `examples/plugins/` pass the schema
      and `ValidatePluginManifest` (FR-020)
- [X] T040 [P] Update `docs/plugin-registry-specification.md`: canonical field names and enum strings,
  resource type form and 256 bound, the full method list, the capability union with the protocol naming
  rule, and that `SaveManifest` output validates as is (FR-019)
- [X] T041 [P] Update `sdk/go/pluginsdk/README.md` manifest section: `SaveManifest`/`LoadManifest`
  behavior, `MarshalManifestJSON`/`MarshalManifestYAML`, accepted legacy inputs, and a compiling example
  that builds capabilities with `registry.ManifestCapabilityName`
- [X] T042 [P] Update `sdk/go/registry/README.md`: `IsValidServiceMethod`, `AllServiceMethods`,
  `ManifestCapabilityName`, `IsValidManifestBillingMode`, `AllManifestBillingModes`,
  `MaxResourceTypeLength`, and the new `supported_resources`/`capabilities` checks
- [X] T043 Add a CLAUDE.md pattern section `### Plugin Manifest Pattern (595-manifest-writer-validator)`
  and a Recent Changes entry (by hand; do not run `update-agent-context.sh`)
- [X] T044 Run the full gates: `make generate && git diff --exit-code -- sdk/`, `make buf-lint`,
  `make test`, `go test -v -tags=integration ./sdk/go/testing/`, `make lint-go`, `make lint-markdown`,
  `make lint-yaml`, `make validate-npm`, and quickstart §4

---

## Dependencies & Execution Order

- **Setup (T001)** → **Foundational (T002-T003)** → stories.
- **US1 (T004-T016, T006a)** depends only on Foundational. It is the MVP.
- **US2 (T017-T020)** depends on Foundational. Its SDK-side check of the 256 bound lands in US3 (T025).
- **US3 (T021-T026)** depends on US2's `MaxResourceTypeLength` (T020) and shares the case table (T017).
- **US4 (T027-T038)** depends on Foundational; T034 changes `AllPluginCapabilities`, so T027's length
  update lands with it. T030 also covers billing modes, so it needs T024 (US3).
- **Polish (T039-T044)** after all stories.

Within each story: tests → confirm failure → implementation → green.

## Parallel Opportunities

- US1 (pluginsdk files) and US2/US3/US4 (registry and schema files) touch different packages and can run
  in parallel after Phase 2, except that T005 imports `registry.ValidatePluginManifest` (unchanged API).
- T027 and T028 (same file, different functions) are sequential; T033 is a new file and can run beside
  T034.
- T040, T041, T042 are independent documents.

```text
# After Phase 2:
Developer A: T004-T016 (US1, sdk/go/pluginsdk)
Developer B: T017-T026 (US2+US3, sdk/go/registry + manifest schema)
Developer C: T027-T038 (US4, after B's T024 for the drift test)
```

## Implementation Strategy

1. MVP: Phases 1-3. Plugins can generate valid manifests; the Azure plugin's shim can go.
2. Add US2 and US3: long tokens and SDK-side resource validation.
3. Add US4: full method and capability vocabularies with drift guards.
4. Polish: examples test, docs, gates.

## Phase 8: Convergence

- [X] T045 Change the US1 test selector in `specs/595-manifest-writer-validator/quickstart.md` §1 and the
  Phase 3 Independent Test in `tasks.md` to `-run 'Manifest'` so it runs `TestSaveManifest*`,
  `TestMarshalManifest*`, and `TestLoadManifest*` too, per plan: quickstart §1 (partial)
- [X] T046 Change the US4 test selector in `specs/595-manifest-writer-validator/quickstart.md` §3 and the
  Phase 6 Independent Test in `tasks.md` to match `TestManifestCapabilityName`,
  `TestValidateMethodsAndCapabilities`, and `TestManifestBillingModes`, per T028 (partial)
- [X] T047 Reject two keys that name one field in `normalizeMessage` (`sdk/go/pluginsdk/manifest_codec.go`),
  with `TestLoadManifestConflictingKeys`, per FR-007b (contradicts; found by the push security review)
