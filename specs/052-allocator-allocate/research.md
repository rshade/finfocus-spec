# Research: Allocator Service (Allocate)

Phase 0 decisions for `specs/052-allocator-allocate/`. Every item in the plan's Technical Context is
resolved here; none remain open.

Sources: [spec.md](./spec.md), issue #506, the #505 implementation (`sdk/go/pluginsdk/usage_source.go`,
`sdk/go/testing/usage_source.go`, `specs/051-usage-source-getstats/research.md`), and the downstream
finfocus plans (SP2 Kubernetes allocator, SP3 `cost cluster`), which already code against the names
fixed below.

## R1. Proto shape and field names

**Decision**: Add `proto/finfocus/v1/allocation.proto` with exactly the shape in #506:
`AllocatorService.Allocate`, `AllocateRequest {usage, priced, policy_json, mode}`, `PricedResource
{resource, cost, currency, priced, note}`, `AllocateResponse {rows, effective_policy_json,
policy_digest, warnings}`, `AllocationRow {subject, cpu_cost, mem_cost, total_cost, currency, note}`.
Add `PLUGIN_CAPABILITY_ALLOCATION = 15` to `enums.proto`. The spec's "fingerprint" is the wire field
`policy_digest`. See [contracts/allocation.proto](./contracts/allocation.proto).

**Rationale**: SP2 and SP3 plans already use `GetPolicyJson`, `GetEffectivePolicyJson`,
`GetPolicyDigest`, and `GetPriced`. Renaming now would break plans written against #506 for no wire
benefit. `bytes` for policy keeps the document opaque to the host (FR-012) and avoids implying a
text encoding or a schema.

**Alternatives considered**:

- `google.protobuf.Struct` for the policy: makes the host parse the policy and loses number
  precision and key order control. Rejected by FR-012.
- `string` policy: equivalent on the wire, but invites hosts to treat it as text to edit. Rejected
  to match #506.
- Reserving GPU/storage/network cost fields now: the spec says additive later (Assumptions).

## R2. Serving: how the allocator joins `Serve`

**Decision**: Mirror `UsageSourceProvider`. Add `AllocatorProvider` to `sdk.go`, unexported adapters
`allocatorGRPCServer` and `allocatorConnectHandler` in a new `allocator.go`, and register both
services when the plugin implements the interface. To avoid a growing parameter list, replace the
`usage UsageSourceProvider` parameter of `serveGRPC`/`serveConnect` with an unexported
`optionalServices` struct (`usage`, `allocator`) built once in `Serve`.

Interceptors (FR-017): on gRPC, `grpc.ChainUnaryInterceptor` is a server option, so every service
registered on that server gets tracing plus the user interceptors. On Connect, the allocator handler
gets the same `handlerOpts` slice as the cost handler. No per-service wiring is needed.

Health (FR-017): append `pbcconnect.AllocatorServiceName` to the static health checker list when
served.

**Rationale**: #505 established this exact pattern, and hosts depend on its behavior. A struct
parameter keeps both serve functions' signatures stable as services are added.

**Alternatives considered**: a service registry (`[]func(*grpc.Server)`) is more general but adds
indirection for two services. It is overkill now.

## R3. Error codes that survive both transports and re-wrapping

**Decision**: Request validation, currency resolution, and policy decoding return a plain Go error
whose concrete type also implements `GRPCStatus() *status.Status` with `codes.InvalidArgument` and
the same message. `Error()` returns only the human message (no `rpc error: code = ...` prefix).

- Returned directly from `Allocate`, the error reaches gRPC clients as InvalidArgument
  (`status.FromError` honors `GRPCStatus`) and Connect clients as InvalidArgument (`toConnectError`
  uses `status.FromError`). This satisfies FR-018 with no new conversion code.
- SP2's pattern `status.Error(codes.InvalidArgument, err.Error())` still yields a clean message.
- `errors.Is` works against exported sentinels (`ErrInvalidAllocateRequest`, `ErrMixedCurrency`,
  `ErrInvalidPolicy`).

**Rationale**: FR-024 says "every rejection uses invalid-argument", and SP2 re-wraps the message.
Only a status-carrying plain error satisfies both without a double prefix.

