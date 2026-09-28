---

description: "Task list for 052-allocator-allocate"
---

# Tasks: Allocator Service (Allocate)

**Input**: Design documents from `specs/052-allocator-allocate/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required. Constitution Principle V (Test-First Protocol) is non-negotiable, and the plan
orders rule tables, serving and parity tests, and broken-allocator conformance tests before
implementation. In every phase, write the test tasks first and confirm they fail (or do not compile)
before starting the implementation tasks.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested
independently. Because the reference allocator needs the request, currency, and policy helpers, those
helpers are foundational (Phase 2) even though they also serve US2.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US6)
- All paths are repo-relative

## Conventions That Apply to Every Task

- Work on branch `052-allocator-allocate` (`git checkout -b 052-allocator-allocate`); the workspace
  starts on `main` (research R14).
- Every new `.go`, `.proto`, and `.ts` file starts with the Apache 2.0 header used by existing files
  (Constitution XI). Copy it from `sdk/go/pluginsdk/usage_source.go` or a sibling file.
- Every exported Go symbol gets a godoc comment (Constitution XIV).
- `sdk/go/testing` must **not** import `sdk/go/pluginsdk` (import cycle: `pluginsdk/conformance.go`
  imports `sdk/go/testing`). The allocation rules live in `sdk/go/testing`; `pluginsdk` delegates
  (FR-032, research R4).
- `sdk/go/internal/refalloc` imports `pluginsdk`, so only **external** test packages
  (`package pluginsdk_test`, `package testing_test`, `package refalloc_test`) may import it.
- Names are fixed by downstream finfocus plans and must not change: `AllocatorProvider`,
  `DecodePolicy`, `CheckConservation`, `ValidateAllocateRequest`, `ResolveCurrency`,
  `RunAllocatorConformance`, and every proto field name in `contracts/allocation.proto`.
- Invalid-input errors from `ResolveCurrency`, `ValidateAllocateRequest`, and `DecodePolicy` are plain
  errors whose type implements `GRPCStatus() *status.Status` returning `codes.InvalidArgument` with the
  same message; `Error()` has no `rpc error:` prefix (research R3). Do not add `GRPCStatus` to the
  existing `ContractError`.
- Tests that call `pluginsdk.Serve` inject a `net.Listener` via `ServeConfig.Listener` rather than a
  `Port` (CLAUDE.md `pluginsdk.Serve` guidance).
- Do not use `t.Parallel()` on subtests that share one harness or one served listener.
- Run `goimports -w <file>` on each new Go file. Use `require.Len` rather than
  `require.Equal(len(...))` (testifylint).

---

## Phase 1: Setup (Proto Contract and Generated Code)

**Purpose**: Land the wire contract first (Constitution I) and regenerate both SDKs so every later
task compiles against real generated types.

- [X] T001 Create `proto/finfocus/v1/allocation.proto` with the content of
  `specs/052-allocator-allocate/contracts/allocation.proto`: Apache header, `syntax`, `package finfocus.v1`,
  `go_package`, imports of `finfocus/v1/costsource.proto` and `finfocus/v1/usage.proto`, `AllocatorService`,
  `PricedResource`, `AllocateRequest`, `AllocateResponse`, and `AllocationRow`, with all inline comments exactly as
  written (FR-001–FR-005, FR-030). Remove only the line `// Target content for proto/finfocus/v1/allocation.proto
  (052-allocator-allocate).` and the trailing `// enums.proto addition (PluginCapability):` comment block
- [X] T002 [P] In `proto/finfocus/v1/enums.proto`, add `PLUGIN_CAPABILITY_ALLOCATION = 15;` to `enum PluginCapability`
  immediately after `PLUGIN_CAPABILITY_USAGE_STATS = 14;`, preceded by the comment
  `// Plugin implements AllocatorService.Allocate.`
- [X] T003 Run `make generate` and confirm `sdk/go/proto/finfocus/v1/allocation.pb.go`,
  `sdk/go/proto/finfocus/v1/allocation_grpc.pb.go`, and `sdk/go/proto/finfocus/v1/pbcconnect/allocation.connect.go`
  exist, `enums.pb.go` defines `PluginCapability_PLUGIN_CAPABILITY_ALLOCATION`, and `go build ./...` passes. If
  unrelated `*.connect.go` files change only in doc comments (unpinned remote plugins), restore them with
  `git checkout` (depends on T001, T002)
- [X] T004 Regenerate TypeScript bindings for only the changed files:
  `cd sdk/typescript && ../../bin/buf generate ../../proto --template buf.gen.yaml`
  `--path ../../proto/finfocus/v1/allocation.proto --path ../../proto/finfocus/v1/enums.proto`. Confirm
  `sdk/typescript/packages/client/src/generated/finfocus/v1/allocation_pb.ts` exports `AllocatorService`,
  `AllocateRequestSchema`, `AllocateResponseSchema`, `PricedResourceSchema`, and `AllocationRowSchema`, and that
  `enums_pb.ts` has `PluginCapability.ALLOCATION`. If buf fails on the missing `protoc-gen-connect-es` plugin, generate
  with a temporary scratchpad template listing only `protoc-gen-es` (`out: packages/client/src/generated`,
  `opt: target=ts`); do not commit template changes. No other generated files may change (depends on T003)
- [X] T005 Run `bin/buf lint` and `bin/buf breaking --against '.git#branch=main'`; both must be clean (FR-006)
  (depends on T003)

**Checkpoint**: Contract generated in Go and TypeScript; lint and breaking-change detection pass.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The provider interface, the request/currency rules, the strict policy decoder, and the
reference allocator. Every user story's code or fixtures depend on them.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Tests (write first, confirm failing) ⚠️

- [X] T006 [P] Create `sdk/go/testing/allocation_test.go` (package `testing_test`, import alias `plugintesting`) with
  table-driven `TestResolveCurrency` over `[]*pbc.PricedResource` (FR-011, data-model "Derived values"). Accept:
  `nil` → `"USD"`; all `priced=true` with empty currency → `"USD"`; `USD`,`""`,`USD` → `"USD"`; single `EUR` → `"EUR"`;
  a `priced=false` entry with `JPY` beside `priced=true` `USD` → `"USD"` (unpriced currency ignored). Reject: `USD`,`""`,
  `EUR` and `USD`,`EUR` → `errors.Is(err, plugintesting.ErrMixedCurrency)`, `status.Code(err) == codes.InvalidArgument`,
  message lists `EUR` and `USD` sorted, and `err.Error()` does not start with `rpc error:`. Case matters: `usd` and
  `USD` are two currencies (research R7)
