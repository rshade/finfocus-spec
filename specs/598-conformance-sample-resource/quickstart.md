# Quickstart: Validate the Conformance Sample Resource

Run from the repository root.

## 1. Default behavior is unchanged

```bash
go test ./sdk/go/testing/ ./sdk/go/pluginsdk/
go test -v -run TestConformance ./sdk/go/testing/
```

Expected: every existing test passes with no edits to its call site.

## 2. A neutral single-provider plugin passes every level

```bash
go test -v -run 'TestSampleResource' ./sdk/go/testing/ ./sdk/go/pluginsdk/
```

Expected:

- With `WithSampleResource` set to a `custom` descriptor, Basic, Standard, and Advanced report 0 failures.
- Without it, Basic reports failures from the default `aws` descriptor (the defect, kept as a guard).
- An invalid sample resource (empty provider) returns an error before any check runs.

## 3. No hard-coded descriptor is left in the registered checks

```bash
grep -nE 'CreateResourceDescriptor\((providerAWS|"aws")' \
  sdk/go/testing/{spec_validation,rpc_correctness,performance,concurrency}.go
```

Expected: no output. The default lives only in `DefaultSampleResource`.

## 4. Provider neutrality (FR-011)

```bash
git diff origin/main -- sdk/ docs/ PLUGIN_DEVELOPER_GUIDE.md | grep -niE '^\+.*(azure|gcp|kubernetes)'
```

Expected: no output.

## 5. Docs examples compile

Compile the README and developer-guide snippets that call `RunConformance` (one file each, as in
issue 623's README compile check). See [contracts/go-api.md](contracts/go-api.md) for the API.
