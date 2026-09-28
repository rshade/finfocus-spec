# Quickstart: Validating Projected Cost Breakdown

**Feature**: 053-projected-cost-breakdown

This guide proves the feature works from end to end. For rules and signatures, see
[data-model.md](./data-model.md), [contracts/proto-changes.md](./contracts/proto-changes.md) and
[contracts/sdk-helpers.md](./contracts/sdk-helpers.md).

## Prerequisites

- Go (per `go.mod`), Node.js 22 or later, and `npm install` run at the repo root
- `make generate` works (it installs buf into `bin/`)

## 1. The proto change is additive

```bash
make generate
git status --short sdk/go/proto sdk/typescript/packages/client/src/generated
bin/buf lint
bin/buf breaking --against '.git#branch=main'
```

**Expected**: only `costsource.pb.go` and `costsource_pb.ts` change (restore any unrelated
`*.connect.go` reformatting), `buf lint` passes, and `buf breaking` reports no breaking changes.

## 2. Validator rules (SC-003)

```bash
go test ./sdk/go/pluginsdk/ -run 'CostBreakdown' -v
```

**Expected**: a passing subtest for each case below.

| Case | Input | Result |
|------|-------|--------|
| EC2 example | total 8.392, `{compute: 7.592, root_volume: 0.80}` | valid |
| Single component | total 8.0, `{storage: 8.0}` | valid |
| Empty | total 8.0, `{}` | valid |
| Rounding | total 10.00, components summing to 10.005 | valid (within 0.01) |
| Zero total | total 0, `{compute: 0}` | valid |
| Mismatch | total 8.392, sum 9.00 | `ErrCostBreakdownSumMismatch` |
| Negative | `{compute: -1}` | `ErrCostBreakdownInvalidValue` |
| NaN / Inf | `{compute: NaN}` | `ErrCostBreakdownInvalidValue` |
| Bad key | `RootVolume`, `root-volume`, `1st`, `""`, a 65-byte key | `ErrCostBreakdownInvalidKey` |
| Too many | 33 entries | `ErrCostBreakdownTooManyEntries` |
| Dry run | `dry_run_result` set and a non-empty breakdown | `ErrCostBreakdownWithDryRun` |

## 3. No performance regression (SC-004)

```bash
go test ./sdk/go/pluginsdk/ -run '^$' -bench 'ValidateGetProjectedCostResponse' -benchmem
```

**Expected**: the new `_WithCostBreakdown` and `_WithCostBreakdown32` benchmarks report
`0 allocs/op`, and the existing `_Valid` benchmark is within 10% of `main`.

## 4. Round trip over gRPC, and the mock and batch paths (SC-001, FR-013)

```bash
go test ./sdk/go/testing/ -run 'CostBreakdown' -v
```

**Expected**: a harness test configures `MockPlugin.ProjectedCostBreakdown` and receives the
breakdown on `GetProjectedCost` and inside the `CostData.projected_cost` results of a batch call.
The received breakdown passes `ValidateGetProjectedCostResponse`.

## 5. Backward compatibility (SC-002)

```bash
make test
```

**Expected**: all existing tests pass without changes, and plugins that don't set the breakdown
still validate.

## 6. TypeScript exposure (FR-010)

```bash
cd sdk/typescript/packages/client && npx vitest run && npx tsc --noEmit
```

**Expected**: the integration test's msw handler returns `costBreakdown`, and the test reads
`response.costBreakdown.compute`. Type-check with `tsc`; `npm run build` has a known tsup failure
(see CLAUDE.md).

## 7. Documentation (SC-005)

```bash
make lint-markdown
```

**Expected**: the README section `## Cost Breakdown Helpers (cost_breakdown)` exists, is in the
table of contents, uses exported symbol names correctly, and passes lint.
