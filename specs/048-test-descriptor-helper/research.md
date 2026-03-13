# Research: Extract Test Helper for ResourceDescriptor Creation

## R-001: Common ResourceDescriptor Pattern in helpers_test.go

**Decision**: The most common test pattern uses `provider="aws"`, `resourceType="ec2"` with optional
SKU, region, ID, and ARN. These are the defaults for the helper.

**Rationale**: Analysis of `helpers_test.go` shows 7 out of 14 `NewResourceDescriptor` calls use
`"aws", "ec2"` as the first two arguments. The remaining calls use different providers/types
intentionally (e.g., `"custom", "resource"` for ARN tests, `"gcp", "compute_engine"` for
composability tests) and should NOT be refactored.

**Alternatives considered**:

- Using `"test", "test:resource:Type"` as defaults — rejected because the majority of existing calls
  use `"aws", "ec2"` and changing would alter test semantics.
- Making provider/resource type configurable — rejected because the spec explicitly states the helper
  accepts ID and ARN as primary parameters, with additional options for overrides.

## R-002: Default SKU and Region Values

**Decision**: Use `sku="t3.micro"` and `region="us-east-1"` as standard test defaults.

**Rationale**: These are the values used in `TestNewResourceDescriptor_WithAllOptions` (line 1723-1724)
and represent realistic AWS test defaults. The `t3.micro` SKU is the most common test instance type
in the codebase, and `us-east-1` is the standard AWS test region.

**Alternatives considered**:

- Not setting SKU/region by default — rejected because providing richer defaults reduces boilerplate
  further and matches the common "fully populated descriptor" pattern.
- Using generic values like `"test-sku"` — rejected because realistic values make test failures easier
  to diagnose and match existing patterns.

## R-003: Empty String Handling for ID and ARN

**Decision**: Only apply `WithID(id)` when `id != ""` and `WithARN(arn)` when `arn != ""`.

**Rationale**: Per FR-004 and FR-005 in the spec. This allows callers to create descriptors with
only base defaults by passing empty strings, which maps to edge case "both empty" producing a
descriptor with only provider, resource type, SKU, and region.

**Alternatives considered**:

- Always setting ID and ARN regardless of value — rejected because it would set empty string fields
  unnecessarily, diverging from the existing pattern where `NewResourceDescriptor("aws", "ec2")`
  produces a descriptor with empty ID/ARN.

## R-004: Which Tests to Refactor

**Decision**: Refactor tests that use the standard `"aws", "ec2"` pattern with various options.
Do NOT refactor:

1. `TestNewResourceDescriptor_Basic` — tests basic construction behavior (FR-008)
2. `TestNewResourceDescriptor_WithIDOption` — tests specific WithID behavior (FR-008)
3. `TestNewResourceDescriptor_WithARNOption` — tests specific WithARN behavior (FR-008)
4. `TestNewResourceDescriptor_WithAllOptions` — tests all options together (FR-008)
5. `TestWithID` table-driven tests — tests WithID specifically with `"aws", "ec2"` (FR-008)
6. `TestWithARN` table-driven tests — uses `"custom", "resource"` (different provider)
7. `TestResourceDescriptorOptions_Composability` — tests option ordering explicitly (FR-008)
8. `FuzzResourceDescriptorID` — uses `"provider", "type"` (generic, intentional)
9. Direct `&pbc.ResourceDescriptor{}` struct literals — these use `"test"` or `"any"` providers
   and don't go through `NewResourceDescriptor` at all

**Rationale**: Per FR-008, tests that directly test `NewResourceDescriptor` construction behavior
must NOT be refactored. The remaining `NewResourceDescriptor` calls in `helpers_test.go` are all
in the ResourceDescriptor Helper Tests section and test the construction API itself.

**Conclusion**: After careful analysis, the primary value of this helper is for **future test
development** (User Story 1) rather than mass refactoring of existing tests. The existing tests
in `helpers_test.go` are intentionally verbose to test the `NewResourceDescriptor` API itself.
The helper provides immediate value for any new tests added to the file that need descriptors
with standard defaults.

**Alternatives considered**:

- Refactoring all `"aws", "ec2"` calls — rejected per FR-008 since those tests exist specifically
  to verify `NewResourceDescriptor` construction behavior.

## R-005: Helper Function Signature

**Decision**: `func newTestDescriptor(id, arn string, opts ...pluginsdk.ResourceDescriptorOption) *pbc.ResourceDescriptor`

**Rationale**: Matches the spec's FR-002 requirement. The variadic options parameter allows callers
to override defaults (FR-006). Using `pluginsdk.ResourceDescriptorOption` as the option type
maintains API consistency with the production `NewResourceDescriptor` function.

**Alternatives considered**:

- Accepting a struct of overrides — rejected because the functional options pattern is already
  established in the codebase and provides better composability.
- Returning `(*pbc.ResourceDescriptor, error)` — rejected because test helpers should be
  simple and panic-free; the underlying `NewResourceDescriptor` never errors.
