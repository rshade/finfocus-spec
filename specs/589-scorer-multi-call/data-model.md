# Data Model: Multi-Call Scorer Identity and Per-Item Error Semantics

## ScorerInfo (changed)

| Field | Number | Type | Rule |
| --- | --- | --- | --- |
| `name` | 1 | string | unchanged |
| `model` | 2 | string | Primary model. Must equal `models[0]` when `models` is non-empty |
| `calibration` | 3 | ScoreCalibration | unchanged |
| `provider_request_id` | 4 | string | **deprecated**. Should equal `provider_request_ids[0]` when both are set |
| `provider_request_ids` | 5 | repeated string | **new**. Every backend request id; no empty entries. Log field, not a cache key part |
| `models` | 6 | repeated string | **new**. Every model used, primary first; no empty entries. Joins the cache key |

## RecommendationScoreResult.error (ResourceError, rules tightened for scoring)

| Field | Rule for scorers |
| --- | --- |
| `code` | Not OK (unchanged) |
| `message` | Hosts SHOULD surface it |
| `resource_type_unsupported` | MUST NOT be set; rejected by validation |

## Validation outcomes (all wrap `ErrInvalidScoreResponse`)

| Case | Message names |
| --- | --- |
| Empty `provider_request_ids[i]` | `scorer.provider_request_ids[i]` |
| Empty `models[i]` | `scorer.models[i]` |
| `models[0] != model` | `scorer.model` |
| `resource_type_unsupported = true` | `results[i].error.resource_type_unsupported` |
