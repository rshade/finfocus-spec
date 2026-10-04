# Quickstart: Include Dismissed Recommendations

## Plugin

```go
resp, err := backend.List(ctx, req)
if err != nil {
    return nil, err
}
resp.Recommendations = pluginsdk.ApplyRecommendationVisibility(resp.Recommendations, pluginsdk.RecommendationVisibility{
    DismissedIDs:     store.DismissedIDs(),
    IncludeDismissed: req.GetIncludeDismissed(),
    ExcludedIDs:      req.GetExcludedRecommendationIds(),
})
return resp, nil
```

A plugin with no dismissal store only needs the exclusion list, which it may already apply. It
ignores `include_dismissed`.

## Host

Set `include_dismissed` when the operator asks to see plugin-side dismissals. Keep sending
`excluded_recommendation_ids` for IDs the host itself dismissed, including when the flag is on.
Older plugins ignore the new field and still honor the exclusion list.

## Checks

```bash
make generate
make buf-lint
buf breaking --against '.git#branch=main'
go test ./sdk/go/pluginsdk/ ./sdk/go/testing/ -count=1 -run 'Visibility|IncludeDismissed'
```

From a worktree, `buf breaking` against `.git#branch=main` compares this branch with the local
`main` ref. Use `origin/main` when `main` is not checked out in this clone.