**Alternatives considered**:

- Returning `status.Error(...)` directly: SP2's re-wrap would produce "rpc error: code =
  InvalidArgument desc = ..." inside another status. Rejected.
- Adding `GRPCStatus` to the existing `ContractError`: it would change the code that existing
  validators' errors map to when returned from handlers. That violates SC-006. Rejected.

## R4. Where each rule lives (one rule, two import paths)

**Decision**:

| Rule | Implementation | Public names |
|------|----------------|--------------|
| Conservation (FR-007, FR-022) | `sdk/go/testing/allocation.go` | `plugintesting.CheckConservation`, `pluginsdk.CheckConservation` (wrapper) |
| Request validation (FR-024) | `sdk/go/testing/allocation.go` | `plugintesting.ValidateAllocateRequest`, `pluginsdk.ValidateAllocateRequest` (wrapper) |
| Currency resolution (FR-011) | `sdk/go/testing/allocation.go` | `plugintesting.ResolveCurrency`, `pluginsdk.ResolveCurrency` (wrapper) |
| Response validation (FR-023) | `sdk/go/testing/allocation.go` | `plugintesting.ValidateAllocateResponse` |
| Strict policy decoding (FR-031) | `sdk/go/pluginsdk/policy.go` | `pluginsdk.DecodePolicy` |

The pluginsdk wrappers are one-line delegations (FR-032). `pluginsdk` already imports
`sdk/go/testing` in production code (`conformance.go`), so the delegation adds no new dependency
edge and no new binary weight.

**Rationale**: FR-032 fixes the delegation direction, and `testing` cannot import `pluginsdk` (the
existing cycle constraint). `DecodePolicy` is allocator production logic, not a test rule, and has no
delegation requirement, so it lives natively in `pluginsdk`. The conformance suite does not need it.
It checks decoding behavior through the wire.

**Alternatives considered**: a leaf `sdk/go/internal/allocation` package that both wrap. This is
cleaner, but it contradicts FR-032's stated delegation and gains nothing, since the edge
`pluginsdk → testing` already exists.

## R5. Strict policy decoder (`DecodePolicy`)

The standard decoder cannot report the path of an unknown field (clarification, verified on Go
1.27.1 including `GOEXPERIMENT=jsonv2`).

**Decision**: two passes.

1. **Syntax and trailing data**: a `json.Decoder` reads one value into `json.RawMessage`. A second
   `Decode` must return `io.EOF`, or the error is "trailing data". An empty, whitespace-only, or
   `null` document returns `nil` and leaves the target untouched.
2. **Unknown-field walk**: walk the parsed document against the target's `reflect.Type` and build
   paths such as `node_split.cpu` and `rules[2].match`. Rules:
   - Struct: keys match the effective JSON names that `encoding/json` would use: the tag name, or
     the Go field name when untagged. Embedded structs are promoted, `json:"-"` is excluded, and
     unexported fields are ignored. Matching is **exact and case-sensitive**. `"Version"` is
     unknown when the field is `version`, which is stricter than `encoding/json`'s
     case-insensitive fold. That is deliberate: a policy should not silently accept a near-miss.
   - Pointer: dereference the type.
   - `map[string]T`: any key is accepted. Recurse into `T` with path `parent.key`.
   - Slice or array: recurse into the element with path `parent[i]`.
   - `any`, `json.RawMessage`, and types implementing `json.Unmarshaler` or
     `encoding.TextUnmarshaler` are opaque. They accept any value and are not walked.
3. **Merge**: `json.Unmarshal` onto the caller's target, which already holds the defaults. Nested
   structs and non-nil struct pointers merge field by field, and maps merge by key. **Arrays are
   replaced wholesale**: before unmarshalling, the walk resets every slice the document supplies to
   `nil`, so no element from the defaults carries over. A dedicated test covers this, because
   `encoding/json` can reuse a slice's backing array. A JSON `null` on a field follows
   `encoding/json` semantics.
4. **Type errors**: a `*json.UnmarshalTypeError` is reported with its `Field` path, for example
   `node_split.cpu: expected number, got string`.

All failures from bad input carry InvalidArgument (R3) and wrap `ErrInvalidPolicy`. A nil or
non-pointer target is a programming error. It returns a plain error with no status, so it surfaces
as Unknown and is never mistaken for bad input.

