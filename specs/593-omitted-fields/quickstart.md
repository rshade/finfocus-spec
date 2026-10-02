# Quickstart

```bash
make generate && git diff --exit-code -- sdk/ ; true   # regenerated files are expected on first run
go test ./sdk/go/testing/ -run 'Scor|Omitted'
go test ./sdk/go/pluginsdk/ -run Scorer
go test -bench=ValidateScoreRecommendationsRequest -benchmem ./sdk/go/testing/
( cd sdk/typescript/packages/client && npm run build && npx vitest run recommendation-scorer )
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
```

Expected: tests pass, empty-list validation allocs/op equal to `main`, `buf breaking` clean.
