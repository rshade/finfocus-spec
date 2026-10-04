# Implementation Plan: Keep Handler Status Codes in the Plugin SDK Server

**Branch**: `600-handler-status-codes` | **Date**: 2026-10-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/600-handler-status-codes/spec.md`

## Summary

`pluginsdk.Server` rewraps every handler error from eight RPCs as `Internal`. One helper,
`handlerStatus`, now classifies the error: a real gRPC status passes through, `Unimplemented` sends
the RPC down its existing not-a-provider path, and anything else stays `Internal` with today's
message. The not-a-provider branches become small methods shared by both paths, so a plugin that
embeds the generated stub behaves like one that does not. Stacked on PR 627. Design:
[research.md](research.md); behavior table: [contracts/server-errors.md](contracts/server-errors.md).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod)

**Primary Dependencies**: google.golang.org/grpc (`status`, `codes`), connectrpc.com/connect
(unchanged `toConnectError`); no new dependencies

**Storage**: N/A

**Testing**: `go test` with testify; `NewServer` calls plus a Connect round trip; `make test`,
`make lint-go`

**Target Platform**: Go library (plugin SDK server)

**Project Type**: library

**Performance Goals**: no change on the success path; the helper runs only when a handler errors

**Constraints**: no exported API change; existing plain-error tests unchanged (SC-003); neutral test
names (FR-008)

**Scale/Scope**: one file of production code (`sdk/go/pluginsdk/sdk.go`), one new test file, README

## Constitution Check

| Principle | Status | Note |
| --- | --- | --- |
| I. Proto first | N/A | No proto change |
| II. Multi-provider | Pass | Provider-agnostic error handling |
| V. Test first | Pass | Table tests over all eight RPCs written first and failing |
| VI. Compatibility | Pass | Codes change only from `Internal` to the handler's own; plain errors unchanged |
| VII. Documentation | Pass | pluginsdk README section |
| VIII. Performance | Pass | Error path only |
| X. Patterns | Pass | Mirrors `toConnectError`'s status handling |
| XI. Headers | Pass | New test file carries the Apache header |
| XIII. SDK sync | N/A | Server-side Go only; the TS client already surfaces any code |
| XIV. Doc integrity | Pass | README example compiles |

Post-design re-check: no violations.

## Project Structure

```text
specs/600-handler-status-codes/          spec, plan, research, contracts/server-errors.md, quickstart, tasks
sdk/go/pluginsdk/sdk.go                  handlerStatus helper; default-path methods; eight wrappers
sdk/go/pluginsdk/handler_status_test.go  new: pass-through, internal, fallback, stub parity, Connect
sdk/go/pluginsdk/README.md               "Handler errors" section
CLAUDE.md                                pattern notes (by hand)
```

## Complexity Tracking

No violations.
