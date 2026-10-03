# Feature Specification: Plugin Manifest Writer and Validator Agree

**Feature Branch**: `595-manifest-writer-validator`

**Created**: 2026-10-03

**Status**: Draft

**Input**: User description: "fix(registry): plugin manifest schema and SDK writer reject valid manifests.
Closes #611. Item 1 (provider keys are clouds) already landed in 56ef520; this feature covers items 2-5:
the resource type length limit, the SDK manifest writer's output shape, validation of
`supported_resources`, and the method and capability lists."

## Clarifications

`/speckit-clarify` was not required: the issue states the acceptance criteria, and the remaining choices
(capability string form, resource type bound, reading the previous writer format) have defaults recorded
under Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A manifest the SDK writes passes the spec's own checks (Priority: P1)

A plugin author builds a manifest in code and saves it with the SDK's manifest writer, as JSON or as
YAML. The saved file passes the SDK manifest validator and the published manifest JSON schema with no
hand translation. Saving the same manifest twice produces identical bytes, so a generated manifest
checked into a plugin repository does not churn between builds.

**Why this priority**: This is the defect that forced the Azure plugin to carry a translation shim. Until
the writer and the validator agree, every plugin that generates its manifest is broken, and the other
stories only widen what a valid manifest may say.

**Independent Test**: Build a manifest that sets every section, save it as JSON and as YAML, then validate
each file with the SDK validator and the JSON schema; save again and compare bytes.

**Acceptance Scenarios**:

1. **Given** a fully populated manifest, **When** the author saves it as JSON, **Then** the file uses the
   schema's field names and enum strings and passes both the SDK validator and the JSON schema.
2. **Given** the same manifest, **When** the author saves it as YAML, **Then** the file has the same field
   names and values as the JSON form, and the JSON equivalent of the YAML passes both checks.
3. **Given** a manifest saved once, **When** the author saves it again unchanged, **Then** the two files are
   byte-identical, for both JSON and YAML.
4. **Given** a file the writer produced, **When** the author loads it with the SDK loader, **Then** the
   loaded manifest equals the original.
5. **Given** a manifest file in the previous writer's format (camelCase JSON keys, prefixed enum names,
   or integer enums in YAML), **When** the author loads it, **Then** it still loads.

---

### User Story 2 - Full resource type tokens fit in a manifest (Priority: P2)

A plugin author lists the resource types the plugin prices under their cloud, using the full resource
type token that hosts send (for example `azure-native:compute:VirtualMachine`). Long tokens are accepted,
and the schema says which form an entry takes.

**Why this priority**: Real Pulumi tokens exceed the current 50-character limit, so a correct manifest is
rejected today. This is a one-line limit plus documentation.

**Independent Test**: Validate a manifest that lists a 60-character `azure-native:` token under the
`azure` key, with the SDK validator and the JSON schema.

**Acceptance Scenarios**:

1. **Given** a manifest that lists a 60-character `azure-native:` resource type under `azure`, **When** it
   is validated, **Then** it passes.
2. **Given** a manifest that lists `azure:compute/linuxVirtualMachine:LinuxVirtualMachine` (53
   characters), **When** it is validated, **Then** it passes.
3. **Given** a manifest that uses the short type names in the existing examples (`ec2`, `vm`), **When** it
   is validated, **Then** it still passes.

---

### User Story 3 - The SDK validator checks resource support (Priority: P2)

A registry or host validates a manifest with the SDK validator and gets the same verdict on
`supported_resources` as the JSON schema gives: unknown provider keys, empty or duplicate resource type
lists, unknown billing modes, and malformed regions are rejected with a message naming the field.

**Why this priority**: Today the SDK validator accepts any `supported_resources`, so a manifest can pass
the SDK and fail the schema. Hosts that use only the SDK get no protection.

**Independent Test**: Run a table of `supported_resources` inputs through the SDK validator and the JSON
schema and check both give the same verdict.

**Acceptance Scenarios**:

1. **Given** a `supported_resources` key that is not a cloud provider (for example `azure-native`),
   **When** the manifest is validated, **Then** it is rejected and the message names the key.
2. **Given** a provider entry with no `resource_types`, an empty list, a duplicate entry, or an empty
   string, **When** validated, **Then** it is rejected with the path of the offending entry.
3. **Given** a billing mode outside the schema's list, or a duplicate billing mode, **When** validated,
   **Then** it is rejected.
4. **Given** a region shorter than 2 or longer than 30 characters, or a duplicate region, **When**
   validated, **Then** it is rejected.
5. **Given** a provider entry with a field other than `resource_types`, `billing_modes`, and `regions`,
   **When** validated, **Then** it is rejected.

---

### User Story 4 - A manifest can declare every RPC and capability the plugin offers (Priority: P3)

A plugin author lists every `CostSourceService` RPC the plugin implements (including `EstimateCost`,
`DryRun`, `GetPluginInfo`, `BatchCost`, and `ResolveResourceTypes`) and every plugin capability it
reports through `GetPluginInfo`, using names derived from the protocol. The lists cannot drift from the
protocol, because a test fails when a new RPC or capability is added to the protocol and not to the
manifest lists.

