# Feature Specification: Structured Attributes on ResourceDescriptor

**Feature Branch**: `596-resource-descriptor-attributes`

**Created**: 2026-10-03

**Status**: Draft

**Input**: User description: "Issue #617: add structured attributes to ResourceDescriptor and raise
MaxTagValueLength. Add `google.protobuf.Struct attributes = 12` with a host redaction rule and a fallback to
`tags`; bound its size with `MaxAttributesBytes` (64 KiB) in both descriptor validators; document or enforce
the batch and transport interaction; raise `MaxTagValueLength` from 256 to 2048 in both validation layers; add
a generic dotted-path read helper; regenerate Go and TypeScript bindings; make conformance exercise the field
while a plugin that ignores it still passes; update every document that describes descriptor properties or tag
limits. Closes #617."

## Clarifications

`/speckit-clarify` was not required. Issue #617 fixes the field number, both limits, and the scope. Its one
open choice, whether to enforce or document the batch and transport interaction, is resolved in Assumptions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A plugin reads a resource's nested inputs on any cost RPC (Priority: P1)

A plugin author prices a resource whose cost depends on a nested input, such as a Kubernetes Deployment's
container CPU request (`spec.template.spec.containers.0.resources.requests.cpu`) or an Azure SQL database's
`sku.capacity`. Today the host can deliver that value only through `EstimateCost`. On `Supports`,
`GetProjectedCost`, `GetPricingSpec`, `GetRecommendations` target resources, and `BatchCost`, the value is
collapsed into a lossy string tag or dropped. With this feature, the host sends the resource's declared
properties unflattened on the descriptor, and the plugin reads the nested value directly.

**Why this priority**: This is the blocking need. rshade/finfocus#1525 (projected cost for Kubernetes
workloads) cannot ship without it, and no tag encoding can carry the inputs within the contract.

**Independent Test**: Send a `GetProjectedCost` request whose descriptor carries nested attributes to a test
plugin over the in-memory harness, and confirm that the plugin receives the nested value intact.

**Acceptance Scenarios**:

1. **Given** a descriptor whose `attributes` hold a nested map and a list, **When** a host sends it on any
   RPC that carries a `ResourceDescriptor`, **Then** the plugin receives the same structure, with no
   flattening or loss.
2. **Given** a descriptor with no `attributes`, **When** a plugin handles it, **Then** the plugin falls back to
   `tags` exactly as it does today.
3. **Given** a descriptor that carries both `attributes` and `tags`, **When** an older plugin that does not
   know the field handles it, **Then** the plugin behaves exactly as before, reading `tags` only.

---

### User Story 2 - Hosts and plugins agree on a size bound (Priority: P1)

A host operator needs to know how much structured data a single resource may carry, and a plugin maintainer
needs the SDK to reject unbounded input before it is processed. Both validation layers (the plugin SDK
denial-of-service guard and the testing contract) enforce the same documented maximum encoded size for
`attributes`. The bound's interaction with batch requests and transport message limits is documented, so
hosts know they must split batches.

**Why this priority**: Without a bound, the new field is an unbounded allocation vector, which is the same
risk that the existing descriptor limits address.

**Independent Test**: Validate descriptors whose attributes are exactly at the limit and one byte over the
limit with both validators, and confirm that the first passes and the second is rejected as an invalid
argument.

**Acceptance Scenarios**:

1. **Given** attributes whose encoded size equals the maximum, **When** either validator checks the
   descriptor, **Then** validation passes.
2. **Given** attributes whose encoded size is one byte over the maximum, **When** either validator checks the
   descriptor, **Then** validation fails with an invalid-argument error that names the field, the size, and
   the limit.
3. **Given** a `BatchCost` request whose resources carry attributes, **When** a host reads the documentation,
   **Then** it finds the rule that the whole encoded request must fit the transport limit and that the host
   splits batches to stay under that limit.

---

### User Story 3 - Long scalar inputs fit in a tag (Priority: P2)

