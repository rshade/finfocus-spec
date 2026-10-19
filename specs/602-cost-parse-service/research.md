# Research: Cost Parse Service

**Date**: 2026-10-18
**Status**: Complete

## 1. Capability discovery and constitution XII

**Decision**: Add `CostParseProvider` to automatic inference, appending capability 19. Keep the four inherited
pricing capabilities in the default inference result. A parser-only plugin explicitly uses
`WithCapabilities(PLUGIN_CAPABILITY_COST_PARSE)` to replace that list, as scorer-only and usage-only plugins do.
Extend the existing startup diagnostic to parsers without an explicit override.

**Rationale**: `sdk/go/pluginsdk/plugin_info.go` already infers eleven optional interfaces, including scoring.
Constitution XII requires automatic interface discovery and expressly permits explicit overrides for advanced cases.
The source design's statement that inference adds only base pricing capabilities is inaccurate for this codebase;
its parser-only deployment goal is preserved without a constitutional exception.

**Alternatives considered**: Manual declaration for every parser violates XII. Removing base pricing inference
would alter unrelated plugin discovery and cannot reliably distinguish inherited stubs from implemented pricing.

## 2. Canonical contract and warnings

**Decision**: Add `proto/finfocus/v1/parse.proto`, importing the existing descriptor and well-known Struct/Timestamp
messages. Preserve the source design's message names and field numbers. Define feature-local severity values
UNSPECIFIED=0, INFO=1, WARNING=2; emitted warnings require a recognized nonzero severity and non-empty message/code.
Use `PLUGIN_CAPABILITY_COST_PARSE = 19` and legacy `supports_cost_parse`.

**Rationale**: `scoring.proto` and `allocation.proto` demonstrate separate optional services. Existing capability
18 is scoring. Informational diagnostics and warnings suffice for nonfatal parsing outcomes; fatal failures use
service errors. No existing message or SourceFormat enum needs modification except the additive capability value.

**Alternatives considered**: Reusing registry validation severity couples unrelated domains. Adding error-level
warnings blurs fatal and partial outcomes. Reserving never-used future fields has no compatibility benefit.

## 3. Strict input exclusivity requires decoding-boundary enforcement

**Decision**: Preserve the input oneof. Typed validators reject absent, nil, or empty selected input. Also add
parse-specific pre-decode checks for a binary payload containing both top-level tags 2 and 3, and JSON objects
containing both input members. This is required to honor the design's no-silent-ignore guarantee.

**Rationale**: Generated oneof setters and normal binary decoding keep a single arm; after decoding, evidence
of both inputs is lost. A unary interceptor is too late. Use the installed gRPC codec extension and Connect codec
extension, delegate ordinary serialization, and inspect only ParseRequest. Use existing protobuf protowire support
for tag scanning, including unknown-field skipping. Do not register a process-wide replacement codec.

**Alternatives considered**: Testing both members on a typed request is impossible. Accepting binary last-wins
semantics weakens the design. Replacing oneof with independent fields changes the accepted wire proposal.

**Validation obligation**: Use encoding.GetCodecV2("proto") with grpc.ForceServerCodecV2 for the server-local
wrapper; delegate all non-ParseRequest messages unchanged. gRPC wraps codec errors as Internal, so after a successful
stock decode of a conflicting payload, clear req.Input and return nil. The parser adapter rejects missing input as
InvalidArgument before calling the provider. Apply the same conflict-to-invalid-request transformation to Connect
binary decoding; its normal JSON decoder rejects conflicting oneof members already. No side tables or generated
edits are required. Test both field orders, explicit empty root, and repeated same-arm tags. Independently malformed
protobuf stays a normal codec error. The conflict path must identify `input` in the application error.

## 4. Limits and validation ownership

**Decision**: Own ParseLimits, metadata formatting/parsing, and validators in `sdk/go/testing`; expose pluginsdk
aliases/delegates. ParseLimits has Formats []string, InputModes []string, MaxInputBytes int64. Require all three
metadata entries together, nonempty unique lists, valid formats, only content/path modes, and positive decimal
MaxInputBytes within int64. No metadata returns nil for non-parser plugins; a declared parser requires full metadata.

**Rationale**: `testing/scorer_limits.go` and `pluginsdk/scorer.go` avoid an import cycle and defer bad configured
metadata to PluginInfo.Validate. Request validation uses proto.Size(options) <= 65536 and overflow-safe subtraction
when totaling content bytes. Path mode byte counting and symlink containment belong to the plugin performing reads;
shared validation must never traverse host paths. Unknown option keys are also a provider responsibility, since formats
are open-ended; shared validation enforces structure and size only.

**Alternatives considered**: A global built-in option allowlist excludes future formats. Having validators open paths
introduces filesystem access into a transport contract check. Using full protobuf request size charges file names
and encoding overhead as content and contradicts the advertised byte definition.

