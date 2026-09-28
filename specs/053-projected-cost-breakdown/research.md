# Research: Projected Cost Breakdown

**Feature**: 053-projected-cost-breakdown | **Date**: 2026-09-27

The Technical Context had no open NEEDS CLARIFICATION items after `/speckit-clarify`. Each entry below
records a design decision that the spec left to planning, and the codebase evidence behind it.

## R1: Wire type and field number

- **Decision**: `map<string, double> cost_breakdown = 15;` on `GetProjectedCostResponse`.
- **Rationale**: Field numbers 1 to 14 are taken; `metadata = 14` is the most recent addition (#427). A
  proto3 map has no presence bit, so an empty map is the natural "no breakdown available" signal
  (FR-002). Adding a field is wire compatible, and `buf breaking` accepts it (Constitution VI).
- **Alternatives considered**:
  - `repeated CostComponent { string name; double monthly_cost; }`: this allows ordering and future
    per-component fields such as a unit or a description. It was rejected because the issue asks for
    a map, duplicate names would need their own validation rule, and nothing in the spec needs
    ordering. A later message-valued field can still be added without breaking this one.
  - Putting components in `metadata` as strings: rejected. That is the string-parsing problem the
    issue exists to remove.

## R2: Sum tolerance constants

- **Decision**: `tolerance = max(0.01, 0.001 * cost_per_month)`. The response is valid when
  `|sum(components) - cost_per_month| <= tolerance`. Both constants are unexported package-level
  values: `costBreakdownAbsTolerance = 0.01` and `costBreakdownRelTolerance = 0.001`.
- **Rationale**: 0.01 absorbs rounding each component to the cent on small totals. The issue's own
  example (7.592 + 0.80 = 8.392) sums exactly, and two components each rounded to the cent are off by
  at most 0.01. The 0.1% relative term absorbs float64 summation error and upstream rounding on large
  totals, for example 0.001 × 50,000 = 50 on a 50,000/month cluster, without making small totals
  permissive. The same shape as the existing `spotRiskEpsilon`, with constants in `validation.go`,
  keeps it consistent (Constitution X).
- **Alternatives considered**: exact equality, which the clarification rejected, since float64 sums of
  decimal values rarely match exactly. Absolute-only: fails on large totals. Relative-only: rejects
  cent rounding on totals under 10.

## R3: Summation determinism

- **Decision**: Plain `float64` accumulation while ranging over the map. No Kahan or sorted summation.
- **Rationale**: Go map iteration order is random, so the rounding error in the sum varies slightly
  between runs. With at most 32 entries, the worst-case error is about 32 × 2⁻⁵³ × total, many orders
  of magnitude below the tolerance, so the valid/invalid outcome is deterministic. Ranging over a map
  does not allocate, which keeps SC-004.
- **Alternatives considered**: sorting the keys before summing (allocates a slice and breaks the
  zero-allocation happy path), and Kahan summation (unnecessary at this scale).

## R4: Key validation without allocation

- **Decision**: A byte loop over each key, checking the first byte is `a-z` and the rest are in
  `a-z`, `0-9` or `_`, with a length of 1 to 64. Maximum 32 entries, reusing the existing
  `maxMetadataEntries` and `maxMetadataKeyLen` values through new named constants set equal to them.
- **Rationale**: The same technique as `validateMetadataEntry` (printable ASCII byte loop), so zero
  allocations. Regular expressions are avoided: they are slower and, per Constitution VIII, not used on
  validation hot paths in this SDK.
- **Alternatives considered**: `regexp.MustCompile("^[a-z][a-z0-9_]{0,63}$")`. It is correct but
  slower, and it gives a worse error position.

## R5: Error reporting

- **Decision**: New sentinel errors in `validation.go`'s error block, wrapped with `fmt.Errorf("%w: …")`
  in the same way as the metadata errors, so callers can `errors.Is` on them:
  `ErrCostBreakdownTooManyEntries`, `ErrCostBreakdownInvalidKey`, `ErrCostBreakdownInvalidValue`
  (NaN, Inf or negative), `ErrCostBreakdownSumMismatch`, `ErrCostBreakdownWithDryRun`. The response
  validator adds the `GetProjectedCostResponse:` prefix, as it does for metadata.
- **Rationale**: FR-007 requires errors that name the rule and the component. Sentinels make each
  SC-003 case testable with `require.ErrorIs`.

## R6: Validation order

- **Decision**: Check the dry-run conflict first, then the entry count, then each entry (key, then
  value) while accumulating the sum, then the sum. Call it right after `validateMetadataMap` in
  `ValidateGetProjectedCostResponse`, and extend that function's godoc rule list.
- **Rationale**: The cheapest checks come first. Checking entries before the sum means a NaN value
  reports as an invalid value, not as a confusing sum mismatch. It runs after the existing
  `cost_per_month` NaN and negative checks, so the tolerance is always computed from a valid total.

## R7: Dry-run interaction

- **Decision**: If `resp.GetDryRunResult() != nil` and `len(resp.GetCostBreakdown()) > 0`, return
  `ErrCostBreakdownWithDryRun`.
- **Rationale**: The proto already says cost fields are empty or zero when `dry_run_result` is
  populated (field 7 comment). The breakdown is a cost field, so FR-011 follows. The mock plugin's
  dry-run path already returns only `DryRunResult`, so it complies without changes.

## R8: Go SDK helper

- **Decision**: `WithProjectedCostBreakdown(breakdown map[string]float64) GetProjectedCostResponseOption`
  in `helpers.go`, placed with the other `WithProjectedCost*` options. It copies the map, and a nil
  or empty input leaves the field empty. It does **not** panic on invalid input.
- **Rationale**: The sum rule depends on `cost_per_month`, which a different option
  (`WithProjectedCostDetails`) may set before or after this one. Only the validator sees the finished
  response, so validation stays there. Copying the map stops a caller's later changes from reaching an
  already-built response.
- **Alternatives considered**: panicking like `WithProjectedCostSpotRisk`. Rejected because the sum
  check cannot be done inside the option, and checking only some rules here would split validation
  across two places.

## R9: TypeScript SDK scope

- **Decision**: Regenerate `costsource_pb.ts`, which exposes `costBreakdown: { [key: string]: number }`
  on `GetProjectedCostResponse`. Extend the msw handler and `integration.test.ts` to show that the
  field round-trips. No TypeScript validator.
- **Rationale**: `CostSourceClient.getProjectedCost` returns the generated type directly, so the field
  is exposed with no wrapper change (FR-010). The TypeScript SDK is a consumer (client) SDK, while
  producer-side validation lives in the Go plugin SDK. The #427 metadata field set this precedent:
  its TypeScript change was only regenerated bindings.
- **Alternatives considered**: a `sumCostBreakdown()` TypeScript utility. Deferred because no consumer
  has asked for it, and it is easy to add later.

## R10: Mock plugin support

- **Decision**: Add a `ProjectedCostBreakdown map[string]float64` field to `MockPlugin`. When the map
  is set, `GetProjectedCost` returns the breakdown with values scaled proportionally, so they sum to
  the computed `cost_per_month`. When the map is nil, the response is unchanged.
- **Rationale**: FR-013. Treating the configured values as weights keeps the mock's responses valid
  under the sum rule without the test needing to predict the mock's rate arithmetic. Batch projected
  results get the breakdown for free through `batchProjectedCost`.
- **Alternatives considered**: returning the configured values exactly. That makes most mock responses
  fail validation unless the test reproduces `BaseHourlyRate × multiplier × 730`.

## R11: Documentation placement

- **Decision**:
  1. Proto field comment (FR-012), in the style of the `metadata` field 14 comment.
  2. A new `## Cost Breakdown Helpers (cost_breakdown)` section in `sdk/go/pluginsdk/README.md`, right
     after `## Caching Hint Helpers (expires_at)`, with the rules, the recommended vocabulary (FR-008),
     and the EC2 and EBS examples, plus a Table of Contents entry.
  3. A pointer from `docs/` if an existing projected-cost field reference covers neighboring fields;
     otherwise the README is the source of truth.
  4. Update the `EstimateCostResponse` comment ("Future versions may add optional breakdown fields")
     so it points at `GetProjectedCostResponse.cost_breakdown` as the model to reuse (spec
     Assumptions).
- **Rationale**: Constitution VII and XIV require the docs to change in the same PR, with README
  examples that compile.

## R12: Versioning and changelog

- **Decision**: Use the conventional commit `feat(proto): add cost_breakdown map to
  GetProjectedCostResponse (#433)`. release-please produces a MINOR bump. CHANGELOG.md is not edited
  by hand.
- **Rationale**: This is a backward-compatible addition (CLAUDE.md Versioning, Constitution VI).
