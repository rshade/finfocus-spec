# Contract and Limits Requirements Checklist: Structured Attributes on ResourceDescriptor

**Purpose**: Unit-test the requirements for the proto contract, the validation limits, and documentation
completeness before implementation
**Created**: 2026-10-03
**Feature**: [spec.md](../spec.md)

**Ownership**: A reviewer owns this checklist. `[x]` means the reviewer judged that the requirement meets
the criterion. It does not mean the implementation is done. `/speckit-implement` reads these markers but
never changes them.

## Requirement Completeness

- [ ] CHK001 - Is every RPC that carries a `ResourceDescriptor` enumerated, or covered by "any RPC", so that delivery
      scope is unambiguous? [Completeness, Spec §US1]
- [ ] CHK002 - Are the host redaction categories (`__` keys, credential-like keys, secret-marked values) defined
      precisely enough for a host to implement them consistently? [Completeness, Spec §FR-003]
- [ ] CHK003 - Is the error content for an oversized `attributes` (field name, size, limit) specified for both
      validation layers? [Completeness, Spec §FR-005]
- [ ] CHK004 - Are requirements stated for how `tags` and `attributes` relate when both are present and disagree? [Gap]
- [ ] CHK005 - Is the list of documents to update complete, including the TypeScript README and CLAUDE.md/AGENTS.md?
      [Completeness, Spec §FR-013–FR-015]

## Requirement Clarity

- [ ] CHK006 - Is "encoded size" defined unambiguously as protobuf wire size of the structure, rather than JSON or
      character count? [Clarity, Spec §FR-004]
- [ ] CHK007 - Is "credential-like key" given examples or a rule, rather than left to interpretation? [Ambiguity, Spec
      §FR-003]
- [ ] CHK008 - Is the accessor's path grammar (separator, numeric segment meaning, empty segments) fully specified?
      [Clarity, Spec §FR-009]
- [ ] CHK009 - Is the unit of `MaxTagValueLength` (bytes compared with characters) stated, given that values may be
      multi-byte UTF-8? [Ambiguity, Spec §FR-007]

## Requirement Consistency

- [ ] CHK010 - Are the limits identical between the plugin SDK and testing contract layers, and do the requirements say
      both change together? [Consistency, Spec §FR-005, §FR-007]
- [ ] CHK011 - Does the updated DoS-guard description stay consistent with the contract limits ("more generous than"
      compared with "equal to")? [Consistency, Spec §FR-008]
- [ ] CHK012 - Is the new field's semantics consistent with `EstimateCostRequest.attributes` (null or empty handling)?
      [Consistency, Spec §Assumptions]
- [ ] CHK013 - Are FR-011 (conformance) and FR-012 (mock ignores attributes) consistent about which plugin proves
      backward compatibility? [Consistency]

## Acceptance Criteria Quality

- [ ] CHK014 - Are the at-limit and over-limit boundaries stated as exact numbers for both limits? [Measurability, Spec
      §SC-002]
- [ ] CHK015 - Is "no extra allocations" measurable against a named baseline? [Measurability, Spec §SC-004, §FR-017]
- [ ] CHK016 - Is the stale-docs criterion expressed as a reproducible search? [Measurability, Spec §SC-005]

## Scenario and Edge Case Coverage

- [ ] CHK017 - Are an empty structure, a null value at a path, and an absent field each given defined behavior?
      [Coverage, Spec §Edge Cases]
- [ ] CHK018 - Is numeric precision loss (integers above 2^53) addressed with host guidance? [Edge Case, Spec §Edge
      Cases]
- [ ] CHK019 - Is the batch case where each resource is within the limit but the total exceeds the transport limit
      specified, including who acts? [Coverage, Spec §FR-006]
- [ ] CHK020 - Is the behavior of an older plugin that receives `attributes` it does not know specified? [Coverage, Spec
      §US1]

## Non-Functional (Security and Compatibility)

- [ ] CHK021 - Is the "do not log verbatim" rule stated for plugins, and is the SDK's own logging behavior with respect
      to descriptors stated? [Security, Spec §FR-003]
- [ ] CHK022 - Is the additive-only and `buf breaking` requirement explicit, including that raised limits accept
      strictly more input? [Compatibility, Spec §FR-016]
- [ ] CHK023 - Is the rationale for 64 KiB and 2048 recorded, so that future changes have a baseline? [Assumption, Spec
      §Assumptions]

## Dependencies and Assumptions

- [ ] CHK024 - Is the dependency on core (rshade/finfocus#1525) for population and redaction documented as out of scope
      here? [Dependency, Spec §Assumptions]
- [ ] CHK025 - Is the "document, don't enforce" decision for batch size justified with the transport ordering it relies
      on? [Assumption, Spec §Assumptions]

## Notes

- Items reference spec sections. `[Gap]` marks a requirement that may be missing.
