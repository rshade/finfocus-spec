# Data Model: Caller-Supplied Billing Account ID on Actual Cost Requests

## GetActualCostRequest (changed)

| Field | Number | Type | Change |
| --- | --- | --- | --- |
| `resource_id` | 1 | string | unchanged |
| `start` | 2 | Timestamp | unchanged |
| `end` | 3 | Timestamp | unchanged |
| `tags` | 4 | map<string,string> | unchanged; never carries the billing account id |
| `arn` | 5 | string | unchanged |
| `dry_run` | 6 | bool | unchanged |
| `page_size` | 7 | int32 | unchanged |
| `page_token` | 8 | string | unchanged |
| `billing_account_id` | 9 | string | **new** |
| (none) | 10 | (none) | held for a later billing account name; not `reserved` |

### `billing_account_id` rules

| Request value | Plugin attaches FOCUS record? | FOCUS `billing_account_id` | Cost |
| --- | --- | --- | --- |
| `""` (default) | MAY omit it; MUST NOT invent an id | Plugin's own data only, never made up | unchanged |
| non-empty `X` | MAY attach it | MUST equal `X` | unchanged; not a filter |

- Not trimmed, normalized, or format-checked by the SDK.
- No effect when `dry_run` is true.
- A caller sends the same value on every page of one paginated query.

## ActualCostResult (unchanged)

`focus_record` stays optional. With this change, the request id is where a plugin gets
`FocusCostRecord.billing_account_id` when it has no account data of its own.

## Reference producer: MockPlugin FOCUS record

When `billing_account_id` is non-empty, `MockPlugin.GetActualCost` attaches a record to each result:

| FOCUS field | Value |
| --- | --- |
| `billing_account_id` | request `billing_account_id` |
| `service_provider_name` | mock `PluginName` |
| `resource_id` | request `resource_id` |
| `billing_period_start` / `_end` | UTC calendar month that contains the result timestamp |
| `charge_period_start` / `_end` | result timestamp, plus one hour |
| `billing_currency` | `USD` |
| `charge_category` / `charge_class` | Usage / Regular |
| `charge_description`, `service_name`, `service_category` | fixed mock values (Compute) |
| `consumed_quantity` / `consumed_unit` | result `usage_amount` (always > 0) / `Hours` |
| `billed_cost`, `effective_cost`, `list_cost` | result `cost` |

The record passes `pluginsdk.ValidateFocusRecord`. The result `cost` is the same whether or not a
record is attached.

## New Go symbols (`sdk/go/testing`)

| Symbol | Kind | Purpose |
| --- | --- | --- |
| `ErrBillingAccountIDMismatch` | sentinel error | FOCUS record id differs from the request id |
| `ValidateActualCostBillingAccount(req, resp) error` | func | Enforces FR-004 for one response |
| `RPCCorrectness_GetActualCostBillingAccount` | conformance test (Standard) | Runs the rule against a plugin |
