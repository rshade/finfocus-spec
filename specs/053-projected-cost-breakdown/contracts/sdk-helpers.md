# SDK Contracts: cost_breakdown

## Go: `sdk/go/pluginsdk` (exported surface)

### Sentinel errors (`validation.go`)

```go
var (
    ErrCostBreakdownTooManyEntries = errors.New("cost_breakdown has too many entries")
    ErrCostBreakdownInvalidKey     = errors.New("cost_breakdown key is invalid")
    ErrCostBreakdownInvalidValue   = errors.New("cost_breakdown value is invalid")
    ErrCostBreakdownSumMismatch    = errors.New("cost_breakdown does not sum to cost_per_month")
    ErrCostBreakdownWithDryRun     = errors.New("cost_breakdown must be empty for dry-run responses")
)
```

Errors wrap these with context: the key, the value, and the sum, total and tolerance for a mismatch.
The response validator adds the `GetProjectedCostResponse:` prefix.

### `ValidateGetProjectedCostResponse` (behavior extended, signature unchanged)

```go
func ValidateGetProjectedCostResponse(resp *pbc.GetProjectedCostResponse) error
```

- The godoc rule list gains "8. cost_breakdown validation (if set): entry count, key format, finite
  non-negative values, sum matches cost_per_month within tolerance, empty for dry-run responses".
- The happy path, with or without a breakdown, has 0 allocations.

### `WithProjectedCostBreakdown` (`helpers.go`)

```go
// WithProjectedCostBreakdown returns a GetProjectedCostResponseOption that sets
// cost_breakdown to a copy of breakdown. A nil or empty map leaves the field empty.
//
// The option does not validate: the sum rule depends on cost_per_month, which
// another option may set. Call ValidateGetProjectedCostResponse on the result.
//
// Usage:
//
//   resp := pluginsdk.NewGetProjectedCostResponse(
//       pluginsdk.WithProjectedCostDetails(0.0104, "USD", 8.392, "On-demand Linux + 8GB gp2 root"),
//       pluginsdk.WithProjectedCostBreakdown(map[string]float64{
//           "compute":     7.592,
//           "root_volume": 0.80,
//       }),
//   )
func WithProjectedCostBreakdown(breakdown map[string]float64) GetProjectedCostResponseOption
```

## Go: `sdk/go/testing`

```go
type MockPlugin struct {
    // ...existing fields...

    // ProjectedCostBreakdown configures cost_breakdown on GetProjectedCost
    // responses. Values are weights: they are scaled so they sum to the
    // computed cost_per_month. Nil means no breakdown.
    ProjectedCostBreakdown map[string]float64
}
```

## TypeScript: `sdk/typescript/packages/client`

- Generated `GetProjectedCostResponse.costBreakdown: { [key: string]: number }`. It is exposed through
  `CostSourceClient.getProjectedCost` with no wrapper change.
- There is no TypeScript validator or helper in this feature (research R9).
