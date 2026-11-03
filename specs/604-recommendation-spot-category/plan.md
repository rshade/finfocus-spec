# Implementation Plan: Spot Pricing Category and Interruption Risk on Recommendation

**Branch**: `604-recommendation-spot-category` | **Date**: 2026-11-03 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/604-recommendation-spot-category/spec.md` (issue #663)

## Summary

Add two optional fields to `Recommendation` (`pricing_category` = 20, `spot_interruption_risk_score` = 21)
that mean exactly what the same-named fields mean on `GetProjectedCostResponse`. `ValidateRecommendation`
rejects a non-zero risk without DYNAMIC and any non-finite or out-of-range risk. The rule lives in
`sdk/go/testing` so the SDK validator and a new Standard conformance check share one implementation,
following spec 603. The scorer request validator is untouched (clarification, FR-016).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new dependencies

**Storage**: N/A (two optional scalar fields on an existing message, plus one validator rule)

**Testing**: `go test` with testify, in-memory bufconn harness, vitest for the TypeScript client,
`buf lint` and `buf breaking`

**Target Platform**: Go SDK library and generated TypeScript bindings

**Project Type**: Protocol specification repository with Go and TypeScript SDKs

**Performance Goals**: Recommendation validation with both fields unset keeps its current time and
0 added allocations (FR-015); the valid spot path is also 0 allocs/op

**Constraints**: Wire-compatible (fields 20 and 21 only, `buf breaking` clean); generated code is never
hand-edited; `SpecVersion` is bumped by release-please; no `RecommendationActionType` value added

**Scale/Scope**: One proto message, one rule function, two sentinels, one conformance check, docs

## Constitution Check

*GATE: Passed before research; re-checked after design (unchanged).*

| Principle | Status | Note |
|-----------|--------|------|
| I. Proto-first | Pass | Fields land in `costsource.proto`; Go and TS code regenerated, never edited |
| II. Multi-provider consistency | Pass | DYNAMIC is the neutral spot signal; no provider names or action values |
| III. Consume, don't calculate | Pass | Risk is plugin-supplied and never recomputed or compared to impact figures |
| IV. Separation of concerns | Pass | Host display and reordering stay in core (out of scope) |
| V. Test-first | Pass | Tasks put failing rule, conformance, and round-trip tests before the proto change |
| VI. Backward compatibility | Pass | Additive fields 20, 21; defaults keep old plugins valid; `buf breaking` gate |
| VII. Documentation | Pass | Same-PR updates to developer guide, SDK READMEs, proto comments |
| VIII. Performance | Pass | Guarded call site, 0-alloc rule, benchmark A/B against `main` |
| IX. Observability and validation | Pass | Named sentinels naming `recommendation.spot_interruption_risk_score` |
| X. Established patterns | Pass | Mirrors spec 603: rule in `testing`, aliases in `pluginsdk` |
| XI. Copyright headers | Pass | New Go files carry the Apache 2.0 header |
| XII. Capability declaration | Pass | No new capability; recommendations reuse `PLUGIN_CAPABILITY_RECOMMENDATIONS` |
| XIII. SDK synchronization | Pass | `make generate` regenerates TS; a vitest round-trip covers it |
| XIV. Documentation integrity | Pass | README snippets use exported names; godoc on new exports |

No violations, so Complexity Tracking is empty.

## Design Decisions

Full reasoning is in [research.md](./research.md).

1. **One rule, in `sdk/go/testing`.** `ValidateRecommendationSpot(category, score)` with sentinels
   `ErrRecommendationSpotInvalidValue` (NaN, Inf, out of range) and `ErrRecommendationSpotInvalidCategory`.
   `pluginsdk` aliases the sentinels and calls the function from `ValidateRecommendation`. The existing
   `validateSpotRiskScore` cannot be shared because it lives in `pluginsdk` and its sentinels name the
   cost-response field.
2. **Same numbers as the cost responses.** Tolerance 1e-9 on the lower bound and on zero, strict 1.0 upper
   bound after the check order NaN/Inf, zero fast path, range, category. A drift test pins the two
   epsilons together.
3. **Guarded call site.** `ValidateRecommendation` calls the rule only when the category is not
   UNSPECIFIED or the score is not exactly zero, which keeps unset recommendations on the old path.
4. **Conformance.** New Standard check `RPCCorrectness_GetRecommendationsSpot`, gated on a declared
   recommendations capability (a custom gate, not `withCapability`; see research R5), runs the rule on every
   returned recommendation. A plugin that never sets the fields passes.
5. **Test helper.** `NewSpotRecommendation(id, risk)` in `sdk/go/testing` builds a valid DYNAMIC
   recommendation; `GenerateSampleRecommendations` is unchanged so existing count and ordering tests hold.
6. **Scorer untouched.** A regression test asserts `ValidateScoreRecommendationsRequest` still accepts
   recommendations carrying an otherwise invalid pair (FR-016).
7. **Release.** `SpecVersion` and the minor bump come from release-please via the conventional commit
   (`feat(proto): ...`). The upgrade note is a section in `PLUGIN_DEVELOPER_GUIDE.md`.

## Project Structure

### Documentation (this feature)

```text
specs/604-recommendation-spot-category/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── recommendation-spot.md
├── checklists/requirements.md
└── tasks.md              # created by /speckit-tasks
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto               # fields 20, 21 on Recommendation
sdk/go/proto/finfocus/v1/costsource.pb.go        # generated (make generate)
sdk/typescript/packages/client/src/generated/    # generated (make generate)
sdk/go/testing/
├── recommendation_spot.go                       # rule, sentinels, NewSpotRecommendation
├── recommendation_spot_test.go                  # rule table, helper, drift pin, scorer regression
├── rpc_correctness.go                           # RPCCorrectness_GetRecommendationsSpot
└── rpc_correctness_test.go                      # check passes and fails as specified
sdk/go/pluginsdk/
├── validation.go                                # sentinel aliases
├── helpers.go                                   # ValidateRecommendation calls the rule
├── helpers_test.go                              # recommendation cases, benchmark A/B
└── README.md
sdk/typescript/packages/client/test/             # round-trip of both fields
PLUGIN_DEVELOPER_GUIDE.md                        # recommendation section and upgrade note
sdk/go/testing/README.md
CLAUDE.md                                        # pattern entry and Recent Changes, by hand
```

**Structure Decision**: Single-repository SDK layout identical to spec 603. No new packages.

## Complexity Tracking

No constitution violations to justify.

## Baseline

`BenchmarkValidateRecommendation` on the pre-change tree (identical Go code to `main` at d24ef26),
`-count=10`: 7.45 to 8.11 ns/op (median about 7.55), 0 B/op, 0 allocs/op. The branch run in T026 is compared
against this using prebuilt test binaries.

## Wire compatibility result

After adding fields 20 and 21: `buf lint` exits 0 and `buf breaking` against an archive of `origin/main`
exits 0 (additive only). `make generate` changed only `costsource.proto`, `costsource.pb.go`, and
`costsource_pb.ts`; `buf.gen.yaml` still pins `buf.build/connectrpc/go:v1.21.0` and no `*.connect.go` changed.

## Benchmark result (T026)

`BenchmarkValidateRecommendation`, fields unset, prebuilt binaries run interleaved, `-count=8` three times
per binary. `locals` of `ValidateRecommendation` is unchanged at 0x70 and allocations stay 0.

| Variant | Median ns/op | Note |
|---------|--------------|------|
| `main` baseline | 7.5 and 8.0 (two sessions) | the box drifted by about 0.5 ns between sessions |
| Getter guard (`rec.GetPricingCategory() != ...`) | 7.8 and 8.25 | about +3 to 4 percent against its paired baseline |
| Direct-field guard (`rec.PricingCategory != ...`, kept) | 7.91 | within noise of the baseline |

The kept guard reads the fields directly because `rec` is already known to be non-nil at that point. No
allocation was added on the unset path or on the valid DYNAMIC path.
