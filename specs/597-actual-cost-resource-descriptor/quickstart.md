# Quickstart: Validate Resource Descriptor on Actual Cost Requests

Run from the repository root (or the feature worktree).

## Prerequisites

- Go per `go.mod`, Node.js >= 22, `npm ci` at the root and in `sdk/typescript`
- `make generate` (buf from mise or `bin/buf`)

## 1. Proto and bindings

```bash
make generate && git diff --stat -- sdk/   # field 11 appears in Go and TS bindings
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'   # no output
```

## 2. Validators

```bash
go test ./sdk/go/pluginsdk/ -run 'TestValidateActualCostRequest'
go test ./sdk/go/testing/ -run 'TestValidateGetActualCostRequest|TestContract'
go test ./sdk/go/pluginsdk/ -run '^$' -bench 'BenchmarkValidateActualCostRequest' -benchmem
```

Expected: an oversized `attributes` value is rejected on both layers; a nil `resource` passes;
`BenchmarkValidateActualCostRequest_Valid` reports 0 allocs/op.

## 3. Mock and conformance

```bash
go test ./sdk/go/testing/ -run 'GetActualCost'
go test -v -run TestConformance ./sdk/go/testing/
```

Expected: `RPCCorrectness_GetActualCostWithResource` passes for the mock and for a plugin that
ignores `resource`; the mock's FOCUS record carries the descriptor's type, region, and SKU.

## 4. TypeScript

```bash
cd sdk/typescript && npm ci && npm run build && npm test
```

Expected: the pagination test shows the same `resource` on every page request.

## 5. Docs

```bash
make lint-markdown
```

See [contracts/proto.md](contracts/proto.md) for the field comment and
[data-model.md](data-model.md) for the rules.
