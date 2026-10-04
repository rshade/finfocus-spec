# Contract: include_dismissed

Add this field at the end of `GetRecommendationsRequest`, after `usage_profile = 7`.

```protobuf
  // include_dismissed tells a plugin that stores dismissals whether to return them.
  //
  // When false (the default), the plugin MUST omit recommendations it has dismissed
  // or snoozed. When true, the plugin MUST include those recommendations.
  //
  // excluded_recommendation_ids still applies in both cases. An ID in that list is
  // omitted even when include_dismissed is true.
  //
  // A plugin that does not store dismissal state ignores this field.
  bool include_dismissed = 8;
```

## Helper

```go
type RecommendationVisibility struct {
    DismissedIDs     []string
    IncludeDismissed bool
    ExcludedIDs      []string
}

func ApplyRecommendationVisibility(
    recommendations []*Recommendation,
    visibility RecommendationVisibility,
) []*Recommendation
```

Order: drop `DismissedIDs` unless `IncludeDismissed`, then drop `ExcludedIDs`. When both ID lists
are empty, return the input slice. Nil recommendation entries stay in the slice.
