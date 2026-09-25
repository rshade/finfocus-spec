---

description: "Task list for 051-usage-source-getstats"
---

# Tasks: Usage Source Service (GetStats)

**Input**: Design documents from `specs/051-usage-source-getstats/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required. Constitution Principle V (Test-First Protocol) is non-negotiable, and the plan
orders serving, capability, validator, and TypeScript tests before implementation. In every story
phase, write the test tasks first and confirm they fail (or do not compile) before starting the
implementation tasks.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested
independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US5)
- All paths are repo-relative

## Conventions That Apply to Every Task

- Every new `.go`, `.proto`, and `.ts` file starts with the Apache 2.0 header used by existing files
  (Constitution XI). Copy it from `proto/finfocus/v1/costsource.proto` or a sibling file.
- Every exported Go symbol gets a godoc comment (Constitution XIV).
- The `sdk/go/testing` package must **not** import `sdk/go/pluginsdk` (import cycle: `pluginsdk/conformance.go`
  imports `sdk/go/testing`).
- Identifier names are fixed by #505: `UsageSourceProvider`, `ValidateStatsResponse`, `Subject*`,
  `Kind*`, `Metric*`. Do not rename them.
- Tests that call `pluginsdk.Serve` inject a `net.Listener` via `ServeConfig.Listener` rather than a
  `Port` (see CLAUDE.md `pluginsdk.Serve` guidance).
- Do not use `t.Parallel()` on subtests that share one harness or one served listener.
- Run `goimports -w <file>` on each new Go file.

---

## Phase 1: Setup (Proto Contract and Generated Code)

**Purpose**: Land the wire contract first (Constitution I) and regenerate both SDKs so every later
task compiles against real generated types.

- [ ] T001 Create `proto/finfocus/v1/usage.proto` with the content of
  `specs/051-usage-source-getstats/contracts/usage.proto`, keeping the Apache header, `syntax`, `package finfocus.v1`,
  `go_package`, both imports, `UsageSourceService`, `StatsMode`, `GetStatsRequest`, `GetStatsResponse`, and `UsageRow`
  with all inline comments exactly as written. Remove only the contract-note comment block that starts
  `// Contract for proto/finfocus/v1/usage.proto` and ends with the `PLUGIN_CAPABILITY_USAGE_STATS = 14;` example
- [ ] T002 [P] In `proto/finfocus/v1/enums.proto`, add `PLUGIN_CAPABILITY_USAGE_STATS = 14;` to `enum PluginCapability`
  immediately after `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES = 13;`, preceded by the comment
  `// Plugin implements UsageSourceService.GetStats for workload usage`
- [ ] T003 Run `make generate` from the repo root and confirm these files exist and compile with `go build ./...`:
  `sdk/go/proto/finfocus/v1/usage.pb.go`, `sdk/go/proto/finfocus/v1/usage_grpc.pb.go`,
  `sdk/go/proto/finfocus/v1/pbcconnect/usage.connect.go`, and that `sdk/go/proto/finfocus/v1/enums.pb.go` defines
  `PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS` (depends on T001, T002)
- [ ] T004 Regenerate TypeScript bindings for only the changed files:
  `cd sdk/typescript && ../../bin/buf generate ../../proto --template buf.gen.yaml` with
  `--path ../../proto/finfocus/v1/usage.proto --path ../../proto/finfocus/v1/enums.proto`.
  Confirm `sdk/typescript/packages/client/src/generated/finfocus/v1/usage_pb.ts` exports `UsageSourceService`,
  `GetStatsRequestSchema`, `GetStatsResponseSchema`, `UsageRowSchema`, and `StatsMode`, and that `enums_pb.ts` has
  `PluginCapability.USAGE_STATS`. Only `protoc-gen-es` is installed under `node_modules/.bin`; if buf fails on the
  missing `protoc-gen-connect-es` plugin, generate with a temporary template in the scratchpad that lists only
  `protoc-gen-es` (`out: packages/client/src/generated`, `opt: target=ts`), and do not commit template changes. No other
  generated files may change (depends on T003)
- [ ] T005 Run `bin/buf lint` and `bin/buf breaking --against '.git#branch=main'` and confirm both are clean (FR-007,
  SC-004) (depends on T003)

