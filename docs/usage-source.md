# Usage Source Service

`UsageSourceService` (`proto/finfocus/v1/usage.proto`) is the contract for **usage sources**:
plugins that report how much CPU and memory Kubernetes workloads request or consume. A usage source
is backed by the Kubernetes API, Prometheus, Datadog, or a similar system, and it carries no prices.

This page is the canonical reference for what a `GetStats` response means. The proto comments
summarize it, and the SDK helpers enforce it.

## How It Fits Together

```text
host ──GetStats──▶ usage source          rows (per workload / per node) + priceable resources
host ──cost RPCs─▶ cost-source plugins   prices for each priceable resource
host              joins rows to prices on the node name, then allocates cost (#506)
```

- **Usage sources** answer "how much of each node does each workload use?"
- **Cost sources** answer "what does each node cost?" through the existing `CostSourceService`
  RPCs, using the `ResourceDescriptor`s that the usage source returned in `priceable`.
- **Allocation**, meaning the split of node cost across workloads plus idle and cluster overhead,
  is a separate host-side step tracked in #506. Usage sources never do cost math.

## Request

| Field | Meaning |
| ----- | ------- |
| `scope` | Source-defined scope, such as a cluster ID or kubeconfig context. Empty means the source's default. |
| `start`, `end` | Historical window. Leave both unset for run-rate. Setting exactly one is `INVALID_ARGUMENT`, and so is `start` after `end`. |
| `selector` | Workload filter; every entry must match. |
| `metrics` | Metric names to return. Empty means the source's defaults. Unknown names are ignored and reported in `warnings`. |

**Selector.** `namespace` is a reserved key that restricts workloads to that namespace. Every other
key is an exact-match label filter. A label literally named `namespace` therefore cannot be
selected. A selector that matches nothing is not an error; the response simply has no rows.

## Modes

| Mode | When | Amount means | Units |
| ---- | ---- | ------------ | ----- |
| `STATS_MODE_RUN_RATE` | `start` and `end` unset | Point-in-time per-hour rate | `core`, `GiB` |
| `STATS_MODE_HISTORICAL` | Both set | Resource-hours integrated over `[start, end]` | `core-hours`, `GiB-hours` |

The response `mode` is the mode actually served and is never `STATS_MODE_UNSPECIFIED`. In
historical mode **every** amount is integrated, including node `cpu_allocatable` and
`mem_allocatable`. A node that ran for half the window reports half its capacity in resource-hours.
A source that only supports run-rate returns `INVALID_ARGUMENT` for a historical window.

## Rows

Each `UsageRow` is one measurement for one subject: `subject` (string map), `metric`, `amount`,
`unit`.

### Subject Keys

| Key | Go constant | Meaning |
| --- | ----------- | ------- |
| `kind` | `SubjectKind` | **Required.** `workload` or `node` |
| `cluster` | `SubjectCluster` | Cluster the row belongs to |
| `namespace` | `SubjectNamespace` | Workload namespace |
| `controller_kind` | `SubjectControllerKind` | Owning controller kind, such as `Deployment` |
| `controller` | `SubjectController` | Owning controller name |
| `pod` | `SubjectPod` | Pod name |
| `node` | `SubjectNode` | Node name; **required** on `kind=node` rows and the join key to priceable nodes |
| `label.<key>` | `SubjectLabelPrefix` | Workload label, such as `label.app.kubernetes.io/name` |

Any other key is invalid. The key `label.` with nothing after the prefix is also invalid.

### Kinds

