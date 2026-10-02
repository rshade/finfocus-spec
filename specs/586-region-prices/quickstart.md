# Quickstart: Validate Per-Region Retail Prices

Run from the repository root.

## 1. Proto is additive and generated code is current

```bash
make generate && git diff --exit-code -- sdk/
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
```

Expected: no diff, no lint findings, and no breaking changes.

## 2. Validation rules (User Stories 1-3)

```bash
go test ./sdk/go/testing/ -run 'TestValidateRegionPrices|TestMockRegionPrices' -v
go test ./sdk/go/pluginsdk/ -run 'RegionPrice' -v
```

Expected: two rows pass with the primary cost unchanged. Each bad-row kind fails with
`ErrInvalidRegionPrice` and a `region_prices[i].<field>` message. Rows on a dry-run projected
response fail with `ErrRegionPricesWithDryRun`.

## 3. Zero-allocation path (SC-004)

```bash
go test -run xxx -bench 'ValidateGetProjectedCostResponse_Valid|ValidateEstimateCostResponse_Valid|RegionPrices' \
  -benchmem -count 6 ./sdk/go/pluginsdk/
```

Expected: 0 allocs/op everywhere, and `_Valid` within noise of a `main` worktree run.

## 4. TypeScript (User Story 4)

```bash
cd sdk/typescript && npm ci && npm run build && npm test && npm run lint
```

## 5. Backward compatibility (SC-002)

```bash
make test && go test -tags=integration ./sdk/go/testing/ && go test -run TestConformance ./sdk/go/testing/
```
