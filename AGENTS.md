# Agent Guidelines for FinFocus Spec

## Build/Test Commands

- **All tests**: `make test` or `go test ./...`
- **Single test**: `go test -run TestName ./path/to/package`
- **Validate all**: `make validate` (tests + linting + schemas)
- **Generate protobuf**: `make generate`
- **Lint Go**: `make lint` or `golangci-lint run`
- **Validate schemas**: `npm run validate:schema && npm run validate:examples`

## Code Style

- **Go version**: 1.27.1 (per go.mod)
- **Formatting**: `goimports` + `golines` (120 char lines)
- **Linting**: 120+ linters via golangci-lint (see `.golangci.yml`)
- **Imports**: Standard library first, then third-party, then local
- **Naming**: CamelCase (protobuf snake_case → Go CamelCase)
- **Functions**: Small, composable; add `String()` for new enums
- **Error handling**: Check all errors; use table-driven tests
- **Tests**: Separate `_test` packages; cover error paths
- **Markdown/YAML**: Follow `.markdownlint.json` and `.yamllint`

## Key Rules

- Never edit generated code (`sdk/go/proto/`, `bin/buf`)
- Run `make validate` before commits
- Use conventional commits: `feat:`, `fix:`, `chore:`
- Sanitize secrets in examples

## Active Technologies

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) + google.golang.org/protobuf,
  buf v1.32.1; no new dependencies (597-actual-cost-resource-descriptor)
- N/A (one optional ResourceDescriptor on GetActualCostRequest) (597-actual-cost-resource-descriptor)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) + google.golang.org/protobuf
  (`proto.Size`, `structpb`), buf v1.32.1; no new dependencies (596-resource-descriptor-attributes)
- N/A (one optional Struct on ResourceDescriptor, two size limits, one read helper)
  (596-resource-descriptor-attributes)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
  buf v1.32.1; no new dependencies (547-invoice-dataset-rpcs)
- N/A (paged RPCs over the existing BillingPeriod and InvoiceDetail messages)
  (547-invoice-dataset-rpcs)

- Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK) +
  google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; stdlib only
  (546-focus-14-billing-invoice)
- N/A (stateless BillingPeriod and InvoiceDetail messages, builders, and per-record validation)
  (546-focus-14-billing-invoice)

- Go 1.27.1 (go.mod) + google.golang.org/protobuf (protojson, protocmp in tests) (055-cost-allocation-lineage)
- N/A (wire contract and in-memory builder only) (055-cost-allocation-lineage)

- Go 1.25.5 + Go standard library (errors, fmt) (047-validation-error-integration)
- N/A (in-memory validation only) (047-validation-error-integration)

- Documentation (Markdown, JSON) - No code implementation + markdownlint-cli2 (for validation), JSON schema validators (001-migration-docs)
- Files - Repository documentation files (MIGRATION.md, CHANGELOG.md, README.md) (001-migration-docs)

- Go 1.25.5+ + gRPC, Protobuf, Buf, ConnectRPC, RS/Zerolog (035-project-rename-finfocus)
- N/A (Protobuf definitions and Go SDK) (035-project-rename-finfocus)

- Go 1.25.5 + gRPC, protobuf, buf v1.32.1 (034-sdk-polish)

- Go 1.25.5 (as specified in go.mod) + gRPC/protobuf (existing), buf v1.32.1 (existing for proto management) (001-sdk-polish-release)
- N/A (SDK does not manage persistent storage) (001-sdk-polish-release)

- Go 1.25.5 (per go.mod) + gRPC, protobuf, buf v1.32.1 (001-get-budgets-rpc)
- JSON Schema (Draft 2020-12) for PricingSpec and BudgetSpec validation (001-get-budgets-rpc)

## Recent Changes

- 597-actual-cost-resource-descriptor: Added GetActualCostRequest.resource (field 11, a
  ResourceDescriptor) with the fallback and tags-precedence rules, descriptor validation in
  pluginsdk.ValidateActualCostRequest and plugintesting.ValidateGetActualCostRequest, mock FOCUS
  records that read type, region, and SKU from it, and the Standard conformance test
  RPCCorrectness_GetActualCostWithResource (issue 620)

- 596-resource-descriptor-attributes: Added ResourceDescriptor.attributes (field 12, a
  google.protobuf.Struct) with the host redaction rule and the tags fallback, MaxAttributesBytes
  (65536) and ErrAttributesTooLarge in both validators, MaxTagValueLength 256 to 2048 in both,
  pluginsdk.AttributeValue, the TypeScript ResourceDescriptorBuilder.withAttributes, and the Basic
  conformance test RPCCorrectness_GetProjectedCostWithAttributes (issue 617)

- 547-invoice-dataset-rpcs: Added GetBillingPeriods and GetInvoiceDetails on
  SupplementalDatasetService, capability 17 (`supports_invoice_data`), and the Go and
  TypeScript clients. One provider implements both RPCs. A missing provider returns
  UNIMPLEMENTED. GetContractCommitments fields are unchanged.

- 546-focus-14-billing-invoice: Added BillingPeriod and InvoiceDetail messages, status enums,
  pluginsdk builders, per-record validators, JSON-LD serializers, and TypeScript builders.
  Messages only; the RPCs are 547.

- 545-focus-14-contract-commitment: Added ContractCommitment fields 13-30, seven FOCUS 1.4 enums,
  optional doubles for null-vs-zero, and FormatContractApplied for the ContractApplied JSON object.

- 055-cost-allocation-lineage: Added Go 1.27.1 (go.mod) + google.golang.org/protobuf (protojson, protocmp in tests)
- 001-get-budgets-rpc: Added Go 1.25.5 (per go.mod) + gRPC, protobuf, buf v1.32.1

## Common Issues & Solutions

- Issue: `make lint` and `make validate` may time out on this project.
  Solution: Run `golangci-lint run` directly for faster Go linting results, or `make test` for unit tests.

## Workflow Optimizations

- For CodeRabbit fixes: Always verify `git log` and file content first; reviews may reference older
  commits that have already been fixed by subsequent pushes.

## Project-Specific Patterns

- `pluginsdk.Serve`: Tests dealing with `Serve` should prefer injecting a `net.Listener` (via
  `ServeConfig.Listener`) rather than relying on `Port` and `listenOnLoopback` to avoid race
  conditions and ensure predictable port binding.
- `pluginsdk.Run`: Plugin binary entry point. Handshake (`--port` / `serve` / no args) calls
  `Serve()` outside `ax.Execute` so stdout stays `PORT=<n>`. `ServeConfig.Logger` remains
  `*zerolog.Logger`. `ParsePortFlag()` is legacy-only.

## CI Variance

GitHub Actions CI runners exhibit high-performance variability (up to 2x for sub-microsecond
benchmarks). Benchmark alerts are informational and should not fail builds (`fail-on-alert: false`).
The alert threshold is set to 150% to reduce noise.
