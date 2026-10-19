# Data Model: On-Demand Equivalent for Spot Results

## Selected price response

Both cost responses retain their selected monthly price and `pricing_category`. Each gains
one optional scalar:

| Response | Field number | Go type | TypeScript type |
| --- | --- | --- | --- |
| Projected | 18 | `OnDemandEquivalentMonthlyCost *float64` | `onDemandEquivalentMonthlyCost?: number` |
| Estimate | 8 | `OnDemandEquivalentMonthlyCost *float64` | `onDemandEquivalentMonthlyCost?: number` |

The field is the STANDARD monthly price of the same resource as the selected DYNAMIC result.
It shares the response currency and monthly basis, including 730 hours for hourly billing.
It is not a breakdown component or another charge.

## Presence and validation states

| State | Meaning | Result |
| --- | --- | --- |
| Absent, any category | Plugin does not know the equivalent | Valid |
| Present zero, DYNAMIC | Real zero on-demand price | Valid |
| Present finite positive, DYNAMIC | Known equivalent | Valid |
| Present, another category | Contradictory category | Category error |
| Present NaN, infinity, or negative | Invalid price | Value error |

Value errors take precedence over category errors. There is no maximum or ordering constraint
against the selected spot price. DYNAMIC with an absent equivalent remains valid and may
produce an advisory consistency warning.

## Relationships

`BatchCostResponse` embeds the existing response messages through `CostData`, so no batch
field is needed. The equivalent may be absent on one call and present on a later call when
the plugin has pricing data. No storage or migration is involved.
