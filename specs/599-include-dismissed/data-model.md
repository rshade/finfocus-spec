# Data Model: Include Dismissed Recommendations

## GetRecommendationsRequest.include_dismissed

| Property | Value |
| --- | --- |
| Field number | 8 |
| Type | `bool` |
| Default | false |
| JSON name | `includeDismissed` |
| Go getter | `GetIncludeDismissed` |

### Rules

1. False or unset: a plugin omits recommendation IDs it has dismissed or snoozed.
2. True: a plugin includes those IDs.
3. IDs in `excluded_recommendation_ids` (field 5) are omitted in both cases.
4. An empty string in either ID list does not match a recommendation whose ID is empty.
5. Plugins with no dismissal store ignore the field.

## RecommendationVisibility

The helper input, not a proto message.

| Field | Meaning |
| --- | --- |
| `DismissedIDs` | IDs the plugin has dismissed or snoozed |
| `IncludeDismissed` | The request field |
| `ExcludedIDs` | `excluded_recommendation_ids` from the request |

`MockPlugin.RecommendationsConfig.DismissedIDs` is the mock's dismissal store.

## Unchanged

`Recommendation` gains no field. `DismissRecommendation` is unchanged. Field numbers 1-7 on
`GetRecommendationsRequest` are unchanged.
