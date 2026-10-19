# Tasks: Cost Parse Service

**Input**: Design documents from `specs/602-cost-parse-service/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/parse.proto](contracts/parse.proto),
[contracts/sdk-and-errors.md](contracts/sdk-and-errors.md), [quickstart.md](quickstart.md)

**Status**: Complete. All 57 tasks implemented and validated on 2026-10-18; see quickstart.md for actual evidence.

**Tests**: Required by FR-019, the acceptance scenarios, the implementation plan and constitution V.
Write each phase's tests first and observe failure before implementing that phase. Missing new generated symbols are
an acceptable initial failure before protocol generation; rerun afterward to distinguish behavioral failures.
Use external Go test packages and table-driven cases, with TestCostParse prefixes for new acceptance tests.

**Organization**: Shared protocol and validation prerequisites precede four story phases in specification priority
order. Each story has its own acceptance checkpoint. Story independence means independently testable on the shared
foundation and stated predecessors; it does not imply that shared transport files can be edited concurrently.

## Format: `[ID] [P?] [Story] Description`

Every actionable task has an unchecked checkbox, sequential ID, optional [P], story label in story phases, and an
exact repository-relative file path. [P] tasks may run together only within the explicitly identified ready batch.
No existing dependencies need upgrading; no new library, database or vendor parser is required.

## Path Conventions

Protocol sources live in proto/finfocus/v1, Go implementation in sdk/go/pluginsdk and sdk/go/testing, TypeScript in
sdk/typescript/packages/client and sdk/typescript/packages/middleware, and schemas in schemas. References below are
repository-relative. New Go/proto sources require Apache headers and exported symbols require documentation.
Generated files are changed only through root make generate. Leave update-agent-context.sh unrun per repository rules.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish a reproducible toolchain and performance baseline without implementation changes.

- [X] T001 Check Go 1.27.1, Node/npm and repository-managed Buf against go.mod, package.json and mise.toml;
  verify buf.gen.yaml keeps buf.build/connectrpc/go:v1.21.0, and record tool versions and baseline revision in
  specs/602-cost-parse-service/quickstart.md without updating dependencies.
- [X] T002 Capture prebuilt main baseline test binaries for sdk/go/testing/contract_test.go and
  sdk/go/pluginsdk/plugin_info_benchmark_test.go using an isolated main checkout with the same toolchain;
  keep binaries under /tmp/finfocus-cost-parse-baseline and record reproduction commands and benchmark names in
  specs/602-cost-parse-service/quickstart.md; preserve unrelated workspace changes.

**Checkpoint**: Tool versions and baseline measurements are reproducible before any source edit.

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the canonical additive protocol and shared validated entities used by all stories.
No story implementation starts before T010. Shared entities belong here because US1, US2 and US3 all consume them.

### Tests first

- [X] T003 Write failing request-to-response protocol/conformance tests in sdk/go/testing/parse_contract_test.go
  for CostParseService.Parse, both valid input arms, source-linked descriptors, warnings, timestamp round-trip and
  additive capability 19; assert proposed message field numbers from specs/602-cost-parse-service/contracts/parse.proto
  and record the initial failure before adding canonical definitions.

### Canonical contract and generated entities

- [X] T004 Add proto/finfocus/v1/parse.proto from the proposed contract with inline comments and Apache header,
  and append PLUGIN_CAPABILITY_COST_PARSE = 19 in proto/finfocus/v1/enums.proto; preserve all existing field numbers,
  SourceFormat and message definitions; use "INFO=1 or WARNING=2; UNSPECIFIED=0 and unknown numbers invalid" for severity.
- [X] T005 Run root make generate using buf.gen.yaml to produce Go, gRPC, Connect and TypeScript bindings;
  inspect sdk/go/proto/finfocus/v1/parse.pb.go and
  sdk/typescript/packages/client/src/generated/finfocus/v1/parse_pb.ts plus generated service adapters;
  reject Connect v2 or unrelated rewrites and rerun T003 without editing generated code by hand.
- [X] T006 [P] Write failing shared request tests in sdk/go/testing/parse_request_test.go for the exact model rules:
  format "Required, 1–32 ASCII bytes, `[a-z][a-z0-9_-]*`, advertised"; files "Exclusive with root_path, nonnil,
  nonempty collection"; root "Exclusive with files, nonempty; may be absolute"; options "Encoded size <=65,536;
  provider checks format keys"; file entries "No nil entries; paths unique"; content "May be empty; contributes to
  total input bytes"; paths "Portable slash-relative name with safe nonempty segments". Include 1/32/33-byte formats,
  typed-nil arms, unsafe paths, exact/over limits, overflow-safe totals, absent configuration and no filesystem I/O.
- [X] T007 [P] Write failing response tests in sdk/go/testing/parse_response_test.go for resource "Required, passes
  existing testing.ValidateResourceDescriptor", source_id "Nonempty; unique across resources in one response",
  unknown paths "Unique, well-formed dynamic attribute paths with absent values", severity "INFO=1 or WARNING=2;
  UNSPECIFIED=0 and unknown numbers invalid", code "Stable and nonempty; example resource_skipped", message
  "Nonempty human-readable explanation", warning source "Required for resource_skipped; optional for document-wide
  diagnostics", both collections "No nil entries; may be empty", and expiry "Valid timestamp; advisory, may already
  be expired". Include nil request/response, present/null leaves, scalar/list/null ancestors, casing and empty results.

### Shared validation implementation

- [X] T008 Define ParseLimits with Formats []string, InputModes []string and MaxInputBytes int64 in
  sdk/go/testing/parse_limits.go and sentinel/status-carrying validation errors in sdk/go/testing/parse.go;
  use newInvalidArgument for invalid requests and errors.Is-compatible ErrInvalidParseLimits,
  ErrInvalidParseRequest and ErrInvalidParseResponse without importing pluginsdk.
- [X] T009 Implement ValidateParseRequest in sdk/go/testing/parse.go to satisfy T006: validate syntax, input,
  portable content paths and proto.Size(options) before advertised format, then mode and raw content bytes;
  reject absolute/UNC/drive paths, backslashes, empty/dot/parent segments and duplicates without cleaning;
  sum bytes without overflow, never open root_path, return the specified categories and ErrorInfo reasons/domain,
  and leave unknown option keys and consumed path bytes to providers.
- [X] T010 Implement ValidateParseResponse in sdk/go/testing/parse.go to satisfy T007, reusing
  ValidateResourceDescriptor and Struct traversal relative to attributes. Missing ancestors permit an unknown leaf;
  present leaves including null and non-object ancestors fail; list indexing and escaped dotted keys are unsupported.
  Reject duplicate/empty paths, invalid warnings, nil entries and invalid timestamps; preserve input/response objects
  and existing descriptor hot paths; require source_id for resource_skipped while permitting whole-document warnings.

**Checkpoint**: Shared valid/invalid message tests pass; protobuf generation remains additive. Other stories can reuse
validated entities without implementing their own descriptor or metadata types.

## Phase 3: User Story 1 - Discover a parser and obtain priceable resources (Priority: P1)

**Goal**: Discover a mixed or parser-only plugin and parse valid content/root input into directly priceable resources.

**Independent Test**: Start a neutral fixture plugin through injected listeners in gRPC and Connect modes, discover its
limits/capability, parse valid fixtures in each advertised mode, and pass descriptors to the existing pricing mock
without ResolveResourceTypes. Missing and generated-stub providers return Unimplemented.

### Tests first

- [X] T011 [P] [US1] Write failing advertisement tests in sdk/go/testing/parse_limits_test.go for parse_formats
  "Nonempty unique names, each matching format syntax", parse_input_modes "Nonempty unique subset of the two modes",
  and parse_max_input_bytes "1 through 9223372036854775807". Assert all three keys together, nil/nil for no metadata,
  canonical lowercase/no-space names and unsigned decimal representation, and rejection of missing keys, empty entries,
  duplicates, whitespace, signs, nondecimal values, zero and overflow; test formatting/parsing round-trips.
- [X] T012 [P] [US1] Write failing inference/override/metadata tests in sdk/go/pluginsdk/parse_info_test.go
  and cost_parse schema/bounds tests in sdk/go/registry/schema_test.go; require valid complete parsing limits when
  COST_PARSE is advertised, auto-inference for mixed plugins, explicit parser-only override, legacy name compatibility
  and unchanged existing capability sets.
- [X] T013 [P] [US1] Write failing Serve acceptance tests in sdk/go/pluginsdk/parse_serve_test.go for both transports,
  injected listeners, valid content/root parsing, parser health registration, absent provider and embedded Unimplemented
  stub outcomes, invalid-success response handling and unchanged Serve signature; use local fixture providers.
- [X] T014 [P] [US1] Write failing neutral mock tests in sdk/go/testing/parse_mock_test.go for deterministic valid
  content/path fixtures and valid source-linked descriptors usable by the pricing mock with zero type-resolution calls;
  assert existing MockPlugin capabilities remain unchanged and caller-supplied resources are cloned.

### Implementation

- [X] T015 [US1] Implement FormatParseLimits, ParseParseLimits, metadata key/domain constants and canonical validation
  in sdk/go/testing/parse_limits.go to pass T011; formatting records supplied configuration without silently repairing it,
  and advertisement remains authoritative because ParseResponse has no duplicate limits fields.
- [X] T016 [US1] Add pluginsdk ParseLimits alias, ParseParseLimits delegate, WithParseLimits option and both validator
  delegates with godoc in sdk/go/pluginsdk/parse.go; wire PluginInfo.Validate to validate metadata whenever present
  and require it for declared COST_PARSE in sdk/go/pluginsdk/plugin_info.go; pass T012 metadata cases.
- [X] T017 [US1] Define CostParseProvider in sdk/go/pluginsdk/sdk.go and unexported gRPC/Connect adapters with shared
  dispatch in sdk/go/pluginsdk/parse.go; validate configured requests before invocation, normalize handler errors,
  validate successful responses, convert invalid/nil provider output to Internal, and preserve Unimplemented.
- [X] T018 [US1] Extend optionalServices.parser, newOptionalServices, gRPC/Connect registration and Connect health
  names in sdk/go/pluginsdk/sdk.go; register only interface-present providers, propagate validated limits to adapters,
  and retain startup behavior for existing plugins. Treat health as registration and never probe Parse at startup.
- [X] T019 [US1] Append parser inference, raise optionalCapabilities from 11 to 12 and maxValidCapability to COST_PARSE
  in sdk/go/pluginsdk/plugin_info.go; add supports_cost_parse in sdk/go/pluginsdk/capability_compat.go and parser-only
  startup guidance in sdk/go/pluginsdk/usage_source.go; preserve base-pricing inference and explicit overrides.
- [X] T020 [US1] Add cost_parse to schemas/plugin_manifest.schema.json and schemas/plugin_registry.schema.json;
  update affected capability bounds/legacy assertions in sdk/go/pluginsdk/capabilities_test.go and
  sdk/go/pluginsdk/capability_compat_test.go and verify registry.AllPluginCapabilities/schema drift without unrelated
  schema or SourceFormat changes.
- [X] T021 [US1] Implement separate MockCostParser in sdk/go/testing/parse_mock.go with an advertised synthetic format,
  explicit limits, fixed valid content/path rules and configurable neutral descriptor; clone fixture state, apply shared
  validators and keep filesystem operations within caller-owned temporary roots; do not alter MockPlugin.
- [X] T022 [US1] Run the US1 tests in sdk/go/pluginsdk/parse_serve_test.go, sdk/go/pluginsdk/parse_info_test.go,
  sdk/go/testing/parse_mock_test.go and sdk/go/registry/schema_test.go; verify nonzero TestCostParse selections and
  record the discovery-to-pricing demonstration and registration-only health meaning in
  specs/602-cost-parse-service/quickstart.md.

**Checkpoint**: A working discovery-to-resource demonstration is available. This is the first MVP increment;
production release also requires US2's strict decoding and negative transport checks.

## Phase 4: User Story 2 - Diagnose invalid or unsupported input (Priority: P1)

**Goal**: Ensure malformed/unsupported input never disappears silently and diagnostics carry predictable categories.

**Independent Test**: Send invalid requests over both transports, including raw payloads with both input fields,
and verify categories, fields, ErrorInfo and zero provider calls for shared-validation failures. Exercise provider-owned
option and path-read limits using the mock, without implying generic validators perform filesystem I/O.

### Tests first

- [X] T023 [P] [US2] Write failing cross-transport errors tests in sdk/go/pluginsdk/parse_errors_test.go for
  InvalidArgument field/index and file/known-line diagnostics, unsupported format/mode FailedPrecondition with
  UNSUPPORTED_FORMAT or UNSUPPORTED_INPUT_MODE and domain finfocus.v1.CostParseService, ResourceExhausted byte limits,
  known status/details preservation, sanitized ordinary errors, nil successes and Unimplemented fallback.
- [X] T024 [P] [US2] Write failing raw decode tests in sdk/go/pluginsdk/parse_codec_test.go for both input tags in both
  orders, explicit empty root, JSON conflicts, repeated same-arm fields, unknown fields and normal malformed protobuf;
  assert conflicts become InvalidArgument naming input before provider invocation in gRPC and Connect binary/JSON,
  and non-ParseRequest messages and other RPCs retain ordinary codec behavior.
- [X] T025 [P] [US2] Write failing provider-policy tests in sdk/go/testing/parse_mock_policy_test.go for unknown option
  keys, format diagnostics, exact/over content and consumed-file limits, escaping symlinks and multiple input files;
  use controlled temporary roots and small limits, and distinguish mock enforcement from universal SDK enforcement.

### Implementation

- [X] T026 [US2] Add parse-request-only binary scanning in sdk/go/pluginsdk/parse_codec.go using protowire;
  delegate the stock decode, clear Input after a successfully decoded conflicting-tag request so application validation
  returns InvalidArgument, preserve unknown fields and repeated same-arm semantics, and delegate other messages unchanged.
  Avoid global codec registration, generated edits, request side tables and direct semantic codec errors that gRPC maps
  to Internal.
- [X] T027 [US2] Integrate the server-local encoding.GetCodecV2("proto") wrapper via grpc.ForceServerCodecV2 and
  parse-handler-only Connect binary codec override in sdk/go/pluginsdk/sdk.go and sdk/go/pluginsdk/parse.go;
  retain normal JSON oneof rejection and install no parser codec when there is no parser service; pass T024.
- [X] T028 [US2] Complete cross-transport handlerStatus/toConnectError normalization and error/detail preservation
  in sdk/go/pluginsdk/parse.go to satisfy T023, including consistent unsupported-service outcomes and sanitized Internal
  messages; retain request validation precedence from contracts/sdk-and-errors.md.
- [X] T029 [US2] Implement synthetic provider option-key rejection, file/known-line errors and bounded path reads in
  sdk/go/testing/parse_mock.go to satisfy T025; count all consumed file/dependency bytes, stop before crossing the limit,
  reject root-escaping symlinks and return ResourceExhausted. Keep generic validators free of file I/O and vendor logic.
- [X] T030 [US2] Create the input/error guidance in docs/cost-parse.md with cfn parameters, helm release_name/namespace/
  values and no kustomize keys, plugin-owned allowlists, field/location diagnostics, strict oneof rejection, portable file
  paths, path containment, raw-byte accounting and transport receive ceilings; state Terraform/Pulumi remain core-native.
- [X] T031 [US2] Run sdk/go/pluginsdk/parse_errors_test.go, sdk/go/pluginsdk/parse_codec_test.go and
  sdk/go/testing/parse_mock_policy_test.go through focused and race-enabled test commands; record actual category/detail,
  provider-call and root-containment outcomes in specs/602-cost-parse-service/quickstart.md.

**Checkpoint**: Both P1 stories pass. Strict conflicting-input handling is proven on real wire decoders, and
provider-owned policy checks have explicit responsibility boundaries.

## Phase 5: User Story 3 - Trace results and understand incomplete resources (Priority: P2)

**Goal**: Demonstrate source-linked partial results and distinguish unresolved properties from absent/known values.

**Independent Test**: Parse a synthetic fixture with a resolved resource, nested unresolved value and skipped resource;
validate descriptors, identities and separate warnings without asserting vendor-format resource contents.

### Tests first

- [X] T032 [P] [US3] Write failing partial-result integration tests in sdk/go/pluginsdk/parse_partial_test.go for
  nested absent unknown values, known zero/empty values, warning severity/code/message/source, empty successful results,
  skipped sources absent from resources, and invalid provider output becoming Internal on both transports.
- [X] T033 [P] [US3] Write failing partial fixture tests in sdk/go/testing/parse_mock_partial_test.go for logical/native
  source identity preservation, duplicate-source rejection, stable resource_skipped codes, whole-document diagnostics,
  unknown-path casing and immutability of caller fixture descriptors; reuse shared validation cases from T007.

### Implementation

- [X] T034 [US3] Extend fixed MockCostParser fixture rules in sdk/go/testing/parse_mock.go to emit partial resources,
  unique input-derived source identities, nested unknown paths with no corresponding attribute value and separate
  source-linked warnings; allow all-skipped/empty results and document the synthetic fixture convention without adding
  any CloudFormation, Helm or Kustomize parsing implementation.
- [X] T035 [US3] Add the partial-result walkthrough to docs/cost-parse.md: native logical IDs or Kind/name, priceable
  Pulumi type token, plugin pricing property casing, dotted attribute paths, zero/absent/unknown differences, stable
  warning matching and source-linked skipped resources; explain unsupported dotted-key escaping/list indexing.
- [X] T036 [US3] Add response validation benchmarks in sdk/go/testing/parse_response_benchmark_test.go for valid,
  rejected, nested-unknown, warning-heavy and descriptor-near-limit cases with allocation reporting; ensure descriptor
  validation is reused and existing validation call sites/hot paths are unchanged.
- [X] T037 [US3] Run sdk/go/pluginsdk/parse_partial_test.go, sdk/go/testing/parse_mock_partial_test.go and shared
  sdk/go/testing/parse_response_test.go; verify response identity/attributes are unmodified and record the actionable
  partial-result demonstration in specs/602-cost-parse-service/quickstart.md.

**Checkpoint**: Users can trace every usable or skipped source and distinguish unresolved values without reading
warning messages as machine codes.

## Phase 6: User Story 4 - Reuse results and integrate consistently (Priority: P2)

**Goal**: Provide portable structural conformance, deterministic results, freshness guidance, and TypeScript/REST parity.

**Independent Test**: Run configurable neutral conformance fixtures, compare 100 identical calls excluding advisory
expiration, and round-trip binary bytes and nanosecond timestamps through generated Go/Connect, TypeScript and REST.

### Tests first

- [X] T038 [P] [US4] Write failing harness/conformance tests in sdk/go/testing/parse_conformance_test.go for
  configurable format/mode fixtures, cloned neutral sample resources, all eight named scenarios in the SDK contract,
  explicit mode/content-fixture skips, bounded probes, advertised metadata cross-checks and 10-second call deadlines;
  test malformed response rejection separately without requiring real parsers to emit synthetic unknowns/skips.
- [X] T039 [P] [US4] Write failing client/limits tests in sdk/typescript/packages/client/test/cost-parse.test.ts
  for default and injected transports, both oneof arms, byte preservation, subsecond Timestamp, Connect ErrorInfo,
  the exact Go metadata validation rules, and bigint precision through 9223372036854775807.
- [X] T040 [P] [US4] Write failing REST route tests in sdk/typescript/packages/middleware/test/cost-parse.test.ts
  for optional costParseClient, service routing, absent-client 404, base64 bytes AP9B, RFC 3339 expiry, JSON input conflicts,
  current HTTP error sanitization/category mapping and the independent 1 MiB body limit.
- [X] T041 [P] [US4] Write failing freshness/determinism tests in sdk/go/testing/parse_freshness_test.go for absent,
  valid and already-expired advisory timestamps; stable resources/warnings and ordering across 100 calls with unchanged
  dependencies; and permissible expiry advancement while semantic results stay equal.

### Implementation

- [X] T042 [US4] Implement ParseServer, CostParseHarness and RunCostParseConformance in
  sdk/go/testing/parse_conformance.go using the shared bufconn infrastructure; accept explicit limits and caller-owned
  valid mode fixtures and optional served metadata. Implement format_rejected, input_mode_xor, unsupported_input_mode,
  path_traversal_rejected, source_ids_unique, unknown_not_absent, descriptors_valid and deterministic; compare 100 calls
  excluding expiration, use 10-second deadlines, and keep raw conflicting-tag tests in SDK transport tests.
- [X] T043 [US4] Implement CostParseClient.parse and parseParseLimits in
  sdk/typescript/packages/client/src/clients/cost-parse.ts using generated service metadata, ClientConfig and existing
  Connect transport conventions; use bigint for maxInputBytes and reject exactly the same malformed metadata as Go.
- [X] T044 [US4] Export CostParseClient, limits helpers, service, message types/schemas and severity from
  sdk/typescript/packages/client/src/index.ts; extend sdk/typescript/packages/client/test/index-exports.test.ts to verify
  the public surface and preserve existing exports.
- [X] T045 [US4] Add optional costParseClient and CostParseService registration in
  sdk/typescript/packages/middleware/src/gateway.ts; preserve fromJson/toJson, existing body limits and error sanitization,
  return existing 404 behavior when unconfigured, and do not claim REST ErrorInfo propagation absent from its envelope.
- [X] T046 [US4] Add controlled expiration/dependency fixture behavior to sdk/go/testing/parse_mock.go to satisfy T041
  and conformance determinism; expose advisory expiration only for the changing-dependency fixture and preserve stable
  resource/warning ordering and immutable fixtures for pure-input results.
- [X] T047 [US4] Document runnable configurable RunCostParseConformance usage in sdk/go/testing/README.md and
  caching/conflict-selection rules in docs/cost-parse.md: explicit user choice, otherwise unique provider match,
  otherwise ambiguity failure; cache keys include input content, format, options and parser identity, not root text alone;
  no parser-output merging or host selection implementation is added.
- [X] T048 [US4] Add request/metadata benchmarks in sdk/go/testing/parse_request_benchmark_test.go and binary guard
  benchmarks in sdk/go/pluginsdk/parse_codec_benchmark_test.go for normal/rejected/near-limit cases and unchanged
  non-ParseRequest delegation; report allocation costs and avoid modifying existing validator fast paths.
- [X] T049 [US4] Run sdk/go/testing/parse_conformance_test.go and sdk/go/testing/parse_freshness_test.go plus
  TypeScript workspace build/test/lint from sdk/typescript/package.json; require nonzero new client/gateway tests,
  verify all sample bytes/timestamps and error details, and update specs/602-cost-parse-service/quickstart.md with evidence.

**Checkpoint**: All four stories work through supported clients and transports, conformance remains format-neutral,
and freshness and host selection guidance are explicit.

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Complete documentation, compatibility, performance evidence and full repository validation.

- [X] T050 [P] Update README.md, sdk/go/README.md and sdk/go/pluginsdk/README.md with optional parsing discovery,
  provider implementation, WithParseLimits, automatic capability inference and parser-only override examples; link
  docs/cost-parse.md and preserve FinFocus identity and the core-native ingestion boundary.
- [X] T051 [P] Update sdk/typescript/README.md with CostParseClient imports, both input modes, bigint limits helpers,
  optional REST configuration and protobuf JSON byte/timestamp mapping; keep examples consistent with generated types
  and describe independent transport caps without changing them.
- [X] T052 Run new documentation examples against exported symbols and verify links, proto comments, Go godoc and
  Apache headers in sdk/go/pluginsdk/parse.go, sdk/go/testing/parse_conformance.go and proto/finfocus/v1/parse.proto;
  use existing documentation checks for 80%+ godoc coverage and validate sdk/go/testing/README.md conformance examples.
- [X] T053 Regenerate using buf.gen.yaml and run make buf-lint, repository-managed Buf breaking against main and
  git diff --check; verify only intended generated SDK/capability additions, synchronized manifest/registry schemas,
  unchanged SourceFormat/type-resolution behavior, and no manually edited sdk/go/proto files.
- [X] T054 Compare prebuilt candidate and T002 main baseline binaries repeatedly with the same toolchain for
  sdk/go/testing/contract_test.go and sdk/go/pluginsdk/plugin_info_benchmark_test.go; run new request/response/codec
  benchmarks with benchmem and at least ten repetitions and record distributions and any repeatable regressions in
  specs/602-cost-parse-service/quickstart.md; retain informational CI benchmark policy.
- [X] T055 Run go test -race ./sdk/go/pluginsdk ./sdk/go/testing, all TypeScript workspace build/test/lint and
  make validate using Makefile and sdk/typescript/package.json; confirm schema/examples and TestSchemaDrift coverage.
  Record actual results and any incomplete aggregate timeout in specs/602-cost-parse-service/quickstart.md;
  use constituent commands if needed without claiming a timed-out aggregate passed.
- [X] T056 Execute every supported scenario in specs/602-cost-parse-service/quickstart.md, verify byte-limit boundaries,
  100-call equality, zero extra type-resolution steps and documented discovery/diagnosis/selection/cache examples;
  replace its forthcoming-implementation status only when the commands and planned symbols actually work.
- [X] T057 Reconcile completed work and evidence against all FR/SC entries in specs/602-cost-parse-service/spec.md,
  re-evaluate constitution gates in specs/602-cost-parse-service/plan.md and update the existing
  specs/602-cost-parse-service/checklists/requirements.md status/notes; leave unfinished tasks unchecked and do not
  change constitutional requirements, infer completed tests, or invoke update-agent-context.sh.

**Checkpoint**: Implementation is reviewable only after actual validation evidence, compatibility and documentation
checks are recorded. Task generation itself does not assert these implementation checks have run.

## Dependencies & Execution Order

### Phase dependencies

```text
Setup T001–T002
  -> Protocol T003 -> T004 -> T005
  -> Shared validation tests T006 || T007
  -> Shared implementation T008 -> T009 -> T010
  -> US1 tests T011 || T012 || T013 || T014
  -> US1 implementation T015 -> T016 -> T017 -> T018 -> T019 -> T020 -> T021 -> T022
  -> US2 tests T023 || T024 || T025
  -> US2 implementation T026 -> T027 -> T028 -> T029 -> T030 -> T031
  -> US3 tests T032 || T033
  -> US3 implementation T034 -> T035 -> T036 -> T037
  -> US4 tests T038 || T039 || T040 || T041
  -> US4 implementation T042 -> T043 -> T044 -> T045 -> T046 -> T047 -> T048 -> T049
  -> Polish T050 || T051 -> T052 -> T053 -> T054 -> T055 -> T056 -> T057
