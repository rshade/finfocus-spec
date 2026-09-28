# Feature Specification: Projected Cost Breakdown

**Feature Branch**: `053-projected-cost-breakdown`

**Created**: 2026-09-27

**Status**: Draft

**Input**: User description: "GitHub issue #433 — Add CostBreakdown map field to GetProjectedCostResponse to
provide structured cost component reporting alongside the existing cost_per_month total."

## Clarifications

### Session 2026-09-27

- Q: When a breakdown does not sum to `cost_per_month`, should the validator reject the response or only
  warn? → A: Reject with a hard validation error when the sum is outside the tolerance.
- Q: May breakdown components be negative (discounts, credits)? → A: No; non-negative only, with discounts
  already applied to each component.
- Q: Should component names be validated as lowercase snake_case or only recommended? → A: Enforced:
  1–64 characters of `[a-z0-9_]`, starting with a lowercase letter.
- Q: Should `EstimateCostResponse` also get the breakdown in this feature? → A: No; projected cost only.
  Record `EstimateCostResponse` as a follow-up that reuses the same rules.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Plugin reports cost components for a multi-part resource (Priority: P1)

A plugin author estimating a resource whose monthly cost combines several parts (for example, an EC2
instance plus its root storage volume) reports each part with its own monthly cost next to the total,
instead of writing the parts into the free-text billing detail.

**Why this priority**: This is the core capability the issue asks for. Without it, no consumer can
get structured component data, and the aws-public plugin keeps putting its compute and root-volume
split into a string.

**Independent Test**: A plugin returns a projected cost with a total of 8.392 and components
`compute = 7.592` and `root_volume = 0.80`. A consumer reading the response over the wire receives
both components with their exact names and values, unchanged.

**Acceptance Scenarios**:

1. **Given** a plugin that computes compute and root-volume costs, **When** it returns a projected
   cost with both components, **Then** the consumer receives the total and a breakdown whose two
   named entries match what the plugin sent.
2. **Given** a single-component resource (for example, a standalone storage volume costing 8.0/month),
   **When** the plugin reports one `storage` component, **Then** the consumer receives a breakdown
   with exactly one entry equal to the total.
3. **Given** a plugin that populates the breakdown, **When** the response passes through the SDK
   response validator, **Then** validation succeeds as long as the components are well-formed and
   add up to the total.

---

### User Story 2 - Existing plugins and consumers keep working unchanged (Priority: P1)

An operator runs an existing plugin that knows nothing about breakdowns against a host that does,
or a new plugin against an older host. Neither side breaks, and costs are reported exactly as
before.

**Why this priority**: The protocol's backward-compatibility guarantee (Constitution VI) cannot be
negotiated. A breakdown that breaks existing plugins or hosts is worse than no breakdown.

**Independent Test**: Run the existing conformance and integration suites against a plugin that
never sets the breakdown. Every test passes, and the response validator accepts the response.

**Acceptance Scenarios**:

1. **Given** a plugin that does not populate the breakdown, **When** a consumer reads the response,
   **Then** the breakdown is empty and signals "no breakdown available". It never signals "zero cost".
2. **Given** an older consumer built before this change, **When** it receives a response with a
   breakdown, **Then** it ignores the unknown data and reads the total, currency, and billing detail
   exactly as before.

---

### User Story 3 - Plugin authors get early feedback on inconsistent breakdowns (Priority: P2)

A plugin author makes a mistake: the components do not add up to the total, a value is negative or
not a number, or a component name is malformed. The SDK validator reports the problem with a clear
message before the response reaches a host.

**Why this priority**: Consumers will display and total the breakdown. If the numbers disagree with
the headline cost, users stop trusting both. Catching this inside the plugin is cheaper than
debugging it in Core.

**Independent Test**: Feed the validator responses with a mismatched sum, a negative component, a
NaN component, an empty key, and too many entries. Each one is rejected with a message naming the
field and the rule it broke.

**Acceptance Scenarios**:

1. **Given** a breakdown whose components sum to 9.00 while the total is 8.392, **When** the
   response is validated, **Then** validation fails and the message reports both values.
2. **Given** a breakdown whose components differ from the total only by rounding (within the
   documented tolerance), **When** the response is validated, **Then** validation succeeds.
3. **Given** a breakdown containing a negative, NaN, or infinite value, **When** the response is
   validated, **Then** validation fails and names the offending component.

---

### User Story 4 - Plugin authors discover and populate the breakdown easily (Priority: P3)

A plugin author building a new multi-component estimate finds the breakdown in the SDK
documentation, sees the recommended component names, and sets the breakdown with the same
option-style helper they already use for other projected-cost fields. They do the same from the
TypeScript SDK.

**Why this priority**: Ergonomics and documentation drive adoption, but the field is usable without
them.

**Independent Test**: A developer following only the SDK README can build a projected-cost response
that carries a two-component breakdown and passes validation.

**Acceptance Scenarios**:

1. **Given** the SDK documentation, **When** a plugin author looks up projected-cost fields, **Then**
   they find the breakdown's meaning, constraints, recommended names, and a worked example.
2. **Given** a TypeScript consumer, **When** it reads a projected cost response, **Then** the
   breakdown is exposed with the same names and values as in the Go SDK.

---

### Edge Cases

- **Empty breakdown**: Means "no breakdown available". It is valid with any total, including zero.
- **Zero total with components**: The components must sum to zero within the absolute tolerance
  (0.01, FR-004). For example, a free-tier resource reports `compute = 0`. A component sum above 0.01
  is a sum mismatch.
- **Single component**: Valid. The one component must equal the total within tolerance.
- **Floating-point rounding**: Components rounded to cents may not add up exactly to the total. A
  documented small tolerance absorbs this, so authors do not have to fudge values.
