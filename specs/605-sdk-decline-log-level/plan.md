# Implementation Plan: Plugin SDK Decline Log Levels

**Branch**: `605-sdk-decline-log-level` | **Date**: 2026-11-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/605-sdk-decline-log-level/spec.md`

## Summary

One unexported classifier in `sdk/go/pluginsdk` maps an error to a log level: debug for the four decline
codes, ERROR for everything else. The unary tracing interceptor ("rpc failed") and the nine
`handlerStatus` wrappers in `sdk.go` call it instead of hard-coding ERROR. No proto, wire, or exported
API change.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod)

**Primary Dependencies**: `google.golang.org/grpc` v1.84.0 (`status.Code` unwraps wrapped errors),
`github.com/rs/zerolog`; no new dependencies

**Storage**: N/A

**Testing**: `go test` with a `zerolog` JSON buffer; table-driven over all 16 non-OK gRPC codes

**Target Platform**: Go plugin binaries built on the SDK

**Project Type**: library (SDK)

**Performance Goals**: no regression on the error path; the classifier does no allocation

**Constraints**: status and message returned to the host unchanged; plugins must rebuild to pick it up

**Scale/Scope**: 2 new files (`log_level.go`, `log_level_test.go`), 10 call sites (1 interceptor, 9 wrappers),
2 extended test files

## Constitution Check

| Principle | Result |
| --- | --- |
| I Proto-first | Pass: no proto change |
| III Spec consumes | Pass: no pricing logic |
| V Test-first | Pass: tests written and shown failing before the change |
| VI Backward compatibility | Pass: `buf breaking` unaffected; wire outcome identical |
| VII Documentation currency | Pass: SDK README logging section and CLAUDE.md updated in the PR |
| VIII Performance | Pass: benchmark on the classifier, 0 allocs/op |
| IX Observability | Pass: improves signal-to-noise; debug entry keeps trace id, operation, error |
| X Established patterns | Pass: follows `handlerStatus` / `logHandlerFallback` beside it, no new pattern |
| XI Copyright headers | Pass: both new files carry the Apache 2.0 header (task T015) |
| XIII SDK sync | Not applicable: no proto change, and the TypeScript client has no server-side logging |
| XIV Documentation integrity | Pass: README note and godoc in the same PR; spec status set to Implemented |
| SDK parity | Not applicable: the TypeScript client has no server-side logging |

Re-checked after design: no violations.

## Project Structure

### Documentation (this feature)

```text
specs/605-sdk-decline-log-level/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/log-levels.md
└── tasks.md
```

### Source Code (repository root)

```text
sdk/go/pluginsdk/
├── log_level.go            # new: handlerLogLevel, declined
├── log_level_test.go       # new: table over all codes, benchmark
├── logging.go              # interceptor uses the level
├── logging_test.go         # interceptor level tests
├── sdk.go                  # eight wrappers use the level
├── recommendation_history.go # the ninth wrapper uses the level
└── handler_status_test.go  # wrapper level tests (Supports plus the rest)
sdk/go/pluginsdk/README.md  # logging levels note
```

**Structure Decision**: Single package. The helper lives beside `handlerStatus` rather than in
`sdk/go/testing` because no host or conformance code needs it.
