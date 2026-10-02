# Feature Specification: Cross-Batch Scoring Sessions

**Feature Branch**: `592-cross-batch-scoring-groups`

**Created**: 2026-10-01

**Status**: Draft

**Input**: GitHub issue 574, "Scoring duplicate groups and pseudonym tokens cannot span batches".
Companion host work: `finfocus` issue 1569.

## Summary

A host that scores more recommendations than a scorer's `max_batch_size` splits them into several
`ScoreRecommendations` calls. Today `duplicate_group_id` is meaningful only within one response and
pseudonym tokens are identical only within one request, so two recommendations that duplicate each
other but land in different batches are never grouped, and the loss is silent. Recommendations the
host already cached are dropped from later requests, so their duplicates are never grouped either.

This feature adds an optional, host-chosen **session** that spans the batches of one host operation.
When a request carries a session id, the scorer derives duplicate group ids deterministically from
that id and the group's duplicate key, so the same duplicate group has the same id in every batch of the
session and the host merges by equality. The host uses one pseudonymization key for the whole
session, so tokens are stable across its batches. The contract also states how grouping behaves for
items the host leaves out of a batch because they are cached.

The change is additive. A request without a session id behaves exactly as it does today.

## Clarifications

- Q: Which mechanism: session id, stable namespace, or a dedupe RPC? -> A: A request-level
  `session_id` (option 1 of the issue). It reuses the existing RPC, needs no scorer state, and
  covers both group ids and pseudonym tokens with one concept. A stable namespace alone does not
  fix tokens. A dedupe RPC adds a service and a second round trip.
- Q: Does the scorer store anything between calls? -> A: No. Group ids are a pure function of
  `session_id` and the duplicate key, so scorers stay stateless.
- Q: Who owns the pseudonymization key? -> A: The host, as today. The session only fixes that the
  host uses one key for every batch carrying the same `session_id`. The key never crosses the wire.
- Q: Are cached items grouped? -> A: Only if the host includes them in some batch of the session.
  The contract says so plainly and gives the host the choice. The scorer cannot see what it was not
  sent.
- Q: Does a group of one member stay cleared? -> A: Without a session, yes (unchanged). With a
  session, a response may hold one member of a group because the other members are in other
  batches.

Clarify pass: not required. The mechanism choice is settled above and no material ambiguity remains.

## User Stories

- **US1, Grouping across batches (P1)**: A host splits a large set into batches, sends the same
  `session_id` with each, and finds that duplicates in different batches share one
  `duplicate_group_id`.
- **US2, Stable tokens (P1)**: A host that pseudonymizes identifiers uses one key per session, so a
  resource has the same token in every batch, and the contract says this is what `session_id`
  promises.
- **US3, Backward compatibility (P1)**: A host or scorer that ignores `session_id` keeps today's
  behavior and validation, byte for byte.
- **US4, Validation (P1)**: Shared validators accept a lone group member only when a session id is
  present, reject a malformed session id, and detect a scorer that changes or drops the echoed id.
- **US5, Conformance (P2)**: `RunScorerConformance` splits a known duplicate pair across two
  batches of one session and expects equal non-empty group ids, and expects different batches of
  different sessions to stay independent.
- **US6, Cached items (P2)**: Docs state the host's options for items it served from cache, and
  that a cached item missing from the session is not grouped.

## Edge Cases

- Empty `session_id`: single-request scope, as today.
- Over-long or non-printable `session_id`: `INVALID_ARGUMENT`.
- Two sessions with the same duplicate key: group ids differ.
- A scorer that does not support sessions: leaves ids per-response; hosts detect this from the
  missing echo and fall back to merging nothing.
- `IDENTIFIER_MODE_OMITTED`: grouping stays unreliable, with or without a session.
- Per-item failure of the only scored member of a group: the id is cleared as today.

## Requirements

- **FR-001**: `ScoreRecommendationsRequest` gains `string session_id = 4`, an opaque host-chosen
  identifier of at most 128 printable ASCII characters. Empty means no session. No existing field
  changes.
- **FR-002**: `ScoreRecommendationsResponse` gains `string session_id = 5`. A scorer that honors
  sessions echoes the request value; it is empty otherwise.
- **FR-003**: With a non-empty `session_id`, a scorer MUST derive each `duplicate_group_id` as a
  deterministic function of the session id and the group's duplicate key (whatever the scorer groups on, for example
  resource and action type), with no state kept between
  calls, so the same duplicate group has the same id in every batch of the session and ids of
  different groups or sessions differ.
- **FR-004**: The rule that a non-empty `duplicate_group_id` is shared by at least two members of
  the response applies only when `session_id` is empty. With a session, a response may hold a
  single member of a group.
- **FR-005**: The contract says a host using a session MUST use one pseudonymization key for every
  batch carrying that `session_id`, and MUST NOT reuse a `session_id` across operations or with a
  different key.
- **FR-006**: The contract says how grouping treats cached items: a recommendation absent from every
  batch of the session is not grouped, and a host that wants cross-cache grouping includes the
  cached item in a batch and discards the other scores.
- **FR-007**: Request validators reject a `session_id` that is longer than 128 characters or holds
  non-printable characters, with `INVALID_ARGUMENT` and no `rpc error:` prefix.
- **FR-008**: Response validators reject an echoed `session_id` that is non-empty and differs from
  the request, and apply FR-004.
- **FR-009**: `MockRecommendationScorer` honors sessions: it echoes the id and derives group ids
  from the session id and the duplicate key.
- **FR-010**: `RunScorerConformance` adds scenarios for cross-batch grouping, session isolation, and
  the echo, and does not require session support from a scorer that declines it.
- **FR-011**: The TypeScript client exposes `session_id` on the request and response, and its tests
  cover it.
- **FR-012**: `docs/recommendation-scoring.md` and the `scoring.proto` comments replace "meaningful
  only within one response" and "within one request" with the session rules, and the score-cache
  section states that group ids from a session still MUST NOT be cached per recommendation.
- **FR-013**: The change is additive: `buf breaking` against `main` reports nothing, and requests
  without a session validate and behave as before.

## Key Entities

- **Session**: A host operation spanning several `ScoreRecommendations` calls, named by `session_id`.
- **Group id**: `duplicate_group_id`, stable across the batches of one session.
- **Pseudonymization key**: Host-held secret behind `IDENTIFIER_MODE_PSEUDONYMIZED` tokens, one per
  session.

## Success Criteria

- **SC-001**: A duplicate pair split across two batches of one session gets equal non-empty group
  ids in 100 percent of mock and conformance runs.
- **SC-002**: Existing scorer tests and conformance pass unchanged for requests without a session.
- **SC-003**: `buf breaking` against `main` reports no breaking change.
- **SC-004**: Validators stay at 0 allocs/op on valid requests and responses without a session.

## Assumptions

- Hosts, not scorers, choose session ids and keys.
- Scorers can compute a duplicate key for a recommendation without calling another service.
- Spec 591 (issue 573, scorer advertised limits) is landing separately and edits the same files;
  this feature adds fields and rules without rewording its text, and the merge is resolved by hand.
