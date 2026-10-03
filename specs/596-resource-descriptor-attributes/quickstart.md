# Quickstart: Validating Structured Attributes

Run these from the repository root.

## 1. Proto and bindings

```bash
make generate && git diff --stat -- sdk/   # the diff touches only the costsource bindings
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
```

Expected: lint is clean, and there are no breaking changes.

## 2. Limits (US2, US3)

```bash
go test ./sdk/go/pluginsdk/ -run 'TestValidateResourceDescriptor'
go test ./sdk/go/testing/ -run 'TestValidateResourceDescriptor|TestValidateTags|TestContract'
```

Expected: attributes at 65,536 bytes and a tag value of 2048 characters pass, and 65,537 bytes and 2049
characters fail, in both packages.

## 3. Accessor (US4)

```bash
go test ./sdk/go/pluginsdk/ -run 'TestAttributeValue'
go test ./sdk/go/pluginsdk/ -run '^$' -bench 'AttributeValue|ValidateResourceDescriptor' -benchmem
```

Expected: every table case passes. `BenchmarkValidateResourceDescriptor_ZeroAllocs` stays at 0 allocs/op.

## 4. Delivery and conformance (US1, US5)

```bash
go test ./sdk/go/testing/ -run 'Attributes|TestConformance|TestRPCCorrectness'
```

Expected: the plugin receives the ten-segment nested value intact. The mock plugin, which ignores
attributes, passes every level.

## 5. TypeScript

```bash
cd sdk/typescript && npm ci && npm run build && npm test
```

## 6. Docs (US6)

```bash
grep -rnE "256 char|value.{0,20}256|only in .EstimateCost|exists in one place only" \
  --include=*.md . | grep -v node_modules | grep -v '^./specs/' | grep -v CHANGELOG
make lint-markdown
```

Expected: no stale tag value limit and no "only in EstimateCost" claim. Matches on the unrelated
256-character `resource_type` limit are fine.
