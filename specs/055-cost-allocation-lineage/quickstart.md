# Quickstart: Standardized Cost Allocation Lineage Metadata

## Prerequisites

- Go 1.27.1 from `go.mod`
- `buf` 1.32.1 (via `mise`, or `bin/buf`)
- Repository root as the working directory

## Prove the contract and builder

From the repository root:

```bash
go test ./sdk/go/pluginsdk/ -count=1 -run 'TestLineage|TestActualCostResultLineage'
```

Full SDK suite:

```bash
go build ./... && go test ./sdk/go/... -count=1
```

Lint:

```bash
golangci-lint run ./sdk/go/pluginsdk/...
buf lint
```

## Regenerate bindings after a proto edit

```bash
make generate
```

Then confirm the new types exist in both SDKs:

```bash
grep -rn "LineageNode" sdk/go/proto | head
grep -rln "LineageNode" sdk/typescript/packages/client/src/generated | head
```

TypeScript type check (quick; the client package's `lint` script):

```bash
cd sdk/typescript/packages/client && npx tsc --noEmit
```

## Outcomes that must hold

See [contracts/lineage-chain.md](./contracts/lineage-chain.md) for the full matrix. The runs
above are enough when these are true:

- `TestLineageBuilderChainOrder` walks a four-level chain (resource, sub-account, billing
  account, organization) in exact leaf-to-root order.
- `TestLineageBuilderMetadataPlacement` shows attributes landing on the most recently added
  node: the resource before any `WithParent`, the newest parent afterward.
- `TestLineageBuilderPartialChain` accepts a resource-only chain.
- `TestLineageBuilderCustomAndDeepChain` walks a ten-level chain with CUSTOM nodes.
- `TestLineageBuilderJSONRoundTrip` and `TestLineageBuilderIntegrationRoundTrip` round-trip
  chains on `ActualCostResult` and `ResourceDescriptor` through protojson with full fidelity.
- `TestActualCostResultLineageDisagreementRoundTrip` accepts a chain that omits the flat
  account fields' values alongside a `FocusCostRecord` that names them; both survive
  unchanged.
- `buf breaking`-style compatibility holds: `ActualCostResult` field 8 still means
  `expires_at`; the new fields are additions at 9 and 11.

## Out of scope for this check

Do not expect lineage on projected-cost or estimate responses, batch wrappers, or the FOCUS
record. Do not expect a conformance fixture, a `WalkLineage` helper, a stored depth, or any
validation of node ids against provider APIs.
