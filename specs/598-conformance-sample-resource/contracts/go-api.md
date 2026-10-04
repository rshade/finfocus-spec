# Go API Contract: Conformance Sample Resource

All additions are new identifiers. No existing exported signature changes.

## Package `github.com/rshade/finfocus-spec/sdk/go/testing`

```go
// SuiteConfig gains one field. Nil means DefaultSampleResource().
type SuiteConfig struct {
    // ...existing fields unchanged...
    SampleResource *pbc.ResourceDescriptor
}

// DefaultSampleResource returns a new copy of the descriptor the suite sends when no
// sample resource is configured: provider "aws", resource type "ec2", SKU "t3.micro",
// region "us-east-1".
func DefaultSampleResource() *pbc.ResourceDescriptor

// ConformanceOption adjusts the configuration RunConformance uses for a level.
type ConformanceOption func(*SuiteConfig)

// WithSampleResource makes every check that sends a resource descriptor send a copy of r.
// A nil r restores the default.
func WithSampleResource(r *pbc.ResourceDescriptor) ConformanceOption

// RunConformance runs the checks for level with that level's preset configuration,
// adjusted by opts. It returns an error for an unknown level or an invalid sample resource.
func RunConformance(
    impl pbc.CostSourceServiceServer, level ConformanceLevel, opts ...ConformanceOption,
) (*ConformanceResult, error)

// SampleResource returns a copy of the run's sample resource, or a new default when the
// harness was not created by a ConformanceSuite. Callers may modify the copy.
func (h *TestHarness) SampleResource() *pbc.ResourceDescriptor
```

Behavior:

- `RunBasicConformance(impl)` equals `RunConformance(impl, ConformanceLevelBasic)`; likewise Standard
  and Advanced. Their results do not change.
- `ConformanceSuite.Run` and `RunCategory` validate a non-nil `SampleResource` with
  `ValidateResourceDescriptor` before creating the harness and return an error wrapping the
  `ContractError` (`invalid sample resource: ...`).
- Accepted error codes per check are unchanged.

## Package `github.com/rshade/finfocus-spec/sdk/go/pluginsdk`

```go
// ConformanceOption is plugintesting.ConformanceOption.
type ConformanceOption = plugintesting.ConformanceOption

// WithSampleResource forwards to plugintesting.WithSampleResource.
func WithSampleResource(r *pbc.ResourceDescriptor) ConformanceOption

// RunConformance runs the given level against plugin; it returns ErrNilPlugin for a nil plugin.
func RunConformance(plugin Plugin, level ConformanceLevel, opts ...ConformanceOption) (*ConformanceResult, error)
```

## Check descriptions that change

| Check | New description |
| --- | --- |
| `RPCCorrectness_InvalidTimeRange` | Validates GetActualCost rejects an end before start for the sample resource (sent as resource) |
| `RPCCorrectness_GetActualCostRPC` | Validates GetActualCost for the sample resource (sent as resource); requests without one are not checked |
| `RPCCorrectness_GetActualCostBillingAccount` | Validates FOCUS records echo billing_account_id for the sample resource (sent as resource) |
