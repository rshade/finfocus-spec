# Go SDK Contract: Usage Source

Public Go API added by this feature. The signatures are binding. Bodies are left to implementation.

## `sdk/go/pluginsdk`

### Optional interface (`sdk.go`, next to `ResolveResourceTypesProvider`)

```go
// UsageSourceProvider is an optional interface for plugins that serve
// UsageSourceService. When ServeConfig.Plugin implements it, Serve registers
// the service in gRPC and Connect modes and PLUGIN_CAPABILITY_USAGE_STATS is
// inferred. Usage-only plugins should set PluginInfo.Capabilities explicitly.
type UsageSourceProvider interface {
    GetStats(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error)
}
```

### Behavior changes (no new exported symbols)

| Location | Change |
|----------|--------|
| `Serve` → `serveGRPC` | `pbc.RegisterUsageSourceServiceServer` with an unexported adapter, only when the plugin implements `UsageSourceProvider`. |
| `Serve` → `serveConnect` | Mount `pbcconnect.NewUsageSourceServiceHandler(adapter, handlerOpts...)`. Append `pbcconnect.UsageSourceServiceName` to the `grpchealth` static checker. Both happen only when the plugin implements the interface. |
| `Serve` | Logs one `Warn` if the plugin implements `UsageSourceProvider`, `PluginInfo` has no explicit `Capabilities`, and the plugin does not implement `PluginInfoProvider`. |
| `inferCapabilities` | Appends `PLUGIN_CAPABILITY_USAGE_STATS`; `optionalCapabilities = 7`. |
| `legacyCapabilityNames` | `PLUGIN_CAPABILITY_USAGE_STATS: "supports_usage_stats"`. |
| `IsValidCapability` | Upper bound `PLUGIN_CAPABILITY_USAGE_STATS` (14). |

Errors that `GetStats` returns reach clients with the same code and message over both transports.
In Connect mode the adapter converts gRPC `status` errors with the unexported helper below
(connect-go would otherwise report them as `CodeUnknown`; see research R8).

### Unexported helper (`connect_errors.go`)

```go
// toConnectError converts an error carrying a gRPC status into a *connect.Error
// with the same code and message. nil, *connect.Error values, and errors
// without a gRPC status are returned unchanged.
func toConnectError(err error) error
```

### Constants (`subjects.go`)

```go
const (
    SubjectCluster        = "cluster"
    SubjectNamespace      = "namespace"
    SubjectControllerKind = "controller_kind"
    SubjectController     = "controller"
    SubjectPod            = "pod"
    SubjectNode           = "node"
    SubjectKind           = "kind"
    SubjectLabelPrefix    = "label."

    KindWorkload = "workload"
    KindNode     = "node"
    KindIdle     = "__idle__"    // allocator output only (#506)
    KindCluster  = "__cluster__" // allocator output only (#506)

    MetricCPURequest     = "cpu_request"
    MetricMemRequest     = "mem_request"
    MetricCPUAllocatable = "cpu_allocatable"
    MetricMemAllocatable = "mem_allocatable"
    MetricCPUUsage       = "cpu_usage"
    MetricMemUsage       = "mem_usage"

    UnitCore      = "core"
    UnitGiB       = "GiB"
    UnitCoreHours = "core-hours"
    UnitGiBHours  = "GiB-hours"
)
```

## `sdk/go/testing` (imported as `plugintesting`)

```go
// In contract.go, next to the existing Validate*Request helpers (research R11).

// ErrInvertedStatsWindow reports a GetStatsRequest whose start is after its end.
var ErrInvertedStatsWindow = errors.New("start time must not be after end time")

// ValidateGetStatsRequest validates a GetStatsRequest message. A request with
// neither start nor end is a run-rate request and is valid. It returns
// ErrNilRequest, or a *ContractError wrapping ErrNilStartTime, ErrNilEndTime,
// or ErrInvertedStatsWindow (rules Q1–Q3 in data-model.md).
func ValidateGetStatsRequest(req *pbc.GetStatsRequest) error
```

```go
// In usage_source.go.

// ErrInvalidStatsResponse is wrapped by every ValidateStatsResponse failure.
var ErrInvalidStatsResponse = errors.New("invalid GetStats response")

// ValidateStatsResponse returns nil if resp satisfies the usage-source
// contract, or the first violation found (rules V1–V10 in data-model.md).
func ValidateStatsResponse(resp *pbc.GetStatsResponse) error

// KnownSubjectKeys returns a copy of the documented, non-label subject keys
// accepted by ValidateStatsResponse. It exists for the pluginsdk drift test.
func KnownSubjectKeys() []string

// UsageStatsServer is satisfied by any type with a GetStats method, including
// plugins implementing pluginsdk.UsageSourceProvider.
type UsageStatsServer interface {
    GetStats(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error)
}

// UsageSourceHarness serves a UsageStatsServer over an in-memory bufconn.
type UsageSourceHarness struct { /* unexported */ }

func NewUsageSourceHarness(impl UsageStatsServer) *UsageSourceHarness
func (h *UsageSourceHarness) Start(t testing.TB)
func (h *UsageSourceHarness) Stop()
func (h *UsageSourceHarness) Client() pbc.UsageSourceServiceClient
```

## Generated (via `make generate`)

- `sdk/go/proto/finfocus/v1/usage.pb.go`: `GetStatsRequest`, `GetStatsResponse`, `UsageRow`,
  `StatsMode`.
- `sdk/go/proto/finfocus/v1/usage_grpc.pb.go`: `UsageSourceServiceServer/Client`,
  `RegisterUsageSourceServiceServer`, `UnimplementedUsageSourceServiceServer`.
- `sdk/go/proto/finfocus/v1/pbcconnect/usage.connect.go`: `UsageSourceServiceName`,
  `NewUsageSourceServiceHandler`, `NewUsageSourceServiceClient`.
- `enums.pb.go`: `PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS`.
