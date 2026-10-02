# Data Model: Shared Conformance Harness and Duplicate-Key Check

No wire or persisted data changes. The "entities" are two internal helpers and the keys
they operate on.

## Duplicate keys

| Dataset | Record | Key type | Key |
| --- | --- | --- | --- |
| Contract commitments | `*pbc.ContractCommitment` | `string` | `contract_commitment_id` |
| Billing periods | `*pbc.BillingPeriod` | `billingPeriodKey` | (`invoice_issuer_name`, `billing_period_start` seconds, nanos) |
| Invoice details | `*pbc.InvoiceDetail` | `string` | `invoice_detail_id` |

Rules (unchanged):

- Records are already non-nil and per-record valid when the duplicate check runs.
- Pages of `pairwiseDuplicateLimit` (64) records or fewer are compared pairwise without
  allocating. Larger pages use a map keyed by the key type.
- The reported pair is the lowest later index `i` with an earlier equal key, and the earliest
  index `first` holding that key.

## Harness

`bufconnHarness[C]`, where `C` is a generated service client interface:

| Field | Meaning |
| --- | --- |
| listener | in-memory `bufconn` listener, 1 MiB (`bufSize`) |
| server | gRPC server serving on listener; one service registered at construction |
| conn | client connection; nil until `Start` |
| client | `C`, built by `newClient(conn)` in `Start`; zero value until then |
| newClient | the generated `pbc.New*Client` constructor |

Lifecycle: constructed (serving) → `Start` (connected) → `Stop` (closed). `Stop` is
idempotent and valid before `Start`.

| Exported harness | Client type `C` |
| --- | --- |
| `TestHarness` | `pbc.CostSourceServiceClient` |
| `AllocatorHarness` | `pbc.AllocatorServiceClient` |
| `ScorerHarness` | `pbc.RecommendationScorerServiceClient` |
| `UsageSourceHarness` | `pbc.UsageSourceServiceClient` |
| `ContractCommitmentHarness` | `pbc.SupplementalDatasetServiceClient` |
| `InvoiceDatasetHarness` | `pbc.SupplementalDatasetServiceClient` |
