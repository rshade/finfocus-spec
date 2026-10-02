# Quickstart: Validating Cross-Batch Scoring Sessions

```bash
make generate && git diff --exit-code -- sdk/ >/dev/null || echo "regenerated"
go test ./sdk/go/testing/ ./sdk/go/pluginsdk/ -run 'Score|Scorer|Session'
go test -v -run TestScorerConformance ./sdk/go/testing/
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
cd sdk/typescript/packages/client && npx vitest run recommendation-scorer
```

Expected: a duplicate pair split across two batches of one session gets equal non-empty ids; two
sessions get different ids; a request without a session behaves as before; `buf breaking` is clean.
See [contracts/session-contract.md](contracts/session-contract.md) and [data-model.md](data-model.md).
