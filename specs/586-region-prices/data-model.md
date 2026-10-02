# Data Model: Per-Region Retail Prices on Cost Responses

## RegionPrice (new)

| Field | Number | Type | Rule |
| --- | --- | --- | --- |
| `region` | 1 | string | Provider region name. Must not be empty. |
| `unit_price` | 2 | double | Same basis as the parent's unit price. Finite and >= 0. Zero is a real price. |
| `monthly_cost` | 3 | double | Same basis as the parent's monthly cost. Finite and >= 0. |
| `currency` | 4 | string | ISO 4217 code for this row. Must be valid. Never converted to the parent's currency. |

A region with no price is omitted, never sent as a zero row.

## GetProjectedCostResponse (changed)

| Field | Number | Change |
| --- | --- | --- |
| (none) | 16 | Left free for `price_options` (issue 588) |
| `region_prices` | 17 | **new** `repeated RegionPrice` |

`cost_per_month`, `unit_price`, and `currency` stay the requested region's price. The rows are never
summed into or compared with them. A response with `dry_run_result` set carries no rows.

## EstimateCostResponse (changed)

| Field | Number | Change |
| --- | --- | --- |
| (none) | 6 | Left free for `price_options` (issue 588) |
| `region_prices` | 7 | **new** `repeated RegionPrice` |

`cost_monthly` stays the requested configuration's price.

## Validation outcomes

| Input | Result |
| --- | --- |
| No rows | Unchanged from before; 0 allocs |
| Rows with any sum | Pass (no sum rule) |
| Nil row | `ErrInvalidRegionPrice`, `region_prices[i]` |
| Empty region | `ErrInvalidRegionPrice`, `region_prices[i].region` |
| NaN, Inf, or negative price | `ErrInvalidRegionPrice`, `region_prices[i].unit_price` or `.monthly_cost` |
| Empty or non-ISO currency | `ErrInvalidRegionPrice`, `region_prices[i].currency` |
| Rows on a dry-run projected response | `ErrRegionPricesWithDryRun` |

## New Go surface

| Package | Symbol |
| --- | --- |
| `sdk/go/testing` | `ValidateRegionPrices`, `ErrInvalidRegionPrice`, `ErrRegionPricesWithDryRun` |
| `sdk/go/pluginsdk` | `ErrInvalidRegionPrice`, `ErrRegionPricesWithDryRun` (aliases), `WithProjectedCostRegionPrices`, `WithEstimateCostRegionPrices` |
| `sdk/go/testing` | `MockPlugin.RegionPrices` |
