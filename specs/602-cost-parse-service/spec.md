# Feature Specification: Cost Parse Service

**Feature Branch**: `recommendations_history` (existing branch; no branch hook configured)

**Created**: 2026-10-18

**Status**: Implemented and validated on 2026-10-18; see [validation evidence](quickstart.md).

**Input**: User description: `docs/superpowers/specs/2026-10-18-cost-parse-service-design.md`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Discover a parser and obtain priceable resources (Priority: P1)

As a host integrator, I want to discover which plugin can parse a cloud-native or Kubernetes input and obtain
resources ready for the existing pricing workflow, so users can estimate costs without manually translating templates.

**Why this priority**: Discovery and parsing establish the core value of the feature.

**Independent Test**: Use a provider-neutral sample parser to discover its accepted format, input modes, and size
limit, submit valid content, and verify that the resulting resource descriptors can enter the existing pricing flow.

**Acceptance Scenarios**:

1. **Given** a parser plugin, **When** the host discovers it, **Then** parsing capability, supported formats,
   accepted input modes, and maximum total input bytes are available.
2. **Given** a supported format and valid file content within the advertised limit, **When** parsing completes,
   **Then** each returned resource has a valid descriptor, a priceable resource type, and a unique source identifier.
3. **Given** a plugin that accepts filesystem input, **When** the host submits an accessible root,
   **Then** the plugin parses the exposed input without reading through symlinks outside that root.
4. **Given** either a parser-only plugin or a pricing plugin with parsing support, **When** the host invokes parsing,
   **Then** the same parsing contract applies without requiring a separate resource-type resolution step.
5. **Given** a plugin without parsing support, including a plugin whose handler reports it as unimplemented,
   **When** parsing is requested, **Then** the host receives the normal unsupported-service outcome.

---

### User Story 2 - Diagnose invalid or unsupported input (Priority: P1)

As a host integrator, I want predictable rejection reasons so I can help users correct their input or choose
another plugin without guessing from free-form messages.

**Why this priority**: Predictable validation prevents silent input loss and unsafe or unsupported parsing.

**Independent Test**: Submit malformed requests, unsupported formats and modes, and inputs at size boundaries;
verify each outcome against its documented category and field or source location.

**Acceptance Scenarios**:

1. **Given** missing or conflicting input selections, an invalid format name, or an unsafe file path,
   **When** parsing is requested, **Then** the request is rejected as invalid and identifies the offending field.
2. **Given** a syntactically valid but unadvertised format, **When** parsing is requested,
   **Then** the request fails a precondition with the stable reason `UNSUPPORTED_FORMAT`.
3. **Given** a supported format and an unadvertised input mode, **When** parsing is requested,
   **Then** the request fails a precondition with the stable reason `UNSUPPORTED_INPUT_MODE`.
4. **Given** a documented format option, **When** it is supplied within the options size limit,
   **Then** it is accepted; an unknown option key is rejected as invalid.
5. **Given** unparseable content, **When** parsing is attempted, **Then** the error identifies the file and the line
   when known; input exceeding the advertised maximum instead receives a resource-limit failure.

---

### User Story 3 - Trace results and understand incomplete resources (Priority: P2)

As a user reviewing estimated resources, I want to connect each result or warning to its source and distinguish
unresolved values from missing properties, so I can understand gaps in an estimate.

**Why this priority**: Traceability makes partial results actionable and prevents unknown values appearing resolved.

**Independent Test**: Return a sample with resolved properties, an unresolved property, and a skipped resource;
verify identifiers, property omission, warning codes, and descriptor validity independently of vendor format logic.

**Acceptance Scenarios**:

1. **Given** parsed resources, **When** results are returned, **Then** every resource has a non-empty source identifier
   unique within the response, using the input's own naming convention.
2. **Given** an unresolved property, **When** its resource is returned, **Then** its dotted path appears in the
   unknown-property list and its value is absent from the descriptor's attributes.
