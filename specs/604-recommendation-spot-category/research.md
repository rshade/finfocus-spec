# Research: Spot Pricing Category and Interruption Risk on Recommendation

All Technical Context items were resolvable from the repository; nothing is marked NEEDS CLARIFICATION.

## R1. Where the shared rule lives

- **Decision**: New `sdk/go/testing/recommendation_spot.go` holds `ValidateRecommendationSpot` and the
  sentinels. `pluginsdk` aliases the sentinels and calls the function.
- **Rationale**: Conformance (`sdk/go/testing`) cannot import `pluginsdk` (import cycle), and the
  conformance check must apply the same rule the SDK validator applies. Spec 603 and 586 solved this the
  same way (`on_demand_equivalent.go`, `region_price.go`).
- **Alternatives**: Reuse `validateSpotRiskScore` in `pluginsdk` (unreachable from conformance, and its
  `ErrSpotRiskScore*` messages name the cost-response field). Move the cost-response rule into `testing`
  and re-point it (touches cost validators and their benchmarks for no user gain).

## R2. Sentinel set

- **Decision**: Two sentinels: `ErrRecommendationSpotInvalidValue` (NaN, Inf, below zero or above 1.0
  beyond tolerance) and `ErrRecommendationSpotInvalidCategory` (non-zero risk without DYNAMIC). Messages
  are wrapped as `recommendation.spot_interruption_risk_score: ...`.
- **Rationale**: The issue asks for the `ErrOnDemandEquivalentInvalid*` style, which is two sentinels.
  Callers need to tell "bad number" from "contradicts category"; NaN versus out-of-range adds nothing.
- **Alternatives**: Three sentinels mirroring `ErrSpotRiskScoreNaN/OutOfRange/InvalidCategory` (more
  surface, no extra caller decision).

## R3. Numeric rule

- **Decision**: Copy the cost-response behaviour: reject NaN/Inf; treat |score| < 1e-9 as zero (valid for
  every category); reject score < -1e-9 or > 1.0; then reject non-DYNAMIC. A drift test asserts the
  `testing` epsilon equals `pluginsdk`'s `spotRiskEpsilon`.
- **Rationale**: FR-003 to FR-006 require identical reading. Note the existing text says "[1.0, 1.0 +
  epsilon] accepted" in the proto comment but the code applies a strict `> 1.0`. The new comment states
  the code's behaviour for recommendations (strict upper bound) and the cost-response comment is not
  edited here.
- **Alternatives**: Add 1.0 + epsilon tolerance (diverges from the implemented cost rule).

## R4. Call-site cost

- **Decision**: In `ValidateRecommendation`, call the rule only when `GetPricingCategory() != UNSPECIFIED
  || GetSpotInterruptionRiskScore() != 0`. NaN is not equal to zero, so it still reaches the rule.
- **Rationale**: Spec 053 and 557 showed a non-inlined call or an inline `fmt.Errorf` wrap can cost
  10 to 30 percent on a 6 ns benchmark. Recommendation validation is heavier (several string checks), but
  FR-015 still demands a measured A/B. Wrapping stays inside a small helper, not the public validator.
- **Alternatives**: Unconditional call (simpler, measurable cost on the unset path).

## R5. Conformance placement

- **Decision**: Add `RPCCorrectness_GetRecommendationsSpot` (Standard, gated on
  `PLUGIN_CAPABILITY_RECOMMENDATIONS`) to `rpc_correctness.go`. It calls `GetRecommendations`, applies the
  rule to each returned recommendation, and passes on an empty list.
- **Gate (implementation finding)**: The check does not use `withCapability`. For a plugin that declares
  capabilities without recommendations, that wrapper calls `checkUnsupportedCapability`, which has no probe
  for recommendations and returns an error. The check instead skips (passes) when capabilities are declared
  and recommendations is not among them, and passes when `GetRecommendations` answers `Unimplemented`.
  A legacy plugin that declares nothing runs the check.
  The check follows `next_page_token` (at most 100 pages, stopping on a repeated token), so every page is
  checked, not only the first.
- **Rationale**: The shared suite has no recommendation check today (the `GetRecommendations_*` tests are
  in `conformance_test.go` and run only against this repo's mock). Standard, not Basic, because spot data
  is optional and Basic is the "every plugin" floor.
- **Alternatives**: Basic level (forces every recommendation plugin to be checked for an optional field);
  extend the existing in-repo test only (plugin authors would not get it).

## R6. Test helper and mock

- **Decision**: Add `NewSpotRecommendation(id, risk)`; leave `GenerateSampleRecommendations` alone.
- **Rationale**: Existing tests assert counts, category and action-type rotation over the sample list;
  inserting DYNAMIC samples would shift them. The mock already accepts any list through
  `RecommendationsConfig`, so tests can inject spot recommendations.
- **Alternatives**: Make some sample recommendations DYNAMIC (broad test churn for little signal).

## R7. Scorer path

- **Decision**: No scorer change; add a regression test that the scorer request validator accepts an
  embedded recommendation carrying an invalid pair.
- **Rationale**: Clarification Q1 and FR-016.

## R8. Versioning and upgrade note

- **Decision**: No manual `SpecVersion` edit; release-please bumps it (`x-release-please-version`) from a
  `feat(proto):` commit. The upgrade note is a short section in `PLUGIN_DEVELOPER_GUIDE.md` naming the
  fields and rules.
- **Rationale**: `version.go` carries the release-please marker; spec 603 shipped the same way. The
  `references/to-vX.Y.Z.md` file named in the issue belongs to the core repository's upgrade skill.
- **Alternatives**: Hand-edit the constant (conflicts with the release PR).

## R9. TypeScript parity

- **Decision**: Regenerate bindings with `make generate`; add one vitest that round-trips both fields.
  There is no recommendation builder in the client, so none is added.
- **Rationale**: Constitution XIII asks for client exposure; the client exposes generated messages for
  recommendations today.

## R10. Proto documentation

- **Decision**: Field comments restate the rule, point to `ValidateRecommendation`, and say the category
  describes the recommended target state and that hosts only annotate (never reorder) by risk.
- **Rationale**: FR-011 and the issue's "hosts never reorder" guidance; keep the wording parallel to field 9.