| Kind | Go constant | Use |
| ---- | ----------- | --- |
| `workload` | `KindWorkload` | A workload's requests or usage |
| `node` | `KindNode` | A node's allocatable capacity |
| `__idle__` | `KindIdle` | Allocator output only (#506). **Invalid in usage rows.** |
| `__cluster__` | `KindCluster` | Allocator output only (#506). **Invalid in usage rows.** |

### Metrics

| Metric | Go constant | Row kind | Notes |
| ------ | ----------- | -------- | ----- |
| `cpu_request` | `MetricCPURequest` | workload | Requested CPU |
| `mem_request` | `MetricMemRequest` | workload | Requested memory |
| `cpu_usage` | `MetricCPUUsage` | workload | Consumed CPU; optional, needs a metrics backend |
| `mem_usage` | `MetricMemUsage` | workload | Consumed memory; optional |
| `cpu_allocatable` | `MetricCPUAllocatable` | node | Allocatable CPU |
| `mem_allocatable` | `MetricMemAllocatable` | node | Allocatable memory |

Custom metric names and units are allowed. Hosts ignore what they do not understand.

### Units

| Unit | Go constant | Mode |
| ---- | ----------- | ---- |
| `core` | `UnitCore` | Run-rate CPU |
| `GiB` | `UnitGiB` | Run-rate memory |
| `core-hours` | `UnitCoreHours` | Historical CPU |
| `GiB-hours` | `UnitGiBHours` | Historical memory |

TypeScript mirrors every constant in `@rshade/finfocus-client` (`SUBJECT_*`, `KIND_*`, `METRIC_*`,
`UNIT_*`).

### Amounts and Duplicates

Amounts are finite and non-negative; `0` is valid. A (subject, metric) pair appears **at most
once** per response. A pod with several containers is reported as one row per metric, so the
source aggregates per-container data before responding.

## Priceable Resources

`priceable` lists the resources the host should price, as ordinary `ResourceDescriptor`s. Every
entry has a non-empty `id`.

- **Nodes** carry `tags.kind = "node"`, `id` equal to the node's `node` subject value,
  `tags.provider_id`, and `tags.capacity_type` (`spot` or `on-demand`), plus the usual `provider`,
  `resource_type`, `sku`, and `region`.
- **Control planes** carry `tags.kind = "cluster"`. No `node` subject match is required.

### AWS Node

```json
{
  "id": "ip-10-0-1-5.ec2.internal",
  "provider": "aws",
  "resource_type": "aws:ec2/instance:Instance",
  "sku": "m5.large",
  "region": "us-east-1",
  "tags": {
    "kind": "node",
    "provider_id": "aws:///us-east-1a/i-0a1b2c3d4e5f60001",
    "capacity_type": "on-demand"
  }
}
```

### Azure Node

```json
{
  "id": "aks-nodepool1-12345678-vmss000000",
  "provider": "azure",
  "resource_type": "azure-native:compute:VirtualMachineScaleSetVM",
  "sku": "Standard_D4s_v5",
  "region": "eastus",
  "tags": {
    "kind": "node",
    "provider_id": "azure:///subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.Compute/virtualMachineScaleSets/aks-nodepool1-12345678-vmss/virtualMachines/0",
    "capacity_type": "spot"
  }
}
```

### GCP Node

```json
{
  "id": "gke-prod-default-pool-1a2b3c4d-x9yz",
  "provider": "gcp",
  "resource_type": "gcp:compute/instance:Instance",
  "sku": "e2-standard-4",
  "region": "us-central1",
  "tags": {
    "kind": "node",
    "provider_id": "gce://my-project/us-central1-a/gke-prod-default-pool-1a2b3c4d-x9yz",
    "capacity_type": "on-demand"
  }
}
```

### EKS Control Plane

```json
{
  "id": "prod-cluster",
  "provider": "aws",
  "resource_type": "aws:eks/cluster:Cluster",
  "region": "us-east-1",
  "tags": { "kind": "cluster" }
}
```

### Unpriceable Nodes

Some nodes cannot be priced as instances, for example EKS Fargate nodes or nodes from an unknown
provider. Report their `cpu_allocatable`/`mem_allocatable` rows and **omit** a priceable entry. The
host can still see the capacity, and the validator accepts node rows without a matching priceable
entry. The reverse is invalid: a priceable node whose `id` matches no row's `node` value.

## Warnings

`warnings` holds human-readable, non-fatal notes, for example `unknown metric "gpu_seconds"
ignored` or `mem_usage unavailable: metrics-server not installed`. A warning never replaces an
error: if the source cannot answer, it returns a status code.

## Errors

| Code | When |
| ---- | ---- |
| `INVALID_ARGUMENT` | Only one of `start`/`end` is set, `start` is after `end`, or the source cannot serve the requested mode |
| `PERMISSION_DENIED` | Credentials lack a permission; the message names the verb and resource, such as `cannot list pods` |
| `UNAUTHENTICATED` | No usable credentials |

`pluginsdk.Serve` delivers the same code and message over gRPC and Connect. Connect clients do not
receive `Unknown` for these errors.

## Capabilities

A plugin that implements `pluginsdk.UsageSourceProvider` has `PLUGIN_CAPABILITY_USAGE_STATS` (14)
inferred, and legacy hosts see `supports_usage_stats=true`. Inference also always reports the four
pricing capabilities implied by the `Plugin` interface, so:

- A **usage-only** plugin must declare
  `WithCapabilities(PLUGIN_CAPABILITY_USAGE_STATS)` explicitly. **Hosts can tell a usage-only plugin
  apart from a pricing plugin only when its capabilities are explicit.**
- `Serve` logs one warning at startup for a usage source that has no explicit capabilities and does
  not implement `PluginInfoProvider`.
- A plugin that really does both pricing and usage can rely on inference.

## Transport Notes

In gRPC mode, `ServeConfig.UnaryInterceptors` and the SDK tracing interceptor wrap `GetStats`, as
they wrap the cost RPCs. In Connect mode (`Web.Enabled`), neither applies, exactly as for
`CostSourceService`. The Connect-mode `grpc.health.v1.Health/Check` reports
`finfocus.v1.UsageSourceService` as `SERVING` only when the plugin implements `GetStats`.

## Self-Checks for Authors

The `sdk/go/testing` package ([README](../sdk/go/testing/README.md)) provides:

- `ValidateGetStatsRequest`: request rules. Call it first in `GetStats` and map failures to
  `codes.InvalidArgument`.
- `ValidateStatsResponse`: response rules V1–V10, returning the first violation.
- `UsageSourceHarness`: serves your `GetStats` over an in-memory connection for tests.

See [sdk/go/pluginsdk/README.md](../sdk/go/pluginsdk/README.md#usage-only-plugins) for a complete
usage-only plugin.