**Rationale**: FR-031 requires the path, nested merge, and wholesale array replacement. SP2's
`Decode` depends on this exact contract.

**Alternatives considered**:

- `DisallowUnknownFields` alone: no path. Rejected by clarification.
- A third-party strict decoder: adds a dependency to `pluginsdk` for about 150 lines of reflection.
  Rejected (constitution IV, minimal dependencies).
- Rejecting duplicate keys: not required by the spec. `encoding/json` behavior (last wins) is kept
  and documented.

## R6. Conservation arithmetic

**Decision**:

- `expected = Σ cost` over `priced == true` entries. `actual = Σ total_cost` over all rows.
- The check passes when `|actual − expected| ≤ max(relEpsilon × |expected|, 1e-9)`.
- Exported constants: `DefaultConservationEpsilon = 1e-6` and `ConservationAbsoluteFloor = 1e-9`.
- **Non-finite guard**: a NaN or ±Inf cost, a NaN or ±Inf row total, or a `relEpsilon` that is
  NaN, ±Inf, or negative returns an error. In IEEE arithmetic, `NaN > tol` is false, so without
  this guard a NaN total would *pass*, which is exactly the failure the check exists to prevent (a
  failure rendering as a plausible number).
- On failure, the error is a `*ConservationError{Expected, Actual, Difference, Currency}` whose
  message states all three numbers. It wraps `ErrConservation`, so hosts can `errors.As` it for
  display.
- The per-row rule (FR-008) uses the same formula with `DefaultConservationEpsilon`, comparing
  `total_cost` to `cpu_cost + mem_cost`.
- A mismatched request currency is not a conservation error. `CheckConservation` first calls
  `ResolveCurrency` and returns that error unchanged.

**Rationale**: FR-007's numbers, plus making the check fail closed on inputs IEEE comparisons would
silently accept.

**Alternatives considered**: summing with Kahan compensation. With realistic row counts (thousands)
the error of a naive `float64` sum is far below 1 ppm. Rejected as unnecessary.

## R7. Request, currency, and response rules in detail

**ResolveCurrency(priced)**: collect distinct non-empty currencies over entries with
`priced == true`. None means `USD`. Exactly one means that code. More than one returns an error
that wraps `ErrMixedCurrency` and carries InvalidArgument, listing the currencies sorted. Currency
strings compare exactly; the SDK does no case folding and no ISO 4217 validation (not in the spec,
and it would reject inputs the spec accepts).

**ValidateAllocateRequest(req)** rejects, with InvalidArgument wrapping
`ErrInvalidAllocateRequest`:

- a nil request or a nil `priced` element
- `priced == false` with a nonzero cost
- a negative or non-finite cost
- mixed currencies (delegates to `ResolveCurrency`)
- two entries with equal `(resource.tags["kind"], resource.id)`. The check applies to all entries,
  priced or not, because FR-024 speaks of priced *resources*, the message type.

Usage rows are not validated here. That is not in FR-024, and the allocator owns the
interpretation.

**ValidateAllocateResponse(req, resp)** (FR-023) rejects, wrapping `ErrInvalidAllocateResponse`:

- a nil response
- an empty `policy_digest` or empty `effective_policy_json`
- a row whose `subject["kind"]` is missing or not `workload`, `__idle__`, or `__cluster__`
- an idle row without `node`
- a negative or non-finite cost field
- a non-cluster row with `total ≠ cpu + mem` beyond tolerance
- a row whose currency is empty or differs from `ResolveCurrency(req.priced)`
- a successfully priced node (`priced`, `tags.kind == "node"`) without exactly one idle row whose
  `node` equals its id

Subject keys beyond `kind` and `node` are not policed. #505's key vocabulary is documented, but
allocators may add keys, and FR-023 does not list them.

## R8. Conformance suite design

**Decision**: `RunAllocatorConformance(t *testing.T, impl AllocateServer)`, where
`AllocateServer` is a one-method interface (`Allocate(ctx, req) (resp, error)`). Every
`pbc.AllocatorServiceServer` and every `pluginsdk.AllocatorProvider` satisfies it, so SP2's call
`RunAllocatorConformance(t, allocServer{...})` compiles unchanged, and plain providers need no
`Unimplemented` embedding. The #506 signature named `pbc.AllocatorServiceServer`; the narrower
interface accepts a strict superset of callers.