A host forwards long scalar inputs, such as IAM policy documents, user data, long ARNs, and connection
metadata, as tag values. Today any tag value over 256 characters makes the whole descriptor invalid. With this
feature, both validation layers accept tag values up to 2048 bytes, which matches the existing ARN bound.

**Why this priority**: This is independent of the structured field and improves flat inputs for every plugin,
but the structured field is the primary unblocker.

**Independent Test**: Validate a descriptor with a 2048-byte tag value (accepted) and with a
2049-byte tag value (rejected) in both validators.

**Acceptance Scenarios**:

1. **Given** a tag value of 2048 bytes, **When** either validator checks the descriptor, **Then**
   validation passes.
2. **Given** a tag value of 2049 bytes, **When** either validator checks the descriptor, **Then**
   validation fails as before, citing the new limit.
3. **Given** 51 tags, **When** the contract validator checks them, **Then** validation fails exactly as
   today. Tag counts do not change.

---

### User Story 4 - Plugins read a nested value without writing a walker (Priority: P2)

A plugin author wants one call that returns the value at a dotted path, such as
`spec.template.spec.containers.0.resources.requests.cpu`, or reports that it is absent. The SDK provides a
generic accessor that treats a numeric segment as a list index and reports absence, rather than an error, when
any segment is missing.

**Why this priority**: Without a shared helper, every plugin writes its own walker, each with different
edge-case behavior. The field is usable without the helper, so this story comes after the field.

**Independent Test**: Table tests of the accessor cover a hit, a missing segment, a list index, an
out-of-range index, a path through a scalar, and an absent attribute set.

**Acceptance Scenarios**:

1. **Given** attributes containing the path, **When** the accessor is called, **Then** it returns the value
   and reports that the path was found.
2. **Given** a path with a missing segment, **When** the accessor is called, **Then** it reports not found and
   raises no error.
3. **Given** a numeric segment applied to a list, **When** the index is in range, **Then** the element is
   returned. When the index is out of range or negative, the accessor reports not found.
4. **Given** absent attributes, **When** the accessor is called, **Then** it reports not found.

---

### User Story 5 - Conformance exercises the field and stays backward compatible (Priority: P2)

A plugin maintainer runs the conformance suite. The suite sends descriptors that carry `attributes`, and a
plugin that ignores the field still passes every level, which proves that the change is additive.

**Why this priority**: This verifies the additive guarantee for every existing plugin.

**Independent Test**: Run the conformance suite against the mock plugin, which ignores `attributes` for
pricing, and confirm that it passes. Confirm that the attributes request reaches the plugin and validates.

**Acceptance Scenarios**:

1. **Given** the conformance suite, **When** it runs against a plugin that never reads `attributes`, **Then**
   the plugin passes.
2. **Given** the conformance suite, **When** it runs, **Then** at least one request carries a descriptor with
   nested `attributes`, and the request is accepted.

---

### User Story 6 - The documentation tells hosts and plugins how to use the field (Priority: P2)

Host and plugin authors read `docs/PROPERTY_MAPPING.md`, `PLUGIN_DEVELOPER_GUIDE.md`, the SDK READMEs, and the
proto comments. Every document that describes how properties reach a plugin, or what the tag limits are,
states the new field, the redaction rule, the "prefer `attributes`, fall back to `tags`" guidance, the size
bound with its batch implication, and the new tag value limit. No document still says that the structured form
exists only on `EstimateCost`, or that tag values are capped at 256.

**Why this priority**: Under the constitution, stale documentation blocks merge, and the redaction rule exists
only in prose.

**Independent Test**: Search the repository's non-spec documentation for the old 256 tag value limit and for
the "only in EstimateCost" claim, and find no remaining occurrences.

**Acceptance Scenarios**:

1. **Given** the updated repository, **When** a reader searches the non-historical docs for the tag value
   limit, **Then** every occurrence states 2048.
2. **Given** the updated property mapping doc, **When** a reader looks up how a host hands over nested inputs,
   **Then** they find the `attributes` section with the redaction rule and the fallback.

### Edge Cases

