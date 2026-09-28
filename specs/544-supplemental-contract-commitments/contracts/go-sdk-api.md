# Go SDK API: Contract Commitments

## Package `pluginsdk`

```go
// ContractCommitmentProvider is an optional interface for plugins that serve
// SupplementalDatasetService.GetContractCommitments.
type ContractCommitmentProvider interface {
    GetContractCommitments(ctx context.Context, req *pbc.GetContractCommitmentsRequest) (
        *pbc.GetContractCommitmentsResponse, error)
}

// Delegating wrappers; each is identical to the sdk/go/testing function of the same name.
func ValidateContractCommitment(c *pbc.ContractCommitment) error
func ValidateGetContractCommitmentsRequest(req *pbc.GetContractCommitmentsRequest) error
func ValidateGetContractCommitmentsResponse(
    req *pbc.GetContractCommitmentsRequest, resp *pbc.GetContractCommitmentsResponse) error
func ContractCommitmentMatchesWindow(c *pbc.ContractCommitment, start, end *timestamppb.Timestamp) bool
func PaginateContractCommitments(commitments []*pbc.ContractCommitment, pageSize int32, pageToken string) (
    page []*pbc.ContractCommitment, nextPageToken string, totalCount int32, err error)
```

Serving (unexported): `optionalServices.commitments`; `contractCommitmentGRPCServer` and
`contractCommitmentConnectHandler` adapters; the Connect health checker lists
`pbcconnect.SupplementalDatasetServiceName` when the provider is implemented.

Capabilities: `inferCapabilities` appends `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS`;
`optionalCapabilities = 9`; `maxValidCapability = PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS`;
`legacyCapabilityNames` maps it to `supports_contract_commitments`.

`ContractCommitmentBuilder.Build` validates with `ValidateContractCommitment` (same messages as
before, plus rejection of non-finite amounts).

## Package `sdk/go/testing` (imported as `plugintesting`)

```go
var (
    ErrInvalidContractCommitment          = errors.New("invalid contract commitment")
    ErrInvalidContractCommitmentsRequest  = errors.New("invalid contract commitments request")
    ErrInvalidContractCommitmentsResponse = errors.New("invalid contract commitments response")
)

func ValidateContractCommitment(c *pbc.ContractCommitment) error
func ValidateGetContractCommitmentsRequest(req *pbc.GetContractCommitmentsRequest) error
func ValidateGetContractCommitmentsResponse(
    req *pbc.GetContractCommitmentsRequest, resp *pbc.GetContractCommitmentsResponse) error
func ContractCommitmentMatchesWindow(c *pbc.ContractCommitment, start, end *timestamppb.Timestamp) bool
func PaginateContractCommitments(commitments []*pbc.ContractCommitment, pageSize int32, pageToken string) (
    []*pbc.ContractCommitment, string, int32, error)

// Reference producer.
type MockContractCommitmentSource struct{ /* unexported */ }
func NewMockContractCommitmentSource(commitments []*pbc.ContractCommitment) (*MockContractCommitmentSource, error)
func (m *MockContractCommitmentSource) GetContractCommitments(
    ctx context.Context, req *pbc.GetContractCommitmentsRequest) (*pbc.GetContractCommitmentsResponse, error)

// Harness and conformance.
type ContractCommitmentServer interface {
    GetContractCommitments(ctx context.Context, req *pbc.GetContractCommitmentsRequest) (
        *pbc.GetContractCommitmentsResponse, error)
}
type ContractCommitmentHarness struct{ /* unexported */ }
func NewContractCommitmentHarness(impl ContractCommitmentServer) *ContractCommitmentHarness
func (h *ContractCommitmentHarness) Start(t testing.TB)
func (h *ContractCommitmentHarness) Stop()
func (h *ContractCommitmentHarness) Client() pbc.SupplementalDatasetServiceClient
func RunContractCommitmentConformance(t *testing.T, impl ContractCommitmentServer)
```

Every validation failure is a plain error wrapping one sentinel and carrying
`codes.InvalidArgument` via `GRPCStatus()`. `ValidateContractCommitment` messages carry no prefix
(they are the builder's messages); request and response messages start with their sentinel, and
response messages name `commitments[i]`.

### Conformance scenarios (subtest names)

| Name | Checks |
| --- | --- |
| `full_walk` | No window, page size 1000: every page passes response validation, no duplicate IDs across pages, `total_count` equals the walked count on every page. |
| `stable_order` | Page size 1 for up to 25 pages returns the same IDs in the same order as `full_walk`. |
| `window_filter` | Windows `[1900-01-01, 1900-02-01)` and `[2000-01-01, 2100-01-01)`: responses pass validation (every record matches the window) and the total is not more than `full_walk`'s. |
| `window_one_bound` | Only `start` set → InvalidArgument. |
| `window_inverted` | `end` before `start` → InvalidArgument. |
| `negative_page_size` | `page_size = -1` → InvalidArgument. |
| `malformed_page_token` | `page_token = "!not-a-token!"` → InvalidArgument. |

`export_test.go` exposes `RunContractCommitmentScenariosForTest` so broken sources can be asserted
to fail specific scenarios.
