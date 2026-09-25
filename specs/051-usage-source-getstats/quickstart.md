# Quickstart: Validating the Usage Source Service

This guide shows how to prove that feature 051 works end to end. Contracts:
[usage.proto](./contracts/usage.proto), [Go API](./contracts/go-sdk-api.md),
[TS API](./contracts/typescript-api.md). Rules: [data-model.md](./data-model.md).

## Prerequisites

- Go per `go.mod`, Node.js ≥ 22, `npm install` done at the repo root and in `sdk/typescript`
- `make generate` (installs `bin/buf`)

## 1. Contract is generated and additive

```bash
make generate
ls sdk/go/proto/finfocus/v1/usage.pb.go sdk/go/proto/finfocus/v1/usage_grpc.pb.go \
   sdk/go/proto/finfocus/v1/pbcconnect/usage.connect.go \
   sdk/typescript/packages/client/src/generated/finfocus/v1/usage_pb.ts
bin/buf lint
bin/buf breaking --against '.git#branch=main'
```

**Expected**: All four files exist, lint is clean, and there are no breaking changes (FR-007,
SC-004).

## 2. Served over both transports (US1-6, US2-1/2, SC-001, SC-002)

```bash
go test ./sdk/go/pluginsdk/ -run 'UsageSource|ToConnectError' -v
```

**Expected** scenarios in `usage_source_test.go`:

| Scenario | Expected |
|----------|----------|
| Plugin implementing only `GetStats` (plus the base `Plugin` stubs), served with `Web.Enabled=false` | `GetStats` over gRPC returns the fixture response, including its warning |
| Same plugin, `Web.Enabled=true` | Connect client returns a `proto.Equal` response; `Health/Check` for `finfocus.v1.UsageSourceService` is `SERVING` |
| Plugin without `GetStats` | `GetStats` → `Unimplemented` on both transports; health check for the usage service → `NOT_FOUND` |
| Reference source: only `start` set / `start > end` / historical on run-rate-only source | `InvalidArgument` on both transports, same message (not `Unknown` over Connect) |
| Reference source: RBAC failure / no creds | `PermissionDenied` with a "cannot list pods" message / `Unauthenticated`, identical code and message over gRPC and Connect |
| `toConnectError` unit test | `nil` → `nil`; `*connect.Error` unchanged; gRPC codes 1–16 map to the same `connect.Code`; plain error unchanged |
| Selector `namespace=payments` | Only `payments` workload rows returned |
| Counting `UnaryInterceptors` entry, gRPC mode | Runs once for `/finfocus.v1.UsageSourceService/GetStats` (FR-010) |
| Reference source with historical enabled, 2-hour window | `STATS_MODE_HISTORICAL`; every row, including allocatable, in `core-hours`/`GiB-hours`; identical over both transports (US1-2) |
| `metrics` includes unknown `gpu_seconds` | No error; one warning naming `gpu_seconds` |

## 3. Capability discovery (US3, SC-005)

```bash
go test ./sdk/go/pluginsdk/ -run 'Capabilit|IsValidCapability|UsageSourceWarning|LegacyMetadata_UsageStats|UsageStatsRoundTrip' -v
```

**Expected**:

- Inferred: the capabilities include `USAGE_STATS`, and the metadata has
  `supports_usage_stats=true`.
- Explicit `WithCapabilities(USAGE_STATS)`: exactly `[USAGE_STATS]`, with no pricing capabilities.
  This is the only configuration in which a host can tell a usage-only plugin from a pricing
  plugin (SC-005).
- `CapabilityToLegacyName(USAGE_STATS)` is `supports_usage_stats`.
- `IsValidCapability(14)` is true and `IsValidCapability(15)` is false.
- A usage source without explicit capabilities is served, and the captured log contains one `warn`
  entry about declaring capabilities explicitly. There is no warning when capabilities are
  explicit or when the plugin implements `PluginInfoProvider`.

## 4. Validation helper and harness (US4, SC-003)

```bash
go test ./sdk/go/testing/ -run 'GetStatsRequest|StatsResponse|UsageSourceHarness' -v
go test ./sdk/go/testing/ -run '^$' -bench 'ValidateStatsResponse' -benchmem
go test ./sdk/go/pluginsdk/ -run 'SubjectVocabulary' -v   # drift guard
```

**Expected**:

- Each of the request rules Q1–Q3 has a rejecting case. Run-rate requests, well-formed windows
  (including windows shorter than an hour), and unknown metric names are accepted.
- Each of the rules V1–V10 has a rejecting case, and the error `errors.Is(err,
  ErrInvalidStatsResponse)`.
- Every documented valid example passes: an unpriceable node, a `kind=cluster` control plane,
  `label.app.kubernetes.io/name`, an empty response in run-rate mode, and a custom metric.
- The harness `Client().GetStats` returns the implementation's response.
- The drift test passes.

## 5. TypeScript (US5, SC-006)

```bash
cd sdk/typescript/packages/client && npx vitest run test/usage-source.test.ts && npx tsc --noEmit
```

**Expected**: The msw handler at `/finfocus.v1.UsageSourceService/GetStats` receives the request.
`UsageSourceClient.getStats` returns rows, priceable entries, and `StatsMode.RUN_RATE`. The type
check is clean.

`npm run build` bundles ESM/CJS successfully but fails at tsup's DTS step with TS5101 (`baseUrl`
deprecated under TypeScript 6). The failure is the same on `main`, so it is not caused by this
feature; `npx tsc --noEmit` is the type-check gate.

## 6. Full regression

```bash
make test
golangci-lint run ./...
make lint-markdown
```

**Expected**: No existing test changes behavior (SC-004), and there are zero lint findings (the
baseline is 0).
