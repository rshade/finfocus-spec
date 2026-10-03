# GEMINI.md - Guiding Principles for finfocus-spec

This document outlines the core principles, architectural guidelines, and development philosophy
for the `finfocus-spec` repository, based on an analysis of existing documentation and direct
feedback. It summarizes the principles that keep contributions aligned with the project's vision.
The authoritative, versioned rules are in the project constitution,
[`.specify/memory/constitution.md`](.specify/memory/constitution.md), which also requires proto-first
changes, test-first conformance tests, Go and TypeScript SDK parity, and documentation in the same PR.

## 1. Project Vision

The `finfocus-spec` repository aims to define the **universal, open-source standard for cloud
cost observability**. It provides the foundational contracts (schemas, Protobufs) and developer
tools to create a robust ecosystem of cost-estimation plugins.

## 2. Core Principles

- **Performance is Paramount:** Code, especially within the Go SDK, must be highly performant and
  memory-efficient. A "zero-allocation" goal for common operations is the standard.
  All new code must be benchmarked.
- **Contracts are Sacred:** The Protobuf definitions and JSON schemas are the source of truth.
  They must be stable, well-documented, and evolved carefully through the established design
  spec process.
- **Developer Experience (DX) for Plugin Creators:** The primary audience for the SDK is the
  plugin developer. The SDK should provide simple, consistent, and powerful building blocks
  that make creating high-quality plugins as easy as possible.
- **Strict Separation of Concerns:** This repository defines the _specification_ and foundational
  tooling. It is not a monolithic application.
  - `finfocus-spec`: Defines the interfaces and data schemas. Provides SDKs for implementation.
  - `finfocus-core`: (Separate repo) Contains higher-level application logic, such as the
    public-facing Plugin Registry service.
  - `finfocus-plugins-*`: (Separate repos) Individual plugins that implement the spec.

## 3. Architectural & Development Guidelines

- **The Spec Consumes, It Does Not Calculate:** The `finfocus-spec` and the plugins that directly
  implement it are not responsible for complex pricing logic (e.g., tiered pricing, committed-use
  discounts). This logic belongs to upstream data providers (like Kubecost, Vantage, Flexera, etc.).
  The spec's role is to consume the final, _adjusted_ cost from these services and provide a
  standardized model for it.
- **Observability is for Maintainers:** Features like metrics (Prometheus) are intended for plugin
  maintainers to diagnose performance and efficiency. They are not primarily for end-users of the
  cost data. Therefore, such features should be implemented as optional, distinct components (e.g.,
  a separate gRPC interceptor) rather than being deeply integrated into core logic like logging.
- **Logging and Metrics are Separate:** `zerolog` is for structured, event-based logging. Prometheus
  is for aggregated, time-series metrics. These serve different purposes and should remain separate
  concerns in the SDK. The existing logging pattern is the standard.
- **Follow Established Patterns:** New contributions must adhere to existing, documented patterns,
  such as the "Standard Domain Enum Pattern" used in the Go SDK for high-performance,
  zero-allocation validation.
- **Changes Require Design Docs:** Significant changes or new features must be proposed and
  documented in a design specification under the `specs/` directory before implementation.

## Active Technologies

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