```

The graph is the conservative default execution order. Each test batch must finish and demonstrate failure before its
implementation batch. T003 defines the initial protocol conformance requirement before canonical proto edits.
T006/T007 then define complete shared validation before either validator is implemented.

### User story dependencies

| Story | Required predecessors | Why | Independently observable outcome |
| --- | --- | --- | --- |
| US1 (P1) | Setup and foundation | Shared messages and validation | Discovery and valid parsing enter pricing directly |
| US2 (P1) | US1 adapters/mock | Exercises actual registration and provider boundaries | Invalid input has precise errors and never silently wins |
| US3 (P2) | US1; default order follows US2 | Shared dispatch and neutral fixtures | Partial results carry sources, unknowns and warnings |
| US4 (P2) | US1, US2 and US3 | Conformance spans all error/result semantics | Determinism, freshness and client/REST parity |
| Polish | All stories | Validates complete delivery | Compatibility, performance and documentation evidence |

US2 and US3 test authoring may proceed once US1 is complete, but their parse_mock.go implementation changes must
serialize. The default list stays sequential to avoid shared-file collisions and preserve test-first observations.
No production release may omit the remaining mandatory protocol or SDK requirements just because an MVP demo works.

### Within each story

Tests precede implementation; shared entities precede adapters; adapters precede registration; registration precedes
wire integration; each checkpoint follows actual focused checks. Existing descriptor limits and hot paths remain shared
infrastructure. File containment, consumed-byte accounting and format option keys remain provider responsibilities.

### Parallel opportunities

The [P] batches below use disjoint files and share only already-completed prerequisites. Integration commands and shared
source edits serialize. Generation, schema rewrites, benchmark comparisons and installs never run concurrently with
work that changes their inputs. Parallel examples describe optional execution and do not themselves dispatch agents.

## Parallel Example: User Story 1

After foundation T010, author and observe the four independent red test sets together:

```text
T011 sdk/go/testing/parse_limits_test.go
T012 sdk/go/pluginsdk/parse_info_test.go + sdk/go/registry/schema_test.go
T013 sdk/go/pluginsdk/parse_serve_test.go
T014 sdk/go/testing/parse_mock_test.go
```

Then implement T015–T021 in dependency order and run T022. Metadata, adapter, inference and mock edits are deliberately
not marked parallel because their public API and configuration behavior depend on earlier work.

## Parallel Example: User Story 2

After US1 T022:

```text
T023 sdk/go/pluginsdk/parse_errors_test.go
T024 sdk/go/pluginsdk/parse_codec_test.go
T025 sdk/go/testing/parse_mock_policy_test.go
```

After all demonstrate failure, implement the codec and dispatch integration sequentially. Do not try to construct both
oneof arms with typed setters: the conflicting-input test must submit raw bytes/JSON to the real decoding boundary.

## Parallel Example: User Story 3

After US1, or after T031 in the default order:

```text
T032 sdk/go/pluginsdk/parse_partial_test.go
T033 sdk/go/testing/parse_mock_partial_test.go
```

Implement the shared mock fixture changes after these tests; serialize them with US2/US4 edits to parse_mock.go.

## Parallel Example: User Story 4

After T037:

```text
T038 sdk/go/testing/parse_conformance_test.go
T039 sdk/typescript/packages/client/test/cost-parse.test.ts
T040 sdk/typescript/packages/middleware/test/cost-parse.test.ts
T041 sdk/go/testing/parse_freshness_test.go
```

After the red test batch, use the conservative implementation sequence above. Go conformance and TypeScript work can
be split only after public signatures/exports are agreed and the shared mock changes are serialized. Final root-doc
T050 and TypeScript-doc T051 tasks are also independent once every story has passed.

## Implementation Strategy

### MVP first

Complete Setup, Foundation and US1 for the first discovery-to-priceable-resource demonstration. Use the neutral mock
or a fixture provider; vendor parsers are outside scope. Run T022 independently. Add US2 before treating the P1 contract
as production-ready because it proves raw conflicting-input rejection and provider policy diagnostics. Shared response
validation is already present in the foundation; no incomplete validation bypass is introduced for the demo.

### Incremental delivery

Deliver the P1 flow, then the US2 negative cases, then source-linked partial output, then deterministic/conformant
multi-client behavior. Run each phase checkpoint and keep previous focused tests passing. Full delivery requires all
four stories and final validation; sdk parity, compatibility and documentation are not optional release polish.

### Requirement coverage

| Requirements / criteria | Principal tasks |
| --- | --- |
| FR-001, FR-021; additive parsing, scope and compatibility | T003–T005, T017–T022, T030, T050, T053 |
| FR-002; discovery, limits and capability compatibility | T011–T016, T019–T020, T039, T043 |
| FR-003–FR-009; request rules and errors | T006, T008–T009, T023–T031, T040, T045 |
| FR-010–FR-013; descriptors, identities, unknowns and warnings | T007, T010, T014, T021, T032–T037 |
| FR-014–FR-015; freshness and determinism | T038, T041–T042, T046–T047 |
| FR-016–FR-017; optional registration and transport parity | T012–T013, T017–T019, T023–T028 |
| FR-018; shared validation and existing performance | T006–T010, T036, T048, T054–T055 |
| FR-019; separate neutral mock and conformance | T014, T021, T025, T029, T033–T034, T038, T042 |
| FR-020; both SDKs, bytes and timestamps | T003–T005, T039–T045, T049 |
| FR-022; documented host/provider responsibilities | T030, T035, T047, T050–T052 |
| SC-001–SC-002; direct pricing and exact rejection outcomes | T022–T031, T056 |
| SC-003–SC-004; source coverage and unresolved values | T007, T010, T032–T037, T056 |
| SC-005–SC-006; repeatability and client preservation | T038–T049, T056 |
| SC-007–SC-008; compatibility, performance and usable guidance | T050–T057 |

## Notes

All 57 tasks have file paths; 39 carry story labels. There are 17 [P] tasks in independent test/doc batches.
No implementation code, generated bindings, dependencies or existing tests are changed by creating this task list.
Constitution XII is satisfied by inference plus its permitted advanced override; no approval or amendment is pending.
Follow the canonical design artifacts for details and keep future evidence honest: unchecked means not yet verified.

## Completion notes

All tasks are complete; validation results and FR/SC reconciliation are in quickstart.md.
T057 respected the skill's read-only checklist gate: requirements.md was verified unchanged with 16/16 checked;
reconciliation notes were recorded in quickstart.md instead. No project dependencies were added, generated
files were changed only by make generate. Publication follows the validated implementation.