- [X] T007 [P] In the same file add table-driven `TestValidateAllocateRequest` (data-model rules Q1–Q5, FR-024). Each
  rejection asserts `errors.Is(err, plugintesting.ErrInvalidAllocateRequest)` (or `ErrMixedCurrency` for Q4) and
  `status.Code(err) == codes.InvalidArgument`. Reject: `nil` request (Q1); a `nil` element in `Priced` (Q1);
  `priced=false` with `Cost: 0.5` (Q2); `Cost: -1` (Q3); `Cost: math.NaN()` and `math.Inf(1)` (Q3); `USD` and `EUR` on
  `priced=true` entries (Q4); two entries with `Resource.Tags["kind"]="node"` and `Id="n1"`, one priced and one not (Q5).
  Accept: empty request; `priced=false` with `Cost: 0`; a node `n1` and a cluster `n1` (same id, different kind);
  `USD`,`""`,`USD`; usage rows with arbitrary contents (usage is not validated)
- [X] T008 [P] Create `sdk/go/pluginsdk/policy_test.go` (package `pluginsdk_test`) with table-driven `TestDecodePolicy`
  against a local target type `testPolicy{Version int "json:\"version\""; NodeSplit struct{CPU float64
  "json:\"cpu_weight\""; Extra map[string]int "json:\"extra\""} "json:\"node_split\""; Tags []string "json:\"tags\"";
  Rules []testRule "json:\"rules\""; Raw json.RawMessage "json:\"raw\""; Skip string "json:\"-\""}` (with
  `testRule{Match string "json:\"match\""}`) whose defaults are `Version 1, CPU 0.5, Extra {"a":1}, Tags
  ["x","y","z"], Rules [{Match:"d"}]`. Cover data-model D1–D6: (D1) `nil`, `""`, `"  \n"`, `"null"`, and `"{}"` leave
  defaults unchanged; (D2) `{`, `{"version":1}{}`, and `{"version":1} x` fail; (D3) `{"node_split":{"cpu":1}}` fails
  with a message containing `node_split.cpu`, `{"rules":[{"match":"a"},{"nope":1}]}` names `rules[1].nope`,
  `{"Version":2}` is unknown (case-sensitive), `{"Skip":"x"}` and `{"-":"x"}` are unknown; (D4)
  `{"node_split":{"cpu_weight":"high"}}` fails naming `node_split.cpu_weight`; (D5) `{"node_split":{"cpu_weight":0.7}}`
  keeps `Extra` and `Version`, `{"node_split":{"extra":{"b":2}}}` yields `{"a":1,"b":2}`, `{"tags":["q"]}` yields exactly
  `["q"]`, `{"rules":[{}]}` yields exactly `[{Match:""}]` (no carried-over `"d"`), `{"raw":{"anything":[1]}}` is accepted
  (opaque); (D6) a non-pointer target and a `nil` pointer return an error that is **not** `ErrInvalidPolicy` and has
  `status.Code(err) == codes.Unknown`. Every D2–D4 failure asserts `errors.Is(err, pluginsdk.ErrInvalidPolicy)` and
  `status.Code(err) == codes.InvalidArgument` (FR-013, FR-031, research R5)
- [X] T009 [P] Create `sdk/go/internal/refalloc/refalloc_test.go` (package `refalloc_test`) testing the reference
  allocator directly: empty request returns no rows, `EffectivePolicyJson` equal to `{"version":1,"node_split":
  {"cpu_weight":0.5}}`, and a 64-char lowercase hex `PolicyDigest` equal to hex SHA-256 of those bytes; `PolicyJson`
  `""` and `"{}"` give identical digests; `{"version":2}` → `codes.InvalidArgument`;
  `{"node_split":{"cpu_weight":1.5}}` → `codes.InvalidArgument`; one priced node `n1` (cost 10, `cpu_allocatable` 4,
  `mem_allocatable` 16) with two workloads requesting 1 core/4 GiB each yields two workload rows with `TotalCost` 2.5
  each and one `__idle__` row for `n1` with `TotalCost` 5, all in `USD`; and workloads requesting 8 cores on that node
  are scaled so the idle row is `≥ 0`

### Implementation

- [X] T010 In `sdk/go/pluginsdk/sdk.go`, directly after the `UsageSourceProvider` interface, declare the exported
  `AllocatorProvider` interface with the single method
  `Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)` and the godoc from
  `contracts/go-sdk-api.md` (FR-017) (depends on T003)
- [X] T011 Create `sdk/go/testing/allocation.go` (package `testing`) with: constants
  `DefaultConservationEpsilon = 1e-6` and `ConservationAbsoluteFloor = 1e-9`; sentinels `ErrConservation`,
  `ErrInvalidAllocateRequest`, `ErrInvalidAllocateResponse`, `ErrMixedCurrency`; an unexported error type
  `invalidArgumentError{msg string; wrapped error}` with `Error() string` (returns `msg`), `Unwrap() error`, and
  `GRPCStatus() *status.Status` (`status.New(codes.InvalidArgument, msg)`); `ResolveCurrency(priced
  []*pbc.PricedResource) (string, error)` per research R7 ("the single distinct non-empty currency over priced=true
  entries, or USD when there is none"; more than one → `invalidArgumentError` wrapping `ErrMixedCurrency`, currencies
  sorted in the message; skip `nil` elements); and `ValidateAllocateRequest(req *pbc.AllocateRequest) error` checking
  Q1–Q5 in order, using `ResolveCurrency` for Q4 and `(resource.GetTags()["kind"], resource.GetId())` as the Q5 key,
  every failure an `invalidArgumentError` wrapping `ErrInvalidAllocateRequest` (Q4 wraps `ErrMixedCurrency`) whose
  message names the entry index. Godoc per `contracts/go-sdk-api.md`. Make T006 and T007 pass (depends on T003)
