# Quickstart: Trace Id on Plugin Failures

Validation guide for the behavior that already shipped. Details are in
[contracts/failure-trace.md](./contracts/failure-trace.md) and
[data-model.md](./data-model.md). This file does not repeat them and it does
not contain an implementation.

## Prerequisites

- Repository root, Go 1.27.1 as in `go.mod`.
- No generated code, no extra module, no running plugin process.
- The tests use an in-memory interceptor. They do not bind a port.

## What to run

From the repository root:

```bash
go test ./sdk/go/pluginsdk/ -count=1 -run 'TestTracingUnaryServerInterceptor_StampsValidationErrorAndLog|TestTracingUnaryServerInterceptor_KeepsHandlerTraceID|TestWithTrace|TestValidationError_TraceID|TestTracingUnaryServerInterceptor_InvalidTraceIDs|TestTracingUnaryServerInterceptor_MissingMetadata|TestTracingUnaryServerInterceptor_ValidTraceIDs'
```

Expected: the package tests pass, including those seven.

## What each test locks

| Test | Proves |
|------|--------|
| `TestTracingUnaryServerInterceptor_ValidTraceIDs` | A valid `x-finfocus-trace-id` is stored unchanged (FR-001). |
| `TestTracingUnaryServerInterceptor_InvalidTraceIDs` | A bad header is replaced, not stored (FR-002). |
| `TestTracingUnaryServerInterceptor_MissingMetadata` | A missing header gets a generated id (FR-002). |
| `TestTracingUnaryServerInterceptor_StampsValidationErrorAndLog` | A failed call with an empty validation id logs the host id and the method, and stamps that id (FR-004, FR-006). |
| `TestTracingUnaryServerInterceptor_KeepsHandlerTraceID` | A handler-set id is not overwritten (FR-007). |
| `TestValidationError_TraceID` | The suffix appears only when `TraceID` is set, and the sentence otherwise matches the pre-change text (FR-008, SC-003). |
| `TestWithTrace` | One attachment copies the id; a context with no id does not add the field (FR-009). |

## What this guide does not ask for

- Do not run `make generate`. Protos are not part of this slice.
- Do not add a Connect test that expects `rpc failed`. Connect is out of the
  contract. A test that expected the line there would be wrong.
- Do not add a benchmark. See the performance decision in
  [research.md](./research.md).
- Do not point these tests at a live host. This repository does not contain
  the host process. The header contract is what the interceptor accepts.

## Reading a failure by hand

1. Send `x-finfocus-trace-id` as 32 lowercase hex characters, not all zeros.
2. Force a validation failure from the handler and leave `TraceID` empty.
3. The plugin error log message is `rpc failed`, `trace_id` is the header, and
   `operation` is the full method name.
4. The returned text ends with a space and `trace_id=` and that same id.
5. Repeat with `TraceID` already set on the validation failure. The text keeps
   the handler's id. The log's `trace_id` stays the stored call id.
