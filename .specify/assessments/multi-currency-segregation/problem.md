# Problem Definition: Mixed-currency amounts can be added as one total

- **Slug**: multi-currency-segregation
- **Created**: 2026-09-27
- **Inputs used**: intake.md, research.md

## Problem Statement

Callers that combine FinFocus cost figures can add amounts that are not in the same currency and
still hold a single total. For recommendation savings, 100 USD plus 50 EUR is stored as 150 with
the currency cleared, and a test locks that in. For allocation, the same kind of mix is rejected.
Nothing in the recorded demand says a production host has been hurt yet. The harm that is shown
is an ambiguous sum beside a conflicting rule, so a caller cannot know whether a total is safe
to treat as one unit.

## Affected Users & Stakeholders

- **Users**: SDK callers that total recommendation savings. They receive one number whose unit
  was blanked because the inputs disagreed, while the number itself is still the arithmetic sum.
  — [source: research.md, `CalculateRecommendationSummary` and its mixed-currency test]
- **Users**: SDK callers that allocate priced resources. A mix of currencies fails the call
  instead of producing a total. The same mix is therefore safe to add in one path and illegal
  in another. — [source: research.md, `ResolveCurrency` / `ErrMixedCurrency`]
- **Users**: Callers that sum actual-cost points. Those points carry a cost and no currency of
  their own; the currency, if any, sits on an optional FOCUS record. A sum of the cost field
  does not show its unit. — [source: research.md, `ActualCostResult`]
- **Stakeholders**: The repository owner (rshade), who opened issue #190 and placed it on the
  discovery roadmap at large effort. Decision power is theirs; no other stakeholder is named.
  — [source: intake.md, research.md issue #190 and `ROADMAP.md`]
- **Stakeholders**: Plugin hosts that render recommendation summaries or allocation results.
  [NEEDS CLARIFICATION: no host is named, and no host report is in the issue]

## Goals

- A summed cost figure is not presented as one amount when its inputs use more than one currency.
- A caller can predict, before adding, whether a set of amounts shares one currency.
- The same mix of currencies has one outcome everywhere the SDK totals money, instead of
  "reject" in allocation and "add, then blank the label" in recommendation summaries.
- Currency conversion stays out of the outcome. Success is refusing or separating unlike units,
  not turning them into one unit by a rate.

## Non-Goals

- Converting amounts between currencies, storing FX rates, or doing exchange math. The issue
  states this boundary, and issue #358 already recorded conversion as unneeded.
  — [source: intake.md; research.md issue #358]
- Changing how a single FOCUS charge may be billed in one currency and priced in another.
  That pair is valid today and is not, by itself, a sum of unlike billed amounts.
  — [source: research.md, focus conformance "valid pricing currency" and spec 027]
- Introducing accounting-grade precision. Amounts stay `double`, which issue #358 accepted
  for display rather than ledger arithmetic. — [source: research.md]
- Locale-specific formatting. The currency package already formats for display.
  — [source: research.md, `sdk/go/currency`]
- Fixing ungrouped `SUM(BilledCost)` examples or the developer-guide converter sample, except
  where they would keep teaching the behavior this problem is about.
  [NEEDS CLARIFICATION: whether doc drift is in scope or only a symptom]

## Success Metrics

- Count of SDK paths that add amounts with two different non-empty currencies into one total:
  baseline 1 (`CalculateRecommendationSummary`, test expects 150). Target: 0.
  — [source: research.md] (measurable against the current test)
- Count of contradictory mixed-currency policies among SDK totaling paths: baseline 2
  (allocation rejects; recommendation summary adds and clears the currency). Target: 1
  written rule that both paths follow. — [source: research.md] (measurable by inspection)
- A caller can name the currency of an actual-cost total, or can tell that the total has no
  single currency. Baseline: `ActualCostResult.cost` has no currency field.
  — [source: research.md] (qualitative until a check exists; the baseline is factual)
- No new conversion routine in the SDK. Baseline: none in `sdk/go/currency`; one example
  converter in the developer guide. — [source: research.md] (measurable by inspection)
- Named production incidents of mixed-currency sums: baseline 0 in the issue.
  [NEEDS CLARIFICATION: there is no usage metric to move]

## Cost of Inaction

Recommendation summaries keep publishing a unitless sum of mixed savings, and the test keeps
that behavior. Allocation keeps rejecting mixed currencies. Callers who copy one path will not
match callers who copy the other. Actual-cost lists remain summable with no unit on the point.
The developer guide keeps showing a rate converter, and the FOCUS column guide keeps showing
`SUM(BilledCost)` without a currency group. No incident, ticket, or usage number says this has
already misled a host. Leaving it iceboxed preserves a known footgun and does not, on the
evidence, delay a reported customer.

## Open Questions

- [NEEDS CLARIFICATION: which totals are in scope — recommendation savings, actual-cost pages,
  projected or estimated cost, FOCUS billed amounts, allocation (already strict), budgets,
  pricing specs, or every sum]
- [NEEDS CLARIFICATION: what a caller should observe when units differ — no total, an error,
  separate totals per currency, or today's sum with a blank currency]
- [NEEDS CLARIFICATION: whether the recommendation-summary test that expects 150 from
  100 USD + 50 EUR is behavior to retire]
- [NEEDS CLARIFICATION: whether a FOCUS row with billing currency different from pricing
  currency stays legal]
- [NEEDS CLARIFICATION: whether an empty currency may still be treated as USD, and whether
  `XXX` ("no currency") is its own unit]
- [NEEDS CLARIFICATION: whether an actual-cost point must carry its own currency, or whether
  the optional FOCUS billing currency is enough]
- [NEEDS CLARIFICATION: whether Go-only behavior is enough, or TypeScript callers must match]
- [NEEDS CLARIFICATION: whether any host has summed mixed currencies in production, or whether
  the only signal is the roadmap item]
