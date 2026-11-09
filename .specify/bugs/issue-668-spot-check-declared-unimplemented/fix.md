# Bug Fix: Spot check passes a plugin that declares recommendations but answers Unimplemented

- **Slug**: issue-668-spot-check-declared-unimplemented
- **Fixed**: 2026-11-08
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

`RPCCorrectness_GetRecommendationsSpot` now fails a plugin that declares `PLUGIN_CAPABILITY_RECOMMENDATIONS`
and answers `GetRecommendations` with `Unimplemented` on the first page. It still passes a legacy plugin
(no declared capabilities) that answers `Unimplemented`, and still skips a plugin that declares other
capabilities without recommendations.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `sdk/go/testing/rpc_correctness.go` | modified | New unexported `unimplementedRecommendationsResult`; doc comment updated |
| `sdk/go/testing/rpc_correctness_test.go` | added test | `unimplementedRecommendationsPlugin` gains `capabilities` and a `GetPluginInfo` override; new row |
| `sdk/go/testing/README.md` | modified | Skip list for the check |
| `PLUGIN_DEVELOPER_GUIDE.md` | modified | Skip rules for the check |
| `CLAUDE.md` | modified | Spot check note |

## Diff Highlights

```go
if status.Code(err) == codes.Unimplemented && page == 0 {
    return unimplementedRecommendationsResult(profile, start)
}

func unimplementedRecommendationsResult(profile *capabilityProfile, start time.Time) TestResult {
    if profile.declares(pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS) {
        return TestResult{..., Error: fmt.Errorf("declared %s but %s returned Unimplemented", ...),
            Details: "Declared capability returned Unimplemented"}
    }
    return TestResult{..., Success: true, Details: "Plugin does not serve recommendations; check skipped"}
}
```

## Tests Added or Updated

- `TestRPCCorrectness_GetRecommendationsSpot/plugin_declaring_recommendations_but_answering_Unimplemented_fails`
  pins the new failure. The error must name `PLUGIN_CAPABILITY_RECOMMENDATIONS`.
- The existing rows `legacy_plugin_declaring_no_capabilities_and_answering_Unimplemented_passes` and
  `plugin_declaring_capabilities_without_recommendations_is_skipped` are unchanged and still pass.
  `MockPlugin.GetPluginInfo` sets no capabilities, so the new `GetPluginInfo` override with nil
  capabilities keeps the legacy row's meaning.

## Local Verification

- Test first: before the fix, the new row failed with
  `Success = true, want false (... details: Plugin does not serve recommendations; check skipped)`.
- `go test ./sdk/go/testing/ -run TestRPCCorrectness_GetRecommendationsSpot -count=1`: all 9 rows and
  `_PageCapReported` pass.
- `golangci-lint run ./sdk/go/testing/...`: 0 issues.
- `npx markdownlint-cli2` on the changed markdown: clean.

## Deviations from Assessment

- The rule lives in a helper, `unimplementedRecommendationsResult`, instead of inline. Inline, the extra
  branch pushed `testGetRecommendationsSpot` to cognitive complexity 21, over the `gocognit` limit of 20.
  The behavior is the same as the preferred remediation.
- `CLAUDE.md` was added to the changed files. The assessment listed it; this note only confirms it was
  done.
- `assessment.md` was edited for formatting only: the Source line was wrapped to satisfy MD013. Its
  content is unchanged.

## Follow-ups

- None for this fix. The PR body should call out the visible conformance result change for plugins that
  declare recommendations but return `Unimplemented`.

## Review Follow-up

The `/code-review` Standards axis prompted two test-only edits. The test godoc now says a plugin passes on
`Unimplemented` only when it does not declare recommendations. The legacy row is now named
`legacy plugin declaring no capabilities and answering Unimplemented passes`. Behavior is unchanged.
