# Bug Verification: Spot check passes a plugin that declares recommendations but answers Unimplemented

- **Slug**: issue-668-spot-check-declared-unimplemented
- **Tested**: 2026-11-08
- **Assessment**: ./assessment.md
- **Fix**: ./fix.md
- **Result**: verified

## Summary

The assessment's reproduction no longer passes the plugin. A plugin that declares
`PLUGIN_CAPABILITY_RECOMMENDATIONS` and answers `Unimplemented` now fails
`RPCCorrectness_GetRecommendationsSpot` with an error that names the capability. The legacy and
undeclared-capability paths are unchanged. No regressions were found.

## Checks Performed

| Check | Command / Action | Result | Notes |
|-------|------------------|--------|-------|
| Reproduction (pre-fix) | new row in `TestRPCCorrectness_GetRecommendationsSpot` | fail (expected) | `Success = true, want false`, the symptom from the issue |
| Reproduction (post-fix) | `go test ./sdk/go/testing/ -run TestRPCCorrectness_GetRecommendationsSpot -count=1` | pass | Same steps as the assessment: harness, declaring plugin, `TestFunc` |
| New / updated tests | same command | pass | 9 rows, plus `_PageCapReported` |
| Generated code current | `make generate` then `git diff --exit-code -- sdk/` (excluding hand-edited files) | pass | No proto change |
| Proto lint / breaking | `make buf-lint`; `buf breaking --against '<repo>.git#branch=main'` | pass | |
| Regression suite | `make test` | pass | |
| Integration | `go test -tags=integration ./sdk/go/testing/` | pass | |
| Conformance | `go test -run TestConformance ./sdk/go/testing/` | pass | `MockPlugin` serves recommendations, so it is unaffected |
| Lint | `make lint-go` | pass | 0 issues (golangci-lint and buf lint) |
| Markdown / YAML / schema | `make lint-markdown`, `make lint-yaml`, `make validate-npm` | pass | |
| TypeScript SDK | n/a | skipped | No TS or proto change |
| Benchmarks | n/a | skipped | Conformance check, not a hot path |

## Output Excerpts

```text
--- PASS: TestRPCCorrectness_GetRecommendationsSpot/legacy_plugin_declaring_no_capabilities_and_answering_Unimplemented_passes
--- PASS: TestRPCCorrectness_GetRecommendationsSpot/plugin_declaring_recommendations_but_answering_Unimplemented_fails
ok   github.com/rshade/finfocus-spec/sdk/go/testing
0 issues.
Summary: 0 issues in 0 files
```

## Residual Risks

- This is an intended, visible conformance result change in a minor release. Third-party plugins that
  declare recommendations but return `Unimplemented` start failing Standard conformance.
- Only the first page is judged. An `Unimplemented` answer on a later page is still reported as an
  unexpected error, which is a failure, as it was before.

## Recommendation

Close the bug once the PR merges. The fix is verified end to end against the assessment's reproduction.
