# Quickstart: Stop Mixed-Currency Recommendation Totals

## Prerequisites

- Go 1.27.1 from `go.mod`
- Repository root as the working directory

## Prove the summary rule

From the repository root:

```bash
go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -count=1
```

Focused cases:

```bash
go test ./sdk/go/pluginsdk/ -count=1 -run 'TestCalculateRecommendationSummary'
go test ./sdk/go/testing/ -count=1 -run 'TestCalculateMockSummary|mixed currencies'
```

## Outcomes that must hold

See [contracts/recommendation-summary.md](./contracts/recommendation-summary.md) for the
full table. The runs above are enough when these are true:

- 100 USD + 50 USD summarizes to 150 USD.
- 100 USD + 50 EUR summarizes to total 0 and an empty currency, and the shared category
  and action buckets are 0. The count is still 2.
- 50 USD in one category plus 75 EUR in another keeps those bucket sums and withholds
  only the grand total (0, empty currency), not 125.
- A category that mixes currencies stores 0 while a different single-currency category
  keeps its sum.
- `ResolveCurrency` tests in `sdk/go/testing` still pass, including empty currency
  resolving to USD for allocation. This slice does not change that rule.

## Out of scope for this check

Do not run `make generate`. Do not expect TypeScript tests, actual-cost currency fields,
or a per-currency subtotal on the summary.