**Why this priority**: Without this a manifest under-describes modern plugins, but nothing breaks at
runtime today, because hosts discover capabilities through `GetPluginInfo`.

**Independent Test**: Validate a manifest listing all `CostSourceService` RPCs and all capability values;
run the drift test.

**Acceptance Scenarios**:

1. **Given** a manifest whose `methods` lists every `CostSourceService` RPC, **When** validated, **Then**
   it passes with the SDK validator and the JSON schema.
2. **Given** a manifest whose `capabilities` lists every plugin capability value (for example `dry_run`,
   `estimate_cost`, `recommendation_scoring`), **When** validated, **Then** it passes.
3. **Given** a manifest that uses the existing capability strings (`cost_retrieval`, `caching`), **When**
   validated, **Then** it still passes.
4. **Given** an RPC name or capability that is not in the protocol, **When** validated, **Then** it is
   rejected.
5. **Given** a new RPC or capability is added to the protocol, **When** the test suite runs without the
   manifest lists being updated, **Then** a test fails and names the missing value.

---

### Edge Cases

- A manifest with no optional sections (only metadata, specification, installation): the writer omits the
  empty sections and the output still passes validation.
- Enum fields left unset (installation method or security level unspecified): the writer omits the field
  rather than writing an empty or `unspecified` string, so the schema's enum check is not violated by the
  writer. A manifest missing a required installation method still fails validation, as before.
- Map ordering: `supported_resources` keys are written in sorted order so output is deterministic.
- Timestamps (`created_at`, `updated_at`) are written as RFC 3339 strings in UTC.
- A YAML file with an extension of `.yml` behaves the same as `.yaml`.
- An unsupported file extension still returns the existing error and writes nothing.
- Capability `pricing_spec` (protocol) and `pricing_specs` (existing manifest string) are both accepted and
  are distinct values; no alias is introduced.

## Requirements *(mandatory)*

### Functional Requirements

#### Writer and loader (User Story 1)

- **FR-001**: The SDK manifest writer MUST emit JSON whose field names are the schema's snake_case names.
- **FR-002**: The writer MUST emit the installation method and security level as the schema's lowercase
  enum strings (`binary`, `container`, `script`, `package`; `untrusted`, `community`, `verified`,
  `official`), and MUST omit an unspecified enum.
- **FR-003**: The writer MUST emit integer fields (`service_definition.port`) as JSON numbers and
  timestamps as RFC 3339 strings in UTC.
- **FR-004**: The writer MUST emit YAML that carries the same field names and values as its JSON output,
  in the same order. A string that YAML would otherwise read as another type (for example `1.0`, `true`,
  or `no`) MUST be quoted so it loads back as a string.
- **FR-005**: The writer MUST be deterministic: the same manifest produces byte-identical output across
  calls and across processes of the same SDK version, for JSON and YAML: two-space indent, map keys in
  sorted order, list entries in their input order, no HTML escaping, and a trailing newline.
- **FR-006**: The SDK manifest loader MUST load every file the writer produces into a manifest equal to
  the one written.
- **FR-006a**: The loader MUST discard unknown keys, as it does today, and MUST NOT validate the manifest;
  validation stays with `ValidatePluginManifest`. Its documentation MUST say so.
- **FR-007**: The loader MUST continue to load JSON files written in the previous format (camelCase keys,
  prefixed enum names such as `INSTALLATION_METHOD_BINARY`) and YAML files in the previous writer's form
  (lowercased field names without underscores, integer enums, and timestamps as `seconds`/`nanos`
  objects).
- **FR-007a**: The loader MUST return an error naming the field when an enum value matches no value in any
  accepted form (for example `installation_method: zip`).
- **FR-007b**: The loader MUST return an error naming the field when two keys in one object name the
  same field (for example `spec_version` and `specVersion`), as `protojson` does, so the loaded manifest
  cannot differ from what a validator read in the same file.
- **FR-008**: The SDK MUST expose a way to obtain the canonical JSON bytes of a manifest without writing a
  file, so a caller can validate before saving.

#### Resource types (User Story 2)

- **FR-009**: The schema MUST accept `resource_types` entries up to 256 characters.
- **FR-010**: The schema description and the registry documentation MUST state that an entry is the
  resource type string the plugin matches against `ResourceDescriptor.resource_type`, that the full type
  token (for example `azure-native:compute:VirtualMachine`) is recommended, and that a native package's
  types are listed under their cloud.

#### Supported resources validation (User Story 3)

- **FR-011**: The SDK validator MUST validate `supported_resources` when present: each key is a valid
  provider; each entry is an object with only `resource_types`, `billing_modes`, and `regions`;
  `resource_types` is required, non-empty, unique, with entries of 1 to 256 characters; `billing_modes`
  entries are unique and in the schema's list; `regions` entries are unique and 2 to 30 characters.
- **FR-012**: Each `supported_resources` error message MUST name the path of the offending value (for
  example `specification.supported_resources.azure.resource_types[2]`).
- **FR-013**: The SDK validator and the JSON schema MUST give the same verdict for every
  `supported_resources` case in the test table. "Same verdict" means both accept or both reject; error
  text may differ.
