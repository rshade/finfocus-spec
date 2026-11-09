---

description: "Task list for capability-scoped Supports requests"
---

# Tasks: Capability-Scoped Supports Requests

**Input**: Design documents from `specs/606-supports-request-capability/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/supports-capability.md, quickstart.md

**Tests**: Required. Constitution principle V makes test-first non-negotiable. One request→response contract
test (T002) is written before the proto change and fails against the current proto, because it does not compile
without `Capability`. The proto change then makes it pass. Each story's remaining tests come after the proto
change and fail first against the missing SDK code. Writing every story's tests before the proto would make the
stories share one uncompilable test package (see CLAUDE.md, 557), so only the contract test comes first.

**Organization**: Tasks are grouped by user story so each story can be built and tested on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: The user story the task belongs to (US1–US4)

## Phase 1: Setup

**Purpose**: A known-good baseline in the worktree

- [X] T001 Record the baseline in the worktree at `origin/main`. Run `make generate && git diff --exit-code -- sdk/`
  and `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/`; both must pass before any change.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The wire field that every story reads or sends

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Write the failing contract test first. Create `sdk/go/pluginsdk/supports_capability_test.go` (Apache 2.0
  header, package `pluginsdk`) with `TestSupportsCapabilityEndToEnd`. A test plugin embeds `*BasePlugin` and
  implements `SupportsProvider`, reading `req.GetCapability()` directly (the SDK helper does not exist yet):
  `ACTUAL_COSTS` answers `{Supported: true}`, `PROJECTED_COSTS` answers
  `{Supported: false, Reason: "projected cost not priced"}`, and `UNSPECIFIED` answers `{Supported: true}`.
  Serve it with an injected listener (`ServeConfig.Listener`), on gRPC and on Connect (`Web.Enabled`). Send raw
  `SupportsRequest{Resource: ..., Capability: ...}` messages through the generated gRPC client and
  `pbcconnect.NewCostSourceServiceClient`, and assert the three answers on both transports (FR-001, FR-002,
  FR-010, SC-001). The Connect cleanup must call `http.DefaultClient.CloseIdleConnections()` before
  `cancel()`, as the other Connect-mode Serve helpers do. Run
  `go test ./sdk/go/pluginsdk/ -run TestSupportsCapabilityEndToEnd` and confirm it fails, because
  `SupportsRequest` has no `Capability` field (constitution V, step 2).
- [X] T003 Add `PluginCapability capability = 2;` to `message SupportsRequest` in `proto/finfocus/v1/costsource.proto`,
  with the inline comment from `contracts/supports-capability.md`: "`PLUGIN_CAPABILITY_UNSPECIFIED` means any
  capability, answer as before this field existed; hosts treat an unspecified answer as covering every
  capability; a plugin that does not recognize the value answers as if it were unspecified." `enums.proto` is
  already imported.
- [X] T004 Run `make generate`. Confirm that `sdk/go/proto/finfocus/v1/costsource.pb.go` has
  `GetCapability() PluginCapability` and that the TS bindings under `sdk/typescript/packages/client/src/generated/`
  have `capability`. Run `make buf-lint`, then `buf breaking` against a `git archive origin/main proto buf.yaml`
  copy in a scratch directory (the worktree workaround from CLAUDE.md). Zero breaking changes are allowed
  (FR-003). `buf.gen.yaml` must still pin `buf.build/connectrpc/go:v1.21.0`; if any `*.connect.go` file shows
  a v2 rewrite, revert it. Re-run T002: it now passes with no SDK change, because `Server.Supports` and
  `ConnectHandler.Supports` forward the request as is.

**Checkpoint**: The field exists in Go and TS, the contract test passes, and nothing else has changed behavior.

---

## Phase 3: User Story 1 - Plugin answers per capability (Priority: P1) 🎯 MVP

**Goal**: A plugin reads the requested capability and answers differently per capability (FR-004–FR-006, FR-010,
FR-011).

**Independent Test**: `RequestedCapability` returns known values unchanged, maps unspecified and unknown values to
unspecified, warns once per unknown value, and allocates nothing on the known path. A failing Supports handler's
log names the capability. Transport pass-through on gRPC and Connect is already proven by T002.

### Tests for User Story 1 (write first, must fail)

- [X] T005 [US1] Add to `sdk/go/pluginsdk/supports_capability_test.go`. `TestRequestedCapability` is a
  table test:
  - A nil request gives `UNSPECIFIED`.
  - `UNSPECIFIED` gives `UNSPECIFIED`.
  - Every value from `PLUGIN_CAPABILITY_PROJECTED_COSTS` through `maxValidCapability` is returned unchanged.
  - `PluginCapability(999)` and `PluginCapability(-1)` give `UNSPECIFIED`.

  `TestRequestedCapabilityWarnsOncePerValue` swaps the zerolog global logger (`log.Logger`) for a buffer
  writer, with `t.Cleanup` to restore it. It sends 997 twice and 998 once, and expects exactly two warnings
  carrying the raw `capability` int32 values 997 and 998 (FR-005, "at most once per distinct value per
  process"). Use values no other test uses, because the seen-set is process-wide. These tests must fail
  because `RequestedCapability` is undefined.
- [X] T006 [US1] Add `BenchmarkRequestedCapability` with sub-benchmarks `known` (ACTUAL_COSTS) and
  `unspecified` to `sdk/go/pluginsdk/supports_capability_test.go`, using `b.ReportAllocs()`. The SC-005 gate
  is 0 allocs/op for both (FR-006). Unknown values are exempt and are not benchmarked.
- [X] T007 [US1] Add `TestSupportsHandlerErrorLogsCapability` to `sdk/go/pluginsdk/supports_capability_test.go`.
  A `SupportsProvider` returns `errors.New("boom")`. Build the server with
  `NewServerWithOptions(plugin, nil, &logger, nil)`, where `logger` writes to a buffer, and call
  `Server.Supports` directly with a request whose capability is `PROJECTED_COSTS`. Assert that the
  "Supports handler error" entry has `"capability":"PLUGIN_CAPABILITY_PROJECTED_COSTS"` (FR-011).

### Implementation for User Story 1

- [X] T008 [US1] Create `sdk/go/pluginsdk/supports_capability.go` (Apache 2.0 header) with the exported
  `RequestedCapability(req *pbc.SupportsRequest) pbc.PluginCapability`, using the godoc from the contract:
  - Return `req.GetCapability()` when `IsValidCapability` reports true.
  - Return `UNSPECIFIED` for `UNSPECIFIED`, with no log.
  - Otherwise, do `LoadOrStore` on a package-level `sync.Map` keyed by the raw `int32`. Log
    `log.Warn().Int32("capability", v).Msg(...)` only when the value was not already loaded, then return
    `UNSPECIFIED`.

  This makes T005 and T006 pass.
- [X] T009 [US1] Add `FieldCapability = "capability"`, with a one-line godoc comment (constitution XIV), to the
  field constants in `sdk/go/pluginsdk/logging.go`, and add
  `Str(FieldCapability, req.GetCapability().String())` to the "Supports handler error" log in `Server.Supports`
  in `sdk/go/pluginsdk/sdk.go`. Change nothing on the success path. This makes T007 pass.

**Checkpoint**: US1 works with raw requests. A plugin can answer per capability.

---

## Phase 4: User Story 2 - Host sends the capability (Priority: P1)

**Goal**: Hosts send a capability with one SDK call; existing calls are unchanged (FR-007–FR-009).

**Independent Test**: Through `pluginsdk.Client`, a recording plugin receives `PROJECTED_COSTS` from `SupportsFor`
and `UNSPECIFIED` from `Supports` and `SupportsResourceType`.

### Tests for User Story 2 (write first, must fail)

- [X] T010 [P] [US2] Create `sdk/go/pluginsdk/client_supports_for_test.go` (Apache 2.0 header). Follow the
  Serve-plus-`Client` setup in `sdk/go/pluginsdk/client_test.go` with a `SupportsProvider` plugin that records
  the last `SupportsRequest`. `TestClientSupportsFor` checks:
  - `SupportsFor(ctx, res, PROJECTED_COSTS)` delivers `PROJECTED_COSTS`.
  - `Supports(ctx, res)` and `SupportsResourceType(ctx, type)` deliver `UNSPECIFIED`.
  - `SupportsFor` with a nil resource, or an empty `resource_type`, returns the same errors as `Supports`
    ("resource descriptor cannot be nil" and "resource type is required"), without calling the plugin (FR-008).
  - `SupportsFor` forwards `PluginCapability(999)` unchanged; the client does not normalize.

  This must fail because `SupportsFor` is undefined.
- [X] T011 [P] [US2] Create `sdk/typescript/packages/client/test/supports-capability.test.ts` in the style of
  `test/recommendation-spot.test.ts`:
  - A default `SupportsRequest` has `capability === PluginCapability.UNSPECIFIED`.
  - A request with `PluginCapability.PROJECTED_COSTS` round-trips through `toJson` and `fromJson`.
  - `CostSourceClient.supports(req)` forwards `capability`. The shared Supports handler
    (`test/mocks/handlers.ts:31`) ignores the request, so the test sets up its own msw server, as
    `test/batch.test.ts` does (`setupServer` plus `server.use(...)`). The override handler for
    `.../finfocus.v1.CostSourceService/Supports` captures `await request.json()`, the pattern at
    `test/mocks/handlers.ts:126`, and the test asserts that the captured body has
    `capability: "PLUGIN_CAPABILITY_PROJECTED_COSTS"`.

### Implementation for User Story 2

- [X] T012 [US2] In `sdk/go/pluginsdk/client.go`, add `SupportsFor(ctx, resource, capability)` with godoc (see the
  contract), moving the existing nil and resource-type checks into it. `Supports` becomes
  `return c.SupportsFor(ctx, resource, pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED)`.
  `SupportsResourceType` is unchanged. This makes T010 pass.
- [X] T013 [US2] In `sdk/typescript`, run `npm ci`, then `npm run build`, then `npx vitest run` in
  `packages/client`. No client source change is expected, because `supports(req)` already forwards the whole
  request. Change `src/clients/cost-source.ts` only if T011 shows the field is dropped.

**Checkpoint**: US1 and US2 together deliver the issue's fix path: hosts ask, plugins answer.

---

## Phase 5: User Story 3 - Existing plugins keep working and pass conformance (Priority: P2)

**Goal**: Plugins that ignore the field still pass; conformance exercises capability-specific questions
(FR-012–FR-015).

**Independent Test**: `RPCCorrectness_SupportsCapability` passes the default mock and a per-capability mock, and
fails a plugin that errors on a capability-specific request, naming the capability.

### Tests for User Story 3 (write first, must fail)

`contract_test.go`, `mock_plugin_test.go`, and `supports_capability_test.go` are all in package `testing_test`.
Once T015 or T016 uses `MockPlugin.UnsupportedCapabilities`, the whole package fails to compile until T017
adds the field. That compile failure is the expected failing state, and T014's pass can only be observed
after T017.

- [X] T014 [P] [US3] Add `TestValidateSupportsRequestAcceptsAnyCapability` to `sdk/go/testing/contract_test.go`. A
  valid descriptor paired with `UNSPECIFIED`, each known value, `PluginCapability(999)`, and
  `PluginCapability(-1)` returns nil (FR-012). This should pass immediately, because it pins existing behavior.
  Record that in the task note rather than forcing it to fail.
- [X] T015 [P] [US3] Add `TestMockPluginUnsupportedCapabilities` to `sdk/go/testing/mock_plugin_test.go`:
  - With `UnsupportedCapabilities: {PROJECTED_COSTS}`, a supported provider and type asked about
    `PROJECTED_COSTS` gets `Supported: false`, with a reason containing `PLUGIN_CAPABILITY_PROJECTED_COSTS`
    and the resource type.
  - The same resource asked about `ACTUAL_COSTS` or `UNSPECIFIED` gets `Supported: true`.
  - An unsupported provider still gets the provider reason, because the capability is considered only after
    the provider and resource checks.
  - A nil list ignores the capability (FR-013).
- [X] T016 [P] [US3] Create `sdk/go/testing/supports_capability_test.go` (Apache 2.0 header, package `testing_test`).
  `TestRPCCorrectness_SupportsCapability` finds the check by name in `plugintesting.RPCCorrectnessTests()`,
  asserts `MinLevel == ConformanceLevelStandard` and `Category == CategoryRPCCorrectness`, and runs each case
  on its own started `TestHarness`, with no `t.Parallel()`. It uses a recording wrapper plugin that embeds
  `*plugintesting.MockPlugin` and overrides `Supports` and `GetPluginInfo`. The cases:
  - (a) The default `NewMockPlugin()` passes.
  - (b) A mock with `UnsupportedCapabilities: {PROJECTED_COSTS}` passes, because answers are never judged.
  - (c) A legacy plugin (no declared capabilities) passes and receives `UNSPECIFIED`, `PROJECTED_COSTS`, and
    `ACTUAL_COSTS`.
  - (d) A plugin declaring `{CARBON, BUDGETS}` receives exactly `UNSPECIFIED` and `CARBON`.
  - (e) A plugin declaring only `{DRY_RUN}` receives only `UNSPECIFIED`, and passes.
  - (f) A plugin returning `codes.Internal` for `PROJECTED_COSTS` fails, and the error contains
    `PLUGIN_CAPABILITY_PROJECTED_COSTS`.
  - (g) A plugin answering `{Supported: false}` with no reason for `ACTUAL_COSTS` fails, naming
    `PLUGIN_CAPABILITY_ACTUAL_COSTS`.
  - (h) A plugin whose `GetPluginInfo` returns `codes.Internal` fails with "Capability discovery failed".
  - (i) A plugin whose `GetPluginInfo` returns `codes.Unimplemented` is treated as legacy, as in (c).
  - (j) An erroring unspecified request fails.

  Use the `custom` provider via `harness.SampleResource()`; no vendor content. This must fail because the
  check is not registered.

### Implementation for User Story 3

- [X] T017 [US3] In `sdk/go/testing/mock_plugin.go`, add `UnsupportedCapabilities []pbc.PluginCapability`, with
  a field comment, to `MockPlugin`. In `MockPlugin.Supports`, after the provider and resource checks pass,
  return `{Supported: false, Reason: fmt.Sprintf("capability %s is not supported for %s",
  req.GetCapability(), resourceType)}` when `slices.Contains(m.UnsupportedCapabilities, req.GetCapability())`.
  This makes T015 pass.
- [X] T018 [US3] Create `sdk/go/testing/supports_capability.go` (Apache 2.0 header) with:
  - a private `resourceScopedCapabilities` slice in enum order: PROJECTED_COSTS, ACTUAL_COSTS, CARBON,
    RECOMMENDATIONS, ENERGY, WATER, PRICING_SPEC, ESTIMATE_COST, BATCH_COST, HISTORICAL_CARBON;
  - `supportsCapabilityTest()` returning a `ConformanceSuiteTest` named `RPCCorrectness_SupportsCapability`,
    `MinLevel: ConformanceLevelStandard`, `Category: CategoryRPCCorrectness`;
  - `testSupportsCapability(harness)`, which implements research R7 using `harness.discoverCapabilities()` and
    `ValidateSupportsResponse`, with each failure's `Error` naming the capability.

  Add `supportsCapabilityTest()` to the append in `RPCCorrectnessTests()` in
  `sdk/go/testing/rpc_correctness.go`; that is the only edit to that file. Keep the function under
  `gocognit` 20 by splitting out a per-capability helper. This makes T016 pass.
- [X] T019 [US3] Run `go test ./sdk/go/testing/`, `go test -tags=integration ./sdk/go/testing/`,
  `go test -run TestConformance ./sdk/go/testing/`, and `go test ./sdk/go/pluginsdk/ -run Conformance` with no
  plugin changes (FR-015, SC-002). Any existing plugin that now fails is a defect in the check, not in the plugin.

**Checkpoint**: Conformance covers both kinds of request, and existing plugins are unaffected.

---

## Phase 6: User Story 4 - Plugin authors and hosts know the rules (Priority: P3)

**Goal**: The docs describe the field, the rules, and the calls (FR-016, FR-017, SC-006).

**Independent Test**: The guide's Supports section answers what an unspecified request means, shows a
per-capability example, and names `SupportsFor`, and its snippets compile.

- [X] T020 [P] [US4] In `PLUGIN_DEVELOPER_GUIDE.md`, under "Granular Capability Discovery (`Supports`)" and
  "Supports RPC", explain:
  - the `capability` field and that unspecified means any;
  - that hosts treat an unspecified answer as covering every capability;
  - that an old plugin answers a capability-specific request with its "any" answer;
  - per-capability answering with `pluginsdk.RequestedCapability` (a Go example);
  - the consistency guidance (a specific "supported" implies "supported" for unspecified);
  - the resource-scoped set;
  - the host call `Client.SupportsFor`.
- [X] T021 [P] [US4] In `sdk/go/pluginsdk/README.md`, in the SupportsProvider section (near "type SupportsProvider
  interface"), document `RequestedCapability` (with the unknown-value and once-per-value warning behavior)
  and `Client.SupportsFor`. Add both to any API index table in that README.
- [X] T022 [P] [US4] In `sdk/go/testing/README.md`, add `RPCCorrectness_SupportsCapability` (Standard) to the check
  list, with its query set and pass and fail rules, and document `MockPlugin.UnsupportedCapabilities`. If
  `docs/conformance.md` lists checks, add it there too.
- [X] T023 [P] [US4] In `sdk/typescript/README.md` (the TS SDK README; the client package has none), add a
  `supports` example that sets
  `capability: PluginCapability.PROJECTED_COSTS`, using `create(SupportsRequestSchema, ...)`.
- [X] T024 [US4] Compile-check every new snippet. Put the Go snippets in a scratch `main` package outside the repo
  that imports the worktree module via a `replace`, and run `go build`. Check the TS snippet with
  `tsc --ignoreConfig` as an `.mts` file with `export {};`, from `sdk/typescript/packages/client` (memory:
  TS README compile check). Then run `make lint-markdown`.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T025 Add a "Capability-Scoped Supports Pattern (606-supports-request-capability)" section to `CLAUDE.md`,
  after the newest pattern section. It records:
  - the field and its unspecified rule;
  - `RequestedCapability` with its once-per-value `sync.Map` warning, and why `NormalizeUsageProfile`'s
    warn-every-call was not copied;
  - `SupportsFor`, rather than a variadic option, and why;
  - the resource-scoped set living privately in `sdk/go/testing/supports_capability.go`, mirrored in the guide;
  - that the check never judges answer values.

  Confirm that the Active Technologies and Recent Changes entries already added to `CLAUDE.md`, `AGENTS.md`,
  and `GEMINI.md` match what shipped.
- [X] T026 Run the full gate list in the worktree:
  - `make generate && git diff --exit-code -- sdk/`
  - `make buf-lint`
  - `buf breaking` against both the archive and `https://github.com/rshade/finfocus-spec.git#branch=main`
  - `make test`
  - `go test -tags=integration ./sdk/go/testing/`
  - `go test -run TestConformance ./sdk/go/testing/`
  - `make lint-go`
  - `make lint-markdown`
  - `make lint-yaml`
  - `make validate-npm`
  - the TS build and vitest for every workspace (`npm test --workspaces` in `sdk/typescript`)
