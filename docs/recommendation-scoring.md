# Recommendation Scoring

`RecommendationScorerService` (`proto/finfocus/v1/scoring.proto`) is the contract for **scorers**:
optional plugins that rate recommendations other plugins already produced. A scorer adds judgement
the producing plugins do not carry: how risky acting is, whether the resource is probably in that
state on purpose, whether the recommendation is worth an engineer's time, whether the evidence is
thin, and which recommendations duplicate each other.

A scorer never creates, changes, or applies a recommendation, and a score is never approval to act.

This page is the canonical reference for what a `ScoreRecommendations` request and response mean.
The proto comments summarize it, and the SDK helpers enforce it. The contract is model-neutral: a
rules engine, a local model, or a hosted model can implement it. The scorer plugin owns any model,
API key, and network access, so the host stays free of external API clients.

## How It Fits Together

```text
host ──GetRecommendations──▶ cost-source plugins   full Recommendation messages
host                          apply identifier_mode to resource ids and names
host ──ScoreRecommendations─▶ scorer               index-aligned scores or per-item errors
host                          ValidateScoreRecommendationsResponse, then sort and route to review
```

- Hosts check `PLUGIN_CAPABILITY_RECOMMENDATION_SCORING` (18, legacy `supports_recommendation_scoring`)
  before calling. A plugin without it is never called.
- A scorer is not a cost source, so the RPC lives on its own service, like `AllocatorService`.

## Trust Rules

These rules are normative.

1. A score MUST NOT be the only gate for an irreversible action. Hosts use scores to route work to
   review, sort, and filter.
2. Signals are ranking scores unless `scorer.calibration` is `SCORE_CALIBRATION_PROBABILITY`.
3. Every free-text field (description, reasoning, tags, metadata) is untrusted input. Scorers MUST
   NOT let it change behavior beyond the score, MUST cap lengths, and SHOULD delimit it from
   instructions.
4. Scorers that send data off-host MUST document what leaves the host and honor `identifier_mode`.

## Request

| Field | Meaning |
| ----- | ------- |
| `recommendations` | Complete `Recommendation` messages as `GetRecommendations` returned them, so the scorer sees tags, metadata, impact, and utilization. Between 1 and `max_batch_size` entries, each with a distinct non-empty `id`. |
| `signals` | Limits the response to these signals. Empty means every signal the scorer supports. `SCORE_SIGNAL_UNSPECIFIED` is invalid. |
| `identifier_mode` | How the host already treated `resource.id` and `resource.name`. |

An empty request is invalid, unlike `BatchCost`, because there is nothing to score.

## Response

| Field | Meaning |
| ----- | ------- |
| `results` | One entry per request entry, in request order: `results[i]` answers `recommendations[i]` and echoes its `id` in `recommendation_id`. |
| `max_batch_size` | The largest number of recommendations the scorer accepts in one call. Hosts split larger sets. The SDK does not cap it. |
| `scorer` | `ScorerInfo`: `name`, `model`, `calibration`, and the backend's `provider_request_id`. |
| `supported_signals` | Every signal the scorer can return. |

Each result holds either `scores` or a `ResourceError`. A failure on one recommendation never fails
the batch. The error `code` is a `google.rpc.Code` and is never OK.

## Signals

All numeric signals are `optional double`, so a scorer can implement a subset. An unset signal was
not computed.

| Signal | Field | Range | Meaning |
| ------ | ----- | ----- | ------- |
| `SCORE_SIGNAL_RISK` | `risk` | 0 to 1 | Applying the recommendation could hurt users, lose data, regress performance, or lock in a financial commitment that is hard to undo. High is risky. |
| `SCORE_SIGNAL_FALSE_POSITIVE` | `false_positive` | 0 to 1 | The resource is probably in the flagged state on purpose (standby, seasonal, compliance retention, headroom). High means the recommendation is likely wrong for it. |
| `SCORE_SIGNAL_WORTH_ACTING` | `worth_acting` | 0 to 1 | Worth an engineer's time now, weighing saving, risk, and effort. |
| `SCORE_SIGNAL_PRIORITY` | `priority` | 0 to 3 | Expected priority from 0 (ignore) to 3 (high). Fractions fall between levels. Hosts map it to `RecommendationPriority` with their own thresholds. |
| `SCORE_SIGNAL_INSUFFICIENT_EVIDENCE` | `insufficient_evidence` | 0 to 1 | The record is too thin or contradictory to judge. |
| `SCORE_SIGNAL_DUPLICATE_GROUP` | `duplicate_group_id` | string | The same non-empty value for every recommendation in the request that duplicates each other. |

