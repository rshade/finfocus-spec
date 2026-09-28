# Supplemental Dataset Service

FOCUS defines *supplemental datasets*: records that sit beside the Cost and Usage rows and join to
them by key. `SupplementalDatasetService` (`proto/finfocus/v1/supplemental.proto`) is the optional
service that delivers them. Each dataset has its own RPC and its own plugin capability, so a
plugin serves only the datasets it holds.

| Dataset | FOCUS | RPC | Capability | Legacy metadata key |
| --- | --- | --- | --- | --- |
| Contract Commitment | 1.3 | `GetContractCommitments` | `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS` (16) | `supports_contract_commitments` |

Billing Period and Invoice Detail (FOCUS 1.4) are not served yet.

## GetContractCommitments

Returns FOCUS Contract Commitment records (`ContractCommitment` in `focus.proto`) one page at a
time. Cost rows reference a commitment through `contract_applied`, which a host joins to
`contract_commitment_id`.

### Request

| Field | Meaning |
| --- | --- |
| `start`, `end` | Optional half-open window `[start, end)`. Leave both unset for every commitment. |
| `page_size` | 0 means 50; values above 1000 are served as 1000; negative is `INVALID_ARGUMENT`. |
| `page_token` | Opaque token from the previous response; empty for the first page. |

There is no "return everything" mode: every response is bounded by the page size. Pass tokens back
verbatim, with the same window and page size.

### Window matching

A commitment matches when its period overlaps the window:

1. The period is the contract commitment period when either
   `contract_commitment_period_start` or `contract_commitment_period_end` is set; otherwise it is
   the contract period (`contract_period_start`, `contract_period_end`).
2. An unset bound is open-ended. A commitment with no period bounds at all matches every window.
3. Periods and windows are half-open, so a commitment ending exactly at the window start does not
   match, and neither does one starting exactly at the window end.

For example, the window June 2025 matches a 2025-2027 reserved instance and a savings plan with no
end date, but not a 2024 committed use discount.

### Response

| Field | Meaning |
| --- | --- |
| `commitments` | At most the effective page size; each record is valid and matches the window. |
| `next_page_token` | Empty on the last page. |
| `total_count` | Exact number of matching commitments across all pages; 0 only when none match. |

Sources return commitments in an order that is stable across calls, and a
`contract_commitment_id` appears at most once across the pages of one walk.

Each call returns the source's current view of the window. In FOCUS 1.4 terms this is Replacement
correction handling with Overwrite delivery: a host replaces what it holds for that window rather
than appending. The response carries no correction or delivery handling field; one can be added
later without breaking clients if a producer needs Delta or Ledger delivery.

### Records

Every record passes the rules `ContractCommitmentBuilder.Build` applies
(`ValidateContractCommitment`): `contract_commitment_id`, `contract_id`, and `billing_currency` are
set; the category is SPEND or USAGE; the currency is ISO 4217; each period's end is not before its
start; cost and quantity are finite and non-negative.

Provider examples:

| Provider | `contract_commitment_type` | Category |
| --- | --- | --- |
| AWS | `Savings Plan`, `Reserved Instance` | SPEND, USAGE |
| Azure | `Reservation`, `Savings Plan`, `Microsoft Azure Consumption Commitment` | USAGE, SPEND |
| GCP | `Committed Use Discount` | USAGE or SPEND (resource-based or spend-based) |

### Errors

| Code | When |
| --- | --- |
| `INVALID_ARGUMENT` | Only one of `start`/`end` set, an invalid timestamp, `end` not after `start`, negative `page_size`, or a `page_token` the source did not issue |
| `PERMISSION_DENIED` | Credentials lack a required permission |
| `UNAUTHENTICATED` | No usable credentials |

A plugin that does not serve the dataset returns `UNIMPLEMENTED`; hosts check the capability first.

## SDK support

- Go plugins implement `pluginsdk.ContractCommitmentProvider`. `Serve` registers the service over
  gRPC and Connect, adds it to the Connect health check, and infers the capability. Helpers:
  `ValidateGetContractCommitmentsRequest`, `ContractCommitmentMatchesWindow`,
  `PaginateContractCommitments`, `ValidateContractCommitment`, and, for hosts,
  `ValidateGetContractCommitmentsResponse`. See
  [the pluginsdk README](../sdk/go/pluginsdk/README.md#serving-contract-commitments-supplementaldatasetservice).
- `plugintesting.RunContractCommitmentConformance` checks a provider end to end, and
  `plugintesting.MockContractCommitmentSource` is the reference producer. See
  [the testing README](../sdk/go/testing/README.md#contract-commitment-testing).
- TypeScript hosts use `SupplementalDatasetClient` (`getContractCommitments` for one page,
  `contractCommitments` to iterate every page).
