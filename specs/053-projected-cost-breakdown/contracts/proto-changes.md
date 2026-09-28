# Proto Contract: cost_breakdown

**File**: `proto/finfocus/v1/costsource.proto`

## GetProjectedCostResponse: new field 15

Append after `map<string, string> metadata = 14;`:

```protobuf
  // cost_breakdown reports the monthly cost of each named component that makes
  // up cost_per_month (e.g., an EC2 instance's compute plus its root volume).
  // It replaces parsing component costs out of billing_detail.
  //
  // Semantics:
  //   - Empty map: No breakdown available. This does NOT mean zero cost.
  //   - Values are monthly costs in the response currency, on the same basis
  //     as cost_per_month (730 hours/month for hourly-billed resources).
  //   - When non-empty, values sum to cost_per_month within a tolerance of
  //     max(0.01, 0.001 * cost_per_month).
  //   - Discounts and credits are already applied to the component they
  //     reduce; there are no negative entries.
  //
  // Producer constraints (enforced by pluginsdk.ValidateGetProjectedCostResponse):
  //   - Max 32 entries.
  //   - Keys: 1-64 bytes, lowercase snake_case: [a-z][a-z0-9_]*.
  //   - Values: finite and non-negative.
  //   - Sum within tolerance of cost_per_month (hard error, not a warning).
  //   - MUST be empty when dry_run_result is set.
  //
  // Recommended keys (non-exhaustive; consumers MUST accept others):
  //   compute, storage, root_volume, network, license, request, data_transfer
  //
  // Example (EC2 t3.micro + 8GB gp2 root volume):
  //   cost_per_month: 8.392
  //   cost_breakdown: { "compute": 7.592, "root_volume": 0.80 }
  //
  // Backward compatibility:
  //   - Empty map when not populated by plugin
  //   - Existing consumers unaffected (safe to ignore)
  //   - billing_detail is unchanged and may still describe components for humans
  map<string, double> cost_breakdown = 15;
```

## EstimateCostResponse: comment update only

Replace the "Future versions may add optional breakdown fields" sentence with:

```protobuf
// Future versions may add an optional cost breakdown (e.g., compute vs storage).
// It MUST follow the rules of GetProjectedCostResponse.cost_breakdown
// (key format, non-negative values, sum tolerance) for consistency.
```

## Compatibility checks

- `buf lint`: must pass (snake_case field name, map type).
- `buf breaking` against `main`: must pass, because this is an additive field.
- Wire: old readers skip field 15, and new readers see an empty map from old writers.

## Generated artifacts (regenerate with `make generate`, never edit by hand)

- `sdk/go/proto/finfocus/v1/costsource.pb.go`: `CostBreakdown map[string]float64` and
  `GetCostBreakdown()`.
- `sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts`:
  `costBreakdown: { [key: string]: number }`.
- Restore any unrelated `*.connect.go` reformatting with `git checkout` (see CLAUDE.md, 051 notes).
