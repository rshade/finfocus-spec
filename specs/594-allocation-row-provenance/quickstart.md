# Quickstart: Allocation Row Provenance

```bash
make generate && git diff --stat -- sdk/
go test ./sdk/go/testing/ ./sdk/go/internal/refalloc/ ./sdk/go/pluginsdk/
go test -v -run 'Allocator' ./sdk/go/testing/
go test -bench=ValidateAllocateResponse -benchmem ./sdk/go/testing/
cd sdk/typescript && npm ci && npm run build && npm test
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
```

Expected: tests pass, the `row_provenance` scenario passes for the reference allocator, a method-only row
fails validation, and `buf breaking` reports nothing.
