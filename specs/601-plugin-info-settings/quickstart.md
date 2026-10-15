# Quickstart: Validate Plugin Settings on Plugin Info

Run from the repository root of this worktree.

## Prerequisites

- Go per `go.mod`, Node.js >= 22
- `npm ci` at the repo root and in `sdk/typescript`
- `make generate` so buf has written the Go and TypeScript bindings

## 1. Proto and bindings

```bash
make generate && git diff --exit-code -- sdk/
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
```

Expected: generate is clean, buf lint is clean, and breaking prints nothing. In a worktree
where `.git#branch=main` cannot be read, archive `origin/main` and pass that directory to
`buf breaking --against`.

## 2. Validator

```bash
go test ./sdk/go/testing/ -run 'TestValidatePluginSettings' -count=1
go test ./sdk/go/testing/ -run '^$' -bench 'BenchmarkValidatePluginSettings' -benchmem -count=1
```

Expected: each malformed rule fails and the error text does not contain the sample default.
`BenchmarkValidatePluginSettings_Empty` reports 0 allocs/op.

## 3. Serve and mock

```bash
go test ./sdk/go/pluginsdk/ -run 'TestPluginSettings|TestPluginInfo' -count=1
go test ./sdk/go/testing/ -run 'TestMockPluginSettings|TestConformance' -count=1
```

Expected: static info with two settings is returned in order; a bad static list fails
`Serve`; a provider that returns a duplicate key gets `Internal` and `plugin settings are
invalid`; the mock copies `Settings`; conformance passes the default mock and fails a stub
that repeats a key.

## 4. TypeScript

```bash
cd sdk/typescript && npm ci && npm run build && npm test --workspace @rshade/finfocus-client
```

Expected: `plugin-settings.test.ts` reads key, type, secret, and env after a JSON round trip.
The known middleware TS7006 build error is pre-existing; do not treat it as this change.

## 5. Docs and schema

```bash
make lint-markdown
make validate-schema
```

Expected: the developer guide, the pluginsdk README, and the manifest `configuration`
description lint and validate clean.
