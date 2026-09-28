# Idea Research: AllocatorService.Allocate for cost allocation plugins

- **Slug**: allocator-allocate
- **Created**: 2026-09-25
- **Evidence confidence (overall)**: high for design intent and repo fit; low for demand
  beyond the owner's own Kubernetes plugin

Local sources are read with paths relative to the finfocus-spec repo root. `../finfocus` is the
sibling local clone of rshade/finfocus. Its design and plan documents are **untracked** in that
clone (`git status` shows `?? docs/superpowers/`), so they are drafts, not reviewed or merged work.

## Users & Demand

- The only named consumer is the owner's own planned `plugins/kubernetes/` allocator (SP2) and the
  `finfocus cost cluster` command (SP3). Both are specified but not built. — [source:
  `../finfocus/docs/superpowers/specs/2026-09-24-k8s-cost-allocation-design.md` §2] (confidence:
  high, cited)
- The design lists later consumers: Prometheus (SP4) and Datadog (SP5) usage sources, which reuse
  the same allocator, and OpenCost (SP6), which would return "pre-allocated rows in the same
  `AllocationRow` shape". — [source: design doc §1, §9] (confidence: medium, cited; these are
  roadmap items, not commitments)
- No external request was found: issue #506 has 0 comments, and no other plugin author has asked
  for allocation. This is *stated* intent from the maintainer, not *observed* demand. — [source:
  `gh issue view 506`] (confidence: high, cited)
- The user-facing goal is to break cluster cost down by namespace, controller, pod, node, and
  label, starting with a run-rate view. — [source: design doc §1] (confidence: high, cited)

## Prior Art

