# Decision: Standardized cost allocation lineage metadata

- **Slug**: cost-allocation-lineage
- **Decided**: 2026-09-27
- **Verdict**: needs-clarification
- **Artifacts reviewed**: intake.md | research.md | problem.md | concept.md, plus
  `.specify/memory/constitution.md` for strategic fit

## Scorecard

| Criterion | Rating | Justification |
| --- | --- | --- |
| Problem validity | adequate | The contract really is flat: billing account, sub-account, resource id, and tags. Ordered parentage above the sub-account cannot be exchanged. The pain itself is only stated in issue 191. No host or plugin is blocked. |
| Evidence strength | weak | Field numbers, FOCUS columns, the `expires_at` collision, and the allocator naming clash are high-confidence and cited. The claim that someone needs a new chain, rather than those existing fields, is one `research:` issue labeled `roadmap/future`, with no consumer, no usage data, and no human comments since 2025-12-22. |
| Value vs. inaction | weak | Inaction keeps a working two-level model and tag chargeback. Nothing in-tree stalls the way GetStats stalled Allocate. The gain is reserved for a drill-down host that does not exist in this assessment. |
| Feasibility / appetite | strong | Option B is a medium, backward-compatible pass-through on actual cost. The stale field 8 must not be reused. Option C's large appetite and hard consistency check are not required to meet the goals. |
| Strategic fit | adequate | Pass-through metadata fits principle III better than allocation math does. Principle IV keeps the CFO click-path out of this repo. Principle VI forbids recycling `expires_at`'s field number. Principle XIII means generated Go and TypeScript bindings move with the proto. No new RPC is required. The words "cost allocation" now mean `AllocatorService`, so the name must not collide. |
| Risk posture | adequate | The serious risks are identified and avoided by Option B: do not reuse field 8, do not infer parents, do not fail on disagreement with `billing_account_id`, do not promise unlimited depth, and do not treat the chain as audit evidence. Residual risk is building it before anyone reads it. |

## Verdict & Rationale

**Needs clarification.** The schema gap is real and Option B is a credible way to close it, but
a go requires evidence strength of adequate or better. Evidence here is weak: issue 191 is
maintainer research parked on `roadmap/future`, not a consumer with a milestone. Value versus
doing nothing is also weak, because FOCUS account fields and tags already cover two-level
attribution and cost-center tags. Specifying now would freeze a hierarchy model, and a GCP
project or Kubernetes namespace classification, before anyone has agreed to populate or read
it. That is not a kill. The problem is not fake, and the December 2025 field number for
`ActualCostResult` (8) is already `expires_at`, so a later spec must not implement the sketch
literally. It is blocked on whether this leaves the icebox and on which interactions the chain
is for.

## If needs-clarification

- **Blocking questions**:
  - [NEEDS CLARIFICATION: Should this leave `roadmap/future` now? Name the host or plugin that
    will write and read the chain, and the milestone. If the answer is still icebox, the
    verdict on a re-run should be kill until that consumer exists.]
  - [NEEDS CLARIFICATION: Confirm the attachment. Option B carries the chain only on actual
    cost, host-bound, and not on the request resource descriptor, projections, or estimates.
    Issue 191 puts it on both the descriptor (field 11, still free) and the actual-cost result
    (field 8, taken). Which set is actually required?]
  - [NEEDS CLARIFICATION: Confirm that a chain which omits or disagrees with
    `billing_account_id` or `sub_account_id` is still valid. The anti-guess boundary says yes.
    Open question 3 on the issue asks for a conformance check that would say no.]
- **Revisit stage**: research

The concept does not need to be redrawn unless the attachment answer rejects Option B.
Non-blocking items already recorded for a later spec: stored depth, a walk helper, a name for
custom levels, and the GCP project / Kubernetes namespace class versus this repo's sub-account
mapping.

## If go — Handoff to `/speckit-specify`

- **Problem**: n/a (verdict is needs-clarification)
- **Chosen approach**: n/a. Concept recommendation, not chosen: Option B.
- **In scope / out of scope**: n/a
- **Success metrics**: n/a
- **Carried-forward open questions**: the blocking questions above
