# Implementation Plan: Cross-Batch Scoring Sessions

**Branch**: `592-cross-batch-scoring-groups` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

## Summary

Add `session_id` (request field 4, response field 5) to the scorer messages. With a session, group
ids are a deterministic function of session id and duplicate key, so they match across batches, and the
"at least two members" rule relaxes. Docs state the pseudonymization-key and cached-item rules.
Additive only.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
buf v1.32.1; stdlib `crypto/sha256` and `encoding/hex` for the mock. No new dependencies.

**Storage**: N/A (stateless; group ids are derived, never stored)

**Testing**: `make test`, `go test ./sdk/go/testing/ ./sdk/go/pluginsdk/`, conformance, `npx vitest run`
in `sdk/typescript/packages/client`, `buf breaking`

**Target Platform**: Library and wire contract

**Project Type**: Proto-first spec repository with Go and TypeScript SDKs

**Performance Goals**: Validators keep 0 allocs/op on valid requests and responses without a session.
The session check is one length test when `session_id` is empty.

**Constraints**: Additive; no existing field, number, or rule changes for session-less requests

**Scale/Scope**: Two proto fields, two validator changes, one mock change, three conformance
scenarios, TS types, one docs file

## Constitution Check

| Principle | Result |
| --------- | ------ |
| I. Proto first | Pass. `scoring.proto` changes first, then `make generate`. |
| II. Multi-provider | Pass. No provider-specific field. |
| III. Spec consumes, does not calculate | Pass. The contract fixes derivation properties; it ships no grouping engine. The mock derives ids only as a reference. |
| SDK parity | Pass. Go, TypeScript, mock, conformance, and docs in one PR. |
| Documentation integrity | Pass. Docs updated; `make lint-markdown` is a gate. |

Post-design re-check: no change.

## Project Structure

### Documentation (this feature)

```text
specs/592-cross-batch-scoring-groups/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/session-contract.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/scoring.proto                         # fields + comments
sdk/go/proto/finfocus/v1/scoring*.go                    # generated
sdk/typescript/packages/client/src/generated/...        # generated
sdk/typescript/packages/client/src/clients/recommendation-scorer.ts
sdk/typescript/packages/client/test/recommendation-scorer.test.ts
sdk/go/testing/scoring.go                               # request + response validators
sdk/go/testing/scorer_mock.go                           # session-aware group ids and echo
sdk/go/testing/scorer_conformance.go                    # session scenarios
sdk/go/testing/scoring_test.go, scorer_mock_test.go, scorer_conformance_test.go
sdk/go/testing/README.md
sdk/go/pluginsdk/scorer.go                              # delegation (doc comments only)
docs/recommendation-scoring.md
```

**Structure Decision**: Follow spec 556 and 052: rules in `sdk/go/testing`, delegation in
`pluginsdk`.

## Conflict Notes

Spec 591 (issue 573) edits `scoring.proto`, `scoring.go`, `scorer_mock.go`, `scorer_conformance.go`,
`export_test.go`, `docs/recommendation-scoring.md`, the TS client, and `CLAUDE.md`. Expect textual
conflicts in those files and in regenerated bindings; regenerate with `make generate` after merging
rather than merging generated code by hand.