3. **Given** a skipped resource, **When** parsing returns other usable resources, **Then** a separate warning identifies
   the skipped source and supplies severity, a stable non-empty code, and an explanatory message.
4. **Given** duplicate or empty source identifiers, an invalid descriptor, malformed unknown paths, a listed unknown
   value present in attributes, or an empty warning code, **When** results are validated, **Then** validation fails.

---

### User Story 4 - Reuse results and integrate consistently (Priority: P2)

As a host integrator or plugin maintainer, I want consistent parsing behavior and clear reuse guidance so integrations
work across supported clients and plugins without format-specific assumptions in the host.

**Why this priority**: Consistency and explicit freshness guidance support reliable adoption across the ecosystem.

**Independent Test**: Run the provider-neutral conformance scenarios, repeat identical requests, and round-trip binary
input and an expiration time through every supported client transport.

**Acceptance Scenarios**:

1. **Given** identical input and options with unchanged dependencies, **When** parsing is repeated,
   **Then** resource and warning results are deterministic.
2. **Given** output dependent on changing external data, **When** the plugin supplies an expiration time,
   **Then** the host can use it as advisory freshness guidance; omission provides no expiration guidance.
3. **Given** binary file content and a response expiration time, **When** a supported client sends and receives them,
   **Then** neither the original bytes nor the timestamp changes in transit.
4. **Given** multiple plugins advertising the same format, **When** the integrator follows the host guidance,
   **Then** explicit user choice wins, otherwise an unambiguous provider match wins, otherwise selection fails
   as ambiguous rather than merging outputs.

### Edge Cases

- Format names of 1 and 32 bytes are valid if they match the permitted pattern; empty names, 33-byte names,
  uppercase letters, and names starting with a digit are invalid.
- Absolute file paths, colons, NUL bytes, any `..` path segment, and duplicate file paths are invalid. This applies to files
  supplied as content; the filesystem root is a separate input whose location is intentionally exposed by the host.
- Neither input selection, conflicting selections, an empty file set, or an empty filesystem root is invalid.
  A zero-byte file is left to the selected format's parsing rules.
- Options exactly 65,536 bytes in the contract encoding meet the size cap; larger options are invalid.
  Total input exactly at the advertised maximum meets that cap; larger input exhausts the resource limit.
- A supported format can still fail because the selected mode is not supported.
- A document may produce no resources, including when every resource is skipped; warnings explain any skips.
- An unresolved nested value remains distinguishable from an absent property and from a known zero or empty value.
- Warnings concerning an entire input may omit a source identifier; warnings for skipped resources must identify it.
- Expiration is advisory; it is neither a guarantee that cached results stay valid nor a requirement to cache.
- File-path validation does not itself prevent a plugin from following an escaping symlink in filesystem mode.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The feature MUST define one optional parsing service for cloud-native and Kubernetes inputs that returns
  resources usable by the existing pricing workflow. Parsing MUST NOT itself calculate costs.
- **FR-002**: Plugins MUST advertise parsing capability, a non-empty set of supported format names, accepted input modes
  (`content`, `path`, or both), and a positive maximum total input size in bytes. Discovery validation MUST reject
  malformed advertisements, and existing plugins' capabilities MUST remain unchanged.
- **FR-003**: A request MUST identify one format and exactly one input mode: a collection of named file contents or
  a filesystem root. Invalid or conflicting input selections MUST NOT be silently ignored.
- **FR-004**: Format names MUST be lowercase, 1 to 32 bytes, and match `[a-z][a-z0-9_-]*`.
- **FR-005**: Each supplied content file MUST have a non-empty relative path with no `..` segment, colon or NUL byte,
  unique within the collection. Filesystem-root input MUST be non-empty. The host exposes only an intended root,
  and plugins MUST NOT follow symlinks outside it; these filesystem-access responsibilities are documented obligations.
