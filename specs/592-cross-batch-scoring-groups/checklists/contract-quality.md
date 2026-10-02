# Contract Quality Checklist: Cross-Batch Scoring Sessions

**Purpose**: Reviewer check of requirements quality for the session contract
**Created**: 2026-10-01
**Feature**: [spec.md](../spec.md)

**Note**: `[x]` means the reviewer judged the requirements-quality criterion satisfied. It does not mean
implementation is complete. Depth: standard. Audience: PR reviewer.

## Wire Compatibility

- CHK001 - Are the new field numbers and names stated, and is it explicit that no existing field changes?
  [Completeness, Spec §FR-001, §FR-002, §FR-013]
- CHK002 - Is the behavior of a session-less request defined as unchanged for both validators and the mock?
  [Consistency, Spec §FR-013]
- CHK003 - Is the interaction with field numbers 4 and 5 in spec 591 (issue 573) recorded as a merge risk?
  [Dependency, Plan Conflict Notes]

## Session Semantics

- CHK004 - Is the `session_id` grammar (length, character set) unambiguous and testable? [Clarity, Spec §FR-001,
  §FR-007]
- CHK005 - Is "deterministic function of session id and content" bounded precisely enough to test without fixing a
  hash? [Measurability, Spec §FR-003]
- CHK006 - Is the single-member relaxation tied to a non-empty request `session_id`, not to the echo? [Ambiguity, Spec
  §FR-004, §FR-008]
- CHK007 - Is the one-key-per-session duty on the host stated with a prohibition on reuse? [Completeness, Spec §FR-005]
- CHK008 - Is the behavior for a scorer that declines sessions (empty echo) defined for hosts? [Coverage, Spec §Edge
  Cases]
- CHK009 - Is the treatment of cached items explicit about what the scorer cannot see? [Clarity, Spec §FR-006]
- CHK010 - Are `IDENTIFIER_MODE_OMITTED` and per-item failure cases defined with a session? [Coverage, Spec §Edge Cases]

## Coverage Across Deliverables

- CHK011 - Does each of validators, mock, conformance, TypeScript, and docs have a numbered requirement?
  [Completeness, Spec §FR-007 to §FR-012]
- CHK012 - Do conformance requirements avoid forcing session support on scorers that decline it? [Consistency, Spec
  §FR-010]
- CHK013 - Is the score-cache rule against caching `duplicate_group_id` restated for session ids? [Consistency, Spec
  §FR-012]

## Measurability

- CHK014 - Are success criteria measurable without naming test files? [Measurability, Spec §SC-001 to §SC-004]
- CHK015 - Is the zero-allocation expectation scoped to session-less valid input? [Clarity, Spec §SC-004]

## Notes

- `/speckit-implement` reads checklist state but does not modify markers.
