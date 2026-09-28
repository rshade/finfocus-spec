# Concept: Cost allocation contract for plugins

- **Slug**: allocator-allocate
- **Created**: 2026-09-25
- **Recommended option**: B — Issue scope, hardened where research found gaps

## Options

### Option A — Minimal contract (smallest thing that could work)

- **Sketch**: Ship only what SP2 and SP3 cannot proceed without. That means the new service and
  its messages, capability 15, `Serve` registering an allocator over gRPC and Connect, and one SDK
  helper that checks conservation. Plugin authors get the contract and a single correctness check.
  The fixture-based conformance suite, the in-repo reference allocator, and the "Writing an
  allocator" guide are deferred until a second allocator exists.
- **Appetite**: small (days to about a week). Serving registration follows the pattern #505 already
  built, so it is mostly repetition.
- **Trade-offs**: Fastest path to unblocking SP2. It avoids writing fixtures that might get
  rewritten when SP2 is built. But it gives up the "authors can confirm correctness before
  shipping" goal and the "a broken allocator is rejected 100% of the time" metric: one helper
  checks the sum, not idle ≥ 0, strict policy rejection, or digest stability. The SP2 allocator
  would be the only real test of the contract. The SDK's own tests would cover serving but not
  allocation behavior.
- **Rabbit holes**: The deferred suite may never get built. The first third-party allocator then
  arrives with no conformance bar, which is the design's warning ("third-party allocators rely on
  `RunAllocatorConformance`", SP1 plan).

### Option B — Issue scope, hardened where research found gaps (recommended)

- **Sketch**: Deliver everything issue #506 lists: contract, capability, serving, conformance
  suite with its fixtures, reference allocator used by the SDK's own tests, docs, and TS
  generation. Also settle, as contract rules, the gaps research and define found:
  - **Zero totals:** conservation has an agreed tolerance at zero totals, and the SDK helper is
    positioned as the one check core can call too.
  - **Currency:** currency consistency is either stated as a protocol rule or explicitly left to
    allocators.
  - **Digest scope:** the policy fingerprint is documented as stable within one allocator only.
  - **Allocator-only plugins:** they get the same capability-declaration safeguard that
    usage-only plugins got in #505.
- **Appetite**: medium (weeks). This matches the issue's `effort/medium` label and the size of its
  sibling #505. The hardening items are rule-level decisions, not new features.
- **Trade-offs**:
  - **Wins:** meets every problem goal and success metric. It removes the planned drift between
    core and SDK conservation checks. It follows a pattern just proven in #505, so risk is low.
    Third-party authors get a real bar.
  - **Costs:** it freezes a protocol shape before SP2 has run against it. Research warns the
    design already needed 13 plan-time amendments. The CPU/memory-only row shape is locked in for
    this version, and new categories become additive changes later.
- **Rabbit holes**:
  - **Digest canonicalization across languages:** adopting RFC 8785 or similar would widen scope.
    Keep the per-allocator guarantee.
  - **Generic cost categories:** designing a map of cost categories for GPU, storage, and network
    now instead of accepting the issue's shape.
  - **Reference allocator scope:** the reference allocator growing into a second real Kubernetes
    allocator. It should stay a minimal test fixture.
  - **Shared-cost concepts:** adding `__shared__` or `__unallocated__` concepts before any policy
    needs them.

### Option C — Defer: prototype first, or rely on external tools

- **Sketch**: Ship nothing to finfocus-spec yet. Either (C1) build SP2's allocator against an
  unmerged finfocus-spec branch through a `replace` directive, and freeze the contract only once
  SP2 and SP3 work end to end in the kind E2E. Or (C2) drop in-house allocation and send users to
  OpenCost/Kubecost, with SP6 importing its already-allocated output.
- **Appetite**: small now (C1 is branch work that is later folded into Option B; C2 is close to
  zero). The total cost is deferred rather than removed.
- **Trade-offs**: C1 addresses the strongest evidence against the idea, freezing before the first
  consumer, at the cost of a long-lived branch and a `replace` directive the design says must
  never merge. #505 already set the conventions and reserved `__idle__`/`__cluster__`, so the
  remaining shape risk is smaller than it looks. C2 abandons the design's core decisions
  (domain-agnostic core, allocator plugin model) and leaves `GetStats` (#505, merged) with no
  consumer.
- **Rabbit holes**: C1 branch drift against `main` while other protocol work lands. C2 needs an
  unanswered design question: does SP6 fit `Allocate` or need a different RPC?

## Recommendation

**Option B.** It is the only option that meets every goal in `problem.md`:

- **Verifiable conservation:** hosts can check that allocated cost adds up to the priced total.
- **Visible idle cost:** idle and control-plane cost stay visible instead of being dropped.
- **Auditable policy:** organizations can review the policy in effect and tell when it changed.
- **Pre-ship confidence:** plugin authors can confirm their allocator is correct before shipping.

It is also the only option that meets the key metrics: the conformance suite rejects broken
allocators, and core and SDK agree on conservation. Its sibling #505 shipped the same pattern, so
the appetite is well-evidenced.

The main risk, freezing too early, is partly addressed inside B. finfocus-spec is pre-1.0
(`v0.6.1`), so minor-version changes remain possible. B can also borrow C1's discipline: the SP2
allocator is prototyped against the branch before release, even if not before merge.

Option A saves days but gives up the conformance goal that justifies putting the contract in the
SDK at all. Option C2 contradicts decisions the maintainer has already made. Option C1 is a
sequencing choice that `/speckit-assess-decide` can impose on B instead of treating it as a
separate concept.

## Out of Scope (for the recommended option)

- Allocation math as an SDK or core product feature. The reference allocator is a test fixture
  only.
- Usage collection (#505), core CLI, grouping and rendering (SP3), and registry and release work
  (SP3b).
- Historical actuals, spot and Fargate pricing, and idle or shared-cost redistribution.
- A shared policy schema across allocators. Each allocator owns its own policy.
- Cost categories beyond CPU, memory, and a total (GPU, storage, network). These are accepted as a
  later additive change.
- A cross-language canonical JSON standard for the policy digest.
- `__shared__`/`__unallocated__` row kinds.
- How SP6 (OpenCost, pre-allocated output) connects to the contract.
- A typed TypeScript allocator client beyond generated code, unless the specify stage finds a TS
  host that needs one.

## Assumptions to Validate

- The #505 serving and conformance pattern extends to a second optional service without
  restructuring `Serve`. (Validate during planning: `serveGRPC`/`serveConnect` registration and
  the health checker.)
- finfocus core (SP3) is willing to call the SDK's conservation helper rather than keep its own.
  (Validate with the maintainer; the SP3 plan currently defines `VerifyConservation` in core.)
- A later additive change can add cost categories beyond CPU and memory without breaking existing
  allocators or hosts.
- Per-allocator digest stability is enough for hosts. No host needs to compare digests across
  allocators or recompute them.
- Shipping `Allocate` alongside or shortly after a `GetStats` release is acceptable. Both would
  sit on `main` unreleased until the next tag.
- Pre-1.0 versioning allows adjusting the contract if SP2 implementation finds problems.
- The fixture set in the issue (single node through digest stability) is enough to catch the
  allocator bugs the design worries about most: over-allocation, under-allocation, negative idle,
  and silently accepting a bad policy.