## 5. Paths, unknown values, and descriptor reuse

**Decision**: File-set paths use portable slash-separated relative names. Reject empty paths, absolute paths,
Windows drive/UNC forms, backslashes, colons, NUL bytes, empty or dot segments, parent segments, and duplicate names.
Root input can be
absolute and is not subject to file-set path restrictions. Reuse testing.ValidateResourceDescriptor for every result.
Unknown paths are unique dot-separated nonempty attribute-key segments rooted at resource.attributes, preserve casing,
and cannot name a present value, including null. A present non-object ancestor also makes a nested unknown invalid.
Dots in attribute keys and array indexing are outside the initial unknown-path convention; plugins report an unresolved
containing property instead. No Recommendation reflection lookup is used for attribute keys.

**Rationale**: `testing/scoring.go` omitted_fields resolves actual Recommendation fields; dynamic Struct attributes
need their own traversal. Existing descriptor validators retain the attributes and tag limits from 596.

**Alternatives considered**: Reusing Recommendation path validation rejects vendor properties. Cleaning unsafe paths
before validation hides traversal and creates aliases. Literal dotted map keys need an escaping convention not in scope.

## 6. Service registration, errors, and generated stubs

**Decision**: Extend optionalServices with a parser field, add unexported gRPC/Connect adapters, and register parsing
and its Connect health name only when the provider interface is present. Preserve provider status/details through
handlerStatus/toConnectError; Unimplemented follows the same unsupported-service outcome as a missing provider.
Do not probe handlers at startup to determine whether an embedded generated method is real.

**Rationale**: `pluginsdk/sdk.go` centralizes service collection and registration. Method-set checks cannot distinguish
a promoted generated stub from a real handler. Health advertises registration, not a guarantee that every operation
succeeds; clients must handle Unimplemented and avoid treating a failed call as successful parsing.

**Alternatives considered**: Reflection by function name is brittle. Startup parsing has side effects and no universally
valid request. Removing a service dynamically after one Unimplemented response is unsafe under concurrency.

## 7. TypeScript and REST

**Decision**: Follow RecommendationScorerClient with CostParseClient.parse(request), ClientConfig, and generated service
metadata. Add exported ParseLimits metadata helpers with bigint MaxInputBytes for int64 parity. Add optional
costParseClient to middleware RESTGatewayConfig and register CostParseService. Preserve existing fromJson/toJson dispatch.

**Rationale**: `client/src/clients/recommendation-scorer.ts` already handles injected and default Connect transports.
`middleware/src/gateway.ts` currently registers only three services, so client generation alone cannot expose REST.
Its existing 1 MiB body cap and the servers' receive limits remain transport ceilings independent of advertised raw
input bytes; document base64 and message overhead. No global body-limit increase is part of this feature.

**Alternatives considered**: Plain JSON.stringify of message objects corrupts bytes and Timestamp representations.
Silently widening existing gateway limits changes other services. Number-only parsing loses int64 precision.

## 8. Mock, fixtures, performance, and generation

**Decision**: Provide separate MockCostParser, CostParseHarness, and RunCostParseConformance using the shared bufconn
harness pattern. Conformance accepts a valid caller-supplied request/fixture for its advertised format and a neutral
sample descriptor; it does not assume arbitrary vendor input is accepted. Use fixed synthetic fixtures for mock tests.
Missing unsupported modes are skipped explicitly when both modes are advertised. Compare resources/warnings across
100 calls, ignoring advisory expiration. Unit tests cover invalid response structures and codec conflicts.

**Rationale**: `testing/scorer_conformance.go` already tests wire behavior via bufconn. Existing DefaultSampleResource
is AWS-specific; feature 598 offers WithSampleResource rather than a universal neutral fixture. Native format logic
is out of scope, so conformance must accept caller-owned valid inputs.

**Alternatives considered**: Changing MockPlugin expands unrelated capabilities. Mandating a universal template forces
every parser to support a test-only format. Asserting vendor resource contents exceeds structural conformance scope.

**Decision**: Generate both SDKs using root make generate and the pinned Connect v1.21.0 generator. Update capability
bounds, preallocation count, legacy maps, registry capabilities, and both manifest/registry schema enums together.
Benchmark new validation and compare existing validator binaries against main with repeated measurements.

**Rationale**: buf.gen.yaml already generates Go, gRPC, Connect, and TypeScript. The TypeScript-local buf config is
stale. CI benchmark alerts are informational and noisy nanosecond measurements require A/B binaries.

**Alternatives considered**: Hand-editing generated outputs violates the project rules. One-shot benchmark comparisons
cannot establish a regression. New dependencies are unnecessary.
