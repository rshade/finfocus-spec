# Idea Intake: AllocatorService.Allocate for cost allocation plugins

- **Slug**: allocator-allocate
- **Created**: 2026-09-25
- **Source**: GitHub issue rshade/finfocus-spec#506 (pasted issue body via `gh issue view`;
  no URL fetched)
- **Type**: new-capability

## Idea (as captured)

> **feat(proto): add AllocatorService.Allocate for cost allocation plugins**
>
> Labels: `effort/medium`, `enhancement` · Author: rshade · State: OPEN
>
> **Context.** Companion to the UsageSourceService.GetStats issue. An *allocator* plugin divides
> priced resources (nodes, control plane) across workloads according to a policy. finfocus core
> stays domain-agnostic: it gathers usage (GetStats), prices `priceable` through existing
> cost-source plugins, then calls `Allocate`. Design: rshade/finfocus
> `docs/superpowers/specs/2026-09-24-k8s-cost-allocation-design.md`.
>
> **Proto** (new file `proto/finfocus/v1/allocation.proto`, importing `costsource.proto` for
> `ResourceDescriptor` and `usage.proto` for `UsageRow`, `StatsMode`):
>
> ```proto
> service AllocatorService {
>   rpc Allocate(AllocateRequest) returns (AllocateResponse);
> }
>
> message PricedResource {
>   ResourceDescriptor resource = 1;
>   double cost = 2;       // cost for the normalized period
>   string currency = 3;
>   bool priced = 4;       // false => pricing failed; cost MUST be 0
>   string note = 5;
> }
>
> message AllocateRequest {
>   repeated UsageRow usage = 1;
>   repeated PricedResource priced = 2;
>   bytes policy_json = 3;   // standard JSON; empty => plugin defaults
>   StatsMode mode = 4;
> }
>
> message AllocateResponse {
>   repeated AllocationRow rows = 1;
>   bytes effective_policy_json = 2;   // defaults + overrides as applied
>   string policy_digest = 3;          // hex sha256 of canonical effective policy
>   repeated string warnings = 4;
> }
>
> message AllocationRow {
>   map<string, string> subject = 1;   // kind: workload | __idle__ | __cluster__
>   double cpu_cost = 2;
>   double mem_cost = 3;
>   double total_cost = 4;
>   string currency = 5;
>   string note = 6;
> }
> ```
>
> Add to `enums.proto` `PluginCapability`: `PLUGIN_CAPABILITY_ALLOCATION = 15;`
>
> **Contract rules (proto comments + docs).**
>
> 1. **Conservation**: Σ `rows.total_cost` = Σ `priced.cost` where `priced=true`, within relative
>    epsilon 1e-6. Hosts verify this and reject violations.
> 2. `total_cost = cpu_cost + mem_cost` for every row (control-plane rows may put everything in
>    `total_cost` with cpu/mem 0 — then the equality is waived for `kind=__cluster__` only).
> 3. No negative costs. Idle is a row (`kind=__idle__`, with `node` subject), never silently dropped.
> 4. **Policy**: `policy_json` is opaque to the host. The allocator owns the schema, decodes
>    strictly (unknown fields → `InvalidArgument` naming the JSON path), rejects unknown
>    `version`, and merges onto its defaults.
> 5. Empty `usage` and empty `priced` → a response with no rows but with `effective_policy_json`
>    and `policy_digest` (hosts use this to show policy).
> 6. The digest is stable: the same effective policy always yields the same digest.
>
> **pluginsdk work.**
>
> 1. `type AllocatorProvider interface { Allocate(ctx, *pbc.AllocateRequest) (*pbc.AllocateResponse, error) }`
> 2. Serving: register `AllocatorService` in `serveGRPC`, `serveConnect`, and the grpchealth
>    checker when the plugin implements `AllocatorProvider` (same mechanism as
>    UsageSourceProvider).
> 3. `inferCapabilities` appends `PLUGIN_CAPABILITY_ALLOCATION`; legacy name
>    `"supports_allocation"`; `maxValidCapability = PLUGIN_CAPABILITY_ALLOCATION`; update the
>    bounds test ("just above max" → 16).
> 4. **Conformance** (`sdk/go/testing/allocator_conformance.go`): `CheckConservation(req, resp,
>    relEpsilon) error` and `RunAllocatorConformance(t, impl pbc.AllocatorServiceServer)` with
>    fixtures: single node; three nodes; empty cluster (all idle); fully packed node (idle 0);
>    node with `priced=false`; control plane present; requests exceeding allocatable (idle ≥ 0);
>    `policy_json` with unknown field (`InvalidArgument`); unknown `version` (`InvalidArgument`);
>    empty usage+priced returns effective policy and digest; digest stable across two identical
>    calls; `{}` policy digest equals empty policy digest.
>
> **Acceptance criteria.** Generated Go/Connect/TS code for `allocation.proto`; `buf lint` and CI
> `buf breaking` pass; allocator served via `pluginsdk.Serve` answers over gRPC and Connect
> (tests); capability 15 inferred and valid, bounds test updated; `RunAllocatorConformance`
> passes against an in-repo reference allocator; `CheckConservation` table-driven tests (pass,
> over, under, priced=false ignored); README capability table + developer guide section
> "Writing an allocator"; speckit folder `specs/052-allocator-allocate/`.
>
> **Depends on**: rshade/finfocus-spec#505 (imports `UsageRow`, `StatsMode`).

