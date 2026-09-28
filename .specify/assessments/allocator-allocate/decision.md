# Decision: Cost allocation contract for plugins

- **Slug**: allocator-allocate
- **Decided**: 2026-09-25
- **Verdict**: go
- **Artifacts reviewed**: intake.md | research.md | problem.md | concept.md, plus
  `.specify/memory/constitution.md` (for strategic fit)

## Scorecard

| Criterion | Rating | Justification |
|-----------|--------|---------------|
| Problem validity | strong | Without a contract the whole Kubernetes allocation effort (SP2-SP6) is blocked, and GetStats (#505, merged) has no consumer. Core's domain-agnostic design leaves no other place for allocation. |
| Evidence strength | adequate | Design intent, repo fit, and precedent are high-confidence and cited (#505 pattern, reserved `KindIdle`/`KindCluster`, OpenCost `__idle__`). Demand is the maintainer's own roadmap only, and the design and plan docs are untracked drafts. That is thin for an ecosystem feature, but enough for a maintainer-owned protocol with a named consumer. |
| Value vs. inaction | strong | Doing nothing means either putting Kubernetes math in core (rejected by the design and by constitution IV) or giving up in-house breakdown entirely. The only thing gained by waiting is avoiding an early freeze, and pre-1.0 versioning reduces that risk. |
| Feasibility / appetite | strong | Option B is medium (weeks) and matches the `effort/medium` label. #505 just delivered the same serving, capability, and conformance pattern, so there is little new engineering. |
| Strategic fit | adequate | Fits I (proto first), IV (the spec defines the interface and core stays out of it), X (follow established patterns), and XII (capability found through a provider interface). **Tension with III ("The Spec Consumes, It Does Not Calculate")**: allocation is calculation, and III's rationale names Kubecost as where such logic belongs. It is reconcilable because the contract only standardizes the *model* for allocated cost, the math lives in plugins, and the SDK reference allocator is a test fixture. The spec must state this justification explicitly. **XIII requires a TS client wrapper and TS integration tests**, which concept.md wrongly listed as optional. |
| Risk posture | adequate | Key risks are identified and mostly mitigated: early freeze (pre-1.0 plus prototyping SP2 against the branch before release), conservation drift between core and SDK (single helper), and scope rabbit holes (explicitly excluded). Two risks remain open: the CPU/memory-only row shape (accepted, with additive growth later) and the unresolved fit for SP6 (OpenCost pre-allocated rows). |

## Verdict & Rationale

**Go**, with Option B from concept.md, amended by the constitution review. Problem validity is
strong, evidence is adequate, and a recommended concept exists, so the go criteria are met. The
problem is real and structurally unavoidable given decisions already made. The appetite is
well-evidenced by #505. The open questions concern contract *details* (zero-total tolerance,
currency, digest scope), which `/speckit-specify` and `/speckit-clarify` are built to resolve, not
whether to build it at all.

Two conditions come with the go:

- **(1) Principle III justification.** The spec must explicitly reconcile the feature with
  constitution principle III, or the plan's constitution check will fail.
- **(2) Principle XIII TypeScript client.** A TS `AllocatorClient` wrapper and TS integration
  tests are in scope because XIII makes them mandatory. This follows #505's `UsageSourceClient`.

Sequencing advice, not a blocker: prototype the SP2 allocator against the branch before
finfocus-spec tags the release that includes `Allocate`.

## If needs-clarification

- **Blocking questions**: none (verdict is go).
- **Revisit stage**: n/a

## If go — Handoff to `/speckit-specify`

- **Problem**: FinFocus hosts can gather usage (GetStats) and price infrastructure, but have no
  shared, verifiable contract for a plugin to divide priced resources across workloads. That
  blocks every Kubernetes cost breakdown, and hosts cannot detect an allocator that loses or
  invents money.
- **Chosen approach**: Option B, the scope of issue #506 (rshade/finfocus-spec#506) hardened where
  research found gaps. It includes:
  - **Contract:** `AllocatorService.Allocate` in `proto/finfocus/v1/allocation.proto` and
    `PLUGIN_CAPABILITY_ALLOCATION = 15`.
  - **Go SDK:** an `AllocatorProvider` interface, with `Serve` registering it in gRPC and Connect
    modes plus the health checker, following the #505 `UsageSourceProvider` pattern.
  - **Conformance:** `CheckConservation` and `RunAllocatorConformance` with the issue's fixture
    list, plus an in-repo reference allocator used as a test fixture.
  - **Docs:** capability table and a "Writing an allocator" guide.
  - **TypeScript:** generated code plus a TS `AllocatorClient` with integration tests
    (constitution XIII).
  - **Contract rules:** a zero-total conservation tolerance; a stated position on currency
    consistency; digest stability defined per allocator; and allocator-only plugins getting the
    #505 capability-declaration safeguard.
  - **Feature folder:** `specs/052-allocator-allocate/`.
- **In scope / out of scope**:
  - **In:** everything in the approach above, plus an explicit constitution III justification.
  - **Out:**
    - allocation math as an SDK or core product feature
    - usage collection (#505)
    - core CLI, grouping and rendering (SP3), and registry and release work (SP3b)
    - historical actuals, spot and Fargate pricing
    - idle or shared-cost redistribution, and `__shared__`/`__unallocated__` kinds
    - a shared cross-allocator policy schema
    - cost categories beyond CPU, memory, and total
    - a cross-language canonical-JSON standard
    - SP6 (OpenCost) integration
- **Success metrics**:
  - SP2's allocator passes the conformance suite unchanged, using a released finfocus-spec with
    no `replace` directive.
  - The kind E2E for `cost cluster` holds conservation and expected per-Deployment cost.
  - Deliberately broken allocators (over-allocation, under-allocation, negative idle, silently
    accepting a bad policy) are rejected 100% of the time.
  - Core and SDK agree on conservation for every input.
  - `buf breaking` passes, and existing plugins and hosts need no changes.
- **Carried-forward open questions**:
  - [NEEDS CLARIFICATION: Conservation tolerance at zero totals: adopt SP3's `1e-9` absolute
    floor? Will core call the SDK helper instead of its own `VerifyConservation`?]
  - [NEEDS CLARIFICATION: Currency: protocol rule (one currency across `priced=true`, error
    otherwise, empty means `USD`) or allocator-specific?]
  - [NEEDS CLARIFICATION: Is the policy digest guaranteed only per allocator? Confirm no host
    compares digests across allocators.]
  - [NEEDS CLARIFICATION: Which gRPC code should an allocator return for inconsistent inputs
    (mixed currency, `priced=false` with nonzero cost): `InvalidArgument` or
    `FailedPrecondition`?]
  - [NEEDS CLARIFICATION: Release sequencing: tag GetStats first, or ship both services in one
    finfocus-spec release?]
  - [NEEDS CLARIFICATION: Future cost categories (GPU, storage, network): confirm an additive
    change is acceptable, and reserve room for it in the shape.]
  - [NEEDS CLARIFICATION: SP6 fit: does a pre-allocated source implement `Allocate` or need its
    own RPC? This is out of scope here, but the contract should not preclude it.]