- **FR-006**: Requests MAY carry structured, format-specific options. Options MUST occupy at most 65,536 bytes in the
  contract encoding, and plugins MUST reject unknown keys. Supported keys MUST be documented per format:
  `parameters` for `cfn`; `release_name`, `namespace`, and `values` for `helm`; none for `kustomize`.
- **FR-007**: Plugins MUST enforce their advertised maximum total input bytes. The first version MUST return a complete
  response per request; streaming and chunked results are outside scope.
- **FR-008**: Invalid format syntax, input selection, paths, or options MUST produce an invalid-argument error naming
  the field. Unparseable content MUST produce an invalid-argument error naming the file and line when known.
- **FR-009**: An unadvertised format MUST produce a failed-precondition error with reason `UNSUPPORTED_FORMAT`.
  An unadvertised input mode MUST produce a failed-precondition error with reason `UNSUPPORTED_INPUT_MODE`.
  Input over the advertised maximum MUST produce a resource-exhausted error.
- **FR-010**: Each parsed resource MUST contain a valid existing resource descriptor, including the pricing type token
  needed to avoid a separate type-resolution call. Attributes MUST use the property shape expected by the plugin's
  pricing behavior, preserving CloudFormation property casing where applicable.
- **FR-011**: Every parsed resource MUST have a non-empty, response-unique source identifier derived from the input:
  the logical identifier for CloudFormation, or `Kind/name` for Helm and Kustomize.
- **FR-012**: Parsed resources MUST identify unresolved properties using the existing omitted-field dotted-path
  convention. Every listed path MUST be well formed, and its value MUST be absent from the descriptor's attributes.
- **FR-013**: Responses MUST carry warnings separately from resources. Warnings MUST expose severity, a stable non-empty
  code, an explanatory message, and a source identifier when applicable. Every skipped resource MUST have a warning
  identifying its source; hosts MUST be able to match warning codes without interpreting message text.
- **FR-014**: Responses MAY provide an advisory expiration time under the existing freshness convention. Omission MUST
  mean no caching guidance. Plugins MUST provide expiration only for results dependent on changing data.
- **FR-015**: For identical input and options with unchanged dependencies, parsed resources and warnings MUST be
  deterministic. Documentation MUST explain when hosts may cache results by input hash.
- **FR-016**: Parsing support MUST be optional for both pricing and parser-only plugins. Plugins without a working
  parsing provider MUST retain the normal unimplemented behavior. Parser-only capability declaration and startup
  diagnostics MUST follow the design's existing optional-service pattern.
- **FR-017**: Supported service transports MUST expose the same validation and error outcomes. Health discovery MUST
  include parsing only when a working parsing provider is registered.
- **FR-018**: Shared request and response validation MUST cover FR-003 through FR-013, including existing descriptor
  rules. Adding parsing validation MUST preserve existing non-parsing validation behavior and performance expectations.
- **FR-019**: Maintainers MUST have a separate provider-neutral mock parser and conformance checks for unsupported
  formats, exclusive input selection, unsupported modes, unsafe paths, unique identifiers, unresolved values,
  valid descriptors, and determinism. Checks MUST assess structure and errors without requiring vendor templates
  or prescribing resource contents. Existing mock-plugin capabilities MUST remain unchanged.
- **FR-020**: Both supported language SDKs MUST support parsing. Client transports MUST preserve binary file content
  and expiration timestamps according to the canonical contract representation.
- **FR-021**: Existing messages, source-format definitions, type-resolution behavior, and native Terraform and Pulumi
  ingestion MUST remain compatible. Individual plugins' parsing logic, allocation, recommendations, and changes to
  the core's internal parser interface are outside this feature.
- **FR-022**: Documentation MUST state the cloud-native and Kubernetes scope, format-specific option ownership,
  parser-only deployment, conflicting-plugin selection, untrusted-input handling, root containment, and caching rules.
  Host selection guidance MUST prioritize explicit user choice, then an unambiguous provider match, then ambiguity
  failure; combining multiple parsers' results is undefined and outside scope.

