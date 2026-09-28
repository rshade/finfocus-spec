# Quickstart: FOCUS 1.4 Cost and Usage Columns

Run every step from the repository root. Each step lists its expected outcome.

## 1. Regenerate and check wire compatibility

```bash
make generate
git status --short
bin/buf lint
bin/buf breaking --against '.git#branch=main'
```

Expected: only `focus.proto`, `focus.pb.go` and `focus_pb.ts` change among proto and generated files
(restore any unrelated `*.connect.go` reformatting with `git checkout --`). Both buf commands print
nothing and exit 0.

`buf.yaml` lists `proto/finfocus/v1/focus.proto` under `breaking.ignore`, so the command above passes
without checking `focus.proto`. To check it for real, extract `main` (`git archive main proto buf.yaml
buf.lock`) and copy the working tree's `proto/`, `buf.yaml` and `buf.lock` into two scratch
directories, delete that `ignore` line from both `buf.yaml` copies, and run
`buf breaking <worktree-copy> --against <main-copy>`. Expected: exit 0 with no output. Adding a
dummy field to the main copy first must produce a "Previously present field ... was deleted"
error, which proves the check sees `focus.proto`.

## 2. FOCUS 1.4 conformance tests

```bash
go test ./sdk/go/testing/ -run 'TestFocus14' -v
```

Expected: all pass, covering rules V1 to V3 from [data-model.md](./data-model.md): a record with only
`service_provider_name` passes, a record with neither provider name fails on `provider_name`, both
new fields round-trip, malformed or non-object eligibility JSON fails `Build()`, and
`FocusFieldNames()` contains both new names.

## 3. Unit tests and the runnable example

```bash
go test ./sdk/go/pluginsdk/ -run 'Focus14|InvoiceDetail|CommitmentProgramEligibility|MandatoryFields|FocusFieldNames|Example' -v
go test ./sdk/go/jsonld/ -run 'Focus14|InvoiceDetail' -v
```

Expected: all pass, including `ExampleFocusRecordBuilder_WithCommitmentProgramEligibilityDetails`.

## 4. Performance

```bash
go test ./sdk/go/pluginsdk/ -run '^$' -bench 'WithInvoiceDetailID|WithCommitmentProgramEligibilityDetails|ValidateFocusRecord_(ValidRecord|Focus14)' -benchmem
```

Expected: both setter benchmarks report under 1 ns/op and `0 allocs/op`. The existing validator
benchmark reports `0 allocs/op`; the one that sets both new fields reports `1 allocs/op` (the
eligibility JSON check).

## 5. TypeScript

```bash
cd sdk/typescript/packages/client
npx vitest run
npx tsc --noEmit
```

Expected: all tests pass, including the actual-cost test that reads `invoiceDetailId` and
`commitmentProgramEligibilityDetails` from `focusRecord`, and the builder tests for the two new
setters. `tsc` reports no errors.

## 6. Full suite and lint

```bash
make test
golangci-lint run ./...
npx markdownlint-cli2 docs/focus-columns.md sdk/go/pluginsdk/README.md CLAUDE.md 'specs/055-focus-14-cost-usage-columns/**/*.md'
```

Expected: all tests pass (see CLAUDE.md for the one known flake), 0 lint issues, 0 markdown errors.
