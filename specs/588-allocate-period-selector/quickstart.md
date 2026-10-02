# Quickstart: Validate the Allocation Period and Selector

Run from the repository root.

```bash
make generate && git diff --exit-code -- sdk/
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
go test ./sdk/go/testing/ -run 'Allocate|Allocator' -v
go test ./sdk/go/internal/refalloc/ -v
go test ./sdk/go/pluginsdk/ -run 'Allocat' -v
cd sdk/typescript && npm run build && npm test && npm run lint
```

Expected: invalid windows are rejected, a missing echo passes, a mismatched echo fails, the reference
allocator echoes and warns under a selector, both new conformance scenarios pass, and all earlier
allocator tests still pass.
