# Quickstart: Validating Alternative Retail Price Options

**Feature**: 557-price-options

This guide proves the feature works from end to end. For rules and signatures, see
[data-model.md](./data-model.md), [contracts/proto-changes.md](./contracts/proto-changes.md), and
[contracts/sdk-helpers.md](./contracts/sdk-helpers.md).

## Prerequisites

- Go (per `go.mod`), Node.js 22 or later, and `npm install` run at the repo root
- `make generate` works (it installs buf into `bin/`)

## 1. The proto change is additive (FR-001 to FR-004, FR-016, FR-017)

```bash
make generate
git status --short sdk/go/proto sdk/typescript/packages/client/src/generated
bin/buf lint
bin/buf breaking --against '.git#branch=main'
grep -n 'reserved 17\|reserved 7' proto/finfocus/v1/costsource.proto
```

**Expected**: only `costsource.pb.go` and `costsource_pb.ts` change (restore any unrelated
`*.connect.go` reformatting with `git checkout`). `buf lint` and `buf breaking` pass. The `grep`
prints nothing, because fields 17 and 7 are held by comment. In a worktree where
`.git#branch=main` cannot be read, `git archive main proto buf.yaml` into a scratch directory and run
`buf breaking --against <dir>`.

## 2. Validator rules (FR-008 to FR-013, SC-002 to SC-004)

```bash
go test ./sdk/go/pluginsdk/ -run 'PriceOption' -v
go test ./sdk/go/pluginsdk/ -run 'ValidateGetProjectedCostResponse|ValidateEstimateCostResponse'
```

**Expected**: a passing subtest for each case below, and every existing validator test still
passing.

| Case | Input | Result |
|------|-------|--------|
| Omitted | no `price_options` (both responses) | valid |
| Two options | `cost_per_month` 70.08; Reservation 1 Year and SavingsPlan 3 Years | valid, `cost_per_month` still 70.08 |
| Two options (estimate) | `cost_monthly` 70.08; the same two options | valid, `cost_monthly` still 70.08 |
| With breakdown | `cost_breakdown` sums to 70.08, plus two options | valid (options not in the sum) |
| Costs more | `savings_fraction` -0.25 | valid |
| Zero primary | `cost_per_month` 0, option `savings_fraction` 0 | valid |
| Unknown category | `category` 99 | valid |
| NaN / ±Inf | each of the four doubles | `ErrPriceOptionInvalidValue`, message names `price_options[i].<field>` |
| Negative price | `unit_price`, `monthly_cost`, or `upfront_cost` < 0 | `ErrPriceOptionInvalidValue` |
| Nil entry | `[]*pbc.PriceOption{nil}` | `ErrPriceOptionNil` |
| Dry run | options plus a dry-run result | `ErrPriceOptionsWithDryRun` |

## 3. Zero allocations (FR-014, SC-005)

```bash
go test ./sdk/go/pluginsdk/ -run 'PriceOptions.*Alloc' -v
go test ./sdk/go/pluginsdk/ -run '^$' -bench 'Validate(GetProjectedCost|EstimateCost)Response_(Valid|WithPriceOptions)' -benchmem
```

**Expected**: the `AllocsPerRun` test reports 0. Every listed benchmark shows `0 allocs/op`. To check
the 10% budget on `_Valid`, compare against a `main` worktree with prebuilt test binaries, as 053
did.

## 4. Options and mock through the harness (FR-015, SC-001, SC-006)

```bash
go test ./sdk/go/pluginsdk/ -run 'PriceOptions' -v
go test ./sdk/go/testing/ -run 'PriceOptions' -v
```

**Expected**:

- `WithProjectedCostPriceOptions` and `WithEstimatePriceOptions` deep-copy their arguments: changing
  the caller's entry afterwards does not change the response. No arguments leave the list empty.
- A `MockPlugin` with three `ProjectedCostPriceOptions` returns them over bufconn on
  `GetProjectedCost`, `cost_per_month` matches a mock with no options, and
  `pluginsdk.ValidateGetProjectedCostResponse` accepts the response. The same holds for
  `EstimateCostPriceOptions` on `EstimateCost`.
- A dry-run `GetProjectedCost` from the same mock carries no options.
- A response with options, encoded and then decoded as a message without field 16 (for example,
  through `protowire` field skipping or an older descriptor), still yields the same
  `cost_per_month` (SC-006).

## 5. TypeScript bindings (FR-016)

```bash
cd sdk/typescript && npm run build --workspaces && npx --workspace packages/client vitest run
```

**Expected**: the integration test's `getProjectedCost` and `estimateCost` responses carry two
`priceOptions` entries with the mocked values, and `costPerMonth` and `costMonthly` are unchanged.

## 6. Full gate

```bash
make test
golangci-lint run ./...
make lint-markdown
```

**Expected**: all pass. `TestUsageSourceNotRegistered/connect` can flake under full `make test`
load (see CLAUDE.md); rerun it in isolation before treating it as a failure.
