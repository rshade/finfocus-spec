# Research: Structured Attributes on ResourceDescriptor

## R1: Field number and type

- **Decision**: `google.protobuf.Struct attributes = 12` on `ResourceDescriptor`.
- **Rationale**: Field 11 (`lineage`) is the highest number in use, and the message has no `reserved` ranges.
  `struct.proto` is already imported by `costsource.proto`. The name and type mirror
  `EstimateCostRequest.attributes` (field 2), so hosts and plugins use one mental model.
- **Alternatives considered**: A `map<string, google.protobuf.Value>` gives the same wire shape with a different
  generated type, and the rest of the API already uses `Struct`, so it was rejected. A JSON string field was
  rejected because it is untyped, it is the same as a long tag, and it puts parsing on every plugin.

## R2: How to measure the size

- **Decision**: `proto.Size(attributes)`, guarded by `attributes != nil`. The limit is
  `MaxAttributesBytes = 64 << 10`.
- **Rationale**: Wire size is what counts against the transport limits (1 MB Connect, 4 MB gRPC default). The
  check runs only when the field is set, so descriptors without attributes keep today's allocation count
  (FR-017, SC-004). `proto.Size` is linear in the encoded size, and the decoder has already done equivalent
  work.
- **Alternatives considered**: A JSON-encoded length was rejected because it allocates and depends on the
  encoder. A key count or depth cap was rejected because it does not bound bytes, and the bytes are the
  DoS-relevant quantity. Decode-time recursion is already bounded by protobuf-go's default recursion limit.

## R3: Batch and transport interaction

- **Decision**: Document the interaction and do not enforce it.
- **Rationale**: grpc-go rejects an oversized message with `ResourceExhausted`, and `payloadLimitMiddleware`
  (`sdk.go`, `maxPayloadSize = 1 << 20`) rejects an oversized Connect request, both before any handler or
  validator runs. A validator-level aggregate check can therefore never fire on the server. On the host,
  `ValidateBatchCostRequest` is not the place where hosts decide how to split a batch. Hosts already split by
  `max_batch_size`, and they now also split by encoded size. Lowering `max_batch_size` automatically would
  silently change advertised capacity for plugins that never see attributes.
- **Alternatives considered**: An aggregate cap in `ValidateBatchCostRequest` was rejected as unreachable
  server-side. An automatic `max_batch_size` reduction was rejected because it is a behavior change for all
  plugins.

## R4: Tag value limit

- **Decision**: 2048 in both `pluginsdk.MaxTagValueLength` and `testing.MaxTagValueLength`.
- **Rationale**: The value matches `MaxARNLength`. The pluginsdk comment says its DoS limits are "more
  generous than" the contract limits. Moving both together keeps that statement true (equal, as today). The
  worst case per descriptor under the DoS guard becomes 256 tags × (128 + 2048) bytes ≈ 557 KB, which is under
  the 1 MB Connect cap for one resource.
- **Alternatives considered**: Raising only the contract limit would make pluginsdk reject what the contract
  accepts, so it was rejected.

## R5: Accessor shape

- **Decision**: `pluginsdk.AttributeValue(attrs *structpb.Struct, path string) (*structpb.Value, bool)`. The
  path is split on `.`. Struct segments index fields. On a list, a segment is parsed as a non-negative decimal
  index. Anything else reports `(nil, false)`. The walk is iterative and allocation-free, because it scans the
  path in place with `strings.IndexByte` rather than calling `strings.Split`.
- **Rationale**: This is generic and has no provider paths (FR-009). Returning `*structpb.Value` lets the
  caller pick the kind (`GetStringValue`, `GetNumberValue`, `GetStructValue`). A found flag distinguishes an
  explicit JSON `null` (found, `NullValue`) from absence.
- **Alternatives considered**: Typed getters per kind can be layered on later, so they were deferred. JSON
  Pointer (`/a/0/b`) was rejected because the issue and the core flattening convention both use dotted paths.
  Accepting `*pbc.ResourceDescriptor` directly was rejected, because the helper then could not be reused for
  `EstimateCostRequest.attributes`.

## R6: TypeScript

- **Decision**: Regenerate the bindings, and add a client round-trip test. Add no read accessor. Add a
  `ResourceDescriptorBuilder.withAttributes(attributes: JsonObject)` setter.
- **Rationale**: protobuf-es v2 generates `attributes?: JsonObject`, as it already does for
  `EstimateCostRequest.attributes`. A plain object is indexable natively, so a read accessor adds nothing.
  The setter is different: the builder is the hand-written way to build a descriptor, and without the setter
  its users cannot set the field at all (Constitution XIII). It copies the input with `structuredClone`, so
  later edits by the caller do not leak into built descriptors. It does not validate, which matches
  `withTags`; the Go validators enforce the size limit.

## R7: Conformance

- **Decision**: Add a Basic-level `RPCCorrectness_GetProjectedCostWithAttributes` test. It sends a
  `GetProjectedCost` request whose descriptor carries a ten-segment nested `attributes` (CronJob depth) plus
  `tags`, and it validates the response with the existing `ValidateProjectedCostResponse`. Add contract-suite
  cases for the attributes bound.
- **Rationale**: The mock ignores attributes, so it proves that a plugin ignoring the field passes (FR-011).
  Basic level means every plugin is exercised.

## R8: Logging and redaction

- **Finding**: No SDK code logs a request descriptor. `cli.go` marshals only dry-run responses. Redaction is
  therefore a host obligation, plus a "do not log verbatim" rule for plugin authors, documented next to the
  per-request credential guidance in `PLUGIN_DEVELOPER_GUIDE.md`.

## R9: Documentation inventory (FR-013 to FR-015)

These non-historical files mention `ResourceDescriptor` or the tag limits and must be audited:
`README.md`, `PLUGIN_DEVELOPER_GUIDE.md`, `ROADMAP.md`, `docs/PROPERTY_MAPPING.md`, `docs/ADVANCED_PATTERNS.md`,
`docs/plugin-registry-specification.md`, `docs/usage-source.md`, `docs/allocator.md`,
`sdk/go/pluginsdk/README.md`, `sdk/go/testing/README.md`, `sdk/go/pricing/README.md`,
`sdk/go/registry/README.md`, `sdk/typescript/README.md`, and the root and SDK `CLAUDE.md` files. Known stale
statements:

- `docs/PROPERTY_MAPPING.md:82-86`
- the `PLUGIN_DEVELOPER_GUIDE.md` sample validator (`len(value) > 256`)
- the `batch.go` doc comments (`MaxTagValueLength (256 characters)`)
