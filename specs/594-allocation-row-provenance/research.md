# Research: Allocation Row Provenance

## Decision 1: Source resource id, not LineageNode

- Decision: add `allocated_resource_id` (string); do not add `LineageNode` now.
- Rationale: mirrors FOCUS 1.3 `AllocatedResourceId` and `FocusCostRecord` field 63; avoids a recursive
  message on a high-volume row. The `LineageNode` chain (costsource.proto) is documented as a host
  pass-through for cost records, a different job.
- Alternatives: `LineageNode lineage` (heavier, no FOCUS column needs it); both (scope creep).

## Decision 2: Field numbers 7, 8, 9; 10 held by comment

- Decision: next free numbers on `AllocationRow` (1-6 used). Field 10 is held by comment, not `reserved`,
  because a `reserved` statement would need a `buf breaking` exception to undo (precedent: 557-price-options).
- Coupling: issue #579 (live claim, spec 588) edits `allocation.proto` too. Its proposed changes are on
  `AllocateRequest` (period, selection signal) and `AllocateResponse` (echo), so row fields 7-9 should not
  collide. If #579 adds row fields, the second PR to merge renumbers.

## Decision 3: Rule

- Decision: method id non-empty requires resource id non-empty (FOCUS 1.3 conditional). Details free-form.
- Rationale: identical to `validateFocusRules`' allocation consistency; the issue states it.

## Decision 4: Error shape

- Decision: reuse `ErrInvalidAllocateResponse` with the existing `fmt.Errorf` wrap inside
  `validateAllocationRow` (not the public validator), message `rows[i]: allocated_method_id requires
  allocated_resource_id`.
- Rationale: matches neighboring row checks; keeps `ValidateAllocateResponse` frame unchanged.

## Decision 5: Conformance

- Decision: add scenario `row_provenance` (single-node fixture) that checks the rule on the allocator's own
  output with a dedicated message, so a broken allocator fails by name. The suite stays model-agnostic: it
  never requires an allocator to emit provenance.
- Alternatives: rely on `allocateAndVerify` only (error is generic, not named).

## Decision 6: Reference allocator values

- Decision: method id `"refalloc.proportional"`, resource id the node id for workload and idle rows, the
  priced resource id for cluster rows.
