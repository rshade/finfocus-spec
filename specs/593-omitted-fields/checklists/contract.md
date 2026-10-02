# Contract Requirements Quality Checklist: Omitted Fields

**Purpose**: Validate the wire contract, validation rules, and documentation semantics before implementation
**Created**: 2026-10-01
**Feature**: [spec.md](../spec.md)

**Note**: Reviewer-owned. `[x]` means a reviewer judged the requirements criterion satisfied; it does not mean
implementation is done.
Depth: standard. Audience: PR reviewer.

## Wire Contract

- [ ] CHK001 Is the field number choice justified against the sibling request field 4 and response field 5?
      [Consistency, Spec Clarifications]
- [ ] CHK002 Is the behavior for old scorers (unknown field) and old hosts (no field) stated? [Completeness, Contract]
- [ ] CHK003 Is "additive only" tied to a verifiable gate (`buf breaking`)? [Measurability, Spec FR-001]

## Path Syntax and Validation

- [ ] CHK004 Is the path grammar precise enough to decide every example valid or invalid? [Clarity, Spec FR-004]
- [ ] CHK005 Is the treatment of map and scalar leaves followed by a further segment defined? [Edge Case, Data Model]
- [ ] CHK006 Are the entry and byte caps stated with the reason they are enough? [Assumption, Spec FR-005]
- [ ] CHK007 Is the error code and sentinel for each rejection defined? [Completeness, Spec US2]
- [ ] CHK008 Is the strictness on unknown paths reconciled with spec-version skew between host and scorer? [Conflict,
      Spec Clarifications]

## Scorer Semantics

- [ ] CHK009 Is "must not lower confidence" defined for each affected signal, not only `insufficient_evidence`?
      [Ambiguity, Spec FR-002]
- [ ] CHK010 Is it stated that a field absent but not listed keeps its current meaning? [Completeness, Spec FR-002]
- [ ] CHK011 Is the request-wide scope (not per recommendation) explicit, with its trade-off? [Clarity, Spec FR-002]
- [ ] CHK012 Is the relationship to `identifier_mode` (which also removes fields) defined? [Gap]

## Conformance, Performance, Docs

- [ ] CHK013 Is conformance scoped to structure, not score values, consistently across spec and plan? [Consistency,
      Spec FR-007]
- [ ] CHK014 Is the zero-allocation requirement for an empty list measurable? [Measurability, Spec FR-006]
- [ ] CHK015 Is the effect on the score cache key documented? [Gap, Plan]
- [ ] CHK016 Are all documents to update named (proto comment, docs, two READMEs, TS test)? [Completeness, Plan Files]
