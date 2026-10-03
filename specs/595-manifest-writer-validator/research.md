# Research: Plugin Manifest Writer and Validator Agree

## R1. Canonical JSON encoding of a `PluginManifest`

- **Decision**: Marshal with `protojson` using `UseProtoNames: true`, decode the result into a generic
  `map[string]any`, rewrite the two enum fields (`installation.installation_method`,
  `security.security_level`) from their proto names to the schema strings, then encode with
  `encoding/json` through an `Encoder` with `SetEscapeHTML(false)` and a two-space indent. The encoder
  sorts map keys and appends a trailing newline.
- **Rationale**: `protojson` already knows field presence, well-known types (`Timestamp` as RFC 3339), and
  zero-value omission, so the canonical form inherits those rules. Re-encoding through `encoding/json`
  removes `protojson`'s deliberately unstable whitespace and gives sorted keys, which is what makes the
  output byte-stable. `SetEscapeHTML(false)` keeps `&` and `<` in URLs and descriptions readable.
- **Alternatives considered**:
  - `protojson` alone with `Indent`: whitespace is randomized on purpose
    (`protojson` inserts a random extra space to stop callers relying on its output), so it fails the
    byte-stability criterion.
  - A hand-written struct mirror of the schema: duplicates every field and drifts when the proto changes.

## R2. Enum strings in the manifest

