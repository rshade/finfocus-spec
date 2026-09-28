# Data Model: Stop Mixed-Currency Recommendation Totals

No new stored types and no protobuf changes. This page records the rules over the summary
that already exists.

## Recommendation summary

| Field | Rule |
|-------|------|
| Count | Number of recommendations, including those with no savings impact. Never reduced because a total was withheld. |
| Count by category | Same. One increment per recommendation in that category. |
| Count by action | Same. One increment per recommendation with that action. |
| Total estimated savings | Sum of savings when the page has at most one distinct non-empty currency. Otherwise 0. |
| Currency | That single non-empty currency, or empty when none was stated or when the total is withheld. |
| Savings by category | Per category, the bucket rule below. |
| Savings by action | Per action, the bucket rule below. |

## Savings amount

- **Amount**: The recommendation's estimated savings. Added only when the recommendation has
  an impact.
- **Currency**: A string. Empty means not stated. It does not count as a distinct currency.
  Any other string, including `XXX` and mixed-case codes, counts. Comparison is exact.

## Bucket rule

A bucket is one category or one action.

1. Start from zero.
2. Add every impact amount that falls in the bucket, including amounts with an empty
   currency.
3. Collect the distinct non-empty currencies in the bucket.
4. If that set has two or more members, the stored savings are 0.
5. Otherwise the stored savings are the sum from step 2.

The grand total uses the same rule across the whole page. When step 4 fires for the page,
currency is empty. When the page has exactly one non-empty currency, currency is that code
even if the sum is 0.

## What does not change

- Allocation's resolved currency, including empty priced currencies becoming USD.
- A FOCUS record with one billing currency and a different pricing currency.
- Actual-cost rows, which still have no currency of their own.
- The summary's fields and the two functions' arguments and result type.

## Signals callers can rely on

| What the caller sees | Meaning |
|----------------------|---------|
| Total 150, currency USD | Every non-empty currency on the page was USD. Empty-currency amounts are included. |
| Total 0, currency USD | A real zero in USD. Not withheld. |
| Total 0, currency empty, and two or more non-empty currencies were in the input | Grand total withheld. |
| Total 150, currency empty | Every currency was empty. Nothing was withheld. This matches today. |
| Category savings 0 | Either that category's amounts sum to 0 in one currency, or that category mixed non-empty currencies. The grand total's currency is the page-level signal, not a per-bucket flag. |
