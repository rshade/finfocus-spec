# Go SDK API: Allocator

Public Go surface added by 052. Names marked **fixed** are already coded against by downstream
finfocus plans (SP2, SP3) and must not change. Design rationale is in [research.md](../research.md).

## Generated (`sdk/go/proto/finfocus/v1`, `pbcconnect`)

From [allocation.proto](./allocation.proto) via `make generate`:

- `pbc.AllocateRequest`, `pbc.AllocateResponse`, `pbc.PricedResource`, `pbc.AllocationRow`
- `pbc.AllocatorServiceServer`, `pbc.UnimplementedAllocatorServiceServer`,
  `pbc.RegisterAllocatorServiceServer`, `pbc.NewAllocatorServiceClient`
- `pbcconnect.NewAllocatorServiceHandler`, `pbcconnect.NewAllocatorServiceClient`,
  `pbcconnect.AllocatorServiceName`
- `pbc.PluginCapability_PLUGIN_CAPABILITY_ALLOCATION` (15)

## `sdk/go/pluginsdk`

```go
// AllocatorProvider is an optional interface for plugins that serve
// AllocatorService. When ServeConfig.Plugin implements it, Serve registers the
// service in gRPC and Connect modes, reports it in the Connect health checker,
// and infers PLUGIN_CAPABILITY_ALLOCATION. Allocation-only plugins should set
// PluginInfo.Capabilities explicitly.
type AllocatorProvider interface { // fixed
    Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)
}

// DecodePolicy applies the JSON policy document data onto target, which must
// be a non-nil pointer already holding the allocator's defaults. Empty,
// whitespace-only, and null documents leave target unchanged. Unknown fields
// (exact, case-sensitive match), malformed JSON, trailing data, and type
// mismatches return an error that wraps ErrInvalidPolicy, names the JSON path
// (for example "node_split.cpu"), and carries codes.InvalidArgument. Nested
// objects merge field by field; arrays replace the default wholesale.
func DecodePolicy(data []byte, target any) error // fixed

// ErrInvalidPolicy is wrapped by every DecodePolicy input error.
var ErrInvalidPolicy error

// Delegating wrappers (FR-032): identical behavior to the sdk/go/testing
// functions of the same name, so hosts need no test tooling import.
func CheckConservation(req *pbc.AllocateRequest, resp *pbc.AllocateResponse, relEpsilon float64) error // fixed
func ValidateAllocateRequest(req *pbc.AllocateRequest) error                                         // fixed
func ResolveCurrency(priced []*pbc.PricedResource) (string, error)                                 // fixed

// Re-exported for hosts; same values as the testing package.
const DefaultConservationEpsilon = 1e-6
```

Serving behavior (no new public API):

- `Serve` registers `AllocatorService` when `config.Plugin` implements `AllocatorProvider`, in both
  modes, alongside `UsageSourceService` when that is also implemented.
- The gRPC server's interceptor chain and the Connect `handlerOpts` apply to the allocator.
  Allocator errors pass through `toConnectError`.
- Startup warning when an `AllocatorProvider` has no explicit `PluginInfo.Capabilities` and does not
  implement `PluginInfoProvider`.
- Legacy metadata: `supports_allocation=true`.
- `IsValidCapability(15)` is true and `IsValidCapability(16)` is false.

## `sdk/go/testing` (import alias `plugintesting`)