- **FR-013a**: Error messages the validator already returns for other fields MUST NOT change.

#### Methods and capabilities (User Story 4)

- **FR-014**: `service_definition.methods` MUST accept every RPC name of `CostSourceService`, and the
  item-count cap of 5 MUST be removed (uniqueness still applies). RPCs of other services (usage source,
  allocator, supplemental datasets, scorer, observability) are not accepted: `service_definition` names a
  single service, and those services are discovered through capabilities.
- **FR-015**: `specification.capabilities` MUST accept every `PluginCapability` value except unspecified,
  written as the lowercase enum name without the `PLUGIN_CAPABILITY_` prefix, and MUST keep accepting the
  existing capability strings. The plugin registry index schema's `capabilities` enum MUST accept at
  least the same set, so a registry entry can repeat its manifest's capabilities. It keeps its own
  registry-only values (`tagging`, `anomaly_detection`, `forecasting`, `alerts`, `custom`), because
  removing them would narrow the schema.
- **FR-016**: The SDK validator MUST check `specification.capabilities` against the same set the schema
  accepts.
- **FR-017**: A test MUST fail when any of these differ from the protocol or from each other: the manifest
  schema's `methods` enum, the manifest schema's `capabilities` enum (equal to the SDK set), the registry
  index schema's `capabilities` enum (a superset of the SDK set), the SDK's method and capability sets, the schema's and
  SDK's billing mode lists, and the schema's and SDK's resource type
  length bound.
- **FR-018**: The SDK MUST provide a function that converts a `PluginCapability` value to its manifest
  string, so plugins can build the capability list from the same enum they report through
  `GetPluginInfo`.

#### Documentation and examples

- **FR-019**: `docs/plugin-registry-specification.md`, `sdk/go/pluginsdk/README.md`, and
  `sdk/go/registry/README.md` MUST describe the manifest field names, enum strings, resource type form, and the expanded
  method and
  capability lists.
- **FR-020**: The example manifests under `examples/plugins/` that pass the manifest schema today
  (`aws-cost-plugin.json`, `gcp-cost-plugin.json`, `kubecost-plugin.json`, `minimal-plugin.json`) MUST
  keep passing the schema and MUST pass the SDK validator, guarded by a test.

### Key Entities

- **Plugin manifest**: The document a plugin publishes to describe itself: metadata, specification
  (providers, supported resources, capabilities, service definition), security, installation, and
  configuration. Its canonical serialized form is defined by the manifest JSON schema.
- **Supported resources entry**: Per-cloud lists of resource types, billing modes, and regions.
- **Method list**: The `CostSourceService` RPC names a plugin implements.
- **Capability list**: Strings naming what a plugin supports; the union of the existing manifest strings
  and the protocol's plugin capability values.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of manifests produced by the SDK writer in the round-trip test pass both the SDK
  validator and the JSON schema, for JSON and YAML.
- **SC-002**: Saving the same manifest twice yields byte-identical files in 100% of runs, including across
  separate test processes.
- **SC-003**: A plugin author needs zero translation steps between saving a manifest and validating it
  (the Azure plugin's shim can be deleted).
- **SC-004**: Every `CostSourceService` RPC and every non-unspecified plugin capability value can be
  declared in a valid manifest, and adding a protocol value without updating the manifest lists fails the
  test suite.
- **SC-005**: The SDK validator and the JSON schema agree on 100% of the `supported_resources` test cases.
- **SC-006**: The four baseline example manifests (FR-020) and every manifest valid before this change
  continue to validate.

## Assumptions

- No `.proto` message changes are needed: the manifest fields are already strings or existing enums.
  Proto comment updates are allowed if they clarify the manifest string form.
- The manifest capability string for a protocol value is the lowercase enum name without the
  `PLUGIN_CAPABILITY_` prefix (`PLUGIN_CAPABILITY_DRY_RUN` becomes `dry_run`). This follows the scorer
  metadata convention (lowercase name, no prefix) and differs from the legacy `supports_*` metadata keys,
  which are a different surface.
- 256 characters is a generous bound for resource type tokens; the schema stays bounded rather than
  unlimited so a registry can size storage.
- Short resource type names in existing examples remain valid; the recommendation for full tokens is
  documentation, not enforcement.
- The writer's previous output format is read for backward compatibility but never written.
- `PluginManifest` has no `requirements` field, so the schema's `requirements` section (with its 64-bit
  integer fields) is never produced by the writer.
- Two example files failed the manifest schema before this feature: `azure-cost-plugin.json` (its
  `download_url` was a container reference, not a URI) and `greenops-plugin.json` (not shaped as a
  manifest). Both were fixed during PR review (an `oci://` URI, and a manifest that uses the new protocol
  capabilities), and `TestExampleManifests` now covers all six files.
- The TypeScript SDK has no manifest writer or validator, so SDK parity needs only the shared schema
  change; no TypeScript code changes are expected.
- Provider normalization by hosts (rshade/finfocus#1645), a per-type input field catalog, and changes to
  the provider enum or key pattern are out of scope.
