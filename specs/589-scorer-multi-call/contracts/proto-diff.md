# Contract: ScorerInfo Lists and Per-Item Error Semantics

File: `proto/finfocus/v1/scoring.proto`

```protobuf
message ScorerInfo {
  string name = 1;
  // model is the primary model or engine, for example "jev-1.13.0". Empty for
  // scorers that use no model. When models is non-empty, model MUST equal
  // models[0].
  string model = 2;
  ScoreCalibration calibration = 3;
  // Deprecated: use provider_request_ids. Scorers that set provider_request_ids
  // SHOULD also set this to its first entry for older hosts.
  string provider_request_id = 4 [deprecated = true];
  // provider_request_ids lists every backend request identifier for this call,
  // for support tickets. A log field, not a cache key part. Hosts read this list
  // first and fall back to provider_request_id. Entries are non-empty.
  repeated string provider_request_ids = 5;
  // models lists every model or engine the call used, primary first. Entries are
  // non-empty. It joins the score cache key.
  repeated string models = 6;
}
```

`RecommendationScoreResult.error` (a `ResourceError`): for scoring, only `code` and `message` carry
meaning. Hosts SHOULD surface `message`. Scorers MUST NOT set `resource_type_unsupported`.

Additive; `buf breaking` against `main` must pass.
