# Implementation Plan: Invoice Dataset RPCs

**Branch**: `547-invoice-dataset-rpcs` | **Date**: 2026-09-28 | **Spec**: [spec.md](spec.md)

## Summary

Add `GetBillingPeriods` and `GetInvoiceDetails` to the existing `SupplementalDatasetService`.
One `InvoiceDatasetProvider` implements both. Capability 17 covers both because the datasets join.
Window, page size, and token handling stay on the helpers `GetContractCommitments` already uses.

## Decisions

- Same service. No second protobuf service and no change to `GetContractCommitments` fields.
- `Serve` registers the service when `ContractCommitmentProvider` or `InvoiceDatasetProvider` is
  present. A nil provider returns `UNIMPLEMENTED` on gRPC and Connect. Health lists the service once.
- Request checks share `validateDatasetRequest`. Pagination shares `paginateRecords`, so commitment
  error text stays the same.
- Billing-period overlap compares seconds and nanos. Ending exactly at the window start does not
  match. Identity is `(invoice_issuer_name, billing_period_start)`. Invoice lines use
  `invoice_detail_id`.
- `MockInvoiceDatasetSource` is its own type. `MockPlugin` capabilities do not change.
- No correction or delivery handling fields.
- A token that does not decode to a non-negative offset is `INVALID_ARGUMENT`. An offset at or
  past the match count is an empty page. `WithCapabilities` replaces discovery only; `Serve` still
  registers from the provider interfaces. The missing-RPC message is
  `method <RpcName> not implemented`.

## Files

- `proto/finfocus/v1/supplemental.proto`, `proto/finfocus/v1/enums.proto`, generated bindings
- `sdk/go/testing/supplemental.go`, `invoice_dataset.go`, `invoice_dataset_mock.go`,
  `invoice_dataset_conformance.go`
- `sdk/go/pluginsdk/sdk.go`, `supplemental.go`, `plugin_info.go`, `capability_compat.go`
- `sdk/typescript/packages/client/src/clients/supplemental-dataset.ts`
- `docs/supplemental-datasets.md`
