# Implementation Plan: Trace Id on Plugin Failures

**Branch**: `055-trace-id-on-failures` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/055-trace-id-on-failures/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

An ordinary served call already stores a valid `x-finfocus-trace-id` on the
handler context. Invalid or missing values are replaced (spec 008). That rule
stays. Before pull request 538, a failed call did not write that id to the
plugin logger, and a validation failure did not carry it unless the handler
set it.

The shipped slice, and the only slice this plan describes, is: on failure,
`serveGRPC` logs `rpc failed` with that stored id and the full method name, and
stamps the id onto a returned `*ValidationError` whose `TraceID` is empty.
`Error()` appends a space and `trace_id=<id>` only when the field is set. `WithTrace`
copies the id onto a child logger once per request. The web serving mode
(`serveConnect`) does not run the interceptor. Specs 051 and 052 left that out
on purpose. It is a non-goal, not a defect.

The code is on main as commit `a5e6790` (pull request 538). This plan is the
design record for that behavior. It does not propose a second implementation.

Words in the spec map as follows. An ordinary served call is `serveGRPC`. The
web serving mode is `serveConnect`. The failure log is the zerolog error event
whose message is `rpc failed`. The validation failure is `*ValidationError`.

## Technical Context

**Language/Version**: Go 1.27.1 (go.mod). No other language.

**Primary Dependencies**: Existing `github.com/rs/zerolog`,
`google.golang.org/grpc` metadata and unary interceptors, and
`sdk/go/pricing.ValidateTraceID`. No new module. No OpenTelemetry module.

**Storage**: N/A. The id lives on the request context and, when stamped, on the
error value the handler already returned. Nothing is persisted.

**Testing**: `go test` in `sdk/go/pluginsdk`, locked by the tests that shipped
in pull request 538. No new benchmark.

**Target Platform**: Library used by FinFocus plugins. The host process is not
in this repository.

**Project Type**: Go SDK library inside a protocol repository.

**Performance Goals**: No work on the success path beyond the context store
spec 008 already did. On failure, one `errors.As` walk and one error log.
Do not allocate a tracer, a span, or a second id.

**Constraints**: Do not change protobuf or run `make generate`. Do not add
`x-pulumicost-trace-parent`. Do not add an OpenTelemetry dependency. Do not
instrument provider billing calls. Do not relax spec 008. Do not run this
interceptor from `serveConnect`. Do not set `FieldTraceID` on a logger that
`WithTrace` already returned. Do not overwrite a `TraceID` the handler set.

**Scale/Scope**: The gRPC server interceptor already wraps every unary method
on that server, including optional usage and allocation services. This slice
adds the failure log and the validation-error stamp at that one site, plus
`WithTrace` for author logs. Connect handlers are untouched.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Proto first**: Pass. No protobuf change. The trace id is logging and
  error text, not a new message field.
- **II. Multi-provider consistency**: Pass. The id is provider-agnostic. No
  billing mode or example payload changes.
- **III. The spec does not calculate**: Pass. No pricing math.
- **IV. Separation of concerns**: Pass. Behavior stays in the plugin SDK. No
  host tracer, exporter, or sampling. No new application dependency.
- **V. Test-first for gRPC changes**: Pass, not applicable as a proto change.
  The behavior is locked by `logging_test.go` and `validation_error_test.go`,
  which landed with the code. This plan does not ask for a new failing test.
- **VI. Protobuf compatibility**: Pass. No wire change. `buf` breaking checks
  are unaffected.
- **VII. Documentation currency**: Pass. `sdk/go/pluginsdk/README.md` shows
  `WithTrace` once and states the failure log, the stamp, and the Connect
  exclusion. `sdk/go/CLAUDE.md` records the same boundary.
- **VIII. Performance**: Pass with note. Principle VIII's benchmark rule is
  aimed at core query and validation logic on large cost payloads. This slice
  does not add a hot-path calculation. A benchmark would not guard a success
  path. None was added, and none is required. See research.
- **IX. Observability**: Pass. Correlation uses the existing zerolog logger.
  Metrics stay separate. No Prometheus change.
- **X. Established patterns**: Pass. The design reuses
  `TracingUnaryServerInterceptor`, `pricing.ValidateTraceID`, and
  `FieldTraceID`. This plan is the design spec the principle asks for. The
  code merged first; the document matches it. No further code change follows
  from the principle.
- **XI. Copyright headers**: Pass. No new source file.
- **XII. Capability declaration**: Pass. No capability enum change.
- **XIII. Multi-language sync**: Pass. The trigger is a gRPC service-definition
  change. There is none, so the TypeScript SDK stays as it is.
- **XIV. Documentation integrity**: Pass. The README example calls `WithTrace`
  and does not also set `FieldTraceID`. Exported `WithTrace` and
  `TracingUnaryServerInterceptorWithLogger` have godoc. No generated symbol
  changed.

## Project Structure

### Documentation (this feature)

```text
specs/055-trace-id-on-failures/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── failure-trace.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
sdk/go/pluginsdk/
├── logging.go                 # interceptor, stampValidationTrace, WithTrace
├── logging_test.go            # failure log, preserved handler id, WithTrace
├── validation_error.go        # TraceID field and Error() suffix
├── validation_error_test.go   # suffix present only when TraceID is set
├── sdk.go                     # serveGRPC wires the plugin logger
├── traceid.go                 # GenerateTraceID, unchanged
└── README.md                  # attach the id once; Connect exclusion

sdk/go/pricing/
└── observability_validate.go  # ValidateTraceID, unchanged (spec 008)

sdk/go/CLAUDE.md               # maintainer note for the same boundary
```

**Structure Decision**: The behavior lives in the existing logging and
validation-error files. `serveGRPC` is the only call site that passes the
plugin logger. `serveConnect` is not given an interceptor. No new package, no
`proto/`, no `sdk/typescript/`.

## Complexity Tracking

No constitution violation requires justification. Connect staying dark is an
earlier decision (specs 051 and 052), recorded here as a non-goal so a later
pass does not "finish" it inside this spec.

## Post-Design Re-check

Research, the data model, and `contracts/failure-trace.md` do not add a header,
a proto field, a tracing module, a Connect interceptor, or a benchmark. Gates
above still pass. Rejected follow-ons, if a later stage proposes them: keeping
invalid host ids, an `x-pulumicost-trace-parent` key, an OpenTelemetry
dependency, provider billing spans, a Connect trace interceptor, a proto
`correlation_id`, and a TypeScript port of this log line.
