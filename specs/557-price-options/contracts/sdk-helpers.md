# SDK Contracts: price_options

## Go: `sdk/go/pluginsdk` (exported surface)

### Sentinel errors (`validation.go`)

```go
var (
    // ErrPriceOptionNil is returned when a price_options entry is nil.
    ErrPriceOptionNil = errors.New("price_options entry is nil")
    // ErrPriceOptionInvalidValue is returned when a price option's unit_price,
    // monthly_cost, or upfront_cost is NaN, infinite, or negative, or its
    // savings_fraction is NaN or infinite.
    ErrPriceOptionInvalidValue = errors.New("price_options value is invalid")
    // ErrPriceOptionsWithDryRun is returned when price_options is set alongside dry_run_result.
    ErrPriceOptionsWithDryRun = errors.New("price_options must be empty for dry-run responses")
)
```

Errors wrap these with the index and field, for example:

```text
GetProjectedCostResponse: price_options value is invalid: price_options[1].upfront_cost -5 must be finite and non-negative
EstimateCostResponse: price_options value is invalid: price_options[0].savings_fraction NaN must be finite
```

### Validators (behavior extended, signatures unchanged)

```go
func ValidateGetProjectedCostResponse(resp *pbc.GetProjectedCostResponse) error
func ValidateEstimateCostResponse(resp *pbc.EstimateCostResponse) error
```

- `ValidateGetProjectedCostResponse` godoc gains "9. price_options validation (if set): empty for
  dry-run responses, no nil entries, finite non-negative prices, finite savings_fraction. Never
  compared to cost_per_month."
- `ValidateEstimateCostResponse` godoc gains "3. price_options validation (if set): no nil entries,
  finite non-negative prices, finite savings_fraction. Never compared to cost_monthly."
- Both keep 0 allocations on valid input, with or without options.
- Responses without `price_options` behave exactly as before.

### Options (`helpers.go`)

```go
// WithProjectedCostPriceOptions returns a GetProjectedCostResponseOption that
// sets price_options to deep copies of options. No arguments leave the field
// empty. The list is advisory and does not change cost_per_month.
//
// The option does not validate. Call ValidateGetProjectedCostResponse on the result.
//
// Usage:
//
//   resp := pluginsdk.NewGetProjectedCostResponse(
//       pluginsdk.WithProjectedCostDetails(0.096, "USD", 70.08, "Consumption"),
//       pluginsdk.WithProjectedCostPriceOptions(
//           &pbc.PriceOption{
//               Category: pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_COMMITTED,
//               Model: "Reservation", Term: "1 Year",
//               UnitPrice: 0.0573, MonthlyCost: 41.83, UpfrontCost: 502.00,
//               SavingsFraction: (0.096 - 0.0573) / 0.096,
//           },
//       ),
//   )
func WithProjectedCostPriceOptions(options ...*pbc.PriceOption) GetProjectedCostResponseOption

// WithEstimatePriceOptions is the EstimateCostResponse counterpart of
// WithProjectedCostPriceOptions. savings_fraction compares monthly costs.
func WithEstimatePriceOptions(options ...*pbc.PriceOption) EstimateCostResponseOption
```

Nil entries are copied as nil (so the validator reports them), not dropped.

## Go: `sdk/go/testing`

```go
type MockPlugin struct {
    // ...existing fields...

    // ProjectedCostPriceOptions configures price_options on GetProjectedCost
    // responses. Each response gets a deep copy. Entries are returned as
    // configured: the mock does not validate or compute them. Dry-run
    // responses never carry options. Nil means no options.
    ProjectedCostPriceOptions []*pbc.PriceOption

    // EstimateCostPriceOptions configures price_options on EstimateCost
    // responses, with the same semantics.
    EstimateCostPriceOptions []*pbc.PriceOption
}
```

`plugintesting.ValidateProjectedCostResponse` and `plugintesting.ValidateEstimateCostResponse`
(`harness.go`) are unchanged (plan: Conformance scope).

## TypeScript: `sdk/typescript/packages/client`

- Generated `PriceOption` and `priceOptions: PriceOption[]` on `GetProjectedCostResponse` and
  `EstimateCostResponse`. These are exposed through `CostSourceClient.getProjectedCost` and
  `estimateCost` with no wrapper change.
- There is no TypeScript validator or helper in this feature (research R9).
