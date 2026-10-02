# Allocator Service

`AllocatorService` (`proto/finfocus/v1/allocation.proto`) is the contract for **allocators**:
plugins that divide priced infrastructure, such as Kubernetes nodes and control planes, across the
workloads that use it. An allocator reports unclaimed capacity as idle cost and shared
infrastructure as cluster cost.

This page is the canonical reference for what an `Allocate` request and response mean. The proto
comments summarize it, and the SDK helpers enforce it.

## How It Fits Together

```text
host ──GetStats──▶ usage source          usage rows + priceable resources      (usage-source.md)
host ──cost RPCs─▶ cost-source plugins   a price for each priceable resource
host ──Allocate──▶ allocator             allocation rows + effective policy + digest
host              CheckConservation, then render
```

- **Usage sources** report how much of each node each workload requests. See
  [usage-source.md](usage-source.md).
- **Cost sources** price each priceable resource through the existing `CostSourceService` RPCs.
- **Allocators** own the math. The host carries no allocation semantics: it forwards usage, prices,
  and an opaque policy, then verifies the invariants below.

The specification fixes the **shape** of the contract and its invariants, not an allocation
algorithm (constitution principle III). Two conforming allocators may split the same cluster
differently; both must conserve cost.

## Request

| Field | Meaning |
| ----- | ------- |
| `usage` | Usage rows exactly as `UsageSourceService.GetStats` returned them. The SDK does not validate them; the allocator interprets them. |
| `priced` | One `PricedResource` per priceable resource, after the host priced it. |
| `policy_json` | Allocator-owned JSON policy, opaque to the host. Empty and `{}` both mean the allocator's defaults. |
| `mode` | The mode the usage was collected in. `STATS_MODE_UNSPECIFIED` is allowed, because allocation uses ratios of usage to capacity. |

## Priced Resources

