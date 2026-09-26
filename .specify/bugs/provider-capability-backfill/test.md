# Bug Verification: PluginInfoProvider responses advertise no inferred capabilities

- **Slug**: provider-capability-backfill
- **Tested**: 2026-09-25
- **Assessment**: ./assessment.md
- **Fix**: ./fix.md
- **Result**: verified

## Summary

The bug reproduces on the unfixed code (HEAD `e0f5afe`). A `PluginInfoProvider` with empty
`Capabilities` returns `nil` capabilities and no `supports_*` keys, and `Serve` with a
`TypeRegistry` does not advertise `RESOLVE_RESOURCE_TYPES`. With the fix, the same checks pass.
The existing `GetPluginInfo` and `Supports` tests, the full Go suite, and lint are all clean.

## Checks Performed

| Check | Command / Action | Result | Notes |
|-------|------------------|--------|-------|
| Reproduction (pre-fix) | Temporary detached worktree at HEAD, with the new provider-path and `Serve` tests copied in; `go test -run 'ProviderCapabilityBackfill\|ProviderResponseNotMutated\|TestServe_TypeRegistry'` | fail (expected) | Reproduction steps 1–4 are automated here; the worktree was removed afterwards |
| Reproduction (post-fix) | Same tests on the fixed tree | pass | |
| New tests | `go test -race -count=5 -run 'Provider\|TypeRegistryCapability\|TestServe_TypeRegistry' ./sdk/go/pluginsdk/` | pass | 5 repeats under the race detector; the end-to-end gRPC test was not flaky |
| Existing regression tests | `go test -v -run 'TestGetPluginInfo$\|..._AutoDiscovery\|..._BackwardCompatibility_...\|TestSupports' ./sdk/go/pluginsdk/` | pass | Includes the three tests the issue names and all 10 `TestSupports_*` tests |
| Full suite | `go test ./...` | pass | 15 packages ok |
| Go lint | `golangci-lint run ./...` | pass | 0 issues |
| Markdown lint | `npx markdownlint-cli2 sdk/go/pluginsdk/README.md .specify/bugs/provider-capability-backfill/*.md` | pass | 0 issues |

## Output Excerpts

Pre-fix, with the capability enum missing entirely:

```text
--- FAIL: TestGetPluginInfo_ProviderCapabilityBackfill/empty_capabilities_inherit_the_inferred_set_and_legacy_keys
    expected: []pbc.PluginCapability{1, 2, 9, 10, 4, 6, 11, 5, 12, 13}
    actual  : []pbc.PluginCapability(nil)
--- FAIL: TestGetPluginInfo_ProviderResponseNotMutated
--- FAIL: TestServe_TypeRegistryAdvertisesResolveResourceTypes/inferred_capabilities
FAIL    github.com/rshade/finfocus-spec/sdk/go/pluginsdk
```

Post-fix:

```text
ok   github.com/rshade/finfocus-spec/sdk/go/pluginsdk  1.233s   (race, count=5)
15 ok   (go test ./...)
0 issues.   (golangci-lint)
```

## Residual Risks

- **The pre-fix run did not directly show the mutation.**
  - Before the fix, `TestGetPluginInfo_ProviderResponseNotMutated` fails on its
    "capabilities not empty" assertion, not on `proto.Equal`. Its shared response has no
    `BATCH_COST`, so the old code never reached the in-place `resp.Metadata` assignment.
  - After the fix, the test does pin that the response is not modified.
  - The old write happened only when a provider listed `BATCH_COST` explicitly.
- **The explicit-list case failed before the fix only because of missing legacy keys.** An
  explicit list was already returned unchanged, so the "no union" rule is pinned for the
  future, not newly fixed.
- **Host-side behaviour is not exercised.** FinFocus Core now calls RPCs it previously skipped
  for affected plugins; that was not tested end to end with Core.
- **Behaviour change for existing plugins.** A provider-based plugin with `Unimplemented`
  stubs, and no explicit list, now advertises those capabilities. It must declare an explicit
  list. The commit message must call this out.

## Recommendation

Close issue #504 when this is merged; the fix is verified at the SDK level. The commit message
should describe the behaviour change, since release-please builds the changelog from it. File
the follow-up in `finfocus-plugin-aws-public` to remove its hand-built legacy metadata after
the SDK bump.