- [X] T027 Run `go test ./sdk/go/pluginsdk/ -run '^$' -bench 'BenchmarkRequestedCapability' -benchmem` and record
  0 allocs/op for both sub-benchmarks in the PR body (SC-005).
- [X] T028 Write `PR_MESSAGE.md` as `feat(proto): ...` (no `!`; additive), describing the field, the unspecified
  rule, `RequestedCapability`, `SupportsFor`, the mock option, and the new Standard check (FR-018). End it
  with `Closes #675`, then validate with `cat PR_MESSAGE.md | npx commitlint`.

---

## Dependencies & Execution Order

- Phase 1 comes before Phase 2, which blocks everything else. Within Phase 2 the order is T002 (fails), T003, T004
  (T002 passes).
- US1 (Phase 3) and US2 (Phase 4) depend only on Phase 2, and their test files are separate, so their test
  tasks can run in parallel. T009 and T012 touch different files (`sdk.go`/`logging.go` vs `client.go`).
- US3 (Phase 5) depends only on Phase 2. Its tests use raw requests and the mock, not `SupportsFor`. T014–T016 are
  separate files but share package `testing_test`, so they compile together only after T017.
- US4 (Phase 6) depends on US1–US3 for the names it documents.
- Phase 7 depends on all stories.

Within each story the order is tests, then implementation, then the checkpoint. T014 is the one test expected to
pass at once, because it pins existing behavior.

