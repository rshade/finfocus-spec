# API Contract Requirements Checklist: Allocation Row Provenance

**Purpose**: Reviewer-owned requirements-quality check for proto compatibility, validator rules, SDK parity, docs
**Created**: 2026-10-01
**Feature**: [spec.md](../spec.md)

**Note**: `[x]` means the reviewer judged the requirement well written. It does not mean work is done.

## Proto Contract Compatibility

- [ ] CHK001 - Are the three new field names, types, and numbers specified unambiguously? [Clarity, Plan §Summary]
- [ ] CHK002 - Is the additive-only guarantee stated with a measurable compatibility check? [Measurability, Spec
  §FR-002, SC-004]
- [ ] CHK003 - Is the treatment of the held field 10 (comment, not reserved) justified and consistent with prior
  specs? [Consistency, Research §Decision 2]
- [ ] CHK004 - Is the exclusion of request-side and response-level fields explicit? [Scope, Spec §FR-010]
- [ ] CHK005 - Is behavior for older hosts that ignore the fields and older allocators that never set them defined?
  [Coverage, Spec §Edge Cases]

## Validator Rules

- [ ] CHK006 - Is the method-needs-source rule stated for every combination of the three values? [Completeness, Spec
  §FR-003, FR-004]
- [ ] CHK007 - Is the error category and the content of the error message (row index, field) specified? [Clarity, Spec
  §FR-003]
- [ ] CHK008 - Does the spec say whether the rule applies equally to workload, idle, and cluster rows? [Coverage, Spec
  §Edge Cases]
- [ ] CHK009 - Is it explicit that provenance does not alter conservation, totals, or currency rules? [Consistency,
  Spec §FR-005]
- [ ] CHK010 - Is whitespace-only handling defined? [Edge Case, Spec §Edge Cases]
- [ ] CHK011 - Is the decision not to cross-check the source id against the request documented with rationale?
  [Assumption, Spec §Clarifications]

## SDK Parity

- [ ] CHK012 - Are the Go validator, conformance suite, and reference allocator each given a testable requirement?
  [Completeness, Spec §FR-003, FR-006, FR-007]
- [ ] CHK013 - Is the TypeScript expectation defined beyond regenerated bindings? [Clarity, Spec §FR-008]
- [ ] CHK014 - Is it stated that the conformance suite never requires an allocator to emit provenance? [Clarity,
  Research §Decision 5]
- [ ] CHK015 - Is a performance bound for the validator expressed in measurable terms? [Measurability, Spec §SC-005]

## Documentation

- [ ] CHK016 - Are the FOCUS 1.3 column mappings required in the field documentation? [Completeness, Spec §FR-009]
- [ ] CHK017 - Is the deferral of the lineage chain stated where readers will find it? [Gap, Spec §FR-009]
- [ ] CHK018 - Are the docs to update (allocator guide, SDK README) enumerated? [Traceability, Plan §Project Structure]

## Dependencies and Coupling

- [ ] CHK019 - Is the shared-file overlap with issue #579 recorded with a resolution rule? [Dependency, Spec
  §Assumptions, Research §Decision 2]
- [ ] CHK020 - Are the reference allocator's chosen values (method id, source id per row kind) specified? [Gap,
  Research §Decision 6]

## Notes

- `/speckit-implement` reads checklist state but does not modify markers.