- [X] T012 Create `sdk/go/pluginsdk/policy.go` with `ErrInvalidPolicy = errors.New("invalid allocation policy")` and
  `DecodePolicy(data []byte, target any) error` implementing research R5 exactly: (1) validate `target` is a non-nil
  pointer, else a plain `fmt.Errorf` (no status); (2) `bytes.TrimSpace`; empty or `null` → return `nil`; (3) decode one
  value with `json.Decoder` into `json.RawMessage`, then require a second `Decode` to return `io.EOF` ("trailing
  data"); (4) walk the parsed `any` against `reflect.Type` of `target` building paths (`a.b`, `a[2].b`) — struct keys
  match exactly (case-sensitive) the effective JSON name (tag name, else Go field name; embedded structs promoted;
  `json:"-"` and unexported fields excluded); pointers dereferenced; `map[string]T` accepts any key and recurses into
  `T`; slices/arrays recurse per element; `any`, `json.RawMessage`, `json.Unmarshaler`, and `encoding.TextUnmarshaler`
  are opaque; while walking, set every slice field the document supplies to `nil` in the target so arrays are replaced
  wholesale; (5) `json.Unmarshal(data, target)`, mapping `*json.UnmarshalTypeError` to a message naming its `Field`
  path. Every input failure returns an error type with `GRPCStatus()` → `codes.InvalidArgument`, `Error()` without an
  `rpc error:` prefix, wrapping `ErrInvalidPolicy`. Keep functions under the `gocognit` threshold (20) by splitting the
  walk per kind. Make T008 pass (FR-031) (depends on T010)
- [X] T013 Create `sdk/go/pluginsdk/allocator.go` with delegating wrappers, each godoc'd as identical to the
  `sdk/go/testing` function of the same name (FR-032): `const DefaultConservationEpsilon =
  plugintesting.DefaultConservationEpsilon`, `func ValidateAllocateRequest(req *pbc.AllocateRequest) error`, and
  `func ResolveCurrency(priced []*pbc.PricedResource) (string, error)`. (`CheckConservation` is added in T024.) Import
  `plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"` as `pluginsdk/conformance.go` does
  (depends on T011)
- [X] T014 Create `sdk/go/internal/refalloc/refalloc.go` (package `refalloc`, godoc stating it is a test fixture only
  and not a template for production allocators, research R9) with `type Allocator struct{}`, `func New() *Allocator`,
  and `Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)`: call
  `pluginsdk.ValidateAllocateRequest` (return its error unchanged); decode `req.GetPolicyJson()` with
  `pluginsdk.DecodePolicy` onto defaults `Policy{Version: 1, NodeSplit: NodeSplit{CPUWeight: 0.5}}` (json names
  `version`, `node_split`, `cpu_weight`); reject `Version != 1` and `CPUWeight` outside `[0, 1]` with
  `status.Error(codes.InvalidArgument, …)`; canonical bytes = `json.Marshal(policy)`, digest = lowercase hex SHA-256;
  currency = `pluginsdk.ResolveCurrency`. Per `priced=true` entry with `tags.kind == "node"`: `cpu_pool = cost ×
  cpu_weight`, `mem_pool = cost − cpu_pool`; each workload on that node (joined by `subject["node"] == resource.id`,
  requests from `cpu_request`/`mem_request` rows, allocatable from the node's `cpu_allocatable`/`mem_allocatable` rows)
  gets `pool × request / allocatable`, scaled by `allocatable / Σ requests` when requests exceed allocatable; a zero
  allocatable gives zero shares; always emit one `__idle__` row with `node` = id, `cpu_cost = max(0, cpu_pool − Σ cpu
  shares)`, same for memory, `total = cpu + mem`. Priced `tags.kind == "cluster"` → one `__cluster__` row with the full
  cost in `total_cost`; any other priced kind → a `__cluster__` row with a note naming the kind. Workloads on an
  unpriced or unknown node → zero-cost workload rows with a note. Every row's currency = the resolved currency; use
  `pluginsdk.Subject*`/`Kind*`/`Metric*` constants; sort node and workload keys so output order is deterministic.
  Make T009 pass (depends on T012, T013)

**Checkpoint**: Interface, request/currency rules, strict decoder, and the reference allocator exist and are tested.

---

## Phase 3: User Story 1 - Host Obtains a Cost Breakdown from an Allocator (Priority: P1) 🎯 MVP

**Goal**: An `AllocatorProvider` served by `pluginsdk.Serve` answers `Allocate` over both gRPC and
Connect with identical results **and** identical error codes and messages.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'Allocator' -v`. Serve `refalloc` with a fixed
two-node request over each transport and compare responses.

### Tests for User Story 1 (write first, confirm failing) ⚠️

- [X] T015 [US1] Create `sdk/go/pluginsdk/allocator_serve_test.go` (package `pluginsdk_test`) with: (a)
  `allocTestPlugin`, a struct embedding `*pluginsdk.BasePlugin` (via `pluginsdk.NewBasePlugin`) and delegating
  `Allocate` to an inner `pluginsdk.AllocatorProvider`; (b) `fixtureAllocateRequest()`, a two-node run-rate cluster
  (`n1` cost 10, `n2` cost 6, both `priced=true`, `USD`, `tags.kind=node`), with `cpu_allocatable`/`mem_allocatable`
  rows per node and two workloads per node in namespaces `payments` and `web`, and a `priced=false` third node `n3`
  (cost 0, note `no price`) running one workload; (c) a helper that serves a plugin via `pluginsdk.Serve` on an
  injected `net.Listener` in gRPC mode and, separately, with `Web.Enabled` (Connect mode), returning a gRPC client
  (`pbc.NewAllocatorServiceClient`) and a Connect client (`pbcconnect.NewAllocatorServiceClient`) respectively, with a
  5-second call timeout and context cancellation for shutdown, mirroring `usage_source_test.go`
- [X] T016 [US1] In the same file add `TestAllocator_TransportParity`: serve `refalloc.New()` (wrapped in
  `allocTestPlugin`); for each transport, call `Allocate` with (i) the fixture and no policy, (ii) the fixture with
  `{"node_split":{"cpu_weight":0.25}}`, and (iii) an empty request. Assert with `proto.Equal` that the gRPC and Connect
  responses are identical (rows, `EffectivePolicyJson`, `PolicyDigest`, `Warnings`); that (i) contains one `__idle__`
  row each for `n1` and `n2` and none required for `n3`, and that the `n3` workload row has `TotalCost == 0` and a
  non-empty `Note` (US1 scenarios 1, 3); and that (ii)'s effective policy contains `"cpu_weight":0.25` (US1
  scenario 4). Sum row totals and assert 16 within `1e-9` (conservation is re-checked with the SDK helper in US2)
  (FR-018, SC-002)
- [X] T017 [US1] In the same file add `TestAllocator_ErrorParity`: over both transports, send (a) policy
  `{"node_split":{"cpu":1}}`, (b) policy `{"version":99}`, (c) a request with a `priced=false` entry costing 1, and (d)
  mixed `USD`/`EUR`. Assert both transports return `codes.InvalidArgument` (Connect: `connect.CodeOf(err) ==
  connect.CodeInvalidArgument`) with byte-identical messages, and that (a)'s message contains `node_split.cpu`. Add
  (e) a plugin whose `Allocate` returns `status.Error(codes.FailedPrecondition, "boom")`, asserting the same code and
  message on both transports (FR-013, FR-018, US1 scenario 5)
- [X] T018 [US1] In the same file add `TestAllocator_ThreeNodesAndControlPlane`: three priced nodes plus a priced
  `tags.kind=cluster` resource (cost 3) over gRPC; assert one `__idle__` row per node, at least one `__cluster__` row,
  and row totals summing to the four prices within `1e-9` (US1 scenario 2)

### Implementation for User Story 1

- [X] T019 [US1] Add unexported adapters to `sdk/go/pluginsdk/allocator.go`: `allocatorGRPCServer` embedding
  `pbc.UnimplementedAllocatorServiceServer` with a `provider AllocatorProvider` field and an `Allocate` method that
  delegates; and `allocatorConnectHandler` with `Allocate(ctx, *connect.Request[pbc.AllocateRequest])
  (*connect.Response[pbc.AllocateResponse], error)` that delegates and converts errors with `toConnectError`, mirroring
  `usage_source.go` (FR-017, FR-018) (depends on T010, T015)
- [X] T020 [US1] In `sdk/go/pluginsdk/sdk.go`, introduce an unexported `optionalServices struct{ usage
  UsageSourceProvider; allocator AllocatorProvider }` built once in `Serve` via type assertions on `config.Plugin`, and
  change `serveGRPC` and `serveConnect` to take `services optionalServices` instead of `usage UsageSourceProvider`
  (update their doc comments). In `serveGRPC`, register `pbc.RegisterAllocatorServiceServer(grpcServer,
  &allocatorGRPCServer{provider: services.allocator})` when non-nil (the existing interceptor chain applies to it). In
  `serveConnect`, register `pbcconnect.NewAllocatorServiceHandler(&allocatorConnectHandler{…}, handlerOpts...)` on the
  mux when non-nil. Keep usage-source registration unchanged. Make T016–T018 pass (research R2) (depends on T019)

**Checkpoint**: A served allocator answers identically over gRPC and Connect, including errors. MVP complete.

---

## Phase 4: User Story 2 - Host Verifies an Allocator Did Not Lose or Invent Money (Priority: P1)

**Goal**: Hosts, plugins, and the conformance suite share one conservation check and one response
validator, available from production code via `pluginsdk`.

**Independent Test**: `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/ -run 'Conservation|ValidateAllocateResponse' -v`
with hand-built request/response pairs.

### Tests for User Story 2 (write first, confirm failing) ⚠️

- [X] T021 [P] [US2] In `sdk/go/testing/allocation_test.go` add table-driven `TestCheckConservation` (data-model C1–C4,
  FR-007, FR-022, US2 scenarios 1–6). Pass: rows equal to priced sum (10 = 6 + 4); a `priced=false` entry excluded
  from the expected total; nothing priced and all-zero rows (zero total); a difference of `5e-6` on an expected total
  of 10 (within 1e-6 relative); expected 0 and actual `5e-10` (absolute floor); `__cluster__` rows with zero
  CPU/memory. Fail: overshoot by `0.01` and shortfall by `0.01` → `errors.As` a `*plugintesting.ConservationError`
  whose `Expected`, `Actual`, `Difference` (actual − expected, signed), and `Currency` are exact and whose message
  contains all three numbers; a row `TotalCost: math.NaN()` → error (not a pass); a priced `Cost: math.Inf(1)` →
  error; `relEpsilon` of `-1`, `math.NaN()`, and `math.Inf(1)` → error; mixed request currencies →
  `errors.Is(err, plugintesting.ErrMixedCurrency)`. Add `BenchmarkCheckConservation` at 1,000 and 10,000 rows
  (Constitution VIII)
- [X] T022 [P] [US2] In the same file add table-driven `TestValidateAllocateResponse` (data-model P1–P7, FR-023). Build
  one valid pair (one priced node `n1`, one workload row, one idle row, one `__cluster__` row with zero CPU/memory, all
  `USD`, non-empty policy and digest) and assert `nil`. Reject, each with `errors.Is(err,
  plugintesting.ErrInvalidAllocateResponse)`: `nil` response (P1); empty `PolicyDigest` (P1); empty
  `EffectivePolicyJson` (P1); a row without `kind` and a row with `kind=node` (P2); an idle row without `node` (P3);
  negative `CpuCost`, `MemCost`, `TotalCost`, and a NaN `TotalCost` (P4); a workload row with total 3 and CPU+memory 2
  (P5); an empty row currency and a `EUR` row currency against a `USD` request (P6); a missing idle row for `n1` and two
  idle rows for `n1` (P7). Accept: a workload row whose total differs from CPU+memory by `1e-12`; no idle row needed for
  a `priced=false` node; empty-currency priced entries resolving to `USD` with `USD` rows. Add
  `BenchmarkValidateAllocateResponse` at 1,000 and 10,000 rows
- [X] T023 [P] [US2] In `sdk/go/pluginsdk/allocator_serve_test.go` add `TestAllocator_HostVerification`: call the
  served `refalloc` over gRPC with the T015 fixture and assert `pluginsdk.CheckConservation(req, resp,
  pluginsdk.DefaultConservationEpsilon)` returns `nil`; mutate one row's `TotalCost` by `+1` and assert it fails with a
  `*plugintesting.ConservationError`. Add a table test proving `pluginsdk.ValidateAllocateRequest`,
  `pluginsdk.ResolveCurrency`, and `pluginsdk.CheckConservation` return results identical to the `plugintesting`
  functions for three inputs each (FR-032)

### Implementation for User Story 2

- [X] T024 [US2] In `sdk/go/testing/allocation.go` add `ConservationError{Expected, Actual, Difference float64;
  Currency string}` with `Error()` (for example `allocation rows total 10.01 USD, expected 10 USD (difference +0.01)`)
  and `Unwrap() error` returning `ErrConservation`; `CheckConservation(req, resp, relEpsilon) error` implementing
  research R6 in this order: reject `relEpsilon` that is NaN, ±Inf, or `< 0`; `ResolveCurrency` (return its error
  unchanged); reject non-finite priced costs and row totals; `expected = Σ cost over priced=true`, `actual = Σ
  total_cost`; pass when `math.Abs(actual−expected) <= math.Max(relEpsilon*math.Abs(expected),
  ConservationAbsoluteFloor)`; else return `*ConservationError`. Nil request or response → error. Make T021 pass
  (depends on T011)
- [X] T025 [US2] In `sdk/go/testing/allocation.go` add `ValidateAllocateResponse(req *pbc.AllocateRequest, resp
  *pbc.AllocateResponse) error` checking P1–P7 in order, with a package-level `validAllocationKinds = []string{"workload",
  "__idle__", "__cluster__"}` (`//nolint:gochecknoglobals` like `usage_source.go`) reusing `containsString`, private
  constants `kindIdle = "__idle__"` and `kindCluster = "__cluster__"` beside the existing private vocabulary, the P5
  tolerance `max(DefaultConservationEpsilon×|total|, ConservationAbsoluteFloor)`, and the resolved currency from
  `ResolveCurrency(req.GetPriced())`. Every failure wraps `ErrInvalidAllocateResponse` and names the row index. Make
  T022 pass (depends on T024)
- [X] T026 [US2] In `sdk/go/pluginsdk/allocator.go` add `func CheckConservation(req *pbc.AllocateRequest, resp
  *pbc.AllocateResponse, relEpsilon float64) error` delegating to `plugintesting.CheckConservation`, godoc'd for host
  use. Make T023 pass (depends on T024, T020)

**Checkpoint**: Conservation and response validation are available from `pluginsdk` and `testing`, with one rule.

---

## Phase 5: User Story 3 - Plugin Developer Builds an Allocator with the SDK (Priority: P1)

**Goal**: A plugin embedding `BasePlugin` and implementing only `Allocate` is served and health-checked
with no extra wiring; plugins without it are unchanged; usage source and allocator coexist.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'AllocatorServe|Example' -v`.

### Tests for User Story 3 (write first, confirm failing) ⚠️

- [X] T027 [US3] In `sdk/go/pluginsdk/allocator_serve_test.go` add `TestAllocatorServe_HealthIncludesAllocator`: in
  Connect mode, call `grpc.health.v1.Health/Check` (via `connectrpc.com/grpchealth` client or a gRPC health client over
  h2c, matching how the 051 usage test checks health) for `finfocus.v1.AllocatorService` and assert `SERVING`
  (US3 scenario 1, FR-017)
- [X] T028 [US3] In the same file add `TestAllocatorServe_NotRegisteredWithoutProvider`: serve a plain
  `pluginsdk.NewBasePlugin("cost-only")` in both modes; assert `Allocate` returns `codes.Unimplemented` over gRPC and
  `connect.CodeUnimplemented` over Connect, that the Connect health check for `finfocus.v1.AllocatorService` does not
  report `SERVING`, and that `Name` still works (US3 scenario 2, SC-006)
- [X] T029 [US3] In the same file add `TestAllocatorServe_WithUsageSource`: a plugin embedding `BasePlugin` that
  implements both `GetStats` (returning a fixed one-row response) and `Allocate` (delegating to `refalloc`); in both
  modes call both RPCs and assert each answers independently, and in Connect mode that health reports both services
  `SERVING` (US3 scenario 3)
- [X] T030 [P] [US3] In `sdk/go/pluginsdk/allocator_serve_test.go` add `TestAllocatorServe_InterceptorsApply`: in gRPC
  mode configure `ServeConfig.UnaryInterceptors` with a counting interceptor and assert it runs for `Allocate`
  (FR-017). Mirror what `usage_source_test.go` documents for Connect mode (interceptors do not apply there, exactly as
  for the cost service)
- [X] T031 [P] [US3] In `sdk/go/pluginsdk/example_test.go` add a compiled `ExampleAllocatorProvider` showing a type that
  embeds `*pluginsdk.BasePlugin`, implements `Allocate` by calling `ValidateAllocateRequest`, `DecodePolicy` onto
  defaults, and `ResolveCurrency`, and is started with `pluginsdk.NewPluginInfo(…,
  pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_ALLOCATION))` and `ServeConfig` (not actually
  served), plus `ExampleDecodePolicy` with an `// Output:` block printing the merged defaults (SC-001, Constitution XIV)

### Implementation for User Story 3

- [X] T032 [US3] In `serveConnect` in `sdk/go/pluginsdk/sdk.go`, append `pbcconnect.AllocatorServiceName` to
  `healthServices` when `services.allocator != nil`, and update the health comment to mention the allocator. Make
  T027–T030 pass (depends on T020)

**Checkpoint**: Minimal allocators need one method and the standard entry point.

---

## Phase 6: User Story 4 - Plugin Developer Proves an Allocator Is Correct (Priority: P2)

**Goal**: `RunAllocatorConformance` runs 12 named, policy-agnostic subtests; the reference allocator
passes all of them and each deliberately broken allocator fails its target scenario.

**Independent Test**: `go test ./sdk/go/testing/ -run 'AllocatorConformance' -v`.

### Tests for User Story 4 (write first, confirm failing) ⚠️

- [X] T033 [P] [US4] Create `sdk/go/testing/export_test.go` (package `testing`) exposing
  `var RunAllocatorScenariosForTest = runAllocatorScenarios`, where `runAllocatorScenarios(ctx context.Context, client
  pbc.AllocatorServiceClient) map[string]error` (defined in T037) returns each scenario's error keyed by subtest name
  (research R8)
- [X] T034 [US4] Create `sdk/go/testing/allocator_conformance_test.go` (package `testing_test`) with
  `TestAllocatorConformance_Reference`, calling `plugintesting.RunAllocatorConformance(t, refalloc.New())`, and
  `TestAllocatorConformance_ScenarioNames` asserting the scenario map from `RunAllocatorScenariosForTest` (via an
  `AllocatorHarness`) has exactly these 12 keys: `single_node`, `three_nodes`, `empty_cluster`, `fully_packed_node`,
  `unpriced_node`, `control_plane`, `over_requested_node`, `policy_unknown_field`, `policy_unknown_version`,
  `empty_request`, `fingerprint_stable`, `fingerprint_empty_equals_braces`, all `nil` for `refalloc` (FR-025, FR-027,
  SC-004)
- [X] T035 [US4] In the same file add `TestAllocatorConformance_RejectsBrokenAllocators`, a table of wrappers around
  `refalloc.New()`, each served through `plugintesting.NewAllocatorHarness` and run through
  `RunAllocatorScenariosForTest`, asserting the named scenario's error is non-nil (SC-003, US4 scenarios 2–6):
  over-allocation (first workload row `TotalCost` and `CpuCost` +1) → `single_node`; under-allocation (drop the first
  workload row) → `single_node`; dropped idle (remove all `__idle__` rows) → `single_node` and `empty_cluster`, with
  the `empty_cluster` error mentioning the idle/conservation shortfall; negative idle (set every idle row's `CpuCost`
  and `TotalCost` to −1) → `over_requested_node`; ignores unknown fields (decode the policy with plain `json.Unmarshal`
  onto defaults and never reject) → `policy_unknown_field`; accepts unknown version (skip the version check) →
  `policy_unknown_version`; unstable digest (append an incrementing counter to `PolicyDigest`) → `fingerprint_stable`;
  braces differ (return a different digest when `PolicyJson` is `{}`) → `fingerprint_empty_equals_braces`
- [X] T036 [P] [US4] In `sdk/go/testing/allocator_conformance_test.go` add `TestAllocatorHarness` (FR-026): start a
  harness over `refalloc.New()`, call `Client().Allocate` with an empty request, and assert a non-empty
  `PolicyDigest`; assert `Stop` is idempotent-safe after `Start`

### Implementation for User Story 4

- [X] T037 [US4] Create `sdk/go/testing/allocator_conformance.go` with: `AllocateServer` interface (single `Allocate`
  method, godoc noting that `pbc.AllocatorServiceServer` and `pluginsdk.AllocatorProvider` both satisfy it); an
  unexported adapter embedding `pbc.UnimplementedAllocatorServiceServer`; `AllocatorHarness` with
  `NewAllocatorHarness(impl AllocateServer)`, `Start(t testing.TB)`, `Stop()`, and `Client()
  pbc.AllocatorServiceClient`, copied from `UsageSourceHarness` in `usage_source.go` (bufconn, `grpc.DialContext` with
  the same `//nolint:staticcheck` comment); fixture builders for the scenarios in `contracts/go-sdk-api.md` (nodes with
  `cpu_allocatable`/`mem_allocatable` rows, workloads with `cpu_request`/`mem_request` rows, priced entries with
  `tags.kind`, `USD`) using the package's private vocabulary constants. Fixture usage MUST be what a valid usage source
  returns: every row carries `kind`; node rows carry `node`; each workload has a distinct `namespace`/`pod` pair plus
  `node`, `controller_kind`, and `controller`; no (subject, metric) pair repeats. Each fixture's usage is wrapped in a
  `GetStatsResponse` (mode `RUN_RATE`, priceable nodes from the fixture) and asserted with `ValidateStatsResponse` before
  the call, so allocators that key workloads by identity are exercised on realistic input. Then `runAllocatorScenarios`
  executing the 12
  scenarios with the common assertions (fixture passes `ValidateAllocateRequest`; call succeeds;
  `ValidateAllocateResponse`; `CheckConservation` at `DefaultConservationEpsilon`) plus the scenario-specific ones
  from the table in `contracts/go-sdk-api.md`. `policy_unknown_field`: call with an empty request, parse
  `EffectivePolicyJson` as `map[string]any`, add `"conformance_unknown_field": true`, expect
  `codes.InvalidArgument` with the key in the message; if any top-level value is an object, pick `f` as the first such
  key in sorted order (so failures name the same path on every run), inject `f.conformance_unknown_field`, and expect
  `f.conformance_unknown_field` in the message. `policy_unknown_version`:
  set `version` to `2147483647`, expect `codes.InvalidArgument`. `empty_request`: no rows, digest matches
  `^[0-9a-f]{64}$`, effective policy is a JSON object whose `version` is an integer (FR-012). Split scenarios into
  small functions to stay under `gocognit` 20. Define `export_test.go`'s target here (depends on T025, T033)
- [X] T038 [US4] In the same file add `RunAllocatorConformance(t *testing.T, impl AllocateServer)`, which starts an
  `AllocatorHarness`, runs `runAllocatorScenarios`, and reports each scenario in `t.Run(name, …)` (sorted by the
  table's order) with `t.Error(err)` on failure, without `t.Parallel()`. Godoc: assertions are policy-agnostic; list
  the scenarios. Make T034–T036 pass (depends on T037)

**Checkpoint**: Third-party allocators can self-certify; broken allocators are caught.

---

## Phase 7: User Story 5 - Host Discovers Allocators and Routes Correctly (Priority: P2)

**Goal**: Capability 15 is inferred, valid, mapped to `supports_allocation`, overridable, and
allocation-only plugins without explicit capabilities get a startup warning.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'Capabilit|LegacyCapability|IsValidCapability|Warn' -v`.

### Tests for User Story 5 (write first, confirm failing) ⚠️

- [X] T039 [P] [US5] In `sdk/go/pluginsdk/conformance_test.go`, add
  `{"ALLOCATION", pbc.PluginCapability_PLUGIN_CAPABILITY_ALLOCATION, true}` beside the `USAGE_STATS` row and change
  `{"just above max (15)", pbc.PluginCapability(15), false}` to `{"just above max (16)", pbc.PluginCapability(16),
  false}` (FR-020, US5 scenario 3)
- [X] T040 [P] [US5] In `sdk/go/pluginsdk/capability_compat_test.go` add `TestLegacyMetadata_Allocation` asserting
  `PLUGIN_CAPABILITY_ALLOCATION` maps to `supports_allocation` and appears as `"true"` in legacy metadata
  (FR-019); `TestLegacyCapabilityMapCompleteness` will fail until T043
- [X] T041 [P] [US5] In `sdk/go/pluginsdk/plugin_info_test.go` add: an `AllocatorProvider` plugin with inferred
  capabilities reports `PLUGIN_CAPABILITY_ALLOCATION` in `GetPluginInfo` and `supports_allocation=true` in metadata
  (US5 scenario 1); with `WithCapabilities(PLUGIN_CAPABILITY_ALLOCATION)` exactly that capability is reported (US5
  scenario 2); a plugin that is neither usage source nor allocator reports no allocation capability (FR-021)
- [X] T042 [P] [US5] Add `TestServe_WarnsAllocatorWithoutExplicitCapabilities` next to the existing usage-source
  warning test (find it with `grep -rn "usage source relies on inferred capabilities" sdk/go/pluginsdk/*_test.go`):
  with a zerolog buffer logger, serving an `AllocatorProvider` without `PluginInfo.Capabilities` logs a warning whose
  `capability` field is `PLUGIN_CAPABILITY_ALLOCATION` and whose message says allocation-only plugins should set
  `PluginInfo.Capabilities` explicitly; no warning with explicit capabilities or for a `PluginInfoProvider`; a plugin
  implementing both providers logs both warnings; and the existing usage-source warning text is byte-identical
  (US5 scenario 4, FR-021, SC-006)

### Implementation for User Story 5

- [X] T043 [US5] In `sdk/go/pluginsdk/capability_compat.go` add
  `pbc.PluginCapability_PLUGIN_CAPABILITY_ALLOCATION: "supports_allocation"` to `legacyCapabilityNames` (depends on
  T040)
- [X] T044 [US5] In `sdk/go/pluginsdk/plugin_info.go`: set `optionalCapabilities = 8` (update its comment to list
  `AllocatorProvider`), set `maxValidCapability = pbc.PluginCapability_PLUGIN_CAPABILITY_ALLOCATION // 15`, and in
  `inferCapabilities` append `PLUGIN_CAPABILITY_ALLOCATION` when `plugin.(AllocatorProvider)` succeeds, after the
  usage-source check; update the function's godoc list. Inference stays a pre-sized slice with zero extra allocations
  (FR-019, Constitution VIII). Make T039–T041 pass (depends on T043)
- [X] T045 [US5] In `sdk/go/pluginsdk/usage_source.go`, replace `warnUsageSourceCapabilities` with
  `warnInferredOnlyCapabilities(logger *zerolog.Logger, plugin Plugin, info *PluginInfo)` that returns early for
  `PluginInfoProvider` plugins or explicit capabilities, then logs the existing usage-source warning unchanged when the
  plugin is a `UsageSourceProvider`, and a second warning with `capability` =
  `PLUGIN_CAPABILITY_ALLOCATION` and message `"allocator relies on inferred capabilities, which include pricing
  capabilities; allocation-only plugins should set PluginInfo.Capabilities explicitly"` when it is an
  `AllocatorProvider`; update the call in `Serve` (`sdk.go`). Make T042 pass (research R10) (depends on T044)

**Checkpoint**: Hosts can discover allocators and distinguish allocation-only plugins.

---

## Phase 8: User Story 6 - TypeScript Consumers Call Allocators (Priority: P3)

**Goal**: An `AllocatorClient` in the TypeScript client package, mirroring `UsageSourceClient`.

**Independent Test**: `cd sdk/typescript/packages/client && npx vitest run test/allocator.test.ts && npx tsc --noEmit`.

### Tests for User Story 6 (write first, confirm failing) ⚠️

- [X] T046 [P] [US6] Create `sdk/typescript/packages/client/test/allocator.test.ts` (vitest + msw, following
  `test/usage-source.test.ts`): mock `POST */finfocus.v1.AllocatorService/Allocate` returning JSON with two rows (one
  `workload`, one `__idle__` with `node`), `effectivePolicyJson` (base64 of `{"version":1}`), `policyDigest` (64 hex
  chars), and one warning; assert `new AllocatorClient({ baseUrl }).allocate(create(AllocateRequestSchema, {...}))`
  returns them intact, with `effectivePolicyJson` decoding back to `{"version":1}`. Second test: the mock returns a
  Connect error `{"code":"invalid_argument","message":"unknown field node_split.cpu"}` with HTTP 400; assert a
  `ConnectError` with `Code.InvalidArgument` and that message (FR-028)

### Implementation for User Story 6

- [X] T047 [US6] Create `sdk/typescript/packages/client/src/clients/allocator.ts` with `AllocatorClient` exactly as in
  `contracts/typescript-api.md` (constructor from `ClientConfig`, default `createConnectTransport({ baseUrl,
  useBinaryFormat: false })`, one `allocate(request)` method), copied in shape from `src/clients/usage-source.ts`
  (depends on T004, T046)
- [X] T048 [US6] In `sdk/typescript/packages/client/src/index.ts` add
  `export * from "./generated/finfocus/v1/allocation_pb.js";` beside the `usage_pb.js` export and
  `export { AllocatorClient } from "./clients/allocator.js";` beside `UsageSourceClient`. Run `npx vitest run` and
  `npx tsc --noEmit` in `sdk/typescript/packages/client` (do not use `npm run build`; it fails at tsup's DTS step on
  `main` too). Make T046 pass (depends on T047)

**Checkpoint**: Go and TypeScript expose `Allocate` together (SC-007).

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Canonical semantics doc, repo docs, benchmarks, and full regression.

- [X] T049 [P] Create `docs/allocator.md` as the canonical semantics page (FR-029, FR-030, research R12): purpose and
  the host → usage source → cost source → allocator flow; the constitution III position (contract shape, not math);
  request fields; `PricedResource` rules (zero cost when unpriced, uniqueness by `(tags.kind, id)`, currency resolution
  with the `USD`,empty,`USD` → `USD` and `USD`,empty,`EUR` → rejected examples, `USD` fallback scoped to this
  contract); row kinds and subject keys; the five invariants with the tolerance formula; policy rules (opaque,
  top-level integer `version`, strict decoding naming paths, `{}` equals empty, never fall back to defaults, digest
  per allocator only); the error table (all `INVALID_ARGUMENT`); a worked single-node example matching T009's numbers;
  AWS, Azure, and GCP node `ResourceDescriptor` examples plus an unpriced node (Constitution II); the explicit
  capabilities rule; and host verification with `pluginsdk.CheckConservation`. Link it from `docs/README.md`
- [X] T050 [P] Update `PLUGIN_DEVELOPER_GUIDE.md` with a "## Allocator Plugins" section after "## Usage Source Plugins"
  containing "### Writing an allocator" (the `AllocatorProvider` interface, embedding `BasePlugin`, calling
  `ValidateAllocateRequest` → `DecodePolicy` → version check → `ResolveCurrency`, the invariants, idle and cluster
  rows), "### Declare Capabilities Explicitly", and "### Testing an Allocator" (`RunAllocatorConformance`, what each
  scenario checks, `AllocatorHarness`). Snippet names must match exported symbols exactly (FR-029, Constitution XIV)
- [X] T051 [P] Update `sdk/go/pluginsdk/README.md`: add a `PLUGIN_CAPABILITY_ALLOCATION` / `AllocatorProvider` /
  `supports_allocation` row to the capability table, the allocation-only explicit-capabilities rule, and a short
  section on `DecodePolicy`, `ValidateAllocateRequest`, `ResolveCurrency`, and `CheckConservation` (FR-029)
- [X] T052 [P] Update `sdk/go/testing/README.md` with `CheckConservation`, `ConservationError`, `ResolveCurrency`,
  `ValidateAllocateRequest`, `ValidateAllocateResponse`, `AllocatorHarness`, and `RunAllocatorConformance` with its 12
  subtest names
- [X] T053 [P] Update root `README.md` (add `allocation.proto` to the `proto/finfocus/v1/` tree, an "[Allocator
  Service](proto/finfocus/v1/allocation.proto): AllocatorService with 1 RPC method (Allocate)" bullet beside the usage
  source bullet, and a short `AllocatorService` subsection linking `docs/allocator.md`) and `sdk/typescript/README.md`
  (a Features line and a `### AllocatorClient` section with an `allocate` snippet whose names match T047) (FR-029)
- [X] T054 [P] Create `sdk/go/pluginsdk/policy_benchmark_test.go` with `BenchmarkDecodePolicy` over a small nested
  policy (`{"version":1,"node_split":{"cpu_weight":0.3}}`) onto the T008 target type, reporting allocations
  (research R13)
- [X] T055 [P] In root `CLAUDE.md`, add an "Allocator SDK Pattern (052-allocator-allocate)" note under the usage-source
  pattern covering: rules live in `sdk/go/testing` and `pluginsdk` delegates; status-carrying plain errors
  (`GRPCStatus`) instead of `status.Error`; `refalloc` importable only from external test packages; conformance
  derives bad policies from the allocator's own effective policy; `export_test.go` exposes the scenario runner.
  Confirm the 052 "Active Technologies" and "Recent Changes" entries added during planning are present (do not run
  `update-agent-context.sh`; it mangles wrapped entries)
- [X] T056 Run the full regression from quickstart.md §8: `make test`, `golangci-lint run ./...` (baseline is 0 issues,
  so every finding comes from this change; use an extended timeout), `make lint-markdown`, and
  `bin/buf breaking --against '.git#branch=main'`. Fix all findings without editing `.golangci-lint.yml`
- [X] T057 Walk through every command and expected result in `specs/052-allocator-allocate/quickstart.md` §1–§7 and
  confirm each matches; fix code or update the quickstart for any deviation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none; T001/T002 → T003 → T004, T005
- **Foundational (Phase 2)**: needs T003. Tests T006–T009 first; then T010, T011 → T012 (needs T010), T013 (needs
  T011) → T014 (needs T012, T013)
- **US1 (Phase 3)**: needs Phase 2 (serves `refalloc`)
- **US2 (Phase 4)**: T021/T022/T024/T025 need only T011; T023/T026 need US1's T020 (served allocator)
- **US3 (Phase 5)**: needs US1 (T020)
- **US4 (Phase 6)**: needs US2's T025 (validation + conservation) and Phase 2's `refalloc`
- **US5 (Phase 7)**: needs only Phase 2 (T010); independent of US1–US4
- **US6 (Phase 8)**: needs only T004; independent of all Go stories
- **Polish (Phase 9)**: after all stories; T056 → T057 last

### User Story Dependencies

```text
Phase 1 ─► Phase 2 ─┬─► US1 ─┬─► US3
                    │        └─► US2 (T023, T026) ─┐
                    ├─► US2 (T021–T022, T024–T025) ─┴─► US4
                    └─► US5
Phase 1 (T004) ─────────► US6
```

### Within Each User Story

- Tests are written first and confirmed failing (or not compiling)
- Rule implementations before the wrappers that delegate to them
- Adapters before registration; registration before health
- Commit after each task or logical group

### Parallel Opportunities

- T001 ∥ T002; T004 ∥ T005 after T003
- T006–T009 are all in different files and can be written in parallel
- After Phase 2: US1, US2 (T021, T022, T024, T025), US5, and US6 can proceed in parallel
- Within US5: T039–T042 in parallel; within US4: T033 ∥ T036
- Polish docs T049–T055 in parallel

---

## Parallel Example: After Phase 2

```text
Developer A (US1 → US3): T015 → T016–T018 → T019 → T020 → T027–T031 → T032
Developer B (US2 → US4): T021 ∥ T022 → T024 → T025 → T033 → T034–T036 → T037 → T038
Developer C (US5):       T039 ∥ T040 ∥ T041 ∥ T042 → T043 → T044 → T045
Developer D (US6):       T046 → T047 → T048
Then: B finishes T023, T026 once A lands T020
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 (contract) and Phase 2 (interface, rules, decoder, reference allocator)
2. Phase 3 (US1): served allocator with transport and error parity
3. **Stop and validate**: `go test ./sdk/go/pluginsdk/ -run Allocator -v`. SP2 can prototype against the branch here

### Incremental Delivery

1. Setup + Foundational → contract and helpers usable
2. US1 → allocators served (MVP)
3. US2 → hosts can verify conservation from production code (SP3 unblocked)
4. US3 → minimal-plugin ergonomics, health, and coexistence proven
5. US4 → third parties can self-certify (SC-003, SC-004)
6. US5 → discovery and routing
7. US6 → TypeScript parity (SC-007); required before release (Constitution XIII)
8. Polish → docs, benchmarks, regression

---

## Notes

- [P] tasks touch different files and have no dependency on incomplete tasks
- The existing usage-source warning text and all existing behavior must stay unchanged (SC-006)
- `CHANGELOG.md` is generated by release-please; do not edit it. Use conventional commits (`feat(proto): …`)
- Commits only when the user asks; never add Claude attribution trailers