**Checkpoint**: Contract generated in Go and TypeScript; lint and breaking-change detection pass.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared Go SDK pieces that every user story's code or test fixtures reference.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T006 [P] Create `sdk/go/pluginsdk/subjects.go` with one exported `const` block, each constant godoc'd, with
  exactly these values: `SubjectCluster = "cluster"`, `SubjectNamespace = "namespace"`,
  `SubjectControllerKind = "controller_kind"`, `SubjectController = "controller"`, `SubjectPod = "pod"`,
  `SubjectNode = "node"`, `SubjectKind = "kind"`, `SubjectLabelPrefix = "label."`, `KindWorkload = "workload"`,
  `KindNode = "node"`, `KindIdle = "__idle__"` (godoc: allocator output only, #506, invalid in usage rows),
  `KindCluster = "__cluster__"` (same note), `MetricCPURequest = "cpu_request"`, `MetricMemRequest = "mem_request"`,
  `MetricCPUAllocatable = "cpu_allocatable"`, `MetricMemAllocatable = "mem_allocatable"`,
  `MetricCPUUsage = "cpu_usage"`, `MetricMemUsage = "mem_usage"`, `UnitCore = "core"`, `UnitGiB = "GiB"`,
  `UnitCoreHours = "core-hours"`, `UnitGiBHours = "GiB-hours"` (FR-014)
- [ ] T007 [P] In `sdk/go/pluginsdk/sdk.go`, directly after the `ResolveResourceTypesProvider` interface, declare the
  exported `UsageSourceProvider` interface with the single method
  `GetStats(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error)` and the godoc from
  `contracts/go-sdk-api.md` ("…Serve registers the service in gRPC and Connect modes and PLUGIN_CAPABILITY_USAGE_STATS
  is inferred. Usage-only plugins should set PluginInfo.Capabilities explicitly.")
- [ ] T046 [P] In `sdk/go/testing/contract_test.go`, add table-driven `TestValidateGetStatsRequest` (research R11,
  data-model rules Q1–Q3). Rejecting cases: `nil` → `errors.Is(err, plugintesting.ErrNilRequest)`; only `Start` set →
  `errors.Is(err, plugintesting.ErrNilEndTime)` with a `*ContractError` whose `Field` is `end`; only `End` set →
  `ErrNilStartTime` with `Field` `start`; `Start` one hour after `End` → `plugintesting.ErrInvertedStatsWindow`.
  Accepting cases (`nil` error): neither set (run-rate); `Start` before `End`; `Start` equal to `End`; a 5-minute
  window (no 1-hour minimum); empty `Scope`; an unknown name in `Metrics`; a `Selector` with `namespace` and a label key
  (FR-020, SC-003)
- [ ] T047 In `sdk/go/testing/contract.go`, add the sentinel
  `ErrInvertedStatsWindow = errors.New("start time must not be after end time")` to the existing error `var` block and
  `ValidateGetStatsRequest(req *pbc.GetStatsRequest) error` after `ValidateGetRecommendationsRequest`, with the godoc
  from `contracts/go-sdk-api.md`. Check in order: `nil` → `ErrNilRequest`; neither timestamp set → `nil`; only `Start`
  → `NewContractError("end", nil, ErrNilEndTime)`; only `End` → `NewContractError("start", nil, ErrNilStartTime)`;
  `Start.AsTime().After(End.AsTime())` → `NewContractError("time_range", …, ErrInvertedStatsWindow)`. Do not call
  `ValidateTimeRange` (it adds 1-hour, 365-day, and future-start rules the spec does not have) (depends on T046). Make
  T046 pass
- [ ] T008 Create the shared test fixture file `sdk/go/pluginsdk/usage_source_fixtures_test.go` (package `pluginsdk`)
  with: (a) `usageTestPlugin`, a struct embedding `*BasePlugin` (from `helpers.go`, built via `NewBasePlugin`) that
  implements `GetStats` by returning a configurable `*pbc.GetStatsResponse` or error; (b) `fixtureStatsResponse()`, a
  two-node cluster in `STATS_MODE_RUN_RATE` with per-workload `cpu_request` (`UnitCore`) and `mem_request` (`UnitGiB`)
  rows in namespaces `payments` and `web`, `cpu_allocatable`/`mem_allocatable` rows per node (`kind=node`,
  `node=<name>`), and one priceable `*pbc.ResourceDescriptor` per node with `Id` equal to the node name and tags
  `kind=node`, `provider_id`, `capacity_type=on-demand`, plus one entry in `Warnings` (for example
  `"mem_usage unavailable: metrics-server not installed"`) so transport tests prove warnings survive; (c)
  `referenceUsageSource`, a source that first calls `plugintesting.ValidateGetStatsRequest(req)` and returns
  `status.Error(codes.InvalidArgument, err.Error())` on failure (partial or inverted window). When both `Start` and
  `End` are set it returns `codes.InvalidArgument` (`historical mode not supported`) unless its `historical` field is
  true, in which case it returns the fixture's rows in `STATS_MODE_HISTORICAL` with every amount multiplied by the
  window's hours and units `UnitCoreHours`/`UnitGiBHours`, including the allocatable rows. It returns
  `codes.PermissionDenied` with message `cannot list pods` when configured to simulate RBAC failure and
  `codes.Unauthenticated` when configured with no credentials. Otherwise it filters workload rows by
  `req.Selector["namespace"]` when present, and appends one warning `unknown metric "<name>" ignored` for each
  `req.Metrics` entry that is not one of the six documented metric names. Use the T006 constants for all keys, kinds,
  metrics, and units (depends on T006, T007, T047)

**Checkpoint**: Vocabulary, interface, request validator, and shared fixtures exist; user stories can begin.

---

## Phase 3: User Story 1 - Host Retrieves Workload Usage from a Usage Source (Priority: P1) 🎯 MVP

**Goal**: A plugin implementing `UsageSourceProvider`, served by `pluginsdk.Serve`, answers
`GetStats` over both gRPC and Connect with identical responses **and** identical error codes and
messages.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'UsageSource|ToConnectError' -v`. Serve the
fixture plugin once with `Web.Enabled=false` and once with `Web.Enabled=true`, call `GetStats`, and
compare results with `proto.Equal`. For each reference-source error, the gRPC `status.Code`/`Message`
must equal the Connect `connect.CodeOf`/`Message`.

### Tests for User Story 1 (write first, confirm failing) ⚠️

- [ ] T009 [P] [US1] Create `sdk/go/pluginsdk/connect_errors_test.go` with a table-driven `TestToConnectError` covering:
  `nil` → `nil`; an existing `connect.NewError(connect.CodeNotFound, …)` returned unchanged (same pointer); each gRPC
  code 1–16 via `status.Error(code, "msg")` → a `*connect.Error` whose `connect.CodeOf` equals `connect.Code(code)` and
  whose `Message()` equals `"msg"`; a plain `errors.New("boom")` returned unchanged (research R8). Also add
  `BenchmarkToConnectError` over a `status.Error(codes.PermissionDenied, "cannot list pods")` input, calling
  `b.ReportAllocs()` (Constitution VIII: benchmarks for all new core SDK logic)
- [ ] T010 [P] [US1] Create `sdk/go/pluginsdk/usage_source_test.go` with `TestUsageSourceServeTransportParity`: for each
  of gRPC mode (`ServeConfig{Plugin: usageTestPlugin, Listener: <net.Listen on 127.0.0.1:0>, Web.Enabled: false}`,
  client from `grpc.NewClient` + `pbc.NewUsageSourceServiceClient`) and Connect mode (`Web.Enabled: true`, client
  `pbcconnect.NewUsageSourceServiceClient(http.DefaultClient, "http://"+addr)`), call `GetStats` with an empty request
  and assert the response is `proto.Equal` to `fixtureStatsResponse()` (including its `Warnings`) and
  `Mode == STATS_MODE_RUN_RATE`; then assert
  the gRPC and Connect responses are `proto.Equal` to each other (US1-1, US1-6, SC-002). Serve in a goroutine with a
  cancellable context and cancel it in `t.Cleanup` (depends on T008)
- [ ] T011 [US1] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceErrorParity`, table-driven over the
  `referenceUsageSource` cases: only `Start` set, `Start` after `End`, both set (historical on run-rate-only source) →
  `codes.InvalidArgument`; RBAC failure → `codes.PermissionDenied` with message `cannot list pods`; no credentials →
  `codes.Unauthenticated`. For each case, serve over both transports and assert the gRPC `status.Code(err)` and
  `status.Convert(err).Message()` equal the Connect `connect.CodeOf(err)` (as `codes.Code`) and `connectErr.Message()`,
  and that neither is `Unknown` (US1-4, US1-5, FR-008, FR-009, SC-002) (depends on T010)
- [ ] T012 [US1] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceSelectorNamespace`: serve
  `referenceUsageSource` and request `Selector: map[string]string{"namespace": "payments"}`; assert every returned
  `kind=workload` row has `subject["namespace"] == "payments"` and at least one such row is returned (US1-3) (depends on
  T010)
- [ ] T048 [US1] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceInterceptors`: serve `usageTestPlugin`
  in gRPC mode with `ServeConfig.UnaryInterceptors` set to one counting interceptor that records `info.FullMethod`; call
  `GetStats` once and assert the interceptor ran exactly once with `/finfocus.v1.UsageSourceService/GetStats`. Connect
  mode needs no case, because the cost service has no Connect interceptors either (FR-010, research R2) (depends on
  T012)
- [ ] T049 [US1] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceHistoricalParity`: serve a
  `referenceUsageSource` with `historical: true` over both transports and request a 2-hour window; assert
  `Mode == STATS_MODE_HISTORICAL`, every row (including `cpu_allocatable`/`mem_allocatable`) has unit `core-hours` or
  `GiB-hours`, and the gRPC and Connect responses are `proto.Equal` (US1-2, FR-004, SC-002) (depends on T048)
- [ ] T050 [US1] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceUnknownMetric`: request
  `Metrics: []string{"cpu_request", "gpu_seconds"}` from `referenceUsageSource`; assert no error and that `Warnings`
  contains exactly one entry naming `gpu_seconds` (edge case "unknown metric requested") (depends on T049)

### Implementation for User Story 1

- [ ] T013 [P] [US1] Create `sdk/go/pluginsdk/connect_errors.go` with the unexported
  `func toConnectError(err error) error` and the godoc from `contracts/go-sdk-api.md`: `nil` → `nil`; `errors.As` a
  `*connect.Error` → return unchanged; `status.FromError(err)` succeeds with code ≠ `codes.OK` →
  `connect.NewError(connect.Code(st.Code()), errors.New(st.Message()))`; anything else → unchanged. Make T009 pass,
  including `BenchmarkToConnectError`
- [ ] T014 [US1] Create `sdk/go/pluginsdk/usage_source.go` with two unexported adapters: `usageSourceGRPCServer` (embeds
  `pbc.UnimplementedUsageSourceServiceServer`, holds a `UsageSourceProvider`, `GetStats` delegates and returns the error
  unchanged) and `usageSourceConnectHandler` (implements `pbcconnect.UsageSourceServiceHandler`;
  `GetStats(ctx, *connect.Request[pbc.GetStatsRequest])` delegates with `req.Msg`, wraps the result in
  `connect.NewResponse`, and returns errors through `toConnectError`) (depends on T013)
- [ ] T015 [US1] In `sdk/go/pluginsdk/sdk.go`, in `Serve`, evaluate
  `usage, hasUsage := config.Plugin.(UsageSourceProvider)` once and pass it to `serveGRPC` and `serveConnect` (add a
  parameter to each). In `serveGRPC`, when present, call
  `pbc.RegisterUsageSourceServiceServer(grpcServer, &usageSourceGRPCServer{…})` next to the existing cost-service
  registration so the server-wide `grpc.ChainUnaryInterceptor` (tracing plus `ServeConfig.UnaryInterceptors`) applies.
  In `serveConnect`, when present, mount
  `pbcconnect.NewUsageSourceServiceHandler(&usageSourceConnectHandler{…}, handlerOpts...)` on the same mux using the
  same `handlerOpts` as the cost handler. Register nothing when the assertion fails. If `gocognit`/`funlen` trips on
  `Serve`, extract a helper instead of adding `//nolint` (depends on T014). Make T010–T012 and T048–T050 pass

**Checkpoint**: `GetStats` is served over both transports with identical results and error codes.
User Story 1 is independently demonstrable (MVP).

---

## Phase 4: User Story 2 - Plugin Developer Builds a Usage Source with the SDK (Priority: P1)

**Goal**: Implementing one method on a `BasePlugin`-embedding struct and calling the standard entry
point yields a served, health-checked usage source. Plugins without `GetStats` are unaffected.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'UsageSourceHealth|UsageSourceNotRegistered|Example' -v`.

### Tests for User Story 2 (write first, confirm failing) ⚠️

- [ ] T016 [US2] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceHealth`: serve `usageTestPlugin` with
  `Web.Enabled=true` and call `grpc.health.v1.Health/Check` over Connect (for example `grpchealth`'s client or a
  `connect.NewClient[healthpb.HealthCheckRequest, healthpb.HealthCheckResponse]` against `/grpc.health.v1.Health/Check`)
  with `Service: pbcconnect.UsageSourceServiceName`; assert `SERVING` (US2-1, FR-010) (depends on T015)
- [ ] T017 [US2] In `sdk/go/pluginsdk/usage_source_test.go`, add `TestUsageSourceNotRegistered`: serve a plugin that
  embeds `*BasePlugin` but has **no** `GetStats`, over both transports; assert `GetStats` returns `codes.Unimplemented`
  over gRPC and `connect.CodeUnimplemented` over Connect, and the Connect health check for
  `pbcconnect.UsageSourceServiceName` returns `NOT_FOUND` (US2-2, research R2) (depends on T015)
- [ ] T018 [P] [US2] In `sdk/go/pluginsdk/example_test.go`, add a compiled example function (for example
  `Example_usageSource`) showing a struct that embeds `*pluginsdk.BasePlugin`, implements only `GetStats` returning one
  `KindWorkload` row with `SubjectNamespace`, `SubjectPod`, `SubjectNode`, `SubjectKind`, `MetricCPURequest`,
  `UnitCore`, and one `KindNode` row, and builds
  `ServeConfig{Plugin: …, PluginInfo: NewPluginInfo(…, WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS))}`
  without calling `Run`. This keeps the README snippet compiled (Constitution XIV, SC-001, US2-3)

### Implementation for User Story 2

- [ ] T019 [US2] In `sdk/go/pluginsdk/sdk.go` `serveConnect`, append `pbcconnect.UsageSourceServiceName` to the
  `grpchealth.NewStaticChecker(…)` arguments only when the plugin implements `UsageSourceProvider` (gRPC mode registers
  no health service today; do not add one) (depends on T015). Make T016 and T017 pass

**Checkpoint**: Usage source is registered, health-checked, and documented by a compiled example.

---

## Phase 5: User Story 3 - Host Discovers Usage Sources and Routes Correctly (Priority: P2)

**Goal**: `GetPluginInfo` advertises `PLUGIN_CAPABILITY_USAGE_STATS` (inferred or explicit) plus the
legacy `supports_usage_stats=true` flag, `IsValidCapability` accepts 14, and usage sources relying on
inference get a startup warning.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'Capabilit|IsValidCapability|UsageSourceWarning' -v`.

### Tests for User Story 3 (write first, confirm failing) ⚠️

- [ ] T020 [P] [US3] In `sdk/go/pluginsdk/capabilities_test.go`, add cases: a plugin implementing `GetStats` with no
  explicit capabilities → inferred list contains `PLUGIN_CAPABILITY_USAGE_STATS` and the legacy metadata contains
  `supports_usage_stats` = `"true"` (US3-1); a plugin with `WithCapabilities(PLUGIN_CAPABILITY_USAGE_STATS)` only →
  capabilities are exactly `[PLUGIN_CAPABILITY_USAGE_STATS]` with no pricing capabilities (US3-2, SC-005). Keep the
  existing capability tests unchanged
- [ ] T021 [P] [US3] In `sdk/go/pluginsdk/conformance_test.go`, extend the `IsValidCapability` bounds test:
  `PLUGIN_CAPABILITY_USAGE_STATS` (14) is valid and `pbc.PluginCapability(15)` is invalid; move any existing "13 is max
  / 14 is invalid" assertion to the new bound (US3-3, FR-012)
- [ ] T022 [P] [US3] In `sdk/go/pluginsdk/plugin_info_test.go`, add a `GetPluginInfo` round-trip case for a usage-stats
  plugin that asserts both the `Capabilities` enum list and the legacy `Metadata["supports_usage_stats"] == "true"` in
  the response
- [ ] T051 [P] [US3] In `sdk/go/pluginsdk/capability_compat_test.go`, add `TestLegacyMetadata_UsageStats` mirroring
  `TestLegacyMetadata_ResolveResourceTypes`: `CapabilityToLegacyName(PLUGIN_CAPABILITY_USAGE_STATS)` returns
  `supports_usage_stats`, and `CapabilitiesToLegacyMetadataWithWarnings([USAGE_STATS])` yields
  `supports_usage_stats=true` with no warnings (FR-011)
- [ ] T023 [US3] In `sdk/go/pluginsdk/usage_source_test.go`, add table-driven `TestUsageSourceWarning` using a
  buffer-backed `zerolog.Logger` passed via `ServeConfig.Logger`: (a) usage source with nil `PluginInfo` → served
  normally and exactly one `warn`-level entry whose message tells usage-only plugins to set `PluginInfo.Capabilities`
  explicitly; (b) `PluginInfo` with explicit `Capabilities` → no such entry; (c) plugin also implementing
  `PluginInfoProvider` → no such entry; (d) plugin without `GetStats` → no such entry (US3-4, FR-013, research R3)
  (depends on T015)

### Implementation for User Story 3

- [ ] T024 [US3] In `sdk/go/pluginsdk/plugin_info.go`: in `inferCapabilities`, append
  `pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS` when `plugin.(UsageSourceProvider)` succeeds, following the
  existing optional-interface type-assertion pattern; change `optionalCapabilities = 6` to `7` so the pre-sized slice
  stays zero-reallocation; change `maxValidCapability` to `pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS // 14`
  with a comment that #506 moves it to 15. Update the comments that go stale with this change: the
  `optionalCapabilities` comment's interface list, the `inferCapabilities` godoc's interface list, the
  "4 base + 5 optional" comment inside `inferCapabilities`, and the "currently defined capabilities (12)" count in the
  `MaxConfiguredCapabilities` comment (now 14). Leave all other inference unchanged (FR-011, FR-012, FR-013). Make
  T020–T022 pass
- [ ] T025 [US3] In `sdk/go/pluginsdk/capability_compat.go`, add
  `pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS: "supports_usage_stats"` to `legacyCapabilityNames`, and update
  the map's godoc from "PluginCapability values (1-13) MUST be included" to "(1-14)" (FR-011). Make T051 pass
- [ ] T026 [US3] In `sdk/go/pluginsdk/usage_source.go`, add unexported
  `warnUsageSourceCapabilities(logger *zerolog.Logger, plugin Plugin, info *PluginInfo)` that logs one `Warn` only when
  the plugin implements `UsageSourceProvider`, `info == nil || len(info.Capabilities) == 0`, and the plugin does **not**
  implement `PluginInfoProvider`. Call it from `Serve` in `sdk/go/pluginsdk/sdk.go` after the server is constructed,
  passing `&server.logger` (the field is a `zerolog.Logger` value) (depends on T015). Make T023 pass
- [ ] T027 [US3] In `sdk/go/pluginsdk/README.md`: add a `UsageSourceProvider` → `PLUGIN_CAPABILITY_USAGE_STATS`
  (`supports_usage_stats`) row to the capability table (around line 695), and a "Usage-only plugins" subsection that
  embeds `*pluginsdk.BasePlugin`, implements `GetStats`, sets `WithCapabilities(PLUGIN_CAPABILITY_USAGE_STATS)`
  explicitly, states that `GetStats` is the only method an author writes, and explains the startup warning. State that
  hosts can tell a usage-only plugin apart only when its capabilities are explicit (SC-005). Keep the snippet identical
  in substance to the T018 example (FR-019) (depends on T018)
- [ ] T028 [P] [US3] In `PLUGIN_DEVELOPER_GUIDE.md`, add a "Usage source plugins" section: what a usage source is, the
  `UsageSourceProvider` interface, the explicit-capabilities rule for usage-only plugins, the startup warning, and a
  link to `docs/usage-source.md` (FR-019)

**Checkpoint**: Hosts can tell usage-only plugins from pricing plugins using metadata alone.

---

## Phase 6: User Story 4 - Plugin Developer Verifies Responses in Tests (Priority: P2)

**Goal**: `plugintesting.ValidateStatsResponse` enforces rules V1–V10, and `UsageSourceHarness` lets
plugin tests call `GetStats` in memory.

**Independent Test**: `go test ./sdk/go/testing/ -run 'GetStatsRequest|StatsResponse|UsageSourceHarness' -v`, plus
`go test ./sdk/go/pluginsdk/ -run 'SubjectVocabulary' -v` for the drift guard.

### Tests for User Story 4 (write first, confirm failing) ⚠️

- [ ] T029 [P] [US4] Create `sdk/go/testing/usage_source_test.go` (package `testing_test`, importing
  `plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"`) with table-driven `TestValidateStatsResponse`. Each
  **rejecting** case asserts `errors.Is(err, plugintesting.ErrInvalidStatsResponse)` and that the message names the
  offending index and key or value: V1 `nil` response; V2 mode `STATS_MODE_UNSPECIFIED`; V3 row subject `{pod: a}`
  (missing `kind`); V4 `kind: __idle__`, `kind: __cluster__`, `kind: pvc`; V10 `{kind: node}` and
  `{kind: node, node: ""}`; V5 unknown key `namespcae` and bare `label.`; V6 amount `-1`, `math.NaN()`, `math.Inf(1)`;
  V7 two rows with identical subject map and metric; V8 priceable `nil` entry and `Id: ""`; V9 priceable with
  `tags.kind=node` and `Id: n2` when no row has `node=n2`. Each **accepting** case asserts `nil`: capacity rows for a
  node with no priceable entry (Fargate-style, FR-016); priceable `tags.kind=cluster` with no matching `node` subject;
  priceable with no `kind` tag; subject key `label.app.kubernetes.io/name`; empty rows in `STATS_MODE_RUN_RATE`; a
  custom metric and unit; same subject with two different metrics; amount `0`; a `STATS_MODE_HISTORICAL` response using
  `core-hours`/`GiB-hours` including allocatable rows. Use string literals, not `pluginsdk` constants (import cycle)
  (SC-003)
- [ ] T030 [US4] In `sdk/go/testing/usage_source_test.go`, add `TestUsageSourceHarness`: define a local struct with a
  `GetStats` method returning a fixed response, `h := plugintesting.NewUsageSourceHarness(impl)`, `h.Start(t)`,
  `defer h.Stop()`, call `h.Client().GetStats(ctx, &pbc.GetStatsRequest{})`, and assert the response is `proto.Equal` to
  the fixture; also assert an implementation error is returned with its gRPC code intact (US4-10, FR-017) (depends on
  T029 file existing)
- [ ] T031 [US4] In `sdk/go/testing/usage_source_test.go`, add `BenchmarkValidateStatsResponse` over a realistic valid
  response (for example 3 nodes × 20 pods × 2 metrics plus allocatable rows and 3 priceable entries), calling
  `b.ReportAllocs()` (Constitution VIII; the helper may allocate)
- [ ] T032 [P] [US4] Create `sdk/go/pluginsdk/subjects_test.go` with `TestSubjectVocabularyMatchesTesting`: build the
  set
  `{SubjectCluster, SubjectNamespace, SubjectControllerKind, SubjectController, SubjectPod, SubjectNode, SubjectKind}`
  and assert it equals the set from `plugintesting.KnownSubjectKeys()` using `assert.ElementsMatch`; also assert
  `SubjectLabelPrefix == "label."`, and that mutating the returned slice does not affect a second call (copy semantics)
  (research R4)

### Implementation for User Story 4

- [ ] T033 [US4] Create `sdk/go/testing/usage_source.go` with: exported
  `var ErrInvalidStatsResponse = errors.New("invalid GetStats response")`; unexported package-level
  `knownSubjectKeys = []string{"cluster", "namespace", "controller_kind", "controller", "pod", "node", "kind"}`,
  `labelPrefix = "label."`, and valid row kinds `workload`/`node` (registry zero-alloc slice pattern); exported
  `KnownSubjectKeys() []string` returning a copy; exported `ValidateStatsResponse(resp *pbc.GetStatsResponse) error`
  returning the **first** violation as `fmt.Errorf("%w: rows[%d]: …", ErrInvalidStatsResponse, …)` (or `priceable[%d]`),
  checking in this exact order: V1 nil response; V2 `mode == STATS_MODE_UNSPECIFIED`; then per row in index order V3
  `kind` missing, V4 `kind` not in {`workload`, `node`}, V10 `kind=node` with missing or empty `node`, V5 each subject
  key neither known nor `label.` followed by a non-empty suffix, V6 amount `< 0`, NaN, or ±Inf, V7 duplicate canonical
  key (sorted `k=v` pairs joined by `\x00`, then `\x00` + metric, tracked in `map[string]struct{}`); collect row `node`
  values into a set; then per priceable in index order V8 nil entry or empty `Id`, V9 `Tags["kind"] == "node"` and `Id`
  not in the node set. Do not check unit/metric consistency or duplicate priceable IDs (spec Assumptions). Make T029,
  T031, T032 pass
- [ ] T034 [US4] In `sdk/go/testing/usage_source.go`, add exported `UsageStatsServer` interface
  (`GetStats(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error)`), exported
  `UsageSourceHarness` struct with unexported fields,
  `NewUsageSourceHarness(impl UsageStatsServer) *UsageSourceHarness`, `Start(t testing.TB)`, `Stop()`, and
  `Client() pbc.UsageSourceServiceClient`. Mirror `TestHarness` in `sdk/go/testing/harness.go`:
  `bufconn.Listen(bufSize)`, the same dial pattern, and register an unexported adapter embedding
  `pbc.UnimplementedUsageSourceServiceServer` that delegates `GetStats`. Leave `NewTestHarness` untouched (SC-004)
  (depends on T033). Make T030 pass
- [ ] T035 [P] [US4] In `sdk/go/testing/README.md`, document `ValidateGetStatsRequest` (rules Q1–Q3,
  `ErrInvertedStatsWindow`, run-rate requests valid), `ValidateStatsResponse` (rules V1–V10 as a table, first-error
  semantics, `ErrInvalidStatsResponse`, what is accepted), and `UsageSourceHarness` with a short usage snippet

**Checkpoint**: Plugin authors can validate usage responses and exercise `GetStats` in memory.

---

## Phase 7: User Story 5 - TypeScript Consumers Call Usage Sources (Priority: P3)

**Goal**: TypeScript clients call `GetStats` through a `UsageSourceClient` wrapper and use mirrored
vocabulary constants.

**Independent Test**: `cd sdk/typescript/packages/client && npx vitest run test/usage-source.test.ts && npm run build`.

### Tests for User Story 5 (write first, confirm failing) ⚠️

- [ ] T036 [P] [US5] Create `sdk/typescript/packages/client/test/usage-source.test.ts` (vitest + msw, following
  `test/resolve-resource-types.test.ts`): an msw handler for `POST {baseUrl}/finfocus.v1.UsageSourceService/GetStats`
  that records the request and returns a JSON `GetStatsResponse` with one workload row, one node row, one priceable
  node, and `mode: "STATS_MODE_RUN_RATE"`; assert
  `new UsageSourceClient({ baseUrl }).getStats(...)` with
  `create(GetStatsRequestSchema, { scope: "c1", selector: { namespace: "payments" } })`
  reaches the handler with that scope and selector and returns rows, priceable entries, and `StatsMode.RUN_RATE`. Add a
  second case where the handler returns a Connect error with code `permission_denied` and assert the promise rejects
  with a `ConnectError` whose `code` is `Code.PermissionDenied`
- [ ] T037 [P] [US5] Add constant-value assertions to `sdk/typescript/packages/client/test/usage-source.test.ts` (or a
  sibling `test/usage-subjects.test.ts`): every constant in `src/utils/usage-subjects.ts` equals its Go counterpart
  value from `sdk/go/pluginsdk/subjects.go`

### Implementation for User Story 5

- [ ] T038 [P] [US5] Create `sdk/typescript/packages/client/src/utils/usage-subjects.ts` exporting `const` strings with
  exactly the Go values: `SUBJECT_CLUSTER`, `SUBJECT_NAMESPACE`, `SUBJECT_CONTROLLER_KIND`, `SUBJECT_CONTROLLER`,
  `SUBJECT_POD`, `SUBJECT_NODE`, `SUBJECT_KIND`, `SUBJECT_LABEL_PREFIX`, `KIND_WORKLOAD`, `KIND_NODE`, `KIND_IDLE`,
  `KIND_CLUSTER`, `METRIC_CPU_REQUEST`, `METRIC_MEM_REQUEST`, `METRIC_CPU_ALLOCATABLE`, `METRIC_MEM_ALLOCATABLE`,
  `METRIC_CPU_USAGE`, `METRIC_MEM_USAGE`, `UNIT_CORE`, `UNIT_GIB`, `UNIT_CORE_HOURS`, `UNIT_GIB_HOURS`, each with a
  TSDoc comment (mirrors `src/utils/usage-profile.ts`)
- [ ] T039 [P] [US5] Create `sdk/typescript/packages/client/src/clients/usage-source.ts` with
  `export class UsageSourceClient`, whose constructor takes the `ClientConfig` exported from `src/clients/auxiliary.ts`
  (`baseUrl`, optional `transport`) and builds the transport the same way `RegistryClient` does, and
  `createClient(UsageSourceService, transport)`; expose
  `async getStats(request: GetStatsRequest): Promise<GetStatsResponse>`. Errors propagate as `ConnectError` with no
  wrapping, and there is no client-side validation (research R9)
- [ ] T040 [US5] In `sdk/typescript/packages/client/src/index.ts`, add
  `export * from "./generated/finfocus/v1/usage_pb.js";`,
  `export { UsageSourceClient } from "./clients/usage-source.js";`, and named exports of every constant from
  `./utils/usage-subjects.js`, following the existing `usage-profile.js` export block. Resolve any export-name collision
  with generated symbols before building (depends on T038, T039). Make T036 and T037 pass and run `npm run build` in
  `sdk/typescript/packages/client`

**Checkpoint**: Go and TypeScript expose `GetStats` in the same release (SC-006).

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Canonical semantics doc, repo docs, and full regression.

- [ ] T041 [P] Create `docs/usage-source.md` as the canonical semantics page (FR-006): purpose and relation to cost
  sources and #506; request fields and the reserved `namespace` selector key (a label named `namespace` cannot be
  selected); run-rate vs historical modes, with the rule that every amount including allocatable is integrated in
  historical mode; subject key table; kinds (note `__idle__`/`__cluster__` are allocator-only and invalid here); metric
  table; unit table per mode; duplicate-row rule and per-container aggregation; priceable tagging with an AWS, an Azure,
  and a GCP node `ResourceDescriptor` example, an EKS control plane (`tags.kind=cluster`), and an unpriceable
  Fargate-style node; error codes (`INVALID_ARGUMENT`, `PERMISSION_DENIED` naming verb and resource, `UNAUTHENTICATED`);
  the usage-only capability rule and that hosts can distinguish usage-only plugins only when capabilities are explicit
  (SC-005); `ValidateGetStatsRequest` and `ValidateStatsResponse` as the author's self-checks; and that in Connect mode
  `ServeConfig.UnaryInterceptors` and tracing do not apply, exactly as for the cost service (FR-010). Link it from
  `docs/README.md`
- [ ] T052 [P] Update the READMEs that list services and clients (FR-019, Constitution VII, research R10). In root
  `README.md`: add `usage.proto` to the `proto/finfocus/v1/` project-structure tree, add a
  "[Usage Source Service](proto/finfocus/v1/usage.proto): UsageSourceService with 1 RPC method (GetStats)" bullet
  beside the existing "gRPC Service" bullet, and add a short `UsageSourceService` subsection after the "gRPC Service
  Interface" section that links `docs/usage-source.md`. In `sdk/typescript/README.md`: add a usage-source line to
  "Features" and a `### UsageSourceClient` section under "Core API" with a `getStats` snippet whose names match T039
  and the `usage-subjects.ts` constants exactly (Constitution XIV)
- [ ] T042 [P] Add a 051 entry to the "Active Technologies" and "Recent Changes" lists in root `CLAUDE.md` (the Active
  Technologies entry already exists; add only what is missing), plus a short "Usage Source SDK Pattern
  (051-usage-source-getstats)" note covering `UsageSourceProvider`, the `sdk/go/testing` vocabulary duplication with its
  drift test, and `toConnectError`
- [ ] T043 Run the full regression from quickstart.md §6: `make test`, `golangci-lint run ./...` (baseline is 0 issues,
  so any finding comes from this change; use an extended timeout), `make lint-markdown`, and
  `bin/buf breaking --against '.git#branch=main'`. Fix all findings without editing `.golangci-lint.yml`
- [ ] T044 Walk through every command and expected result in `specs/051-usage-source-getstats/quickstart.md` §1–§5 and
  confirm each matches; record any deviation by fixing code or updating the quickstart
- [ ] T045 Open a follow-up GitHub issue (with the user's approval) for the pre-existing `ConnectHandler` defect in
  `sdk/go/pluginsdk/connect.go`, where `CostSourceService` RPC errors reach Connect clients as `Unknown`. Reference
  `toConnectError` as the one-line-per-method fix (research R8 scope boundary)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies. T001 and T002 in parallel, then T003 → T004 and T005.
- **Foundational (Phase 2)**: Depends on Phase 1 (generated types). T006 ∥ T007 ∥ T046; T046 → T047;
  T006, T007, T047 → T008.
- **US1 (Phase 3)**: Depends on Phase 2. This is the MVP.
- **US2 (Phase 4)**: Depends on T015 (US1 registration in `Serve`), because health and
  not-registered behavior are properties of the same registration code.
- **US3 (Phase 5)**: T020–T022, T024, T025, T028, T051 depend only on Phase 2. T023 and T026 depend
  on T015. T027 depends on T018 (its snippet mirrors the US2 example).
- **US4 (Phase 6)**: Depends only on Phase 1 (generated `pbc` types); independent of US1–US3. T032
  also needs T006. T035 documents T047's request validator, so it runs after Phase 2.
- **US5 (Phase 7)**: Depends only on T004 (TS bindings); independent of all Go stories.
- **Polish (Phase 8)**: Depends on all stories. T052 also needs T039 (the client it documents).

### User Story Dependencies

```text
Phase 1 ─┬─> Phase 2 (T006–T008, T046, T047) ─┬─> US1 (T009–T015, T048–T050) ─┬─> US2 (T016–T019)
         │                                     │                               ├─> US3 T023, T026
         │                                     │                  US2 T018 ────┴─> US3 T027
         │                                     └─> US3 (T020–T022, T024, T025, T028, T051)
         ├─> US4 (T029–T035)          [T032 also needs T006; T035 after T047]
         └─> US5 (T036–T040)          [needs T004 only]
All ─> Phase 8 (T041–T045, T052)
```

### Within Each User Story

- Tests are written first and must fail (or not compile) before implementation.
- `sdk/go/pluginsdk/usage_source_test.go` is touched by T010–T012, T048–T050, T016, T017, and T023.
  These tasks run sequentially.
- `sdk/go/testing/contract_test.go` and `contract.go` are touched by T046 then T047.
- `sdk/go/pluginsdk/capability_compat.go` and its test are touched by T051 then T025.
- `sdk/go/pluginsdk/sdk.go` is touched by T007, T015, T019, and T026 (sequential).
- `sdk/go/pluginsdk/usage_source.go` is touched by T014 and T026 (sequential).
- `sdk/go/testing/usage_source.go` is touched by T033 then T034.

### Parallel Opportunities

- Phase 1: T001 ∥ T002; after T003, T004 ∥ T005.
- Phase 2: T006 ∥ T007 ∥ T046.
- US1: T009 ∥ T010 (different files); T013 can start with T009.
- US3: T020 ∥ T021 ∥ T022 ∥ T028 ∥ T051 (five different files).
- US4 and US5 can run entirely in parallel with US1–US3 after Phase 1.
- Polish: T041 ∥ T042 ∥ T052.

---

## Parallel Example: After Phase 2

```bash
# Three independent tracks once Phase 2 is done:
Track A (US1 → US2): T009, T010 → T011 → T012 → T048 → T049 → T050 → T013 → T014 → T015 → T016 → T017 → T019
Track B (US4):       T029 → T030 → T031, T032 → T033 → T034 → T035
Track C (US5):       T036, T037, T038, T039 → T040

# Within US3, launch the file-disjoint tests together:
Task: "IsValidCapability bounds in sdk/go/pluginsdk/conformance_test.go"        (T021)
Task: "Inferred/explicit capability cases in sdk/go/pluginsdk/capabilities_test.go" (T020)
Task: "GetPluginInfo round-trip in sdk/go/pluginsdk/plugin_info_test.go"        (T022)
Task: "Legacy name for USAGE_STATS in sdk/go/pluginsdk/capability_compat_test.go" (T051)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: proto, generate, lint, breaking.
2. Phase 2: constants, interface, request validator, fixtures.
3. Phase 3 (US1): `GetStats` served over gRPC and Connect with error-code, historical-mode, and warning parity.
4. **Stop and validate**: `go test ./sdk/go/pluginsdk/ -run 'UsageSource|ToConnectError' -v`.

### Incremental Delivery

1. Setup + Foundational → contract ready.
2. US1 → served usage source (MVP).
3. US2 → health checks, not-registered guarantee, compiled example.
4. US3 → capability discovery, legacy flag, startup warning, README + guide.
5. US4 → validator, harness, drift guard.
6. US5 → TypeScript client and constants.
7. Polish → `docs/usage-source.md`, root and TypeScript READMEs, full regression, follow-up issue.

All stories ship in one PR (Constitution VII requires docs in the same PR as the feature, and
SC-006 requires Go and TS in the same release). The phases are checkpoints, not separate releases.

---

## Notes

- [P] means different files and no dependency on an incomplete task.
- Commit only when the user asks. Use conventional commits (`feat(proto): …`, `feat(pluginsdk): …`,
  `feat(sdk-ts): …`, `docs: …`). CHANGELOG.md is generated by release-please, so do not edit it.
- Do not edit generated code under `sdk/go/proto/` or `src/generated/`; regenerate instead.
