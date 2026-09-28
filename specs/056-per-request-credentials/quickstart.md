# Quickstart: Validate Opt-In Per-Request Credentials

Validation guide only. Behavior under test is defined in [spec.md](./spec.md),
[data-model.md](./data-model.md), and [contracts/](./contracts/).

## Prerequisites

- This worktree, branch `056-per-request-credentials`.
- Go toolchain that matches `go.mod`.
- No new module download is required for this slice.
- Do not point tests at a real cloud account. The only secret string is the fixture
  `test-secret-value`.

## Unit and serve tests

From the repository root:

```bash
go test -count=1 -timeout 120s ./sdk/go/pluginsdk/ -run 'Credential|PerRequest'
```

Expected:

- A valid one-entry set round-trips on the native server and on the web-compatible
  server. The second call, with no set, reports nothing attached and a nil error.
- `NewCredentials` rejects an empty map, a duplicate name, a non-ASCII value, and a set
  over the limits in [data-model.md](./data-model.md). Each error string is one of the
  fixed sentences in [contracts/credentials-api.md](./contracts/credentials-api.md).
- Opted-in plugin information contains `supports_per_request_credentials=true` and does
  not add a capability. A plugin that did not opt in omits the key even if its notes
  claimed it. See [contracts/plugin-info-metadata.md](./contracts/plugin-info-metadata.md).
- The process environment is unchanged by attach and by a served call.
- The log buffer, every error string, and gathered metric labels contain zero
  occurrences of `test-secret-value`.

## Absent-path benchmark

```bash
go test -bench=BenchmarkCredentialAbsent -benchmem -run '^$' ./sdk/go/pluginsdk/
```

Expected: the absent-header parse reports 0 B/op and 0 allocs/op. A one-entry
`NewCredentials` may allocate the copied map. There is no allocation that grows with
unrelated metadata keys when none of those keys use the credential prefix.

## Review checks that are not tests

- `git diff -- proto sdk/go/proto sdk/typescript` is empty.
- No new identifier contains `PULUMICOST_`.
- No `os.Setenv` in `sdk/go/pluginsdk/credentials.go`.
- No `connect.WithInterceptors` added in `sdk/go/pluginsdk/sdk.go`.
- `DefaultAllowedHeaders` is unchanged.
- The diff does not add a process pool, a router, or an instance cap.

## Out of scope for this guide

Do not start a multi-tenant host, do not measure process RSS, and do not call a cloud
API. Those are the host follow-ons this slice does not claim to finish.
