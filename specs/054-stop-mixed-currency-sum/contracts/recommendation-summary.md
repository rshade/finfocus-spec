# Contract: Recommendation Summary Currency Rule

Both of these functions follow this contract. Their names, arguments, and result type do
not change, and they remain separate copies:

- `pluginsdk.CalculateRecommendationSummary(recommendations, projectionPeriod)`
- `testing.CalculateMockSummary(recommendations, projectionPeriod)`

`projectionPeriod` is copied onto the summary and does not affect currency rules.

## Inputs

Each recommendation contributes:

- its category and action, always, to the counts
- its impact's estimated savings and currency, only when the impact is present

A nil impact adds no savings and no currency. An empty currency adds savings and does not
add a currency.

## Outputs

| Input shape | Total estimated savings | Currency | Bucket savings |
|-------------|-------------------------|----------|----------------|
| No recommendations | 0 | empty | no buckets |
| 100 USD and 50 USD | 150 | USD | each bucket keeps its single-currency sum |
| 100 USD and 50 with an empty currency | 150 | USD | USD buckets include the empty-currency amount |
| Only empty currencies adding to 150 | 150 | empty | those buckets keep 150 |
| All amounts 0 in USD | 0 | USD | 0 in the buckets that had rows |
| 100 USD and 50 EUR, same category and same action | 0 | empty | that category and that action store 0 |
| 50 USD cost/modify and 75 EUR performance/rightsize | 0 | empty | cost 50, performance 75, modify 50, rightsize 75 |
| One category mixes USD and EUR, another category is only USD | 0 | empty | mixed category 0, single-currency category keeps its sum |

Counts always include every recommendation passed in.

## Non-behavior

- No error return and no transport failure.
- No currency conversion.
- No extra field for per-currency subtotals.
- No change to allocation currency resolution.
