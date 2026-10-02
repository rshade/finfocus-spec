# Implementation Plan: Omitted Fields on Scoring Requests

**Branch**: `593-omitted-fields` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

## Summary

Add `repeated string omitted_fields = 5` to `ScoreRecommendationsRequest`, validate it in the shared
request validator, make the mock scorer honor the rule, extend conformance, document the rule, and
carry the field through the TypeScript bindings. Additive only.

## Technical Context

Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript SDK, `google.golang.org/protobuf`
(`protoreflect` to resolve paths), buf v1.32.1. No new dependencies. Storage: N/A.

## Constitution Check

- I/VI Proto first, backward compatible: one new field, number 5, no renumbering. Passes.
- III Spec consumes, does not calculate: the SDK validates the list, never scores. Passes.
- V Test first: tests precede the validator and mock changes in tasks.md. Passes.
- VIII Performance: the empty list adds one `len()` check; the empty-list check adds no allocations
  (the validator already allocates its duplicate-id map). A benchmark A/B against `main` guards it.
- XIII SDK sync: Go and TypeScript in the same PR. XIV docs: `docs/recommendation-scoring.md`,
  `sdk/go/pluginsdk/README.md`, `sdk/go/testing/README.md` updated.
- XI headers: no new source file needs one beyond tests, which carry it.

## Decisions

- Field 5 on the request avoids request field 4 (`session_id`, issue 574 branch 592). The
  other open scorer changes (573 limits via plugin metadata, 580 `ScorerInfo`/per-item errors) do not
  add request fields. If 592 lands first, rebase is a one-line proto adjacency, not a renumber.
- Validation lives in `sdk/go/testing/scoring.go`; `pluginsdk/scorer.go` already delegates. The
  validator resolves each path against `(&pbc.Recommendation{}).ProtoReflect().Descriptor()`, only
  when the list is non-empty. A split on `.` is done with `strings.Cut` loops, no allocation beyond
  the error path.
- Constants: `maxOmittedFields = 64`, `maxOmittedFieldBytes = 128`.
- Path rules: the `action_detail` oneof name is accepted via `Descriptor.Oneofs().ByName` and covers its members. Path
  leaves: a map or scalar field ends a path; a further segment after it is invalid. A message
  field may end a path (omits its whole subtree) or continue.
- Mock: `mockInsufficientEvidence` treats `resource.utilization` / `reasoning` listed in
  `omitted_fields` as not evidence of thinness (returns the "enough" value for that missing input).
- Conformance adds `omitted_fields_accepted` (valid list, valid response) and
  `omitted_fields_rejected` (empty entry, unknown path, duplicate) scenarios. No score comparison.
- Caching: docs add `omitted_fields` to the cache key table.
- TypeScript: bindings regenerate through `make generate`; add a client test asserting the field
  round-trips.

## Files

- `proto/finfocus/v1/scoring.proto` and generated Go, TS bindings
- `sdk/go/testing/scoring.go`, `scorer_mock.go`, `scorer_conformance.go` and their tests
- `docs/recommendation-scoring.md`, `sdk/go/pluginsdk/README.md`, `sdk/go/testing/README.md`
- `sdk/typescript/packages/client/test/recommendation-scorer.test.ts`
- `CLAUDE.md` (Recent Changes), per the project convention; `CHANGELOG.md` is never edited

## Likely conflicts with sibling work

`scoring.proto` request message (574), `scoring.go` request validator (574 session rules, 573), `scorer_conformance.go`
scenario list (573, 574, 580), `scorer_mock.go` (580), `docs/recommendation-scoring.md`, and `CLAUDE.md`.