- Calls go through `AllocatorHarness` (bufconn, FR-026), so every scenario exercises proto
  serialization and the status codes clients really see.
- Scenarios are an unexported table `[]allocatorScenario{name, run func(ctx, client) error}`.
  `RunAllocatorConformance` runs each one as `t.Run(name, …)`. A tiny `export_test.go` exposes the
  table runner to the package's external tests, so broken allocators can be asserted to fail
  *specific* scenarios without a fake `*testing.T` (SC-003).
- **Policy-agnostic fixtures** (FR-025): unknown-field and unknown-version policies are derived from
  the allocator's own effective policy. The suite first calls with an empty request, parses
  `effective_policy_json` as an object, and asserts it has an integer `version` (FR-012). It then
  sends:
  - effective policy plus `"conformance_unknown_field": true` (expect InvalidArgument, message
    contains `conformance_unknown_field`)
  - if the policy has object-valued fields: inject into the first one in sorted key order, `f`, as
    `f.conformance_unknown_field` (expect the message to contain `f.conformance_unknown_field`, which
    proves the path; the sorted choice keeps failures reproducible)
  - effective policy with `version` set to 2147483647 (expect InvalidArgument)

  This works for any allocator without knowing its schema.
- **Realistic usage fixtures**: fixture usage rows are what a valid usage source returns (`kind` on
  every row, `node` on node rows, a distinct `namespace`/`pod` per workload, no repeated
  (subject, metric)), checked with `ValidateStatsResponse` before each call. An allocator that
  groups usage by workload identity (as the finfocus Kubernetes allocator does, by
  `namespace/pod`) would otherwise see workloads collapse and still pass conservation, so the
  scenarios would prove little.
- Scenario names: `single_node`, `three_nodes`, `empty_cluster`, `fully_packed_node`,
  `unpriced_node`, `control_plane`, `over_requested_node`, `policy_unknown_field`,
  `policy_unknown_version`, `empty_request`, `fingerprint_stable`, `fingerprint_empty_equals_braces`.
- Common assertions per allocation scenario: `ValidateAllocateRequest` on the fixture (a guard
  against suite bugs), no error, `ValidateAllocateResponse`, and `CheckConservation` at
  `DefaultConservationEpsilon`. Scenario-specific assertions stay value-free:
  - `empty_cluster`: the sum of idle rows equals the priced node total.
  - `control_plane`: at least one `__cluster__` row exists.
  - `over_requested_node`: the idle row exists and is non-negative.
  - `empty_request`: no rows, and the digest is 64 lowercase hex characters.
  - Fingerprint scenarios: string equality.

**Rationale**: FR-025 forbids asserting allocator-specific values, and FR-013 requires the path in
the message. Deriving the policy from the allocator itself is the only way to test rejection
without knowing each schema.

**Alternatives considered**:

- Hard-coding `{"version":1,"x":1}`: a correct allocator whose current version is 2 would fail on
  version, not the unknown field. Rejected.
- Exporting a results-returning API (`CheckAllocatorConformance`): new public surface only the SDK's
  own tests need. Rejected in favor of `export_test.go`.

## R9. Reference allocator

**Decision**: `sdk/go/internal/refalloc` (Go's `internal` rule makes it unimportable outside
`sdk/go`, so it is not published as a product). It deliberately uses the public helpers:
`pluginsdk.ValidateAllocateRequest`, `pluginsdk.DecodePolicy`, and `pluginsdk.ResolveCurrency`.
That makes it a smoke test of the same path SP2 takes.

- Policy: `{"version": 1, "node_split": {"cpu_weight": 0.5}}`. `cpu_weight` must be in [0, 1], and
  memory gets `1 − cpu_weight`. The nested object gives the conformance suite a path to prove.
- Canonical form: `json.Marshal` of the effective policy struct (fixed field order). The digest is
  hex SHA-256 of those bytes.
