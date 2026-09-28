# Research: Stop Mixed-Currency Recommendation Totals

## Decision: Withhold the grand total instead of summing and clearing the label

- **Decision**: If two or more distinct non-empty currencies appear, set
  `TotalEstimatedSavings` to 0 and `Currency` to `""`. Counts stay. The function still
  returns a summary. It does not return an error.
- **Rationale**: The parent resolved "error versus omitted total" as a withheld total with
  no signature change. Callers tell this from a real zero because a real zero still sets
  `Currency`. The old behavior (100 USD + 50 EUR = 150, currency empty) is what
  `TestCalculateRecommendationSummaryMixedCurrency` locks in today.
- **Alternatives considered**: Return an error (rejected: signature and RPC-failure change).
  Keep the sum and only clear the label (rejected: that is the bug). Return one subtotal
  per currency (rejected: new response shape, out of scope).

## Decision: Apply the same rule per category and per action

- **Decision**: Each savings bucket keeps its sum when it has at most one non-empty
  currency. A bucket with two or more non-empty currencies stores 0. Other buckets are
  independent.
- **Rationale**: The existing mock fixture uses COST/USD 50 and PERFORMANCE/EUR 75. Those
  buckets each have one currency, so they stay 50 and 75, and only the grand total changes
  from 125 to 0. The pluginsdk fixture uses two COST/RIGHTSIZE rows (USD and EUR), so that
  category bucket and that action bucket are withheld too.
- **Alternatives considered**: Withhold every bucket whenever the grand total is mixed
  (rejected: would zero the 50 and 75 that the parent said stay). Add a per-bucket currency
  (rejected: the maps are amount-only and the message does not change).

## Decision: Empty currency is not a second currency; XXX is

- **Decision**: Skip empty strings when collecting distinct currencies. Still add those
  amounts into a bucket that is not withheld. Compare non-empty codes with exact string
  equality, so XXX is its own code and "usd" is not "USD".
- **Rationale**: Parent: empty currency and XXX keep today's meaning. Today's summary
  ignores empty when detecting a mismatch and still adds the amount. Allocation's
  empty-as-USD rule is a different function and must not change.
- **Alternatives considered**: Treat empty as USD inside the summary (rejected: that would
  copy allocation into a path that today leaves the currency blank when nothing is set).
  Treat XXX as empty (rejected: XXX is a real ISO code and is non-empty).

## Decision: Keep two copies

- **Decision**: Update `pluginsdk.CalculateRecommendationSummary` and
  `testing.CalculateMockSummary` in place. Update both comments so they describe the same
  rule. Do not extract a shared helper.
- **Rationale**: `mock_plugin.go` already says the copy exists because `pluginsdk` imports
  `testing`. A new shared internal package is out of this slice.
- **Alternatives considered**: One implementation in `pluginsdk` called from tests only
  (rejected: the mock plugin calls `CalculateMockSummary` from the testing package, and
  the import cycle remains). Delete the mock copy (rejected: not required, and it would
  widen the change).

## Decision: No documentation rewrite unless a sentence states the old rule

- **Decision**: Search README and developer-guide markdown for a statement that mixed
  recommendation currencies are summed and only the label is cleared. Correct that
  sentence if it exists. Do not edit the currency-converter FAQ, FOCUS SQL samples,
  historical spec quickstarts, or the roadmap checkbox for the larger pattern.
- **Rationale**: FR-011 and the parent instruction. A search of `sdk/go/pluginsdk/README.md`
  and `PLUGIN_DEVELOPER_GUIDE.md` during planning did not find that sentence. The old
  behavior lives in the function comment and the tests, which this slice updates.
- **Alternatives considered**: Add a new summary section to the plugin README (rejected:
  the parent said not to rewrite unrelated docs, and no stale sentence was found).

## Decision: Tests that must move

- **Decision**: Change `TestCalculateRecommendationSummaryMixedCurrency` so 100 USD + 50 EUR
  expects total 0, empty currency, cost-bucket 0, and rightsize-bucket 0, with count 2.
  Change the mock case `"mixed currencies clears currency field"` so the total is 0 and
  the 50 and 75 bucket sums stay. Add one case, on both copies, where one category mixes
  currencies (bucket 0) and another category stays single-currency (sum kept). Keep
  `TestCalculateRecommendationSummaryConsistentCurrency` at 150 USD.
- **Rationale**: Those are the fixtures the parent named. The extra case is required
  because neither existing fixture shows a withheld bucket beside a kept bucket.
- **Alternatives considered**: Only change the grand total assertions (rejected: the
  pluginsdk fixture's category and action buckets must also become 0, or the test would
  still allow a mixed bucket sum).
