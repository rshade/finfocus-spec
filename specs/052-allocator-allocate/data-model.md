# Data Model: Allocator Service (Allocate)

Wire definitions are in [contracts/allocation.proto](./contracts/allocation.proto). This page lists
entities, relationships, and the validation rules the SDK helpers enforce, each mapped to its
requirement.

## Entities

### AllocateRequest

| Field | Type | Notes |
|-------|------|-------|
| `usage` | `repeated UsageRow` | From `GetStats` (#505). Not validated by the SDK (the allocator interprets it). |
| `priced` | `repeated PricedResource` | Host output from pricing `GetStatsResponse.priceable`. |
| `policy_json` | `bytes` | Opaque to the host. Empty and `{}` are equivalent. |
| `mode` | `StatsMode` | Passed through; `UNSPECIFIED` allowed. |

### PricedResource

| Field | Type | Notes |
|-------|------|-------|
| `resource` | `ResourceDescriptor` | `tags.kind` is `node` or `cluster` (others allowed; they become cluster rows). `id` is the node name for nodes. |
| `cost` | `double` | For the normalized period; ≥ 0 and finite; 0 when `priced=false`. |
| `currency` | `string` | Empty takes the resolved currency; ignored when `priced=false`. |
| `priced` | `bool` | `false` means pricing failed; counts 0 toward conservation. |
| `note` | `string` | Why pricing failed, etc. |

**Identity**: `(resource.tags["kind"], resource.id)`, unique across all entries.

### AllocateResponse

| Field | Type | Notes |
|-------|------|-------|
| `rows` | `repeated AllocationRow` | Empty only when nothing is priced and there is no usage (or the allocator emits none). |
| `effective_policy_json` | `bytes` | Canonical, allocator-owned; never empty; an object with integer `version`. |
| `policy_digest` | `string` | Lowercase hex SHA-256 of `effective_policy_json` (64 chars). |
| `warnings` | `repeated string` | Non-fatal. |

### AllocationRow

| Field | Type | Notes |
|-------|------|-------|
| `subject` | `map<string,string>` | `kind` ∈ {`workload`, `__idle__`, `__cluster__`}; idle rows require `node`. Other keys follow the #505 vocabulary (`cluster`, `namespace`, `controller_kind`, `controller`, `pod`, `node`, `label.<k>`). |
| `cpu_cost`, `mem_cost`, `total_cost` | `double` | ≥ 0, finite. |
| `currency` | `string` | The resolved currency; never empty. |
| `note` | `string` | Why a workload's cost is zero, etc. |

**Row kinds**:

- **workload**: cost attributed to a workload subject.
- **`__idle__`**: unclaimed capacity of exactly one node. Exactly one row per successfully priced
  node, even at zero cost.
- **`__cluster__`**: shared infrastructure, such as a control plane, or any priced resource that is
  not a node. The CPU/memory split rule is waived for these rows.

### Allocation policy (allocator-owned)

This is a JSON object with a required top-level integer `version`. The host treats it as bytes. For
the reference allocator (research R9): `{"version": 1, "node_split": {"cpu_weight": 0.5}}`, where
`cpu_weight` must be in [0, 1].

### Derived values

- **Resolved currency** = `ResolveCurrency(priced)`. It is the single distinct non-empty currency
  over `priced=true` entries, or `USD` when there is none.
- **Expected total** = Σ `cost` over `priced=true` entries.
- **Actual total** = Σ `total_cost` over all rows.
- **Tolerance** `tol(x, ε) = max(ε·abs(x), 1e-9)`, with `ε = 1e-6` by default.

### Capability

`PLUGIN_CAPABILITY_ALLOCATION = 15` has the legacy key `supports_allocation`. It is inferred from
`AllocatorProvider`.

## Relationships

```text
UsageSourceService.GetStats ──► GetStatsResponse{rows: UsageRow[], priceable: ResourceDescriptor[]}
                                        │                        │ host prices each via CostSourceService
                                        │                        ▼
                                        │               PricedResource{resource, cost, currency, priced}
                                        ▼                        │
                         AllocateRequest{usage, priced, policy_json, mode}
                                        │  AllocatorService.Allocate
                                        ▼
                AllocateResponse{rows: AllocationRow[], effective_policy_json, policy_digest}
                                        │  host: CheckConservation → render
```

Joins used by allocators: a workload `UsageRow.subject["node"]` equals the node's
`PricedResource.resource.id` (`tags.kind="node"`), which equals the idle row's `subject["node"]`.

## Validation rules

### Request: `ValidateAllocateRequest` (all InvalidArgument, wraps `ErrInvalidAllocateRequest`)

| # | Rule | Source |
|---|------|--------|
| Q1 | Request is non-nil; no `priced` element is nil. | FR-024 |
| Q2 | `priced=false` ⇒ `cost == 0`. | FR-003, FR-024 |
| Q3 | `cost` is ≥ 0 and finite. | FR-024 (non-finite: R6) |
| Q4 | At most one distinct non-empty currency over `priced=true` entries (wraps `ErrMixedCurrency`). | FR-011, FR-024 |
| Q5 | No two entries share `(tags.kind, id)`. | FR-024, clarification |

### Response: `ValidateAllocateResponse(req, resp)` (wraps `ErrInvalidAllocateResponse`)

| # | Rule | Source |
|---|------|--------|
| P1 | Response non-nil; `policy_digest` and `effective_policy_json` non-empty. | FR-023 |
| P2 | Every row has `kind` ∈ {`workload`, `__idle__`, `__cluster__`}. | FR-005 |
| P3 | Idle rows carry `node`. | FR-005 |
| P4 | `cpu_cost`, `mem_cost`, `total_cost` ≥ 0 and finite. | FR-009 |
| P5 | Non-cluster rows: `abs(total − (cpu + mem)) ≤ tol(total, 1e-6)`. | FR-008 |
| P6 | Row currency non-empty and equal to the resolved currency. | FR-011 |
| P7 | Each `priced=true`, `tags.kind="node"` entry has exactly one idle row with `node == id`. | FR-010 |

### Conservation: `CheckConservation(req, resp, ε)`

| # | Rule | Source |
|---|------|--------|
| C1 | `ε` is finite and ≥ 0, else error. | R6 |
| C2 | Request currency resolves (else the `ResolveCurrency` error is returned). | FR-011 |
| C3 | All costs and totals are finite, else error. | R6 |
| C4 | `abs(actual − expected) ≤ tol(expected, ε)`, else `*ConservationError{Expected, Actual, Difference, Currency}`. | FR-007, FR-022 |

### Policy: `DecodePolicy(data, target)` (InvalidArgument, wraps `ErrInvalidPolicy`)

| # | Rule | Source |
|---|------|--------|
| D1 | Empty, whitespace-only, or `null` ⇒ no change, no error. | FR-031, FR-014 |
| D2 | Malformed JSON or trailing data ⇒ error. | FR-013, FR-031 |
| D3 | Unknown field (exact, case-sensitive) ⇒ error naming the full path (`a.b`, `a[2].b`). | FR-013, FR-031 |
| D4 | Type mismatch ⇒ error naming the path. | R5 |
| D5 | Nested objects merge field by field; maps merge by key; arrays are replaced wholesale. | FR-031 |
| D6 | Nil or non-pointer target ⇒ plain error (programming error; no status). | R5 |

The `version` check is the allocator's own, after decoding (FR-013).

## State transitions

None. `Allocate` is a stateless function of its request. The only cross-call property is digest
stability: an identical effective policy yields an identical digest (FR-015).