- Attributes set to an empty structure, with no keys: treated the same as absent. Plugins fall back to
  `tags`.
- Attributes nested very deeply: the size bound limits total work. The accessor walks the path iteratively,
  without recursion.
- A key that itself contains a dot: the dotted accessor cannot address it. The documentation states this, and
  plugins can read such a key directly from the structure.
- Numeric values in attributes are double-precision on the wire. Integers above 2^53 lose precision. The
  documentation states that hosts send such values as strings.
- Secret-marked values: the host omits them. A plugin cannot distinguish an omitted secret from an absent
  property, and it must not try.
- A batch where each resource is within the limit but the total exceeds the transport limit: the transport
  rejects the request (gRPC `ResourceExhausted`, Connect payload limit) before any SDK validator runs. The
  documentation tells hosts to split batches.
- An older host that never sets the field: no behavior change.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The resource descriptor MUST gain an optional structured `attributes` field (field number 12, a
  generic JSON-like structure) that carries the resource's declared properties without flattening.
- **FR-002**: The field's contract documentation MUST state that absent or empty attributes mean the host sent
  none and plugins fall back to `tags`, that `tags` keep their current meaning, and that hosts SHOULD keep
  sending `tags` so that older plugins are unaffected.
- **FR-002a**: When the same property appears in both `attributes` and `tags` with different values, plugins
  MUST prefer `attributes`, because it is the unflattened source. Tags with no counterpart in `attributes`,
  such as resource labels, keep their meaning.
- **FR-003**: The field's contract documentation MUST require host redaction. Hosts MUST omit keys that start
  with `__`, credential-like keys, and values that the IaC tool marks secret (for Pulumi, the secret signature
  `4dabf18193072939515e22adb298388d`). A credential-like key is one whose name, matched case-insensitively,
  contains `password`, `secret`, `token`, `credential`, `privatekey`, `accesskey`, or `connectionstring`. The
  host may redact more. Plugins MUST NOT log the field verbatim. The SDK itself never logs request
  descriptors.
- **FR-004**: The SDK MUST define a maximum encoded size for `attributes` of 65,536 bytes (64 KiB), measured as
  the protobuf wire size of the structure, and give the reason in the constant's documentation.
- **FR-005**: The plugin SDK's descriptor validator and the testing contract's descriptor validator MUST both
  reject attributes larger than the maximum with an invalid-argument error that names `attributes`, the actual
  size, and the limit. Both MUST accept attributes exactly at the limit.
- **FR-006**: The batch and transport interaction MUST be documented. A batch's whole encoded request must fit
  the transport message limit (1 MB on the Connect/HTTP path and 4 MB by default on gRPC), and hosts split
  batches whose resources carry large attributes. The SDK does not lower `max_batch_size` automatically.
- **FR-007**: The tag value length limit MUST be 2048 in both validation layers, and both MUST change
  together. As with every existing descriptor length limit, it is measured in bytes of the UTF-8 encoding,
  so a multi-byte value reaches the limit in fewer characters. Tag count limits (50 in the contract, 256 in the DoS
  guard) and tag key limits (128) MUST NOT
  change.
- **FR-008**: The comment that describes the DoS-guard limits MUST be updated so that its comparison with the
  contract limits stays accurate.
- **FR-009**: The plugin SDK MUST provide a generic accessor that returns the value at a dot-separated path
  within `attributes`, together with a found flag. A numeric segment indexes a list. A missing key, an
  out-of-range or non-numeric index, a path through a scalar, and absent attributes all report not found
  without an error. The accessor MUST NOT contain any provider-specific paths.
- **FR-010**: The Go and TypeScript bindings MUST be regenerated from the proto, and the TypeScript SDK tests
  MUST pass, including a test that a descriptor with attributes round-trips through the TypeScript client.
- **FR-011**: The conformance harness MUST send at least one request whose descriptor carries nested
  `attributes`. A plugin that ignores `attributes` MUST pass every conformance level.
