# Specification Quality Checklist: Allocator Service (Allocate)

**Purpose**: Validate specification completeness and quality before proceeding to planning

**Created**: 2026-09-25

**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- This repository's product *is* a protocol and SDK, so transports (gRPC, Connect), the TypeScript
  SDK, capability values, gRPC status codes, and the SHA-256 fingerprint are part of the user-facing
  contract rather than implementation details. This matches prior specs (049, 051). File paths,
  type signatures, and line references from #506 are left for `/speckit-plan`.
- The constitution principle III justification is in the spec Overview, as the assessment decision
  requires. Principle XIII (TypeScript client wrapper) is FR-028.
- Both markers resolved 2026-09-25 (see spec Clarifications): mixed currencies use
  invalid-argument, and an empty currency takes the others' single currency, falling back to
  `USD` when all are empty. FR-011, FR-023, and FR-024 were updated to match.
- The assessment's other open questions were resolved as informed defaults and recorded in
  Assumptions:
  - zero-total tolerance: the 1e-9 floor
  - fingerprint scope: per allocator
  - cost categories: CPU and memory now, others additive later
  - release sequencing
  - SP6: out of scope
- Spec review 2026-09-25 added FR-031 (SDK strict policy decoder with JSON paths, since Go's
  decoder omits them), FR-032 (host-facing helpers in the plugin SDK package), duplicate priced
  resource rejection, and conformance message scope; SC-001 now mentions the base plugin.
- All items complete; ready for `/speckit-plan`.
