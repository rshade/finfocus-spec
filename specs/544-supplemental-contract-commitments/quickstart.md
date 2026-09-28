# Quickstart: Validate the Contract Commitments Service

Run every command from the repository root unless a step says otherwise.

## Prerequisites

- Go (version per `go.mod`), Node.js 22 or later
- `npm ci` in the repository root and in `sdk/typescript` if `node_modules` is missing

## 1. Generate and check the protocol

```bash
make generate
mise exec -- buf lint
mise exec -- buf breaking --against '.git#branch=main'
git status --short sdk/go/proto sdk/typescript/packages/client/src/generated
```

Expected: lint and breaking checks are clean. Only `supplemental*` files, `pbcconnect/supplemental.connect.go`,
`enums.pb.go`, `supplemental_pb.ts`, and `enums_pb.ts` are new or changed.

## 2. Go SDK

```bash
go test ./sdk/go/testing/ -run 'ContractCommitment|Supplemental' -v
go test ./sdk/go/pluginsdk/ -run 'ContractCommitment|Supplemental|Capabilit|Legacy' -v
go test -run '^$' -bench 'ContractCommitment' -benchmem ./sdk/go/testing/ ./sdk/go/pluginsdk/
```

Expected:

- The reference producer passes every `RunContractCommitmentConformance` subtest, and each broken
  source fails the scenario that targets its defect.
- A plugin implementing `ContractCommitmentProvider` serves the RPC over gRPC and Connect with
  identical results and error codes, reports it in the Connect health check, and advertises
  `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS` and `supports_contract_commitments`.
- A plugin without the provider returns `Unimplemented` and advertises neither.
- Validator benchmarks report `0 allocs/op`.

## 3. Full Go suite and lint

```bash
make test
golangci-lint run ./...
```

## 4. TypeScript

```bash
cd sdk/typescript/packages/client
npx vitest run
npx tsc --noEmit
```

Expected: `supplemental-dataset.test.ts` passes (single page, multi-page iteration, error code).

## 5. Docs

```bash
npx markdownlint-cli2 docs/supplemental-datasets.md sdk/go/pluginsdk/README.md sdk/go/testing/README.md \
  sdk/typescript/README.md specs/544-supplemental-contract-commitments/*.md
```
