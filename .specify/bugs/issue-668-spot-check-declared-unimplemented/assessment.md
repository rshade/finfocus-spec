# Bug Assessment: Spot check passes a plugin that declares recommendations but answers Unimplemented

- **Slug**: issue-668-spot-check-declared-unimplemented
- **Created**: 2026-11-08
- **Source**: <https://github.com/rshade/finfocus-spec/issues/668>
  (host: `github.com`, allowlisted; read with `gh issue view`)
- **Verdict**: valid
- **Severity**: low

## Report (summarized)

Issue 668, "RPCCorrectness_GetRecommendationsSpot passes a plugin that declares recommendations but answers
Unimplemented". It was filed from the review of PR #665. A plugin whose `GetPluginInfo` declares
`PLUGIN_CAPABILITY_RECOMMENDATIONS` and whose `GetRecommendations` returns `codes.Unimplemented` gets
`Success=true` with `Details="Plugin does not serve recommendations; check skipped"`. The precedent is
`testGetBudgetsRPC`, which accepts `Unimplemented` only when `PLUGIN_CAPABILITY_BUDGETS` is not declared.
The issue proposes failing that combination, keeping the legacy pass and the undeclared skip, adding a
test case, and updating the doc comment, the guide, and the README skip list.

## Symptom

A plugin that advertises recommendations and then refuses to serve them passes Standard conformance. Its
recommendations are never spot-checked. Expected: the check fails and names the contradiction between the
declared capability and the `Unimplemented` answer.

## Reproduction

1. In `sdk/go/testing`, build a `spotRecommendationsPlugin` (`rpc_correctness_test.go`) with
   `capabilities: {PLUGIN_CAPABILITY_RECOMMENDATIONS}` and a `GetRecommendations` that returns
   `status.Error(codes.Unimplemented, "no")`.
2. Start a `TestHarness` on it and call the `RPCCorrectness_GetRecommendationsSpot` `TestFunc`.
3. Observe `Success=true` and `Details="Plugin does not serve recommendations; check skipped"`.

This was confirmed by reading the code on `origin/main` at `3e93f0f`. The branch has no condition on the
capability profile.

## Suspected Code Paths

- `sdk/go/testing/rpc_correctness.go:541-546` (`testGetRecommendationsSpot`). The issue cites `:542`; the
  code has since moved about three lines. `status.Code(err) == codes.Unimplemented && page == 0` returns
  `Success: true` without consulting `profile`, even though `profile` was loaded at the top of the
  function.
- `sdk/go/testing/rpc_correctness.go:467-468` (`testGetBudgetsRPC`) is the precedent. It guards the
  `Unimplemented` pass with `!harness.capabilities.declares(PLUGIN_CAPABILITY_BUDGETS)`.
- `sdk/go/testing/rpc_correctness.go:512-517` is the doc comment. It states that a plugin answering
  Unimplemented "has no recommendations to check".

## Root Cause Hypothesis

The early-skip branch for `Unimplemented` was written for legacy plugins, which declare nothing, and for
plugins that do not serve the RPC. It never re-checks the declared capability. The undeclared case already
returns before the loop (`len(profile.values) > 0 && !declares(...)`). So the only plugins that still reach
the `Unimplemented` branch either declare no capabilities (legacy) or declare recommendations. The
second group is the contradiction. Confidence: high.

## Proposed Remediation

**Preferred**: Inside the `page == 0 && Unimplemented` branch, check
`profile.declares(PLUGIN_CAPABILITY_RECOMMENDATIONS)`. When it is declared, return `Success: false` with an
error that names the contradiction. For example: `declared PLUGIN_CAPABILITY_RECOMMENDATIONS but
GetRecommendations returned Unimplemented`, with Details `Declared capability returned Unimplemented`.
Otherwise keep the existing legacy pass. This mirrors the shape of `testGetBudgetsRPC`, adds no RPC call,
and leaves the undeclared skip untouched because that one returns earlier.

**Alternatives**:

- Switch the check to `withCapability`. Rejected: that wrapper probes undeclared capabilities with an RPC.
  Recommendations have no probe in `checkUnsupportedCapability`, and adding one would change the result for
  plugins that declare other capabilities, which the issue says to keep skipping.

**Files likely to change**:

- `sdk/go/testing/rpc_correctness.go` (branch and doc comment)
- `sdk/go/testing/rpc_correctness_test.go` (new case)
- `sdk/go/testing/README.md:870-873` (skip list)
- `PLUGIN_DEVELOPER_GUIDE.md:1017-1020` (skip rules)
- `CLAUDE.md:1157-1159` (the "passes on `Unimplemented`" note)

**Tests to add or update**:

- In `TestRPCCorrectness_GetRecommendationsSpot`, add a case for a plugin that declares
  `PLUGIN_CAPABILITY_RECOMMENDATIONS` and returns `Unimplemented`. It expects `Success=false` and an error
  naming `PLUGIN_CAPABILITY_RECOMMENDATIONS`. This needs a test plugin that combines declared capabilities
  with an `Unimplemented` answer. Extend `unimplementedRecommendationsPlugin` with a `capabilities` field
  and a `GetPluginInfo` override, so the legacy case (nil capabilities) keeps passing.
- Keep the existing rows `legacy plugin answering Unimplemented passes` and `plugin declaring capabilities
  without recommendations is skipped` unchanged as regression guards.

## Risks & Considerations

- **Visible result change in a minor release.** A plugin that declares recommendations but returns
  `Unimplemented` starts failing Standard conformance. The issue intends this and flags it. It is a
  conformance verdict, not a wire, proto, or exported-API change. Note it in the PR body so the
  release-please `fix:` entry is not a surprise.
- No proto, generated code, TypeScript, or exported Go signature change. No hot path is touched.
- `MockPlugin` serves recommendations, so the default-mock row is unaffected.

## Open Questions

None.
