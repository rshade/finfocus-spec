# Research: Multi-Call Scorer Identity and Per-Item Error Semantics

## R1: Fields

- **Decision**: `ScorerInfo`: `repeated string provider_request_ids = 5`, `repeated string models = 6`,
  and `string provider_request_id = 4 [deprecated = true]`. No request or response fields change.
- **Rationale**: The issue's proposed direction. Open PRs 602-604 add `ScoreRecommendationsRequest`
  fields 4 and 5 and `ScoreRecommendationsResponse` field 5. None touches `ScorerInfo`, so 5 and 6 there
  cannot collide. No Go or TypeScript source reads `provider_request_id`, so the deprecation raises no
  staticcheck findings outside the mock option that sets it on purpose.

## R2: Rules

- **Decision**: In `ValidateScoreRecommendationsResponse`, through a new `validateScorerInfo` called from
  `validateScoreEnvelope`: reject an empty entry in `provider_request_ids[i]` or `models[i]`, and reject a
  non-empty `models` whose first entry differs from `model`. In `validateScoreResult`'s error branch,
  reject `resource_type_unsupported = true` with `results[i].error.resource_type_unsupported`. Errors wrap
  `ErrInvalidScoreResponse`.
- **Rationale**: The primary-model rule keeps `model` meaningful for hosts that read only it, and gives
  the list a defined order. Not requiring `provider_request_id == provider_request_ids[0]` keeps scorers
  that set only one of them valid.
- **Alternatives considered**: A scoring-specific error message (the issue's other option; it adds a type
  for one flag); ignoring the flag (it leaves a meaningless field settable).

## R3: Conformance

- **Decision**: Every scenario validates responses with `ValidateScoreRecommendationsResponse`, so broken
  scorers with an empty request id or a mismatched primary model fail `single_recommendation`. No existing
  scenario triggers a per-item error, so a new scenario, `unscorable_item`, sends a normal recommendation
  next to one with no resource. It accepts scores for both, a per-item error, or a whole-request
  `InvalidArgument`, and validates whatever comes back, so a scorer that sets `resource_type_unsupported`
  fails it.
- **Rationale**: FR-010 requires conformance to catch the flag, which needs a request that elicits a
  per-item error. Accepting every lawful outcome keeps conformance model-agnostic, as 556 requires.
- **Correction**: The first draft of this decision said no new scenario was needed. Implementation showed
  that no scenario produced a per-item error, so the flag rule was not reachable from conformance.

## R4: Mock

- **Decision**: `WithScorerModels(models ...string)` sets `models` and `model = models[0]`.
  `WithScorerProviderRequestIDs(ids ...string)` sets `provider_request_ids` and the deprecated scalar to
  `ids[0]`, following the SHOULD. Empty and blank arguments are ignored.
- **Rationale**: The reference scorer shows the recommended way to fill both lists.

## R5: Docs and cache key

- **Decision**: `docs/recommendation-scoring.md` gains `scorer.models` in the cache key table, describes
  `provider_request_ids` as a log field next to the deprecated scalar, and states that hosts SHOULD show
  a per-item error's message and that scorers MUST NOT set `resource_type_unsupported`.
- **Rationale**: Spec 581's rule says that when `ScorerInfo` gains a field that changes scores, that
  field joins the key.
