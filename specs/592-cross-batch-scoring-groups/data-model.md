# Data Model: Cross-Batch Scoring Sessions

## ScoreRecommendationsRequest (existing, one field added)

| Field | Number | Type | Rule |
| ----- | ------ | ---- | ---- |
| `session_id` | 4 | string | Empty means no session. Otherwise 1 to 128 printable ASCII characters (0x21-0x7E). One host operation, one key. |

## ScoreRecommendationsResponse (existing, one field added)

| Field | Number | Type | Rule |
| ----- | ------ | ---- | ---- |
| `session_id` | 5 | string | Echo of the request value when the scorer honors sessions; empty otherwise. A non-empty value that differs from the request is invalid. |

## RecommendationScores.duplicate_group_id (rule change, no wire change)

| Request | Rule |
| ------- | ---- |
| `session_id` empty | Unchanged: shared by at least two members of the response. |
| `session_id` set and echoed | Deterministic in `(session_id, duplicate key)`. May appear on one member. Equal across the batches of the session. |

## Validation rules

- Request: `len(session_id) <= 128` and every byte in 0x21-0x7E, else `ErrInvalidScoreRequest`
  (`InvalidArgument`).
- Response: echo is empty or equals the request; the two-member check runs only when the request
  `session_id` is empty.
