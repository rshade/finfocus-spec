# Contract: `RegionPrice` and `region_prices`

File: `proto/finfocus/v1/costsource.proto`

```protobuf
// RegionPrice is one region's retail price for the same resource and basis as the
// response that carries it. It is advisory: it is never summed into, compared with,
// or required to match the response's primary cost.
message RegionPrice {
  // region is the provider's region name, for example "eastus". Required.
  string region = 1;
  // unit_price uses the same basis as the parent response's unit price.
  // Finite and non-negative. Zero is a real price, for example a free tier.
  double unit_price = 2;
  // monthly_cost uses the same basis as the parent response's monthly cost.
  // Finite and non-negative.
  double monthly_cost = 3;
  // currency is the ISO 4217 code for this row. It may differ from the parent
  // response's currency. Rows are never converted.
  string currency = 4;
}

message GetProjectedCostResponse {
  // ... fields 1-15 unchanged ...
  repeated PriceOption price_options = 16;  // issue 588 (PR 599)

  // region_prices lists retail prices for the same resource in other regions.
  // Advisory: it is never summed into cost_per_month, and cost_per_month, unit_price,
  // and currency stay the requested region's price. Empty means the plugin supplied
  // no other regions. A region with no price is omitted, never sent as a zero row.
  // Must be empty when dry_run_result is set.
  repeated RegionPrice region_prices = 17;
}

message EstimateCostResponse {
  // ... fields 1-5 unchanged ...
  repeated PriceOption price_options = 6;  // issue 588 (PR 599)

  // region_prices lists retail prices for the same resource in other regions.
  // Advisory: it is never summed into cost_monthly, which stays the requested
  // configuration's price. Empty means the plugin supplied no other regions. A region
  // with no price is omitted, never sent as a zero row.
  repeated RegionPrice region_prices = 7;
}
```

## Compatibility

Additive. `buf breaking` against `main` must pass. Old hosts ignore tags 17 and 7. New hosts read an
empty list from old plugins.

## Go SDK surface

```go
// sdk/go/testing
var ErrInvalidRegionPrice = errors.New("invalid region price")
var ErrRegionPricesWithDryRun = errors.New("region_prices must be empty for dry-run responses")
func ValidateRegionPrices(rows []*pbc.RegionPrice) error

// sdk/go/pluginsdk
var ErrInvalidRegionPrice = plugintesting.ErrInvalidRegionPrice
var ErrRegionPricesWithDryRun = plugintesting.ErrRegionPricesWithDryRun
func WithProjectedCostRegionPrices(rows ...*pbc.RegionPrice) GetProjectedCostResponseOption
func WithEstimateCostRegionPrices(rows ...*pbc.RegionPrice) EstimateCostResponseOption
```

## TypeScript

Generated `RegionPrice` type; `GetProjectedCostResponse.regionPrices` and
`EstimateCostResponse.regionPrices` (`RegionPrice[]`, default `[]`).
