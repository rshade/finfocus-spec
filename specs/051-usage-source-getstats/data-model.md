# Data Model: Usage Source Service (GetStats)

**Feature**: 051-usage-source-getstats | **Proto**: [contracts/usage.proto](./contracts/usage.proto)

## Entities

### GetStatsRequest

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `scope` | string (1) | Source-defined | Cluster ID, kubeconfig context, or similar. Empty means the source default. |
| `start` | Timestamp (2) | With `end` | Both unset → run-rate. Both set → historical. Exactly one set → `InvalidArgument`. |
| `end` | Timestamp (3) | With `start` | `start > end` → `InvalidArgument`. |
| `selector` | map<string,string> (4) | No | `namespace` is a reserved key: it restricts workloads to that namespace. Every other key is an exact-match label filter. All entries must match (AND). |
| `metrics` | repeated string (5) | No | Empty → source defaults. Unknown names are ignored and added to `warnings`. |

### GetStatsResponse

| Field | Type | Notes |
|-------|------|-------|
| `rows` | repeated UsageRow (1) | May be empty (empty cluster or selector matched nothing). |
| `priceable` | repeated ResourceDescriptor (2) | Nodes (`tags.kind=node`) and control planes (`tags.kind=cluster`). |
| `mode` | StatsMode (3) | The mode actually served. Must not be `UNSPECIFIED`. |
| `warnings` | repeated string (4) | Human-readable, non-fatal (for example, unknown metric requested). |

### UsageRow

| Field | Type | Notes |
|-------|------|-------|
| `subject` | map<string,string> (1) | Must contain `kind`. The other keys come from the vocabulary below or have the form `label.<key>`. |
| `metric` | string (2) | Documented names below. Custom names are allowed. |
| `amount` | double (3) | Finite and ≥ 0. A per-hour rate (run-rate) or integrated resource-hours (historical), for **every** row including allocatable. |
| `unit` | string (4) | `core`, `GiB` (run-rate); `core-hours`, `GiB-hours` (historical). Custom units are allowed. |

**Identity**: (subject map, metric) is unique within a response. Sources aggregate per-container data
before responding.

### StatsMode (enum)

| Value | Number | Meaning |
|-------|--------|---------|
| `STATS_MODE_UNSPECIFIED` | 0 | Invalid in responses. |
| `STATS_MODE_RUN_RATE` | 1 | Point-in-time per-hour rates. |
| `STATS_MODE_HISTORICAL` | 2 | Integrated over `[start, end]`. |

### Priceable resource (existing `ResourceDescriptor`, no schema change)

| Kind | Required shape |
|------|----------------|
| Node | `tags.kind = "node"`, `id` = node name (must equal a row's `node` subject value), `tags.provider_id`, `tags.capacity_type ∈ {spot, on-demand}`, plus the usual `provider`/`resource_type`/`sku`/`region`. |
| Control plane | `tags.kind = "cluster"`, `id` non-empty. No `node` subject match required. |

Every priceable entry needs a non-empty `id`.

### PluginCapability (existing enum, +1 value)

| Value | Number | Legacy metadata key |
|-------|--------|---------------------|
| `PLUGIN_CAPABILITY_USAGE_STATS` | 14 | `supports_usage_stats` |

`IsValidCapability` upper bound: 13 → 14 (→ 15 when #506 lands).

## Vocabulary (Go constants in `pluginsdk/subjects.go`, mirrored in TS)

| Group | Constant | Value |
|-------|----------|-------|
| Subject | `SubjectCluster` | `cluster` |
| Subject | `SubjectNamespace` | `namespace` |
| Subject | `SubjectControllerKind` | `controller_kind` |
| Subject | `SubjectController` | `controller` |
| Subject | `SubjectPod` | `pod` |
| Subject | `SubjectNode` | `node` |
| Subject | `SubjectKind` | `kind` |
| Subject | `SubjectLabelPrefix` | `label.` |
| Kind | `KindWorkload` | `workload` |
| Kind | `KindNode` | `node` |
| Kind | `KindIdle` | `__idle__` (allocator output only; invalid in usage rows) |
| Kind | `KindCluster` | `__cluster__` (allocator output only; invalid in usage rows) |
| Metric | `MetricCPURequest` | `cpu_request` (workload) |
| Metric | `MetricMemRequest` | `mem_request` (workload) |
| Metric | `MetricCPUUsage` | `cpu_usage` (workload, optional) |
| Metric | `MetricMemUsage` | `mem_usage` (workload, optional) |
| Metric | `MetricCPUAllocatable` | `cpu_allocatable` (node) |
| Metric | `MetricMemAllocatable` | `mem_allocatable` (node) |
| Unit | `UnitCore` / `UnitGiB` | `core` / `GiB` |
| Unit | `UnitCoreHours` / `UnitGiBHours` | `core-hours` / `GiB-hours` |

## Request validation rules (`ValidateGetStatsRequest`, first violation wins)

Lives in `sdk/go/testing/contract.go` next to the other `Validate*Request` helpers (research R11).

| # | Rule | Source | Reject example | Accept example |
|---|------|--------|----------------|----------------|
| Q1 | Request non-nil (`ErrNilRequest`) | FR-020 | `nil` | any request |
| Q2 | `start` and `end` set together or not at all (`ErrNilStartTime` / `ErrNilEndTime`) | FR-008, FR-020 | only `start`, only `end` | neither (run-rate), both (historical) |
| Q3 | `start` not after `end` (`ErrInvertedStatsWindow`) | FR-008, FR-020 | `start` = 12:00, `end` = 11:00 | `start` = 11:00, `end` = 12:00 |
| — | `scope`, `selector`, `metrics` unchecked | FR-002 | — | empty scope, unknown metric |

## Validation rules (`ValidateStatsResponse`, first violation wins)

| # | Rule | Source | Reject example | Accept example |
|---|------|--------|----------------|----------------|
| V1 | Response non-nil | — | `nil` | any response |
| V2 | `mode ≠ UNSPECIFIED` | FR-015, edge case | mode 0 | `RUN_RATE`, empty rows |
| V3 | Row has `kind` | FR-004/015 | `{pod: a}` | `{kind: workload, pod: a}` |
| V4 | `kind ∈ {workload, node}` | FR-015 | `kind: __idle__`, `kind: pvc` | `kind: node` |
| V10 | `kind=node` row has a non-empty `node` key (checked right after V4) | FR-015 | `{kind: node}` | `{kind: node, node: n1}` |
| V5 | Subject keys known or `label.<non-empty>` | FR-015 | `namespcae`, `label.` | `label.app.kubernetes.io/name` |
| V6 | Amount finite and ≥ 0 | FR-004/015 | `-1`, `NaN`, `+Inf` | `0` |
| V7 | (subject, metric) unique | FR-004/015 | two identical rows | same subject, different metric |
| V8 | Priceable non-nil with non-empty `id` | FR-015 | `id: ""` | `id: ip-10-0-1-5` |
| V9 | Priceable `kind=node` → `id` ∈ row `node` values | FR-015 | node `n2` with no rows | node `n1` with rows |
| — | Node rows without priceable entry | FR-016 | — | Fargate node capacity rows only |
| — | `kind=cluster` priceable | edge case | — | control plane, no `node` match |

## State and relationships

- Stateless: one request produces one response, and nothing is persisted.
- Join key: a priceable node's `id` equals the `node` subject value (a 1-to-many relation from the
  priceable node to the workload and allocatable rows on that node). The reverse is optional.
- The capability is advertised via `GetPluginInfo`. It is inferred from `UsageSourceProvider`
  unless `PluginInfo.Capabilities` is set explicitly.