| Field | Meaning |
| ----- | ------- |
| `resource` | The `ResourceDescriptor` from `GetStatsResponse.priceable`. Nodes carry `tags.kind = "node"` and an `id` equal to their `node` subject value; a control plane carries `tags.kind = "cluster"`. |
| `cost` | Cost for the normalized period; finite and non-negative. |
| `currency` | ISO 4217 code; see [Currency](#currency). |
| `priced` | `true` when pricing succeeded. |
| `note` | Human-readable note, such as why pricing failed. |

Rules, enforced by `ValidateAllocateRequest`:

- **Unpriced means zero.** An entry with `priced = false` has `cost = 0`, and it counts 0 toward
  conservation. Its workloads still appear, at zero cost, usually with a note.
- **Unique identity.** No two entries share `(resource.tags["kind"], resource.id)`. A node `n1` and
  a cluster `n1` are distinct.
- **Priced nodes are named.** An entry with `priced = true` and `tags["kind"] = "node"` has a
  non-empty `resource.id`, because its `__idle__` row is keyed by that id.
- **Finite, non-negative cost.** NaN, infinite, or negative costs are rejected.

### Currency

Across entries with `priced = true` there is exactly one **resolved currency**:

- the single distinct non-empty `currency`, or
- `USD` when every priced entry leaves it empty.

An empty value takes the resolved currency. Unpriced entries are ignored. Currencies compare
exactly, so `usd` and `USD` are two currencies.

| Priced entries' currencies | Result |
| -------------------------- | ------ |
| `USD`, empty, `USD` | `USD` |
| empty, empty | `USD` |
| `EUR` | `EUR` |
| `USD`, empty, `EUR` | `INVALID_ARGUMENT` (mixed currencies) |

The `USD` fallback applies only within this contract. `ResolveCurrency` implements the rule.

### Node Examples

AWS:

```json
{
  "resource": {
    "id": "ip-10-0-1-5.ec2.internal",
    "provider": "aws",
    "resource_type": "aws:ec2/instance:Instance",
    "sku": "m5.large",
    "region": "us-east-1",
    "tags": { "kind": "node", "capacity_type": "on-demand" }
  },
  "cost": 0.096,
  "currency": "USD",
  "priced": true
}
```

Azure:

```json
{
  "resource": {
    "id": "aks-nodepool1-12345678-vmss000000",
    "provider": "azure",
    "resource_type": "azure-native:compute:VirtualMachineScaleSetVM",
    "sku": "Standard_D4s_v5",
    "region": "eastus",
    "tags": { "kind": "node", "capacity_type": "spot" }
  },
  "cost": 0.0384,
  "currency": "USD",
  "priced": true
}
```

GCP:

```json
{
  "resource": {
    "id": "gke-prod-default-pool-1a2b3c4d-x9yz",
    "provider": "gcp",
    "resource_type": "gcp:compute/instance:Instance",
    "sku": "e2-standard-4",
    "region": "us-central1",
    "tags": { "kind": "node", "capacity_type": "on-demand" }
  },
  "cost": 0.134,
  "currency": "USD",
  "priced": true
}
```

An unpriced node, for example one whose SKU no cost source recognizes:

```json
{
  "resource": {
    "id": "fargate-ip-10-0-9-12",
    "provider": "aws",
    "tags": { "kind": "node" }
  },
  "cost": 0,
  "priced": false,
  "note": "no cost source supports this node"
}
```

## Response

| Field | Meaning |
| ----- | ------- |
| `rows` | Allocation rows. Empty only when there is nothing to allocate. |
| `effective_policy_json` | The policy actually applied (defaults with overrides) in the allocator's canonical JSON form. Never empty. |
| `policy_digest` | Lowercase hex SHA-256 of `effective_policy_json` (64 characters). |
| `warnings` | Human-readable, non-fatal notes. |

A request with no usage and no priced resources succeeds with no rows, but it still returns the
effective policy and digest, so hosts can display the policy.

### Rows

Each `AllocationRow` has `subject`, `cpu_cost`, `mem_cost`, `total_cost`, `currency`, and `note`, plus
optional provenance fields (see [Row provenance](#row-provenance)).
The `subject` uses the [usage-source subject keys](usage-source.md#subject-keys); `kind` is
required.

| Kind | Go constant | Meaning |
| ---- | ----------- | ------- |
| `workload` | `KindWorkload` | Cost attributed to a workload. Other subject keys identify it (`namespace`, `pod`, `node`, ...). |
| `__idle__` | `KindIdle` | Unclaimed capacity of exactly one node. Requires the `node` key. |
| `__cluster__` | `KindCluster` | Shared infrastructure, such as a control plane, or any priced resource that is not a node. |

### Row provenance

Three optional fields let a host fill the FOCUS 1.3 split-cost columns without guessing. They are
opaque strings the allocator chooses.

| Field | FOCUS 1.3 column | Meaning |
| ----- | ---------------- | ------- |
| `allocated_method_id` (7) | `AllocatedMethodId` | Identifies the method that produced the row. |
| `allocated_method_details` (8) | `AllocatedMethodDetails` | Free-form description of how the cost was split. Allowed without a method id. |
| `allocated_resource_id` (9) | `AllocatedResourceId` | The priced resource the cost came from. By convention the `resource.id` of the `PricedResource` (a node id for workload and idle rows). Not cross-checked against the request. |

Provenance never affects conservation, portion totals, or currency. Allocators that set none of
the three stay valid, and the conformance suite never requires them. The reference allocator
reports method id `refalloc.proportional` and the node (or priced resource) id on every row.
A lineage chain (`LineageNode`) is a possible later addition; field 10 is held for it.

## Invariants

Every response satisfies all six. Hosts verify them and reject a response that violates any.

1. **Conservation.** `Σ rows.total_cost` equals `Σ priced.cost` over entries with `priced = true`,
   within `max(1e-6 × |expected|, 1e-9)`. The absolute floor lets zero totals compare equal.
2. **Portions add up.** Every row has `total_cost = cpu_cost + mem_cost` within
   `max(1e-6 × |total_cost|, 1e-9)`, except `__cluster__` rows, which may carry all cost in
   `total_cost` with zero portions.
3. **No negative cost.** `cpu_cost`, `mem_cost`, and `total_cost` are finite and non-negative.
4. **One idle row per priced node.** Every entry with `priced = true` and `tags.kind = "node"` has
   exactly one `__idle__` row whose `node` equals its `id`, even when the idle cost is zero.
   Unpriced nodes need none.
5. **One currency.** Every row carries the resolved currency, never empty.
6. **Provenance is consistent.** A row with a non-empty `allocated_method_id` also has a non-empty
   `allocated_resource_id`. Rows with no provenance, or with only a resource id or method details,
   are valid. See [Row provenance](#row-provenance).

`CheckConservation` checks invariant 1, and `ValidateAllocateResponse` checks the others plus the
presence of the effective policy and digest. Both fail closed on NaN and infinite values: in IEEE
arithmetic `NaN > tolerance` is false, so an unguarded check would pass a NaN total.

## Policy

The policy is a JSON object owned by the allocator. The host treats it as bytes.

- Every allocator's schema has a required top-level integer `version`.
- **Strict decoding.** Malformed JSON, trailing data, an unknown field, or a type mismatch is
  rejected with `INVALID_ARGUMENT`. The message names the field's JSON path, for example
  `node_split.cpu` or `rules[2].match`. Field names match exactly (case-sensitive).
- An unknown or unsupported `version` is rejected with `INVALID_ARGUMENT`.
- **Never fall back to defaults** on a bad policy. A typo must fail loudly, not silently change
  results.
- **Empty equals `{}`.** Both mean "the defaults" and yield identical effective policy and digest.
- **Overrides merge.** Nested objects merge field by field and maps merge by key, including object
  values under an existing key; arrays replace the default wholesale.
- **Digests are per allocator.** The same effective policy always yields the same digest from the
  same allocator. Digests from different allocators are not comparable.

`pluginsdk.DecodePolicy` implements every decoding rule; the version check is the allocator's own.

## Errors

| Code | When |
| ---- | ---- |
| `INVALID_ARGUMENT` | An unpriced entry with nonzero cost; a negative or non-finite cost; mixed currencies; a duplicate `(tags.kind, id)`; a malformed policy, trailing data, an unknown field (message names its path), or an unknown version |

The SDK's validation errors carry `codes.InvalidArgument` themselves and have no `rpc error:`
prefix, so an allocator can return them directly. `pluginsdk.Serve` delivers the same code and
message over gRPC and Connect.

## Worked Example

One node `n1` costs 10 USD and has 4 allocatable cores and 16 GiB. Two workloads each request 1
core and 4 GiB. The reference allocator's default policy is
`{"version":1,"node_split":{"cpu_weight":0.5}}`.

| Step | CPU | Memory |
| ---- | --- | ------ |
| Pool (`cost × cpu_weight`, remainder to memory) | 5 | 5 |
| Each workload (`pool × request / allocatable`) | 5 × 1/4 = 1.25 | 5 × 4/16 = 1.25 |
| Idle (`pool − Σ shares`) | 5 − 2.5 = 2.5 | 5 − 2.5 = 2.5 |

| Row | cpu_cost | mem_cost | total_cost |
| --- | -------- | -------- | ---------- |
| workload `a` | 1.25 | 1.25 | 2.50 |
| workload `b` | 1.25 | 1.25 | 2.50 |
| `__idle__` `n1` | 2.50 | 2.50 | 5.00 |
| **Total** | | | **10.00** |

When requests exceed allocatable, the reference allocator scales shares by
`allocatable / Σ requests`, so idle stays non-negative. Other allocators may choose other math;
the invariants are what matter.

## Capabilities

A plugin that implements `pluginsdk.AllocatorProvider` has `PLUGIN_CAPABILITY_ALLOCATION` (15)
inferred, and legacy hosts see `supports_allocation=true`. Inference also always reports the four
pricing capabilities implied by the `Plugin` interface, so:

- An **allocation-only** plugin must declare
  `WithCapabilities(PLUGIN_CAPABILITY_ALLOCATION)` explicitly. **Hosts can tell an allocation-only
  plugin apart from a pricing plugin only when its capabilities are explicit.**
- `Serve` logs one warning at startup for an allocator that has no explicit capabilities and does
  not implement `PluginInfoProvider`.

In gRPC mode, `ServeConfig.UnaryInterceptors` and the SDK tracing interceptor wrap `Allocate`. In
Connect mode, the health check reports `finfocus.v1.AllocatorService` as `SERVING` only when the
plugin implements `Allocate`.

## Host Verification

Hosts verify every response from production code, without importing test tooling:

```go
resp, err := client.Allocate(ctx, req)
if err != nil {
    return err
}
if err := pluginsdk.ValidateAllocateResponse(req, resp); err != nil {
    // For example: priced[0]: node "n1" has 0 "__idle__" rows, want exactly 1
    return fmt.Errorf("allocator broke an invariant: %w", err)
}
if err := pluginsdk.CheckConservation(req, resp, pluginsdk.DefaultConservationEpsilon); err != nil {
    // For example: allocation rows total 10.01 USD, expected 10 USD (difference +0.01)
    return fmt.Errorf("allocator broke conservation: %w", err)
}
```

A mismatch is a `*ConservationError` from `sdk/go/testing` with `Expected`, `Actual`,
`Difference`, and `Currency` fields, for hosts that want to display them separately.

## Self-Checks for Authors

- `pluginsdk.ValidateAllocateRequest`, `pluginsdk.DecodePolicy`, and `pluginsdk.ResolveCurrency`:
  call them in this order at the top of `Allocate`.
- `plugintesting.RunAllocatorConformance`: thirteen policy-agnostic scenarios over an in-memory
  connection. See [sdk/go/testing/README.md](../sdk/go/testing/README.md#allocator-conformance).
- The [Plugin Developer Guide](../PLUGIN_DEVELOPER_GUIDE.md#allocator-plugins) walks through a
  complete allocator.
