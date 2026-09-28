# Problem Definition: Cost allocation contract for plugins

- **Slug**: allocator-allocate
- **Created**: 2026-09-25
- **Inputs used**: intake.md, research.md

## Problem Statement

FinFocus hosts can gather workload usage (via `GetStats`, #505) and price infrastructure (via
cost-source plugins). They still have no agreed, verifiable way to have a plugin divide those
priced resources across workloads. The design keeps allocation math out of core, so every
Kubernetes cost breakdown on the roadmap is blocked until hosts and plugins share one contract.
Hosts also need a way to detect an allocator that loses or invents money.

## Affected Users & Stakeholders

- **Users**:
  - **FinFocus CLI users running Kubernetes**: cannot see cluster cost by namespace, controller,
    pod, node, or label. The design's `finfocus cost cluster` needs this contract.
    [source: research, design doc §1]
  - **Allocator plugin authors**: today that is only the maintainer's planned
    `plugins/kubernetes/` allocator (SP2). They have no contract to implement and no conformance
    suite to show their output is trustworthy. [source: research, Users & Demand]
  - **Future usage-source and allocation integrators** (Prometheus SP4, Datadog SP5, OpenCost SP6):
    need allocation output in one shape across sources. Roadmap only, not confirmed demand.
    [source: research, design doc §9]
- **Stakeholders**:
  - **Maintainer (rshade)**: owns finfocus-spec and finfocus and decides protocol stability. Wants
    core to stay independent of Kubernetes. [source: intake, design doc §1]
  - **finfocus core (SP3)**: must verify allocator output and render it without understanding
    Kubernetes. Needs a check it can rely on, and needs policy to be displayable.
    [source: design doc §4-5]
  - **Third-party plugin ecosystem**: any published contract becomes a compatibility commitment
    under semantic versioning. [source: CLAUDE.md "Versioning"] [NEEDS CLARIFICATION: whether any
    third party intends to build an allocator]

## Goals

- A host can obtain per-workload cost from a plugin without core understanding Kubernetes (or any
  other domain) semantics.
- A host can verify, mechanically, that allocated cost adds back to the priced total, so a buggy
  allocator is never rendered as a plausible number. [source: design doc §5 "a failure must never
  render as a plausible number"]
- Unallocated capacity (idle) and shared infrastructure (control plane) stay visible instead of
  being dropped silently.
- Organizations can review and audit the allocation policy in effect, and tell when it changed.
- Plugin authors can confirm their allocator is correct before shipping, without writing their own
  test harness.
- Adding the contract does not break existing plugins or hosts.

## Non-Goals

- Doing the allocation math inside core or inside finfocus-spec's SDK as a product feature (the
  reference allocator exists only to test the SDK).
- Collecting usage data (covered by #505).
- Historical actuals, spot pricing, Fargate pricing, and redistributing shared or idle cost across
  workloads. These are design non-goals for this slice. [source: design doc §1]
- CLI commands, grouping, and rendering in finfocus core (SP3), plus registry and release work
  (SP3b).
- Defining one policy schema that every allocator must share. Each allocator owns its own policy.
  [source: intake, contract rule 4]
- Supporting GPU, storage, or network cost categories in this slice. [NEEDS CLARIFICATION: see
  Open Questions; research flags this as a future-proofing risk]

## Success Metrics

- The Kubernetes plugin's allocator (SP2) passes the SDK allocator conformance suite unchanged,
  using a released finfocus-spec version with no `replace` directive. (baseline: no contract, 0
  allocators)
- finfocus `cost cluster` (SP3) runs end to end in the kind E2E. Conservation holds, and a
  Deployment's cost equals `request / allocatable × node monthly price` within epsilon.
  (baseline: not possible) [source: design doc §6 E2E]
- A deliberately broken allocator (over-allocation, under-allocation, negative idle) is rejected
  by the conformance suite 100% of the time. (baseline: no check exists)
- Core and SDK agree on conservation: no input exists that core rejects and the SDK accepts, or
  the reverse. (baseline: two separate planned implementations with different zero-total rules,
  per research)
- Existing plugins and hosts need no code change. `buf breaking` passes. (baseline: passing)
- Qualitative: a plugin author can write a working allocator by following only the SDK's
  documentation and conformance output. [NEEDS CLARIFICATION: how to measure without an outside
  author]

## Cost of Inaction

The finfocus Kubernetes allocation effort stops at SP1. The run-rate `cost cluster` command
(SP2/SP3) cannot be built, and the planned Prometheus, Datadog, and OpenCost integrations lose
their common output shape. Doing nothing leaves two fallbacks, both rejected by the design: put
Kubernetes allocation math into core, which breaks the "core stays domain-agnostic" decision, or
leave cluster cost breakdown to external tools such as OpenCost or Kubecost with no FinFocus
integration. `GetStats` (#505), already merged, would have no consumer in the pipeline. Doing
nothing also avoids the risk research raised: freezing a protocol shape before its only consumer
exists.

## Open Questions

- [NEEDS CLARIFICATION: Should verifying conservation have one source of truth (the SDK helper
  used by both conformance tests and core), and what is the tolerance for zero totals?]
- [NEEDS CLARIFICATION: Is currency consistency a protocol-level guarantee or allocator-specific
  behavior?]
- [NEEDS CLARIFICATION: Do hosts only need to show and compare a policy fingerprint from one
  allocator, or must it be reproducible across allocators and languages?]
- [NEEDS CLARIFICATION: Must the problem cover cost categories beyond CPU and memory (GPU,
  storage, network) now, or is a later additive change acceptable?]
- [NEEDS CLARIFICATION: Is "pre-allocated" output from external tools (OpenCost, SP6) in scope for
  this problem, or a separate problem?]
- [NEEDS CLARIFICATION: Should the contract be frozen only after SP2's allocator is prototyped
  against it, given that the design docs are still untracked drafts?]
- [NEEDS CLARIFICATION: Release sequencing: ship with `GetStats` in one finfocus-spec release, or
  tag `GetStats` first?]
- [NEEDS CLARIFICATION: Should allocator-only plugins get the same capability-declaration
  safeguard that usage-only plugins received in #505?]
- [NEEDS CLARIFICATION: Is TypeScript host support (beyond generated code) needed for this
  problem?]
