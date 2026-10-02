# Data Model

`ScoreRecommendationsRequest.omitted_fields` (field 5): `repeated string`.

| Rule | Value |
| ---- | ----- |
| Entries | 0 to 64, unique |
| Entry | 1 to 128 bytes, `.`-joined segments, each `[a-z][a-z0-9_]*` |
| Resolution | First segment is a `Recommendation` field or the `action_detail` oneof name, e.g. `resource.tags`, `metadata`, `kubernetes.cluster_id`; later segments descend message fields; map and scalar fields are leaves |
| Meaning | The host removed this field by policy from every recommendation in the request |
| Scorer duty | An empty omitted field is not evidence: do not lower confidence, raise `insufficient_evidence`, or raise `false_positive` for it |

Errors: `InvalidArgument` wrapping `ErrInvalidScoreRequest`, naming `omitted_fields[i]`.
