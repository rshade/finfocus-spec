# Idea Research: Multi-Currency Cost Segregation Pattern

- **Slug**: multi-currency-segregation
- **Created**: 2026-09-27
- **Evidence confidence (overall)**: medium

Overall confidence is medium: in-repo behavior is cited and concrete, but demand is a single
discovery issue with no incident, metric, or external user report.

## Users & Demand

- The only open request is GitHub issue #190, filed by the repo owner (rshade) on 2025-12-22.
  Title prefix is `discovery:`. Labels are `enhancement`, `roadmap/future`, and `effort/large`.
  The body is two sentences: standardize multi-currency record handling so different units are
  not aggregated, and limit the work to validation/typing with no conversion math.
  No milestone, assignee, or follow-up from a plugin author.
  — [source: https://github.com/rshade/finfocus-spec/issues/190] (confidence: high, cited)
- `ROADMAP.md` lists the same issue under "Proposed for Discussion (Discovery)", unchecked,
  effort tag `[L]`. It is not in a completed or committed milestone.
  — [source: `ROADMAP.md` lines 83–89] (confidence: high, cited)
- The only issue comment is an auto-generated CodeRabbit plan widget. It is not a user report.
  It names `https://github.com/rshade/pulumicost-spec/issues/101`, which was not fetched.
  — [source: issue #190 comments via `gh issue view --comments`] (confidence: high, cited)
- No support ticket, usage metric, or host bug is attached to the issue.
  Strength of observed demand is therefore low; the want is stated, not measured.
  — [source: issue #190 body] (confidence: high, cited)

## Prior Art

- `specs/012-iso4217-currency/` (directory name; internal branch label `013-iso4217-currency`)
  specified a reusable ISO 4217 package: validate codes, metadata, list currencies. It does not
  define aggregation or segregation. Closed as issue #101,
  "feat: Extract ISO 4217 currency validation as reusable package".
  — [source: `specs/012-iso4217-currency/spec.md`; issue #101 via `gh issue list`]
  (confidence: high, cited)
- `sdk/go/currency` implements that scope: `IsValid`, `GetCurrency`, `AllCurrencies`,
  `GetSymbol`, `FormatAmount`. Package docs describe display formatting, not unit-safe sums.
  `FormatAmount` uses each currency's minor units (JPY 0, USD 2, KWD 3).
  — [source: `sdk/go/currency/doc.go`, `sdk/go/currency/README.md`, `sdk/go/currency/symbol.go`]
  (confidence: high, cited)
- Issue #358 (closed 2026-02-04) evaluated `golang.org/x/text/currency` and
  `github.com/bojanz/currency` and kept the custom package. Explicit non-goals included
  currency conversion and arbitrary-precision arithmetic. Revisit `bojanz/currency` only if
  "precise financial calculations" become a requirement. `pkg.go.dev` was not fetched
  (host not on the safe list). The `bojanz/currency` repository page was not fetched.
  — [source: https://github.com/rshade/finfocus-spec/issues/358;
  `sdk/go/currency/CLAUDE.md` "Why Custom Implementation vs External Libraries";
  `ROADMAP.md` lines 76–78] (confidence: high, cited)
- Allocation already refuses mixed currencies. `ResolveCurrency` returns the single distinct
  non-empty currency on `priced=true` entries, or `"USD"` when every such currency is empty.
  More than one distinct value wraps `ErrMixedCurrency` and `codes.InvalidArgument`.
  Response rows must use that resolved currency. Empty currency is not case-folded.
  — [source: `sdk/go/testing/allocation.go` `ResolveCurrency`; `docs/allocator.md`;
  `specs/052-allocator-allocate/data-model.md` "Resolved currency"] (confidence: high, cited)
- Recommendation summaries do the opposite. `CalculateRecommendationSummary` adds
  `estimated_savings` across recommendations, then sets `currency` to `""` when two non-empty
  currencies differ. The test `TestCalculateRecommendationSummaryMixedCurrency` expects
  100 USD + 50 EUR to yield `TotalEstimatedSavings == 150` and an empty currency.
  The same logic is copied in `CalculateMockSummary` to avoid an import cycle.
  — [source: `sdk/go/pluginsdk/helpers.go` lines 1173–1220;
  `sdk/go/pluginsdk/helpers_test.go` lines 883–911;
  `sdk/go/testing/mock_plugin.go` lines 1766–1803] (confidence: high, cited)
- FOCUS conformance validates each currency code independently. `billing_currency` must be a
  valid ISO 4217 code. `pricing_currency`, if non-empty, must also be valid. A record with
  billing `USD` and pricing `EUR` is a passing case
  (`TestValidateFocusRecord_ISO4217Currency`, "valid pricing currency"). Spec 027 lists
  "mixed currency scenarios (BillingCurrency vs PricingCurrency)" as an edge case and then
  puts "Currency conversion or multi-currency validation" out of scope. It also puts
  cross-record validation out of scope.
  — [source: `sdk/go/pluginsdk/focus_conformance.go` `validateCurrencyFields`;
  `sdk/go/pluginsdk/focus_conformance_test.go` around the ISO 4217 table;
  `specs/027-finops-validation/spec.md` edge cases and Out of Scope] (confidence: high, cited)
- `specs/003-getpricingspec/spec.md` assumed "Currency will initially be USD; multi-currency
  support may be added later" and listed "Currency conversion between different currencies"
  as out of scope. The schema still accepts any string currency, described as an ISO 4217 code,
  and does not restrict a document to USD.
  — [source: `specs/003-getpricingspec/spec.md` Assumptions and Out of Scope;
  `schemas/pricing_spec.schema.json` `currency`] (confidence: high, cited)
- `docs/focus-columns.md` section "Multi-Currency Billing" shows one FOCUS builder call with
  billing period currency `EUR` and pricing currency `USD`. Later SQL examples
  `SUM(BilledCost)` grouped only by service, commitment, or tag, with no `BillingCurrency`
  in the `GROUP BY`. — [source: `docs/focus-columns.md` lines 431–443 and 487–515]
  (confidence: high, cited)
- `PLUGIN_DEVELOPER_GUIDE.md` FAQ "How do I handle different currency conversions?" shows a
  `CurrencyConverter` that divides and multiplies by stored rates. That is conversion math,
  which issue #190 excludes. — [source: `PLUGIN_DEVELOPER_GUIDE.md` FAQ around lines 3433–3451]
  (confidence: high, cited)
- No TypeScript helper matches `CalculateRecommendationSummary` or `currencyMismatch`.
  Currency appears on generated messages and in a focus-record builder that rejects an empty
  billing currency code but does not check ISO 4217 or mixed sets.
  — [source: search of `sdk/typescript` for `currencyMismatch` / `CalculateRecommendation`
  (no matches); `sdk/typescript/packages/client/src/builders/focus-record.ts`]
  (confidence: medium, cited)

## Market & Context

- The cost of doing nothing, inside this repo, is two live policies: allocation fails closed
  on mixed currency; recommendation summaries add the numbers and drop the label. A host that
  copies the summary helper will publish a unitless total. A host that copies allocation will
  reject the same shape. — [source: helpers.go and allocation.go, cited above]
  (confidence: high, cited)
- Official FOCUS text on summing across `BillingCurrency` was not fetched. In-repo FOCUS docs
  already treat billing currency and pricing currency as different columns on one row, and
  they demonstrate ungrouped `SUM(BilledCost)`.
  — [source: `docs/focus-columns.md`; focus.finops.org fetch skipped, host not on safe list]
  (confidence: medium, cited for the in-repo docs only)
- Issue #358 is the closest external-library survey. It treated a money type as unnecessary
  for display. A segregation feature that adopts that library would reverse that decision.
  — [source: issue #358] (confidence: high, cited)
- Coping pattern today is per-message `string` currency plus optional ISO checks, not a
  typed amount. — [source: proto currency fields listed under Data & Constraints]
  (confidence: high, cited)

## Data & Constraints

- Monetary amounts in the public protos are `double` (`float64`), not decimal.
  Spec 027 states that assumption. Issue #358 declined arbitrary precision.
  — [source: `specs/027-finops-validation/spec.md` Assumptions; issue #358] (confidence: high, cited)
- Currency fields found, each a single `string`, not a repeated currency:
  `GetProjectedCostResponse.currency`, `PricingSpec.currency`, `EstimateCostResponse.currency`,
  `RecommendationImpact.currency`, `RecommendationSummary.currency`,
  `FocusCostRecord.billing_currency` and `pricing_currency`,
  `PricedResource.currency`, `AllocationRow.currency`, budget amount currency.
  — [source: `proto/finfocus/v1/costsource.proto`, `focus.proto`, `allocation.proto`,
  `budget.proto`] (confidence: high, cited)
- `ActualCostResult` has `cost` (`double`) and optional `focus_record`, and no currency field
  of its own. `GetActualCostResponse` is a repeated list of those results with no summary
  currency. Summing `results[i].cost` cannot see a unit unless the caller opens
  `focus_record.billing_currency`, which may be unset.
  — [source: `proto/finfocus/v1/costsource.proto` `ActualCostResult` and
  `GetActualCostResponse`] (confidence: high, cited)
- `RecommendationSummary` is defined as one total and one currency for a page
  (`total_estimated_savings`, `currency`, plus per-category savings maps with no currency).
  The helper's mismatch behavior is to keep the summed double and clear `currency`.
  — [source: `proto/finfocus/v1/costsource.proto` `RecommendationSummary`; helpers.go]
  (confidence: high, cited)
- ISO 4217 validation is case-sensitive and accepts `XXX` ("no currency") as valid.
  Allocation compares currency strings exactly and does not call `currency.IsValid`.
  — [source: `specs/012-iso4217-currency/spec.md` edge case for `XXX`;
  `sdk/go/testing/allocation.go` `ResolveCurrency`] (confidence: high, cited)
- Empty priced currency becomes `"USD"` in allocation. Recommendation summary ignores empty
  impact currency and still adds that impact's savings into the detected currency's total.
  — [source: `ResolveCurrency`; `CalculateRecommendationSummary`] (confidence: high, cited)
- No FX rate table, converter type, or exchange RPC exists under `sdk/go/currency`.
  The developer-guide converter is an example, not an SDK API.
  — [source: `sdk/go/currency/` file list; `PLUGIN_DEVELOPER_GUIDE.md`] (confidence: high, cited)

## Evidence Against the Idea

- The issue does not name a failing RPC, a host, or a numeric example. Labels put it in the
  future icebox at large effort. Building a cross-SDK pattern from that text risks a wide
  proto and behavior change for a problem that is not yet scoped.
  — [source: issue #190; `ROADMAP.md`] (confidence: high, cited)
- A tested behavior already aggregates mixed recommendation savings on purpose (150 from
  100 USD + 50 EUR). Standardizing on "do not aggregate" would change that contract and the
  duplicated mock. That is a compatibility question the issue does not mention.
  — [source: `helpers_test.go` `TestCalculateRecommendationSummaryMixedCurrency`]
  (confidence: high, cited)
- Spec 027 explicitly deferred multi-currency and cross-record validation. FOCUS records are
  allowed to carry two different valid currencies (billing vs pricing). A rule that rejects
  "a record with multiple currencies" would contradict that passing test unless the rule is
  only about sums, not about the FOCUS pair.
  — [source: `specs/027-finops-validation/spec.md`; focus conformance test "valid pricing currency"]
  (confidence: high, cited)
- Issue #358 said the SDK does not need conversion or a money library. Segregation can stay
  inside validation, which matches the issue boundary, but a typed `Money` amount would
  reopen the library decision the project just closed.
  — [source: issue #358] (confidence: medium, cited)
- `ActualCostResult` cannot express a currency without a proto change or a convention that
  reads `focus_record`. The issue says validation/typing only. Whether that allows a new
  field is not stated. Proto and generated SDK files are shared across other work, so a
  field add is not a local helper.
  — [source: `ActualCostResult` in `costsource.proto`] (confidence: medium, cited; the
  "shared files" constraint is from this assessment's tasking, not from the issue)
- The developer guide currently teaches conversion. That is evidence the public docs disagree
  with the issue boundary. It is also evidence that "no conversion" is not yet the documented
  plugin contract, so a segregation feature may be mistaken for an FX feature.
  — [source: `PLUGIN_DEVELOPER_GUIDE.md` FAQ] (confidence: medium, cited)

## Gaps & Open Questions

- [NEEDS CLARIFICATION: which messages are in scope — recommendation summaries, actual-cost
  pages, projected/estimate responses, FOCUS rows, allocation (already strict), budgets,
  pricing specs, or all of them]
- [NEEDS CLARIFICATION: required outcome when units differ — hard error, omit the sum,
  return per-currency subtotals, or keep today's "sum and blank the currency"]
- [NEEDS CLARIFICATION: whether `CalculateRecommendationSummary` must stop adding mixed
  savings, given the test that locks in 100+50=150]
- [NEEDS CLARIFICATION: whether billing currency and pricing currency on one FOCUS row stay
  legal, with segregation applying only when amounts are added]
- [NEEDS CLARIFICATION: whether empty currency may still default to USD, and whether `XXX`
  is its own unit]
- [NEEDS CLARIFICATION: whether `ActualCostResult` gains a currency field (proto) or callers
  must use `focus_record.billing_currency`]
- [NEEDS CLARIFICATION: Go-only versus TypeScript parity]
- [NEEDS CLARIFICATION: any production host that summed mixed currencies, or only the roadmap
  note]

## Sources

- [finfocus-spec issue 190](https://github.com/rshade/finfocus-spec/issues/190)
  (host: github.com, policy: allowlisted)
- [finfocus-spec issue 358](https://github.com/rshade/finfocus-spec/issues/358)
  (host: github.com, policy: allowlisted)
- [finfocus-spec issue 101](https://github.com/rshade/finfocus-spec/issues/101)
  (host: github.com, policy: allowlisted; title and state from `gh issue list`, body not opened)
- [pulumicost-spec issue 101](https://github.com/rshade/pulumicost-spec/issues/101)
  (host: github.com, policy: allowlisted but not fetched; linked from an untrusted comment)
- [golang.org/x/text/currency](https://pkg.go.dev/golang.org/x/text/currency)
  (host: pkg.go.dev, policy: auto-refused: host not on safe list)
- [ISO 4217 currency codes](https://www.iso.org/iso-4217-currency-codes.html)
  (host: iso.org, policy: auto-refused: host not on safe list; named inside
  `sdk/go/currency/doc.go`, not fetched)
- [FOCUS specification site](https://focus.finops.org)
  (host: focus.finops.org, policy: auto-refused: host not on safe list)
- In-repo paths cited above, read at commit `fc1402d` on branch
  `assess/190-multi-currency-segregation`