```go
const (
    DefaultConservationEpsilon = 1e-6 // relative tolerance (FR-007)
    ConservationAbsoluteFloor  = 1e-9 // absolute floor so zero totals compare
)

var (
    ErrConservation            error // wrapped by *ConservationError
    ErrInvalidAllocateRequest  error // wrapped by ValidateAllocateRequest failures
    ErrInvalidAllocateResponse error // wrapped by ValidateAllocateResponse failures
    ErrMixedCurrency           error // wrapped by ResolveCurrency failures
)

// ConservationError reports a conservation failure (FR-022).
type ConservationError struct {
    Expected   float64 // sum of cost over priced=true entries
    Actual     float64 // sum of row total_cost
    Difference float64 // Actual - Expected (positive: overshoot)
    Currency   string  // resolved currency
}
func (e *ConservationError) Error() string // states expected, actual, difference
func (e *ConservationError) Unwrap() error // ErrConservation

// CheckConservation verifies FR-007 with tolerance
// max(relEpsilon*|expected|, ConservationAbsoluteFloor). It returns an error
// (never passes) on non-finite costs or totals, and on a negative or
// non-finite relEpsilon. Currency errors from ResolveCurrency are returned
// unchanged.
func CheckConservation(req *pbc.AllocateRequest, resp *pbc.AllocateResponse, relEpsilon float64) error // fixed

// ResolveCurrency applies FR-011 to priced=true entries: the single distinct
// non-empty currency, or "USD" if all are empty. More than one distinct value
// returns an error wrapping ErrMixedCurrency with codes.InvalidArgument.
func ResolveCurrency(priced []*pbc.PricedResource) (string, error)

// ValidateAllocateRequest applies FR-024. Every failure wraps
// ErrInvalidAllocateRequest (or ErrMixedCurrency) and carries
// codes.InvalidArgument, so allocators may return it directly.
func ValidateAllocateRequest(req *pbc.AllocateRequest) error

// ValidateAllocateResponse applies FR-005 and FR-008 through FR-011, plus the
// presence of effective policy and digest (FR-023). Failures wrap
// ErrInvalidAllocateResponse.
func ValidateAllocateResponse(req *pbc.AllocateRequest, resp *pbc.AllocateResponse) error

// AllocateServer is satisfied by any type with an Allocate method, including
// pbc.AllocatorServiceServer implementations and pluginsdk.AllocatorProvider.
type AllocateServer interface {
    Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)
}

// AllocatorHarness serves an AllocateServer over an in-memory bufconn (FR-026).
type AllocatorHarness struct{ /* unexported */ }
func NewAllocatorHarness(impl AllocateServer) *AllocatorHarness
func (h *AllocatorHarness) Start(t testing.TB)
func (h *AllocatorHarness) Stop()
func (h *AllocatorHarness) Client() pbc.AllocatorServiceClient

// RunAllocatorConformance runs the standard allocator scenarios as subtests
// (FR-025) over an AllocatorHarness. Assertions are policy-agnostic.
func RunAllocatorConformance(t *testing.T, impl AllocateServer) // fixed name
```

### Conformance subtests

Each subtest is named exactly as below:

| Subtest | Fixture | Assertions beyond the common set |
|---------|---------|----------------------------------|
| `single_node` | 1 priced node, 2 workloads, no policy | — |
| `three_nodes` | 3 priced nodes, workloads on each | — |
| `empty_cluster` | 2 priced nodes, no workload usage | Σ idle totals = Σ node costs |
| `fully_packed_node` | requests = allocatable | idle row present |
| `unpriced_node` | 1 priced + 1 `priced=false` node | idle row only required for the priced node |
| `control_plane` | 3 nodes + `tags.kind=cluster` | ≥ 1 `__cluster__` row |
| `over_requested_node` | requests > allocatable | idle row present and ≥ 0 |
| `policy_unknown_field` | effective policy + unknown top-level key | InvalidArgument; message contains the key |
| `policy_unknown_version` | effective policy with `version` = 2147483647 | InvalidArgument |
| `empty_request` | no usage, no priced | no rows; digest is 64 lowercase hex; effective policy is an object with integer `version` |
| `fingerprint_stable` | same request twice | equal digests and effective policies |
| `fingerprint_empty_equals_braces` | `policy_json` empty vs `{}` | equal digests |

Fixture usage is valid usage-source output: every row has `kind`, node rows have `node`, each workload has a
distinct `namespace`/`pod` pair plus `node`, and no (subject, metric) pair repeats. The suite checks each fixture with
`ValidateStatsResponse` (051) before calling the allocator. `policy_unknown_field` probes only the top level: a nested
object may be a map field that accepts any key.

The common set for allocation scenarios is: the request passes `ValidateAllocateRequest`, the call
succeeds, the response passes `ValidateAllocateResponse`, and `CheckConservation` passes at
`DefaultConservationEpsilon`.
