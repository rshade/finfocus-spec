# Quickstart: Validate Multi-Call Scorer Identity

```bash
make generate && git diff --exit-code -- sdk/
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
go test ./sdk/go/testing/ -run 'Scor' -v
go test ./sdk/go/pluginsdk/ -run 'Scor' -v
cd sdk/typescript && npm run build && npm test && npm run lint
```

Expected: a response with three ids and two models (primary first) passes; empty entries, a mismatched
primary model, and the unsupported-type flag each fail with a field-named error; the three broken
scorers fail conformance; and every existing scoring test passes.
