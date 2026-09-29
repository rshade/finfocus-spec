# Data Model: Recommendation Scoring

Wire types are in [contracts/scoring.proto](contracts/scoring.proto).

## Request

| Field | Rule |
| ----- | ---- |
| `recommendations` | 1 to `max_batch_size` entries, none nil, each with a distinct non-empty `id`. |
| `signals` | Defined values only, never `SCORE_SIGNAL_UNSPECIFIED`. Empty means all supported. |
| `identifier_mode` | A defined value. Unspecified means pseudonymized. |

## Response

| Field | Rule |
| ----- | ---- |
| `results` | Same length as the request. `results[i].recommendation_id` equals `recommendations[i].id`. |
| `max_batch_size` | At least 1 and at least the request size. |
| `supported_signals` | Defined, unique, never unspecified. |
| `scorer.calibration` | A defined `ScoreCalibration`. |

## Result

Exactly one of `scores` or `error`. A `ResourceError` code is never OK.

## Scores

| Field | Range | Set only when |
| ----- | ----- | ------------- |
| `risk`, `false_positive`, `worth_acting`, `insufficient_evidence` | `[0, 1]`, finite | Supported and requested |
| `priority` | `[0, 3]`, finite | Supported and requested |
| `duplicate_group_id` | non-empty string, shared by at least two results | `SCORE_SIGNAL_DUPLICATE_GROUP` supported and requested |

## Capability

`PLUGIN_CAPABILITY_RECOMMENDATION_SCORING = 18`, legacy `supports_recommendation_scoring`.
