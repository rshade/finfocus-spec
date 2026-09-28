# Concept: Multi-Currency Cost Segregation

- **Slug**: multi-currency-segregation
- **Created**: 2026-09-27
- **Recommended option**: Option B — Stop the one mixed sum

## Options

### Option A — Leave it iceboxed

- **Sketch**: Ship nothing. Allocation keeps rejecting mixed currencies. Recommendation
  summaries keep adding mixed savings and clearing the currency label. Actual-cost points
  stay without their own currency. The discovery issue stays on the future roadmap.
- **Appetite**: small
- **Trade-offs**: Wins by spending no compatibility budget and by honoring the thin demand
  (one owner-filed discovery issue, no named host, no incident). Sacrifices every success
  metric in `problem.md`: the mixed sum stays, and the two policies stay. Risk is low of
  building the wrong rule. Risk is that the next caller copies the summary behavior.
- **Rabbit holes**: Treating the developer-guide converter example or the ungrouped SQL
  samples as a reason to start a docs-and-SDK program while the issue is still a sketch.

### Option B — Stop the one mixed sum

- **Sketch**: The smallest change that moves the metrics. Where the SDK already totals
  money in more than one currency, it stops presenting one number. Today that is the
  recommendation summary (100 USD + 50 EUR becomes 150 with a blank currency). That path
  would follow the allocation path: unlike currencies are not added. Callers can already
  see a single currency on each recommendation impact and on each priced allocation row.
  No exchange rates. No new amount type. No new fields on actual-cost points. The FOCUS
  case of billing in one currency and pricing in another stays legal, because that is not
  a sum. The locked test that expects 150 would change.
- **Appetite**: small
- **Trade-offs**: Wins the two measurable targets that have a baseline: mixed-sum paths
  from 1 to 0, and contradictory totaling policies from 2 to 1, without conversion and
  without reopening the closed library decision (issue #358). Sacrifices the actual-cost
  metric (a caller still cannot name the currency of a summed actual-cost page) and does
  not invent per-currency subtotals. Risk: a host may depend on the unitless 150. That
  dependency is not in evidence, but the test treats the sum as intended. The issue label
  `effort/large` describes an unbounded pattern, not this slice; the slice's appetite is
  small only if it stays on the existing summary behavior.
- **Rabbit holes**:
  - Extending the same rule onto actual-cost pages, which have no currency on the point.
    That pulls in a protocol change and every generated binding.
  - Returning a total per currency instead of withholding the total. That is a new
    response shape, not a guard on the current one.
  - Redefining empty currency and `XXX`, or rejecting a FOCUS row whose billing currency
    differs from its pricing currency.
  - Matching the rule in TypeScript, where the summary helper does not exist.
  - Editing the developer-guide converter and the FOCUS SQL samples in the same change.

### Option C — Buy a money type

- **Sketch**: Replace bare amounts-plus-currency-strings with an external money type that
  refuses to add unlike units. Issue #358 already compared `github.com/bojanz/currency`
  and `golang.org/x/text/currency` and kept the in-repo validator. This option reverses
  that: callers would hold a typed amount, and mixed addition would fail in the type
  rather than in one helper.
- **Appetite**: large
- **Trade-offs**: Wins a single rule for every future sum, which is what "everywhere" in
  the problem statement asks for. Sacrifices the #358 decision (zero dependencies,
  display-only scope, no arbitrary precision) and almost certainly touches public
  signatures across cost, recommendation, budget, and allocation messages. Risk of scope
  is the issue's `effort/large` label made concrete. It also drifts toward arithmetic the
  issue said not to do: a money type is how conversion usually arrives later.
- **Rabbit holes**: Decimal versus `float64` migration, generated protobuf still using
  `double` plus `string`, TypeScript having no equivalent type, and relitigating #358.

## Recommendation

Option B. It is the only option that changes the cited harm (a summed total of unlike
currencies) inside the issue's boundary (no conversion math) and inside the appetite the
evidence can support. Option A leaves the tested 150 in place. Option C buys a library
the project already turned down, for a problem that one totaling path demonstrates.

Option B does not claim the actual-cost metric. That stays unmet on purpose. If the
desired outcome is per-currency subtotals, or a currency on every actual-cost point, B is
the wrong shape and the work should wait.

## Out of Scope (for the recommended option)

- Currency conversion, FX rates, and exchange math.
- Adopting `bojanz/currency`, `x/text/currency`, or any other money library.
- Changing `float64` amounts to fixed-point or decimal.
- Locale-specific formatting.
- Making billing currency and pricing currency on one FOCUS row illegal.
- Adding a currency to actual-cost points, or any other protocol field.
- Per-currency subtotals as a new response.
- TypeScript parity, unless specification finds a TypeScript path that already sums mixed
  currencies (research found none).
- Rewriting the developer-guide converter or the ungrouped SQL examples, except a
  one-line warning if they sit next to the behavior being corrected.

## Assumptions to Validate

- The outcome callers need is "do not add unlike currencies," matching allocation, not
  "return one total per currency" and not "keep the sum and blank the label."
- Retiring the test that expects 100 USD + 50 EUR = 150 is acceptable. No host relies on
  that unitless total. This is not shown; it is assumed.
- Empty currency may keep today's meaning (ignored on recommendation impacts; default USD
  on priced allocation entries). `XXX` stays a normal valid code.
- A FOCUS row with two different currencies remains valid.
- Go is enough for this slice.
- No protocol change is required to stop the recommendation-summary sum.
- The discovery issue's `effort/large` label is about an unbounded pattern. It is not a
  requirement to boil every cost RPC into this slice.
