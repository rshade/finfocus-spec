# Feature Specification: Omitted Fields on Scoring Requests

**Feature Branch**: `593-omitted-fields`

**Created**: 2026-10-01

**Status**: Complete

**Input**: GitHub issue 576, "Scoring request cannot tell fields the host removed from fields that
were never present". Consumer context: `finfocus` issue 1569 (scoring in core).

## Summary

A host may clear `Recommendation` fields before scoring because an operator allowlist says they must
not leave the process. Today the scorer sees a cleared field exactly like a field the resource never
had. An empty tag map after clearing looks like an untagged resource, and a scorer can raise
`insufficient_evidence` or lower its confidence for the wrong reason. Add one repeated string on the
request, `omitted_fields`, listing the field paths the host removed by policy, and define what a
scorer must do with it. The change is additive.

## Clarifications

`/speckit-clarify` was not required: no material ambiguity remains after the answers below.

- Q: Which request field number? → A: 5. Request field 4 is `session_id` and response field 5 is its
  echo in the cross-batch grouping work (issue 574, spec 592), so 5 on the request leaves both free.
  Response fields are untouched here.
- Q: What is a path? → A: A dot-separated chain of proto field names (snake_case, as in the
  `.proto`), starting at `Recommendation`, for example `resource.tags` or `kubernetes.cluster_id`. No
  indexes, no map keys, no wildcards. Naming a message field omits everything beneath it. The oneof name
  `action_detail` is accepted and
  means every member of that oneof.
- Q: Does the validator check that the host really cleared the field? → A: No. It checks only the
  list: well-formed, resolvable against `Recommendation`, unique, bounded.
- Q: Does an unknown path fail the request? → A: Yes, in `ValidateScoreRecommendationsRequest`. A
  host and scorer built from different spec versions may disagree; the scorer decides whether to
  call the validator or ignore unknown paths. The validator is strict because it is the shared rule.
- Q: Does it change scores? → A: Only through the rule below. The SDK computes no scores.

## User Stories

- **US1, Signal (P1)**: A host that cleared `tags` and `metadata` sends
  `omitted_fields = ["resource.tags", "metadata"]`. A scorer reads it and does not treat the empty maps as
  evidence about the resource.
- **US2, Validation (P1)**: `ValidateScoreRecommendationsRequest` accepts a well-formed list, and
  rejects an empty entry, a path that does not resolve against `Recommendation`, a duplicate, an
  over-long path, and too many entries, each with `InvalidArgument` wrapping
  `ErrInvalidScoreRequest`.
- **US3, Mock and conformance (P2)**: The mock scorer follows the rule and its own tests prove it.
  `RunScorerConformance` stays model-agnostic: it checks that a scorer accepts a valid list and
  returns a structurally valid response, and rejects a malformed list. It never compares score
  values.
- **US4, Documentation and clients (P2)**: `docs/recommendation-scoring.md`, the `pluginsdk` README,
  and the proto comment state the rule. The TypeScript client carries the field.

## Requirements

- **FR-001**: `ScoreRecommendationsRequest` gains `repeated string omitted_fields = 5`. No existing
  field, message, or enum changes; `buf breaking` against `main` passes.
- **FR-002**: Docs and proto comment state: an entry in `omitted_fields` means the host removed that
  field by policy. A scorer must not lower confidence, raise `insufficient_evidence`, or raise
  `false_positive` because an omitted field is empty. Absence of a field that is not listed keeps its
  current meaning. The list applies to every recommendation in the request.
- **FR-003**: `omitted_fields` is advisory input. The host performs the clearing; the field neither
  clears nor restores anything.
- **FR-004**: Path syntax: one or more field names joined by `.`, each matching `[a-z][a-z0-9_]*`,
  at most 128 bytes, resolving to a field of `Recommendation` (or the oneof name `action_detail`), or of a message
  reachable through singular or repeated message fields. Map fields and scalars are leaves.
- **FR-005**: The list has at most 64 entries, each unique. An empty list means the host removed
  nothing it can name, and is the default.
- **FR-006**: `ValidateScoreRecommendationsRequest` enforces FR-004 and FR-005 and adds no allocations
  when the list is empty (the existing duplicate-id map is unchanged).
- **FR-007**: The mock scorer does not raise `insufficient_evidence` because an omitted field is empty
  (its tests prove this), and `RunScorerConformance` covers the accepting and rejecting cases without
  comparing score values.
- **FR-008**: The TypeScript client and its tests carry the field; no hand-edited generated code.

## Success Criteria

- `make generate` leaves no diff, `buf lint` and `buf breaking` pass.
- `make test`, the integration and conformance runs, and `make lint-go` pass.
- A request with an empty `omitted_fields` validates and scores exactly as before.
- Each malformed-list case fails a named validator or conformance case.

## Assumptions

- Core's `applyAllowlist` can produce proto field names for what it clears (issue 1569 side).
- The cap of 64 entries and 128 bytes comfortably exceeds the number of fields in `Recommendation`.

## Out of Scope

Reporting what a scorer ignored, per-recommendation omission lists, restoring cleared values,
changing `identifier_mode`, and any change to scoring rules beyond the stated one.
