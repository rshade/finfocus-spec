# Contract: Exported API of `sdk/go/testing` Is Unchanged

The public contract of this feature is that nothing in it changes. This exported surface
must keep compiling for existing callers:

```go
func NewTestHarness(impl pbc.CostSourceServiceServer) *TestHarness
func NewAllocatorHarness(impl AllocateServer) *AllocatorHarness
func NewScorerHarness(impl ScoreServer) *ScorerHarness
func NewUsageSourceHarness(impl UsageStatsServer) *UsageSourceHarness
func NewContractCommitmentHarness(impl ContractCommitmentServer) *ContractCommitmentHarness
func NewInvoiceDatasetHarness(impl InvoiceDatasetServer) *InvoiceDatasetHarness

// For every harness H with client type C (see data-model.md):
func (h *H) Start(t testing.TB)
func (h *H) Stop()
func (h *H) Client() C
```

Unchanged as well:

- `ValidateGetContractCommitmentsResponse`, `ValidateGetBillingPeriodsResponse`, and
  `ValidateGetInvoiceDetailsResponse`: the same errors, sentinels, and codes, and 0 allocs on
  valid pages of up to 64 records.
- `pluginsdk` re-exports of these validators.

Check: `go doc ./sdk/go/testing <Harness>` lists `Start`, `Stop`, and `Client` for all six,
and `go build ./...` (including `pluginsdk` and the examples) succeeds without edits outside
`sdk/go/testing`.