## Parallel Example

```text
After T004:
  T005, T006, T007        (US1 tests, appended to the T002 file - write sequentially, independent of others)
  T010                    (US2 Go test, separate file)
  T011                    (US2 TS test)
  T014, T015, T016        (US3 tests, three different files)
Docs after T018:
  T020, T021, T022, T023  (four different files)
```

## Implementation Strategy

1. **MVP**: Phases 1–3 (field plus plugin read path). After Phase 2, a plugin can already answer per capability
   with `req.GetCapability()` for hosts that build raw requests; US1 adds the forward-compatible helper.
2. **Usable fix path**: add Phase 4, so hosts send the capability with one call. This is what the core repo's
   follow-up needs.
3. **Safe release**: add Phase 5, where conformance proves existing plugins are unaffected.
4. **Done**: Phases 6–7 (docs, gates, PR text). Everything ships in one PR, as constitution principles VII and
   XIII require.

## Phase 8: Convergence

- [X] T029 Justify `SetSampleResourceForTest` in `sdk/go/testing/export_test.go`. It is a test-only seam (an
  `_test.go` file, so not exported API) that lets external tests set the harness sample, as `ConformanceSuite`
  does, so T016 can run one check with the `custom` provider. Record this in the CLAUDE.md 606 pattern section
  per plan: Project Structure (unrequested)
- [X] T030 Justify `detailsCapabilityDiscoveryFailed` in `sdk/go/testing/capabilities.go`. goconst required it
  once the new check added a third copy of "Capability discovery failed", and the T026 lint gate needs 0
  issues. Confirm the detail text is unchanged at all three sites (`capabilities.go`, `rpc_correctness.go`,
  `supports_capability.go`) per plan: Project Structure (unrequested)

## Phase 9: Post-Commit Security Review

- [X] T031 Bound the unknown-capability dedup set in `sdk/go/pluginsdk/supports_capability.go` at 64 values
  (`maxUnknownCapabilitiesTracked`, counted by an `atomic.Int32`), logging one "no longer logging" warning at
  the bound. Test first: `TestRequestedCapabilityBoundsTrackedValues` per FR-005 (partial)