- **FR-012**: The mock plugin MUST accept descriptors with attributes and ignore them when pricing, so that it
  serves as the conformance suite's "plugin that ignores `attributes`". The mock validates no requests today,
  and it gains no new validation.
- **FR-013**: `docs/PROPERTY_MAPPING.md` MUST gain an `attributes` section that covers the redaction rule, the
  fallback, the size bound, and the accessor. It MUST no longer state that the structured form exists only on
  `EstimateCost`, and it MUST give the new tag value limit.
- **FR-014**: `PLUGIN_DEVELOPER_GUIDE.md` MUST update its sample limits and its GetProjectedCost guidance to
  "prefer `attributes`, fall back to `tags`". `sdk/go/pluginsdk/README.md` and `sdk/go/testing/README.md` MUST
  document the new constant, the accessor, and the new tag limit.
- **FR-015**: Every other non-historical document, and every code comment, that states the tag value limit or
  lists the descriptor's fields MUST be updated so that no stale value remains. Historical `specs/` records and
  `CHANGELOG.md` are exempt.
- **FR-016**: The change MUST be additive. `buf lint` and `buf breaking` against `main` MUST pass, and no
  existing field may change meaning.
- **FR-017**: Validating a descriptor without attributes MUST cost no more allocations than it does today.
  The size check runs only when attributes are present.

### Key Entities

- **ResourceDescriptor.attributes**: the resource's declared properties as a nested structure of maps, lists,
  strings, numbers, booleans, and nulls. It is optional, bounded by the maximum encoded size, and redacted by
  the host.
- **MaxAttributesBytes**: the maximum encoded size of `attributes` per descriptor (64 KiB). Both validation
  layers share the same value.
- **MaxTagValueLength**: the maximum tag value length in bytes, 2048 in both validation layers.
- **Attribute accessor**: a read-only lookup from a dot-separated path to a value plus a found flag.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin receives a ten-segment nested input (the depth of a CronJob container's CPU request)
  intact on every RPC that carries a resource descriptor, verified by a harness test.
- **SC-002**: Both validators accept attributes of exactly 65,536 encoded bytes and reject 65,537 bytes, and
  both accept a 2048-byte tag value and reject 2049 bytes, with 100% agreement between the two
  layers.
- **SC-003**: Every existing conformance and unit test passes unchanged, apart from tests that assert the old
  256-character boundary, so plugins that ignore the field lose no compatibility.
- **SC-004**: Validating a descriptor without attributes stays at its current allocation count (0 extra
  allocations per operation).
- **SC-005**: A search of non-historical documentation finds zero statements of a 256-byte or 256-character tag value
  limit and zero claims that the structured form exists only on `EstimateCost`.
- **SC-006**: The accessor's table tests cover a hit, a missing segment, a list index, an out-of-range index, a
  scalar in mid-path, and absent attributes, and all of them pass.

## Assumptions

- The 64 KiB bound and the 2048-byte tag value bound are the values proposed in #617. 2048 matches the
  existing `MaxARNLength`. Neither value was picked by an ADR, and both are documented in their constants.
- Batch and transport interaction is **documented, not enforced**. The transport rejects an oversized request
  before any SDK validator runs, so a validator-level aggregate check could never fire on the server. Hosts
  own batch splitting, which they already do for `max_batch_size`.
- Redaction is a host obligation stated in the contract. The SDK cannot detect secrets that the host already
  dropped, and it does not attempt host-side redaction. Building `attributes` from IaC inputs is core work
  (rshade/finfocus#1525).
- The TypeScript SDK gains no accessor. protobuf-es exposes a structure field as a plain JSON object, which a
  TypeScript caller can index natively. TypeScript parity is the regenerated binding plus a round-trip test.
- The size is measured as the protobuf wire size of the structure, as #617 proposes. That is the size that
  counts against the transport limit.
- `EstimateCostRequest.attributes` is unchanged. The new field mirrors its name and semantics.
- Out of scope: any provider-specific schema for `attributes`, changes to tag counts, making `attributes`
  required, and host-side population (core work).