- **Decision**: The schema string is the enum value name with the enum's type prefix removed, lowercased:
  `INSTALLATION_METHOD_BINARY` → `binary`, `SECURITY_LEVEL_VERIFIED` → `verified`. The prefix is derived
  from the enum descriptor (the `UNSPECIFIED` value's name minus `UNSPECIFIED`). An unspecified value is
  omitted, which `protojson` already does for proto3 zero values.
- **Rationale**: These are exactly the schema's enum lists and the `registry.InstallationMethod` and
  `registry.SecurityLevel` constants, so no table is maintained by hand.
- **Alternatives considered**: A per-field lookup table. Rejected because the descriptor already holds the
  names and a table would need its own drift test.

## R3. YAML encoding

- **Decision**: Build YAML from the same canonical JSON: decode the canonical JSON bytes into a
  `yaml.Node` tree (`yaml.Unmarshal` of JSON, which YAML 1.2 accepts as a subset), clear the flow style on
  every node, and encode with `yaml.v3` at a two-space indent.
- **Rationale**: Decoding into a `yaml.Node` keeps the key order of the canonical JSON (already sorted),
  keeps numbers as numbers and strings as strings exactly as JSON typed them, and needs no Go-type
  round trip. Both formats therefore carry the same names and values by construction (FR-004).
- **Alternatives considered**:
  - `yaml.Marshal` of the proto struct (today's behavior): ignores `json` tags, lowercases Go field names,
    writes enums as integers.
  - `yaml.Marshal` of a `map[string]any`: works, but `float64` numbers and key ordering depend on Go-type
    marshalling rules; the node tree is more direct.

## R4. Loading every accepted format

- **Decision**: The loader decodes JSON or YAML into a generic value, normalizes it against the
  `PluginManifest` message descriptor, re-encodes it as JSON, and unmarshals with `protojson`
  (`DiscardUnknown: true`, as today). Normalization walks the descriptor:
  - Field keys match a field when the key equals the proto name, the JSON name, or the proto name with
    underscores removed compared case-insensitively. The last form covers the old YAML writer, which
    used lowercased Go field names (`specversion`, `supportedresources`).
  - Enum values accept the schema string (`binary`), the full proto name (`INSTALLATION_METHOD_BINARY`),
    or the number (`1`), and become the full proto name.
  - A `Timestamp` given as an object with `seconds` and `nanos` (old YAML) becomes RFC 3339.
  - Map fields (`supported_resources`) keep their keys and normalize their values against the value
    message.
- **Rationale**: One descriptor-driven pass reads the new format, the old `protojson` format, and the old
  YAML format, so FR-006 and FR-007 hold without a format sniffer. It also needs no maintenance when a
  field is added.
- **Alternatives considered**: Keep `yaml.Unmarshal` straight into the proto struct for legacy YAML and
  add a format detector. Rejected: detection is fragile and the old path cannot read the new snake_case
  YAML.

## R5. Method list from the protocol

- **Decision**: Build the method set once at package init from
  `pbc.File_finfocus_v1_costsource_proto.Services().ByName("CostSourceService").Methods()` and keep it in a
  package-level slice. The schema's `methods` enum lists the same names; a test compares the two and fails
  with the missing or extra name.
- **Rationale**: The SDK list cannot drift (it is derived), and the static schema is guarded by the test
  (FR-017). The registry package may import the generated proto package: the generated code does not
  import `registry`, so there is no cycle.
- **Alternatives considered**: A hand-written list plus a test. Equivalent safety, more maintenance.

## R6. Capability strings from the protocol

- **Decision**: The manifest string for a `PluginCapability` is the lowercase value name without the
  `PLUGIN_CAPABILITY_` prefix (`dry_run`, `estimate_cost`, `recommendation_scoring`).
  `registry.ManifestCapabilityName(pbc.PluginCapability) string` returns it (empty for unspecified or
  unknown values). The set of protocol-derived strings is built once at package init from the enum
  descriptor. `IsValidPluginCapability` accepts the existing 14 strings or a protocol-derived string.
  `AllPluginCapabilities` returns the existing 14 followed by the protocol-derived values in enum-number
  order. Both schema enums (manifest and registry index) list the same union; a test compares them.
- **Rationale**: Lowercase-without-prefix matches the scorer metadata convention already in the SDK
  (`scorer_supported_signals`). Deriving at init keeps validation a slice scan with zero allocations,
  matching the registry package's enum pattern.
- **Alternatives considered**:
  - The legacy metadata names (`supports_dry_run`): a different surface with a `supports_` prefix that
    reads badly in a capability list, and not every value has one.
  - Replacing the 14 existing strings: breaks every existing manifest (Principle VI spirit).
  - Exported constants for the 18 new values: 18 more exported identifiers, including
    `PluginCapabilityPricingSpec` beside the existing `PluginCapabilityPricingSpecs`, which invites
    confusion. The conversion function covers the use case.

## R7. Resource type bound

- **Decision**: `maxLength` 256 in the schema and the SDK validator, with a shared constant
  `registry.MaxResourceTypeLength = 256`. The description states the entry is the string matched against
  `ResourceDescriptor.resource_type`, recommends the full type token, and repeats that native packages go
  under their cloud.
- **Rationale**: Covers every Pulumi and Terraform token seen in practice with room to spare and keeps the
  field bounded for registries.
- **Alternatives considered**: Removing the bound. Rejected: an unbounded string in a registry index
  invites abuse and the issue allows either option.

## R8. Validator agreement with the schema

- **Decision**: Add `validateSupportedResources` and `validateCapabilities` to
  `registry.ValidatePluginManifest`, following the existing map-walking style and error format
  (`specification.supported_resources.<key>.<field>[i]: ...`). Billing modes reuse the schema's list as a
  package-level slice. A table test runs each case through the SDK validator and the compiled JSON schema
  (`santhosh-tekuri/jsonschema/v6`, already a dependency) and asserts the same verdict.
- **Rationale**: The existing validator is hand-written; matching its style keeps the diff reviewable, and
  the agreement test is what proves FR-013.
- **Alternatives considered**: Make `ValidatePluginManifest` delegate to the JSON schema. Rejected for this
  feature: it changes every error message the validator returns, which callers may match on.

## R9. Example manifests baseline

- **Finding**: Against the manifest schema today, `aws-cost-plugin.json`, `gcp-cost-plugin.json`,
  `kubecost-plugin.json`, and `minimal-plugin.json` pass. `azure-cost-plugin.json` fails on
  `download_url` (a container reference, not a URI) and `greenops-plugin.json` is not shaped as a
  manifest. A Go test validates the four passing files with the schema and the SDK validator.
- **Follow-up (PR review)**: `azure-cost-plugin.json` now uses `oci://registry.hub.docker.com/...`, and
  `greenops-plugin.json` became a manifest: its `supported_metrics` (a `SupportsResponse` field, not a
  manifest field) gave way to the `carbon`, `energy`, and `water` capabilities. The test covers all six.

## R10. TypeScript SDK parity

- **Finding**: The TypeScript SDK has no manifest writer, loader, or validator (only the generated
  registry client). The schema change is the shared contract; no TypeScript code change is needed, and no
  `.proto` message changes.