- **Companion #505 has merged.** `UsageSourceService.GetStats` landed in commit `d314c2a`
  (spec `specs/051-usage-source-getstats/`), and #505 closed 2026-09-25. It established the
  pattern this idea copies: optional provider interface, unexported adapters registered in gRPC
  and Connect modes plus the health checker, a new capability value, and a conformance helper in
  `sdk/go/testing`. — [source: `git log`, `gh issue view 505`, CLAUDE.md "Usage Source SDK
  Pattern"] (confidence: high, cited)
- **#505 already reserved allocator vocabulary.** `KindIdle = "__idle__"` and
  `KindCluster = "__cluster__"` exist in `sdk/go/pluginsdk/subjects.go:44-49`, marked "reserved
  for allocator output (#506)" and invalid in `GetStats` rows. The subject keys (`cluster`,
  `namespace`, `controller_kind`, `controller`, `pod`, `node`, `label.<key>`, `kind`) are defined
  there too. — [source: `sdk/go/pluginsdk/subjects.go`, `specs/051-usage-source-getstats/spec.md:216`]
  (confidence: high, cited)
- **The `usage.proto` contract says how priceables are tagged:** nodes carry `tags.kind="node"`
  with `id` equal to the node's `node` subject; a control plane carries `tags.kind="cluster"`.
  An allocator depends on this join rule. — [source: `proto/finfocus/v1/usage.proto:84-86`]
  (confidence: high, cited)
- **The reference allocator is already planned in detail** in finfocus SP2: join, CPU/memory
  split, `max(request, usage)` share, normalizing shares above 1, a per-node idle row, unpriced
  and spot notes, Fargate rows, a `__cluster__` row for unknown priced kinds (to keep
  conservation), and deterministic output order. — [source:
  `../finfocus/docs/superpowers/plans/2026-09-24-k8s-cost-allocation-sp2-kubernetes-plugin.md`
  lines ~296-308] (confidence: high, cited)
- **OpenCost uses the same sentinel naming.** `core/pkg/opencost/allocation.go` defines
  `IdleSuffix = "__idle__"`, `SharedSuffix = "__shared__"`, and
  `UnallocatedSuffix = "__unallocated__"`, plus share modes such as `ShareWeighted`. The `__idle__`
  convention and "idle as its own row" are established practice in open-source Kubernetes cost
  tools. OpenCost has an `__unallocated__` concept with no match in this proposal. — [source:
  github.com/opencost/opencost `core/pkg/opencost/allocation.go` lines 27-41] (confidence: high,
  cited)
- No earlier allocation contract exists in this repo. The FOCUS 1.3 `Allocated*` builder fields
  (`focus_builder.go`) describe allocation *metadata on billing records*, not an allocation
  computation service. — [source: `sdk/go/pluginsdk/focus_builder.go:50`] (confidence: high, cited)

## Market & Context

- **Cost of doing nothing:** the finfocus Kubernetes allocation effort (SP2, SP3, and SP4-SP6)
  is blocked. The design deliberately keeps allocation math out of core, so the design has no
  fallback that works without this contract. — [source: design doc §1 "Decisions already made",
  §2 "Depends on SP1"] (confidence: high, cited)
- **Alternatives users rely on today:** OpenCost and Kubecost compute Kubernetes allocation
  themselves. The design plans to ingest OpenCost output (SP6) rather than compete with it. —
  [source: design doc §1, §9; OpenCost repo] (confidence: medium, cited for the plan; ASSUMPTION
  that target users already run OpenCost/Kubecost)
- The design keeps core free of Kubernetes knowledge ("nodes arrive as ordinary
  `ResourceDescriptor`s"), so the contract has to be generic string-map subjects rather than
  typed Kubernetes fields. — [source: design doc §3 "Well-known subject keys", §4] (confidence:
  high, cited)

## Data & Constraints

- **Capability number is free:** the highest value is `PLUGIN_CAPABILITY_USAGE_STATS = 14`
  (`proto/finfocus/v1/enums.proto:191`), and `maxValidCapability` is at
  `sdk/go/pluginsdk/plugin_info.go:219`. The bounds test is `TestIsValidCapability` in
  `sdk/go/pluginsdk/conformance_test.go:813`. — [source: repo] (confidence: high, cited)
- **Release gap:** no tag contains `d314c2a`. The latest tag is `v0.6.1`, and finfocus pins
  `github.com/rshade/finfocus-spec v0.6.1`, so `GetStats` is not released yet. The design says
  consumers use a *released* finfocus-spec, with `replace` directives only during development. —
  [source: `git tag --contains d314c2a`, `../finfocus/go.mod:16`, design doc §2] (confidence:
  high, cited)
- **Currency:** the SP2 plan has the allocator return `FailedPrecondition` when `priced=true`
  currencies differ, and treat an empty currency as `USD`. The SP3 core also applies its existing
  `ErrMixedCurrencies`. The issue's contract rules say nothing about currency. — [source: SP2 plan
  rule 8; design doc §5] (confidence: high, cited)
- **Zero-total conservation:** the SP3 plan uses relative epsilon `1e-6` **plus `1e-9` absolute
  for zero totals**. The issue specifies only the relative epsilon. — [source:
  `../finfocus/docs/superpowers/plans/2026-09-24-k8s-cost-allocation-sp3-cost-cluster.md:27`]
  (confidence: high, cited)
- **Digest canonicalization** in the SP2 reference is `json.Marshal` of a Go struct, followed by
  hex SHA-256. That is "canonical" only because Go struct field order is stable. A plugin in
  another language would not reproduce it. The digest is per-plugin, so this only matters if
  hosts compare digests across allocators. — [source: SP2 plan `Policy.Canonical`, lines ~259-267]
  (confidence: high for the code; ASSUMPTION that cross-plugin comparison is not needed)
- **`{}`, comments-only, and version-less policies** decode to the defaults and must give the
  default digest (SP2 Task 1). That matches the issue's fixture "`{}` policy digest equals empty
  policy digest". — [source: SP2 plan line 34] (confidence: high, cited)
- **Capability inference hazard:** `inferCapabilities` always adds projected, actual, pricing,
  and estimate. An allocator-only plugin must declare its capabilities explicitly, or hosts will
  send it price queries. #505 added a `Serve` warning for usage-only plugins, and the SP2 plugin
  declares exactly `[USAGE_STATS, ALLOCATION]`. — [source: SP1 plan line 28; CLAUDE.md "Usage
  Source SDK Pattern"; SP2 plan line 9] (confidence: high, cited)
- **Money as `double`:** existing cost fields are already `double` (`costsource.proto:375, 715,
  1105`), so float `cost` in `PricedResource`/`AllocationRow` follows precedent. Hence the epsilon
  in the conservation rule. — [source: repo] (confidence: high, cited)
- **Connect coverage:** every Connect handler must convert gRPC status errors with
  `toConnectError`, or `InvalidArgument` policy errors reach clients as `Unknown`. — [source:
  CLAUDE.md "Usage Source SDK Pattern"] (confidence: high, cited)

## Evidence Against the Idea

- **Single consumer, not yet built.** The only concrete consumer is the maintainer's own
  unimplemented Kubernetes plugin. A cross-language protocol contract fixed before its first real
  consumer runs risks locking in shapes that SP2/SP3 implementation will want to change. Draft
  plans already needed 13 "plan-time amendments" (design doc §8). — [source: design doc §8]
  (confidence: medium, cited)
- **Conservation logic would live in two places.** SP3 plans its own `VerifyConservation` in core
  (`internal/engine`), while this issue adds `CheckConservation` to the SDK. The two could drift:
  the SP3 version already has an absolute zero-total floor that the issue does not. — [source: SP3
  plan lines 27, 175-178, 435; issue #506 contract rule 1] (confidence: high, cited)
- **CPU/memory-only row shape.** `AllocationRow` hardcodes `cpu_cost` and `mem_cost`. GPU,
  storage (PV), and network cost need new fields or have to be folded into `total_cost`. The
  `__cluster__` exemption to `total = cpu + mem` already shows this strain. OpenCost's allocation
  model carries GPU, PV, network, and load-balancer cost separately. — [source: issue #506
  `AllocationRow`; ASSUMPTION about OpenCost field breadth beyond the constants read] (confidence:
  medium)
- **The SP6 "pre-allocated rows" path is unclear.** An OpenCost plugin that already has allocated
  rows would either implement `Allocate` and ignore its inputs, or need a different RPC. The
  design does not say which. — [source: design doc §9] (confidence: medium, cited)
- **No `__unallocated__`/`__shared__` concepts.** Shared-cost redistribution is an explicit
  non-goal for this slice. Adding it later could need new sentinel kinds or policy fields. The
  opaque policy absorbs the second but not the first. — [source: design doc §1 non-goals;
  OpenCost `allocation.go`] (confidence: medium, cited)
- **Depends on an unreleased contract.** `usage.proto` is on `main` but not in a tagged release,
  so `allocation.proto` would stack a second unreleased service on top of it. — [source:
  `git tag --contains d314c2a`] (confidence: high, cited)

## Gaps & Open Questions

- [NEEDS CLARIFICATION: Should the SDK's `CheckConservation` include the SP3 absolute floor
  (`1e-9`) for zero totals, so the SDK and core agree? Or should core call the SDK helper instead
  of its own `VerifyConservation`?]
- [NEEDS CLARIFICATION: Should currency rules (all `priced=true` entries share one currency,
  `FailedPrecondition` otherwise, empty means `USD`) become part of the protocol contract, or stay
  allocator-specific behavior?]
- [NEEDS CLARIFICATION: Is `policy_digest` defined only per allocator (so any stable
  canonicalization is fine), or must hosts be able to recompute or compare it? If the latter, name
  a canonical form such as RFC 8785 JCS.]
- [NEEDS CLARIFICATION: Should the `Serve` warning for plugins without explicit capabilities
  extend to allocator-only plugins, as #505 did for usage-only plugins?]
- [NEEDS CLARIFICATION: Should `AllocationRow` stay CPU/memory-specific, or carry a
  per-resource-type cost map for future GPU/storage without a breaking change?]
- [NEEDS CLARIFICATION: How does SP6 (OpenCost, pre-allocated) fit: implementing `Allocate`, or a
  separate RPC?]
- [NEEDS CLARIFICATION: Release sequencing: tag a finfocus-spec release with `GetStats` first, or
  ship `GetStats` and `Allocate` together in one release?]
- [NEEDS CLARIFICATION: Do the design and plan docs get committed and reviewed in rshade/finfocus
  before this contract is frozen? They are currently untracked drafts ("Status: Draft, pending
  review").]
- [NEEDS CLARIFICATION: TypeScript scope: generated code only, or a typed `AllocatorClient` like
  the `UsageSourceClient` from #505?]

## Sources

- GitHub issue rshade/finfocus-spec#506 (via `gh issue view`; host: github.com, policy: allowlisted)
- GitHub issue rshade/finfocus-spec#505 (via `gh issue view`; host: github.com, policy: allowlisted)
- <https://github.com/opencost/opencost/blob/develop/core/pkg/opencost/allocation.go> (fetched via
  `gh api` contents endpoint; host: github.com, policy: allowlisted)
- Local: `../finfocus/docs/superpowers/specs/2026-09-24-k8s-cost-allocation-design.md` (untracked
  draft)
- Local: `../finfocus/docs/superpowers/plans/2026-09-24-k8s-cost-allocation-sp1-spec-issues.md`
- Local: `../finfocus/docs/superpowers/plans/2026-09-24-k8s-cost-allocation-sp2-kubernetes-plugin.md`
- Local: `../finfocus/docs/superpowers/plans/2026-09-24-k8s-cost-allocation-sp3-cost-cluster.md`
- Local: `../finfocus/go.mod`
- Repo: `proto/finfocus/v1/usage.proto`, `proto/finfocus/v1/enums.proto`,
  `proto/finfocus/v1/costsource.proto`, `sdk/go/pluginsdk/subjects.go`,
  `sdk/go/pluginsdk/plugin_info.go`, `sdk/go/pluginsdk/conformance_test.go`,
  `specs/051-usage-source-getstats/`
