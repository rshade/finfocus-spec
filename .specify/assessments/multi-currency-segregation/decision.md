# Decision: Multi-Currency Cost Segregation

- **Slug**: multi-currency-segregation
- **Decided**: 2026-09-27
- **Verdict**: go
- **Artifacts reviewed**: intake.md, research.md, problem.md, concept.md

## Scorecard

| Criterion | Rating | Justification |
|-----------|--------|---------------|
| Problem validity | adequate | A recommendation summary adds 100 USD and 50 EUR into 150 and clears the currency, and a test locks that in. Allocation rejects the same kind of mix. No host incident is recorded, so the problem is real but not shown to be urgent. |
| Evidence strength | adequate | The two behaviors, the ISO 4217 package, the FOCUS billing-versus-pricing pair, issue #358, and the missing currency on actual-cost points are cited from the repo. Demand is one discovery issue. Richer outcomes (subtotals, a new field) were treated as out of scope rather than as facts. |
| Value vs. inaction | adequate | Doing nothing keeps a tested unitless sum and two policies. Option B is a small guard on that sum. The icebox label fits an unbounded pattern, not this slice. |
| Feasibility / appetite | strong | Option B stays on the existing summary helper, its duplicated mock, and the test that expects 150. It needs no protocol change, no money library, and no conversion. |
| Strategic fit | strong | Constitution principle III says the spec consumes costs and does not embed financial math. Principle I requires validation of protobuf messages. A no-conversion guard matches both. Option B does not change the proto contract. |
| Risk posture | adequate | The behavior change is named, and nothing in the issue shows a host depending on the unitless 150. Proto, FX, and a bought money type are excluded. How the withheld total is reported (error versus omitted number) is a specification detail, not a different concept. |

## Verdict & Rationale

Go, limited to Option B. The scorecard clears the gate: problem validity and evidence are
adequate, not weak, and a concept is already chosen. The cited harm is one SDK total that adds
unlike currencies, which the issue text exists to prevent, inside a boundary of validation and
typing with no conversion math. Allocation already refuses that mix, so the narrow
standardization is to stop the summary from doing the opposite. The `effort/large` and
`roadmap/future` labels describe an open-ended pattern this verdict does not accept.
Actual-cost currency, per-currency subtotals, TypeScript, and an external money type stay out.
Demand is thin, and that is why the go is for the slice with a measured baseline, not for a
cross-RPC redesign.

## If go — Handoff to `/speckit-specify`

- **Problem**: Callers can add FinFocus amounts that are not in the same currency and still
  hold one total. Recommendation savings do this today (100 USD + 50 EUR = 150, currency
  blank). Allocation does not.
- **Chosen approach**: Option B — Stop the one mixed sum. Unlike currencies are not added.
  Match the allocation outcome. Do not add a money type or a protocol field.
- **In scope / out of scope**: In scope: the recommendation-summary total (and the mock that
  copies it) must not present one number for mixed non-empty currencies; the test that expects
  150 changes; no conversion. Out of scope: FX and rate tables; `bojanz/currency` and
  `x/text/currency`; decimal amounts; locale formatting; forbidding a billing currency that
  differs from pricing currency on one FOCUS row; a currency field on actual-cost points;
  per-currency subtotal responses; TypeScript parity unless a TypeScript sum of mixed
  currencies is found; a rewrite of the developer-guide converter or the ungrouped SQL samples
  beyond a short warning if they sit next to the corrected behavior.
- **Success metrics**: SDK paths that add two different non-empty currencies into one total
  go from 1 to 0. Contradictory totaling policies go from 2 to 1. No new conversion routine.
  The actual-cost "name the currency of the total" metric is explicitly not in this slice.
- **Carried-forward open questions**:
  - Whether the mixed-summary failure is reported as an error or as a summary with no total.
    Either way the 150 must not appear.
  - Whether any host depends on the unitless 150. None is named. If one appears during
    specification, record the compatibility constraint rather than widening scope.
  - Empty currency and `XXX` keep today's meaning unless specification finds that an empty
    currency is being added into a labeled total in a way the issue's "different units"
    wording must cover. Default assumption: leave empty-as-ignored on impacts, and leave
    allocation's empty-as-USD rule alone.
