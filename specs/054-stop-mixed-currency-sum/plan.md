# Implementation Plan: Stop Mixed-Currency Recommendation Totals

**Branch**: `054-stop-mixed-currency-sum` | **Date**: 2026-09-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/054-stop-mixed-currency-sum/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Recommendation summaries currently add savings across currencies and then blank the currency
label (100 USD + 50 EUR becomes 150). This slice withholds that grand total (0 and an empty
currency) and withholds any category or action bucket that itself mixes non-empty currencies.
Single-currency buckets keep their sums. Empty currency is not a second currency. Both copies
of the helper change together and keep their signatures. No protocol, conversion, or
per-currency subtotal response.

## Technical Context

**Language/Version**: Go 1.27.1 (go.mod)

**Primary Dependencies**: Existing `google.golang.org/protobuf` recommendation messages. No new
dependencies.

**Storage**: N/A (stateless in-memory summary)

**Testing**: `go test` on `sdk/go/pluginsdk` and `sdk/go/testing`

**Target Platform**: Library used by FinFocus plugins and hosts

**Project Type**: Go SDK library inside a protocol repository

**Performance Goals**: Stay a single pass over the recommendation page. The function already
allocates the summary maps. Do not add a second pass or a money library.

**Constraints**: No proto edit, no `make generate`, no TypeScript, no FX, no decimal migration,
no actual-cost currency field, no per-currency subtotal response. Do not change allocation's
empty-as-USD rule. Do not reject a FOCUS row whose billing currency differs from its pricing
currency. Do not merge the two summary copies (`pluginsdk` importing `testing` cycles).

**Scale/Scope**: Two functions, their comments, and the tests that lock the old 150 and 125
sums. A doc sentence only if a README or developer guide still says mixed currencies are summed
and only the label is cleared.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Proto first**: Pass. No protobuf change. The summary message already has one total and
  one currency. Withholding uses 0 and an empty currency.
- **II. Multi-provider consistency**: Pass. Currency withholding is provider-agnostic.
- **III. The spec does not calculate**: Pass. The change refuses to add unlike units. It does
  not convert, discount, or price.
- **IV. Separation of concerns**: Pass. Behavior stays in the plugin SDK helper and the test
  mock copy. No host application logic.
- **V. Test-first for gRPC changes**: Pass, not applicable as a proto change. Tests still come
  before the helper edit, because the existing tests encode the behavior being replaced.
- **VI. Protobuf compatibility**: Pass. No wire change. Call signatures stay the same.
- **VII. Documentation currency**: Pass, conditional. FR-011 corrects a sentence only when a
  README or developer guide states the old sum-and-clear behavior. Unrelated conversion
  examples and the larger roadmap item stay untouched.
- **VIII. Performance**: Pass with note. This is not new core logic. No new benchmark. The
  hot path remains one linear pass plus the maps the summary already builds.
- **IX. Observability**: Pass. No new logs or metrics. A withheld total is visible on the
  summary itself.
- **X. Established patterns**: Pass. Same duplicate-helper pattern already documented on
  `CalculateMockSummary`. Exact string compare, matching allocation's currency compare, and
  no case folding.

## Project Structure

### Documentation (this feature)

```text
specs/054-stop-mixed-currency-sum/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── recommendation-summary.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
sdk/go/pluginsdk/
├── helpers.go            # CalculateRecommendationSummary
└── helpers_test.go       # mixed-currency and same-currency cases

sdk/go/testing/
├── mock_plugin.go        # CalculateMockSummary (duplicate, import cycle)
└── mock_plugin_test.go   # "mixed currencies clears currency field"
```

**Structure Decision**: Edit only the two existing summary functions and their tests. No new
package. No `proto/`, no `sdk/typescript/`, no `sdk/go/currency/`.

## Complexity Tracking

No constitution violation requires justification. The duplicated function is an existing
import-cycle constraint, not a new pattern.

## Post-Design Re-check

Design artifacts do not add a protocol field, a subtotal map, a converter, or a shared
package. Gates above still pass. Rejected follow-ons, if a later stage proposes them: proto
or generated SDK edits, TypeScript, FX, an actual-cost currency, and per-currency subtotals.