- **Dry-run responses**: When the response is a dry-run result (cost fields empty or zero), the
  breakdown must also be empty.
- **Currency**: Components are always in the response's `currency`. There is no per-component
  currency.
- **Unrecognized component names**: Consumers must still display, total, or pass through names they
  do not recognize. Names are an open vocabulary, not an enumeration.
- **Duplicate meaning under different names** (for example, `storage` and `root_volume` both
  describing the same disk): This is the plugin author's responsibility. The validator checks only
  structure and the sum, not what the names mean.
- **Discounts or credits**: Negative components are not allowed. Following Constitution III, plugins
  report components that already include discounts, so each component is already the net amount.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The projected-cost response MUST carry an optional breakdown that maps component names
  to monthly costs, alongside the existing total, without changing or renumbering any existing field.
- **FR-002**: An absent or empty breakdown MUST mean "no breakdown available". It MUST NOT be read
  as a zero cost, and existing plugins that never populate it MUST remain valid.
- **FR-003**: Each component value MUST be a monthly cost in the same currency as the response's
  `currency` field.
- **FR-004**: When the breakdown is non-empty, the sum of its component values MUST equal
  `cost_per_month` within a tolerance of 0.01 currency units or 0.1% of the total, whichever is
  larger. A sum outside this tolerance is a hard validation error, not a warning.
- **FR-005**: Each component value MUST be finite (not NaN or infinite) and non-negative. Discounts and
  credits MUST already be applied to the component they reduce, not reported as separate negative entries.
- **FR-006**: Component names MUST be lowercase snake_case: 1–64 characters drawn from `a-z`, `0-9`,
  and `_`, starting with a lowercase letter. The breakdown MUST contain no more than 32 entries (the
  same limit as the response `metadata` map).
- **FR-007**: The SDK response validator MUST enforce FR-004 through FR-006 and MUST return an error
  that names the violated rule and the offending component or values. Valid responses MUST keep the
  validator's existing zero-allocation happy path.
- **FR-008**: The documentation MUST publish a recommended, non-exhaustive vocabulary of component
  names: `compute`, `storage`, `root_volume`, `network`, `license`, `request`, `data_transfer`. The
  documentation MUST state the enforced name format (FR-006), and that consumers MUST tolerate
  well-formed names outside the list.
- **FR-009**: The Go SDK MUST offer an option-style helper for setting the breakdown on a
  projected-cost response, matching the existing projected-cost option helpers.
- **FR-010**: The TypeScript SDK MUST expose the breakdown with identical names and semantics
  (Constitution XIII).
- **FR-011**: When a response is a dry-run result, the breakdown MUST be empty, and the validator MUST
  reject a populated breakdown alongside a dry-run result.
- **FR-012**: The protocol's field documentation MUST state the field's meaning, units, currency
  rule, sum rule, tolerance, empty-map meaning, and constraints, in the same style as the
  neighboring `metadata` and prediction-interval fields.
- **FR-013**: The test mock plugin MUST be able to return a configurable breakdown, so that hosts and
  conformance tests can exercise the field.
- **FR-014**: The free-text `billing_detail` MUST remain unchanged and independent. Plugins MAY keep
  describing components there for human readers.

### Key Entities

- **Cost Breakdown**: An unordered set of named cost components attached to one projected-cost
  response. When present, it partitions the monthly total.
- **Cost Component**: A name (for example, `compute` or `root_volume`) paired with a non-negative
  monthly cost in the response currency. Component names are an open vocabulary with recommended
  conventions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A plugin can report a two-component estimate (compute plus root volume), and a consumer
  can read each component's value without parsing any free text. The issue's EC2 example round-trips
  exactly.
- **SC-002**: 100% of existing conformance, integration, and validation tests pass unchanged against
  plugins that do not populate the breakdown.
- **SC-003**: Each malformed-breakdown case (sum mismatch, negative, NaN/Inf, bad key, too many
  entries, populated during dry run) is rejected by the validator with a message that identifies the
  problem, verified by at least one test per case.
- **SC-004**: Validating a well-formed response, with or without a breakdown, adds no memory
  allocations. Existing validator benchmarks show no regression beyond 10%.
- **SC-005**: By following only the published documentation, a developer can populate and validate a
  breakdown with the Go SDK and read it with the TypeScript SDK. TypeScript-side validation is out of
  scope (research R9).

## Assumptions

- The aws-public plugin and FinFocus Core are the first producer and consumer. Changes to those
  repositories are follow-up work and outside the scope of this spec.
- The breakdown applies only to the projected-cost response. Extending it to actual cost or estimate
  cost responses is out of scope. Batch responses that embed projected-cost results inherit the field
  automatically.
- `EstimateCostResponse` already expects a future compute and storage breakdown (its proto comment
  says so). That is a follow-up feature, and it MUST reuse this spec's rules: the name format
  (FR-006), the sum tolerance and hard error (FR-004), and non-negative values (FR-005). This keeps the
  two RPCs consistent.
- Component values are monthly figures on the same basis as `cost_per_month` (a typical 730-hour
  month). No per-period breakdown is included.
- The sum tolerance (FR-004) is chosen to absorb per-component rounding to cents on small totals and
  float error on large totals. The planning phase may adjust the exact constants, but must keep the
  rule documented and tested.
- Negative components are excluded, following Constitution III and the existing non-negative total
  rule. If credit line items are needed later, they can be added as a separate backward-compatible
  change.
- The name format (FR-006) is validated, but the recommended vocabulary is not. New services can
  introduce well-formed names without a spec change.
- Adding a map field is a backward-compatible minor change (Constitution VI). No capability flag is
  needed, because an empty map already signals that no breakdown is available.
