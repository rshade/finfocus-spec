# Quickstart: Validating the Recommendation Spot Fields

## Prerequisites

- `make generate` has run (Go and TypeScript bindings include fields 20 and 21).
- `npm ci` has run in `sdk/typescript` for the client test.

## Scenarios

1. **Rule table**: `go test -run 'RecommendationSpot' ./sdk/go/testing/ ./sdk/go/pluginsdk/`.
   Expect every cell of the state table in [data-model.md](./data-model.md) to pass, including NaN, Inf,
   negative, tolerance edges, and exactly 1.0.
2. **Conformance**: `go test -run 'GetRecommendationsSpot' ./sdk/go/testing/`. Expect a pass for a plugin
   that never sets the fields and for one returning a valid DYNAMIC recommendation; expect a named
   failure for a non-zero risk without DYNAMIC.
3. **Round trip**: `go test ./sdk/go/testing/` includes a bufconn call that returns a spot recommendation
   with both values intact. In `sdk/typescript/packages/client`, `npx vitest run` covers the same through
   the generated message.
4. **Scorer pass-through**: the scorer regression test shows `ValidateScoreRecommendationsRequest`
   accepts an embedded recommendation with an invalid pair.
5. **Compatibility gate**: `buf lint` and `buf breaking --against '.git#branch=main'` (or an archive of
   `origin/main` in a worktree) both pass.
6. **Performance**: no `BenchmarkValidateRecommendation` exists today, so add it first. Build `main` and
   the branch test binaries and compare it with the fields unset; expect no measurable change and
   0 added allocs/op.
7. **Full gate**: `golangci-lint run ./...`, `make test`, `make lint-markdown`.

Contract details: [contracts/recommendation-spot.md](./contracts/recommendation-spot.md).
