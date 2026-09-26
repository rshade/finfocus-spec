# Bug Verification: Supports always fails with the default registry

- **Slug**: supports-default-registry
- **Tested**: 2026-09-25
- **Assessment**: ./assessment.md
- **Fix**: ./fix.md
- **Result**: verified

## Summary

The bug reproduces on the unfixed code (HEAD `6e31a4f`): with no registry, every `Supports`
call returns `InvalidArgument: no plugin registered…`, directly and over both gRPC and Connect.
With the fix, the plugin is reached on all three paths. A configured registry still validates.
The full suite and lint are clean.

## Checks Performed

| Check | Command / Action | Result | Notes |
|-------|------------------|--------|-------|
| Reproduction (pre-fix) | Temporary detached worktree at HEAD, with the new `supports_registry_test.go` and updated `client_test.go` copied in; `go test -run 'TestSupports\|TestServe_Supports\|TestClient_Supports'` | fail (expected) | All 16 constructor × case cells, gRPC, and Connect fail with "no plugin registered". Reproduction steps 1–4 are automated here; the worktree was removed afterwards |
| Reproduction (post-fix) | Same tests on the fixed tree | pass | |
| New and updated tests | `go test -race -count=5 -run '…NoRegistry\|…ConfiguredRegistry\|…NilResourceStill\|TestServe_SupportsWithoutRegistry\|TestClient_Supports' ./sdk/go/pluginsdk/` | pass | 5 repeats under the race detector |
| Existing registry and `Supports` tests | `go test -v -run 'TestSupports_\|TestNewServerWithRegistry\|TestNewServerWithOptions' ./sdk/go/pluginsdk/` | pass | 15 top-level tests, including the unchanged `mockRegistry` validation tests |
| Full suite | `go test ./...` | pass | 15 packages ok |
| Go lint | `golangci-lint run ./...` | pass | 0 issues |
| Markdown lint | `npx markdownlint-cli2 sdk/go/pluginsdk/README.md .specify/bugs/supports-default-registry/*.md` | pass | 0 issues |

## Output Excerpts

Pre-fix:

```text
--- FAIL: TestSupports_NoRegistryDelegatesToPlugin      (16/16 subtests)
    rpc error: code = InvalidArgument desc = no plugin registered for provider "" and region ""
--- FAIL: TestServe_SupportsWithoutRegistryReachesPlugin
--- FAIL: TestClient_Supports
    Supports RPC failed: unknown: rpc error: code = InvalidArgument desc = no plugin registered for provider "aws" and region "us-east-1"
```

Post-fix:

```text
ok   github.com/rshade/finfocus-spec/sdk/go/pluginsdk  1.395s   (race, count=5)
15 ok   (go test ./...)
0 issues.   (golangci-lint)
```

## Residual Risks

- **Host behavior is not exercised.** Plugins without `SupportsProvider` now return a clean
  `Supported: false` instead of an error. Core fails open on errors, so these plugins were
  previously treated as "supported"; how Core treats the new `false` has not been verified.
  This is the assessment's open question, and it must be confirmed in Core before release.
- **A separate, pre-existing bug was found, out of scope.** The pre-fix Connect error arrived as
  `unknown: rpc error: code = InvalidArgument …`. `ConnectHandler` (`connect.go`) returns the
  gRPC `status` errors from `Server` unconverted, and Connect maps them to `CodeUnknown`. This
  affects every RPC over Connect, not just `Supports`, so Connect clients cannot tell
  `InvalidArgument` or `NotFound` apart from internal errors. It needs its own issue.
- **Callers that pass `&DefaultRegistryLookup{}` explicitly** to reject everything get the new
  behavior. This is documented and considered implausible.

## Recommendation

Close issue #507 when this is merged; the fix is verified at the SDK level over direct calls,
gRPC, and Connect. Before a release, confirm in `rshade/finfocus` that `checkPluginSupports`
handles `Supported: false` with `DefaultSupportsNotImplementedReason` as intended. File a new
issue for the Connect status-code mapping in `connect.go`.