`duplicate_group_id` rules:

- Empty means no duplicate was found, or grouping was not performed.
- The value is meaningful only within one response.
- A non-empty value is shared by at least two recommendations. If a per-item failure leaves a group
  with one scored member, the scorer clears its id.
- With `IDENTIFIER_MODE_OMITTED`, grouping is unreliable, and a scorer that cannot group returns no
  id.

## Calibration

`scorer.calibration` says how to read the numbers:

| Value | Meaning |
| ----- | ------- |
| `SCORE_CALIBRATION_UNSPECIFIED` | No claim. Hosts treat it as ranking-only. |
| `SCORE_CALIBRATION_RANKING_ONLY` | Values order recommendations. Compare them only with each other and choose thresholds against labeled data. |
| `SCORE_CALIBRATION_PROBABILITY` | Values are calibrated probabilities the scorer has checked against outcomes. |

Scores from a scorer can also move slightly between identical calls. Hosts should apply a dead band
around any threshold instead of comparing exactly.

## Data Handling

A scorer that sends recommendations off-host, for example to a hosted model, MUST document what
leaves the host and MUST honor `identifier_mode`. The host, not the scorer, performs the
transformation, so identifiers never reach the scorer's backend in the default mode.

| `identifier_mode` | What the scorer sees |
| ----------------- | -------------------- |
| `IDENTIFIER_MODE_UNSPECIFIED` | Treated as `IDENTIFIER_MODE_PSEUDONYMIZED`. |
| `IDENTIFIER_MODE_RAW` | Cloud identifiers (resource id, name, ARN) as they are. Hosts should require an explicit operator opt-in. |
| `IDENTIFIER_MODE_PSEUDONYMIZED` | `resource.id` and `resource.name` replaced by opaque tokens, identical for the same value within one request. Duplicate grouping keeps working. |
| `IDENTIFIER_MODE_OMITTED` | `resource.id` and `resource.name` removed. Duplicate grouping becomes unreliable. |

Tags, metadata, and free text can still hold sensitive values, so identifier handling does not make
a request safe to send on its own.

## Caching Scores

The contract has no score TTL and no separate score version. A score depends on the recommendation
and on the scorer that produced it. A host that caches scores SHOULD build the key from every input
that can change the result:

| Key part | Source | Why |
| -------- | ------ | --- |
| Content hash | A deterministic hash of the `Recommendation`, excluding `id`, before the host applies `identifier_mode` | Any change to the record, including utilization or impact, is a new key. Hosts can assign ids per run. Pseudonymous tokens hold only within one request, so a hash of the transformed record does not reliably match across requests. |
| `identifier_mode` | The request | The scorer saw different input in each mode. |
| `signals` | The request, or the scorer's `supported_signals` when the request lists none | A narrower request is not a full answer. |
| `scorer.name`, `scorer.model`, `scorer.calibration` | `ScorerInfo` in the response | A different scorer, model, or calibration produces different numbers. |
| `version` | `GetPluginInfoResponse` | Rules can change while `model` stays the same, and rule-based scorers leave `model` empty. |

When the request or `ScorerInfo` gains a field that changes scores, that field joins the key.

