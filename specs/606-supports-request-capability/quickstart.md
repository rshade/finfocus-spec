# Quickstart: Validate Capability-Scoped Supports

All commands run from the repository root of the feature worktree.

## Prerequisites

- Run `npm ci` at the root, and in `sdk/typescript` for the TS checks.
- Run `make generate` after the proto change. Generated Go and TS bindings must show `capability` on
  `SupportsRequest`.

## Scenarios

| # | Proves | Command | Expected |
|---|--------|---------|----------|
| 1 | The field exists and is additive (FR-001, FR-003) | `make buf-lint` and `buf breaking --against <archive of origin/main>` | Both pass with no breaking changes |
| 2 | Plugins read the capability; unknown values become unspecified (US1, FR-005) | `go test ./sdk/go/pluginsdk/ -run TestRequestedCapability -count=1` | Pass |
| 3 | Reading allocates nothing (SC-005) | `go test ./sdk/go/pluginsdk/ -run '^$' -bench BenchmarkRequestedCapability -benchmem` | `0 allocs/op` |
| 4 | Hosts send it; `Supports` sends unspecified (US2, FR-007) | `go test ./sdk/go/pluginsdk/ -run 'TestClientSupportsFor' -count=1` | Pass over gRPC and Connect |
| 5 | Per-capability answers end to end (SC-001) | `go test ./sdk/go/pluginsdk/ -run 'TestSupportsCapabilityEndToEnd' -count=1` | Actual cost is supported, projected cost is not, for the same resource |
| 6 | The handler error log names the capability (FR-011) | `go test ./sdk/go/pluginsdk/ -run 'TestSupportsHandlerErrorLogsCapability' -count=1` | Pass |
| 7 | Validators accept unknown values (FR-012) | `go test ./sdk/go/testing/ -run 'TestValidateSupportsRequest' -count=1` | Pass |
| 8 | The conformance check passes ignoring and per-capability plugins, and fails erroring ones (US3) | `go test ./sdk/go/testing/ -run 'TestRPCCorrectness_SupportsCapability' -count=1` | Pass |
| 9 | Existing plugins unchanged (FR-015, SC-002) | `make test`, `go test -tags=integration ./sdk/go/testing/`, `go test -run TestConformance ./sdk/go/testing/` | All pass |
| 10 | TS passes the field (FR-009) | `cd sdk/typescript && npm run build && npx vitest run --root packages/client` | Pass, including `supports-capability.test.ts` |
| 11 | Docs are accurate (FR-016) | `make lint-markdown`; compile the README Go and TS snippets | Clean; snippets compile |

## Manual check

Read the guide's Supports section and confirm it answers three questions: what an unspecified request means,
how a plugin answers per capability, and which call a host uses to send one.
