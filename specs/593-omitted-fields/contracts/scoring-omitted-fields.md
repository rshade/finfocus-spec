# Contract: omitted_fields

```proto
message ScoreRecommendationsRequest {
  // ... fields 1-3 unchanged (4 is session_id, issue 574, merged)
  repeated string omitted_fields = 5;
}
```

Additive. Old scorers ignore it (unknown field); old hosts send none. `buf breaking` passes.