- Math per priced node: `cpu_pool = cost × cpu_weight` and `mem_pool = cost − cpu_pool`. Each
  workload gets `pool × request / allocatable`. If the summed requests exceed allocatable, shares
  are scaled by `allocatable / Σ requests`. Idle is `pool − Σ shares`, clamped at 0, and one idle
  row is always emitted.
- Cluster rows: priced resources with `tags.kind == "cluster"` become a `__cluster__` row carrying
  the full cost in `total_cost`. Any other kind becomes a `__cluster__` row with a note.
- Zero-cost rows with a note: workloads on an unpriced or unknown node.

**Import consequence**: `refalloc` imports `pluginsdk`, so only *external* test packages
(`pluginsdk_test`, `testing_test`) may import it. Both styles already exist in the repository.

**Rationale**: FR-027. The simplest math that satisfies the invariants (spec Assumptions).

**Alternatives considered**: a `_test.go` file inside `sdk/go/testing`. `pluginsdk`'s serve tests
could not reuse it, and US1's independent test serves the reference allocator through the SDK.

## R10. Capability, legacy flag, and startup warning

**Decision**:

- Bump `optionalCapabilities` from 7 to 8 and set `maxValidCapability = PLUGIN_CAPABILITY_ALLOCATION`.
  `inferCapabilities` gains one type assertion.
- `legacyCapabilityNames` gains `"supports_allocation"`.
- Update the bounds test: 15 valid, "just above max (16)" invalid.
- Generalize `warnUsageSourceCapabilities` into
  `warnInferredOnlyCapabilities(logger, plugin, info)`. It logs one warning per implemented
  service-only provider (usage source, allocator) when the plugin has no explicit capabilities and
  is not a `PluginInfoProvider`. The existing usage-source message text is unchanged, so existing
  tests keep passing.

**Rationale**: FR-019 through FR-021. `TestLegacyCapabilityMapCompleteness` fails until the legacy
name exists (CLAUDE.md, 051 pattern). Inference stays zero-allocation into a pre-sized slice
(constitution VIII).

## R11. TypeScript

**Decision**: Regenerate with `make generate`, which produces `allocation_pb.ts`. Add
`src/clients/allocator.ts` with `AllocatorClient` and one `allocate(request)` method, mirroring
`UsageSourceClient`. Export it and the generated module from `index.ts`, and add the test
`test/allocator.test.ts` (msw round-trip plus error-code propagation). `KIND_IDLE` and
`KIND_CLUSTER` already exist in `usage-subjects.ts`.

TypeScript conservation and validation helpers are **not** added. FR-028 requires only the client,
and no TypeScript allocation host exists (US6 priority rationale).

**Known gotcha**: type-check with `npx tsc --noEmit`, because `npm run build` fails at tsup's DTS step
on `main` (CLAUDE.md).

## R12. Documentation

**Decision**:

- New `docs/allocator.md`: contract semantics, invariants, policy rules, currency, errors, and
  worked single-node example numbers.
- Updated pages:
  - `sdk/go/pluginsdk/README.md`: capability table row and the allocator-only rule
  - root `README.md`: service list
  - `sdk/typescript/README.md`: `AllocatorClient` section
  - `PLUGIN_DEVELOPER_GUIDE.md`: a "Writing an allocator" section
  - `sdk/go/testing/README.md`: helpers, harness, and conformance
- Compiled examples in `pluginsdk/example_test.go` for `AllocatorProvider` with `BasePlugin`
  embedding and for `DecodePolicy` (constitution XIV README sync).

## R13. Performance

**Decision**: add benchmarks for `CheckConservation` and `ValidateAllocateResponse` at 1,000 and
10,000 rows, and for `DecodePolicy` on a small nested policy (constitution VIII). The only
hot-path change is capability inference, which stays zero-allocation. There are no latency targets
beyond linear time: these helpers run once per allocation, not per row of cost data.

## R14. Branch and release sequencing

The workspace is on `main`, while the feature directory is `052-allocator-allocate`. Implementation
should happen on a `052-allocator-allocate` branch (`git checkout -b 052-allocator-allocate`) before
`/speckit-implement`. Shipping in the same release as `GetStats` or right after it is acceptable
(spec Assumptions). SP1 Task 4's version bump must name a release that contains both.