## Restated

Add a new optional gRPC/Connect service, `AllocatorService.Allocate`, so that a plugin can take
workload usage rows plus already-priced infrastructure resources and a plugin-owned JSON policy,
and return per-workload (plus idle and cluster) cost rows that sum back to the priced total. The
SDK would gain a provider interface, auto-registration in `Serve`, a new capability value (15),
and a conformance suite with a conservation checker.

## Origin & Context

- **Raised by**: rshade (repository owner), issue #506, labelled `effort/medium`, `enhancement`.
- **Trigger**: Second half of a Kubernetes cost-allocation design in the host repo
  (rshade/finfocus `docs/superpowers/specs/2026-09-24-k8s-cost-allocation-design.md`, dated
  2026-09-24); the first half is the UsageSourceService.GetStats issue (#505).
- **Repo state observed at intake (read-only)**:
  - #505's work appears to have landed on `main` (commit `d314c2a`, spec
    `specs/051-usage-source-getstats/`): `proto/finfocus/v1/usage.proto` defines `StatsMode`
    and `UsageRow`; `PLUGIN_CAPABILITY_USAGE_STATS = 14` is the current highest capability;
    `maxValidCapability` is at `sdk/go/pluginsdk/plugin_info.go:219`.
  - `ResourceDescriptor` exists in `proto/finfocus/v1/costsource.proto:564`.
  - The `IsValidCapability` bounds test is in `sdk/go/pluginsdk/conformance_test.go`
    (`TestIsValidCapability`, ~line 813), not `sdk/go/testing/conformance_test.go` as the issue
    wording could suggest.
  - `specs/052-*` does not exist yet.

## First-Glance Unknowns

- [NEEDS CLARIFICATION: The design doc lives in a different repo (rshade/finfocus) and was not
  read at intake — does it add constraints (e.g. policy schema, subject keys, currency rules) not
  captured in the issue?]
- [NEEDS CLARIFICATION: Is #505 considered fully closed/released, so `allocation.proto` can import
  `usage.proto` from a published spec version, or does the dependency still need to ship?]
- [NEEDS CLARIFICATION: Currency handling — what should happen when `priced` entries carry mixed
  currencies, or when `AllocationRow.currency` differs from its inputs?]
- [NEEDS CLARIFICATION: Conservation when every `priced` entry is `priced=false` (sum is 0) —
  relative epsilon is undefined at zero; is an absolute floor intended?]
- [NEEDS CLARIFICATION: What "canonical" JSON means for `policy_digest` (key ordering, number
  formatting, whitespace) so hosts and plugins in other languages compute matching digests?]
- [NEEDS CLARIFICATION: Subject key vocabulary for `AllocationRow.subject` (`kind`, `node`,
  workload identifiers) — reused from `UsageRow` subject keys, or a new set?]
- [NEEDS CLARIFICATION: Which `StatsMode` values change allocator behaviour, and how?]
- [NEEDS CLARIFICATION: Scope of the in-repo reference allocator — test-only fixture, or a
  published example plugin other authors can copy?]
- [NEEDS CLARIFICATION: Whether allocation-only plugins must use `WithCapabilities(...)` like
  usage-only plugins do, given inference always adds the base pricing capabilities.]
- [NEEDS CLARIFICATION: TypeScript SDK scope — only generated code, or also a typed
  `AllocatorClient` like the `UsageSourceClient` added for #505?]