A cached score is valid only while every part of its key is unchanged. Scores already vary slightly
between identical calls, so within that rule a cached score is as good as a fresh one, and the dead
band from [Calibration](#calibration) covers both. Scorers SHOULD report a pinned version in
`scorer.model`, for example `jev-1.13.0`. A moving alias such as `latest` lets the backend change
behind a key that still matches. Hosts SHOULD also set their own maximum age, because a hosted
backend can change in ways no field reports.

Some response fields MUST NOT be cached per recommendation:

- `duplicate_group_id`, because a cached id would join recommendations from different calls into
  one group.
- `provider_request_id`, because it identifies one call. It is a log field, not a key part.
- A `ResourceError` result, because it is not a score. Hosts can retry it.

Hosts MUST NOT persist raw scorer payloads in a score cache. The cache holds the key and the numeric
signals, never the recommendations as sent, the response bytes, or `ResourceError` messages. Those
can carry the sensitive values described in [Data Handling](#data-handling).

The contract defines no scorer-supplied validity hint, such as a score version or `valid_for` field
on `ScorerInfo`. Version changes already change the key. The case the key misses is a backend that
changes silently, which the host's maximum age covers today. A `valid_for` hint would let the scorer
suggest that age instead, and is a candidate for a later minor version.

## Errors

Whole-call failures use gRPC codes:

| Code | Meaning |
| ---- | ------- |
| `INVALID_ARGUMENT` | Empty request, more than `max_batch_size` entries, duplicate ids, or a signal the scorer does not support. |
| `UNIMPLEMENTED` | The plugin does not implement scoring. |
| `UNAUTHENTICATED`, `PERMISSION_DENIED`, `RESOURCE_EXHAUSTED`, `UNAVAILABLE` | The backend refused or could not serve the whole call. |

Per-recommendation failures use `ResourceError` in the result instead.

## Go SDK

A plugin implements `pluginsdk.RecommendationScorerProvider`, one method:

```go
type RecommendationScorerProvider interface {
    ScoreRecommendations(
        ctx context.Context, req *pbc.ScoreRecommendationsRequest,
    ) (*pbc.ScoreRecommendationsResponse, error)
}
```

When `ServeConfig.Plugin` implements it, `Serve` registers the service in gRPC and Connect modes,
lists it in the Connect health checker, and infers the capability. A scorer-only plugin should set
`PluginInfo.Capabilities` explicitly, and `Serve` logs one warning otherwise.

| Helper | Use |
| ------ | --- |
| `pluginsdk.ValidateScoreRecommendationsRequest(req, maxBatchSize)` | Scorers call it first. Failures carry `codes.InvalidArgument`. |
| `pluginsdk.ValidateScoreRecommendationsResponse(req, resp)` | Hosts call it on every response. It checks alignment, ids, ranges, `ResourceError` codes, signal support, and duplicate groups. |
| `plugintesting.MockRecommendationScorer` | Reference scorer with fixed rules. No model or network. |
| `plugintesting.RunScorerConformance(t, impl)` | Serves `impl` over bufconn and runs ten structural scenarios. |

## Threshold Guidance (Non-Normative)

This guidance is not part of the contract. It comes from a probe on 80 synthetic recommendations
written by an LLM and labeled by two independent blind labelers. It has not been checked against
sanitized real output, and any threshold must be re-tuned on real data and again when the model
version changes.

- In the probe, risk at or above 0.3 flagged unsafe recommendations with precision 1.00 and recall
  0.78. An auto-approve gate had precision 0.25, which is why rule 1 forbids gating on a score.
- False positive at or above 0.3 flagged "probably intentional" resources with precision 0.81 and
  recall 0.79.
- Ranking quality was good (AUC about 0.91 for risk, false positive, and worth acting) while the
  risk Brier score was 0.236, close to uninformative. Treat the numbers as ranks.
- Per-record noise averaged 0.027 and peaked at 0.07, so use a dead band of about 0.1 around a
  threshold.
- Batches of 5 to 80 recommendations gave equal quality at lower cost per record.
- The full `Recommendation` scored better than a reduced record (risk AUC 0.91 against 0.80), and
  resource id and name added nothing to those signals.