### Key Entities *(include if feature involves data)*

- **Parser advertisement**: A plugin's parsing capability, supported formats, accepted input modes, and input size limit.
- **Parse request**: A format, exactly one input selection, and optional format-specific settings.
- **Input file set**: Named file contents whose paths are safe and unique within a request.
- **Filesystem root**: A host-exposed location delimiting the input a plugin may read.
- **Parsed resource**: An existing priceable resource descriptor plus source identity and unresolved property paths.
- **Parse warning**: A severity, stable code, human-readable explanation, and optional source identity describing
  incomplete or skipped work independently of returned resources.
- **Parse result**: A collection of parsed resources, a separate warning collection, and optional advisory expiration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every valid provider-neutral sample within advertised limits, a host can discover a suitable parser
  and submit all returned descriptors directly to the existing pricing workflow with zero extra type-resolution steps.
- **SC-002**: All documented invalid-request and unsupported-input scenarios return the specified error category,
  stable reason where required, and field or source location where available.
- **SC-003**: Across all accepted sample responses, 100% of returned resources have valid descriptors and unique,
  non-empty source identities; 100% of skipped resources have source-linked warnings.
- **SC-004**: In every unresolved-property sample, all unresolved paths are reported and none also has a value
  in the returned attributes, allowing users to distinguish unknown values from absent and known values.
- **SC-005**: Across 100 repetitions of each fixed sample with unchanged dependencies, resource and warning results
  are identical; both supported input modes obey their advertised byte boundaries.
- **SC-006**: Every supported client transport preserves 100% of sample input bytes and expiration timestamps,
  and passes the same provider-neutral parsing scenarios.
- **SC-007**: Existing non-parsing compatibility and validation checks continue to pass, and comparative measurements
  show no repeatable slowdown attributable to parsing in existing validation workflows.
- **SC-008**: An integrator can complete each documented discovery, partial-result diagnosis, parser-selection,
  and caching example with no undocumented selection rule or warning-message interpretation.

## Assumptions

- The [source design](../../docs/superpowers/specs/2026-10-18-cost-parse-service-design.md) records the accepted scope
  and contract decisions for this specification. Wire layout, generated bindings, registration details, and concrete
  capability identifiers remain planning inputs in that document.
- This repository delivers a contract, SDK support, mock, conformance tooling, and documentation. Actual CloudFormation,
  Helm, Kustomize, and future ARM or Bicep parsing implementations belong to individual plugins.
- An empty file collection and an empty root provide no usable input and are treated as invalid. File contents may
  be empty if a format permits them. Filesystem roots may be absolute; content-file paths may not.
- Total input bytes mean the sum of submitted content bytes in content mode, or the bytes consumed from input files
  in path mode. Options have their separate encoding-size cap; they do not count as file contents.
- Format-specific option meanings are owned by the plugin. This feature documents known keys and rejection behavior;
  it does not add automated checks for drift between each plugin's documentation and accepted keys.
- Determinism concerns resources and warnings. An advisory expiration may advance with time while the underlying
  parsed result stays unchanged. External dependency changes may legitimately change results.
- Without expiration, parsing is assumed to depend only on supplied input and options. Hosts caching by input hash
  must include the format, options, and parser identity and account for filesystem content changes.
- Symlink containment is a plugin contract responsibility rather than a guarantee of shared validation or conformance.
  Conflicting-parser selection is host guidance rather than behavior enforced by the SDK.
- Existing descriptor validation, omitted-field paths, freshness semantics, optional-service error handling,
  and provider-neutral samples are dependencies. The implementation plan must check the design's explicit
  parser-only capability declaration against constitution principle XII on automatic discovery and record any
  necessary compliance decision before implementation.
