# Data Model: Resource Descriptor on Actual Cost Requests

## GetActualCostRequest (changed)

| # | Field | Type | Change |
| --- | --- | --- | --- |
| 1 | `resource_id` | string | Unchanged; still required |
| 2-3 | `start`, `end` | Timestamp | Unchanged |
| 4 | `tags` | map<string,string> | Meaning unchanged: cloud tags, billing filters |
| 5 | `arn` | string | Unchanged |
| 6 | `dry_run` | bool | Unchanged |
| 7-8 | `page_size`, `page_token` | int32, string | Unchanged |
| 9 | `billing_account_id` | string | Unchanged |
| 10 | (free) | - | Held by comment for a billing account name |
| 11 | `resource` | `ResourceDescriptor` | **New**, optional |

### Rules for `resource`

- Unset: the host sent none. Plugins use `tags`, `resource_id`, and `arn`, as before.
- Set: same meaning as `GetProjectedCostRequest.resource`, including `attributes` and the host
  redaction rules. Plugins read pricing dimensions (provider, type, SKU, region, attributes, the
  descriptor's own tags) from it, not from the request `tags`.
- The request `tags` are not merged into the descriptor and do not override it.
- Same value on every page of one paginated query. Dry run is otherwise unchanged.
- `resource.id` is not compared with `resource_id`.

### Validation

| Validator | Unset | Set |
| --- | --- | --- |
| `pluginsdk.ValidateActualCostRequest` | skipped, 0 allocs | `pluginsdk.ValidateResourceDescriptor` (lengths, tag limits, `MaxAttributesBytes`) |
| `plugintesting.ValidateGetActualCostRequest` | skipped | `plugintesting.ValidateResourceDescriptor` (also provider and type required) |

Both run the descriptor check last, after the existing checks, so existing error precedence is
unchanged.

## ResourceDescriptor (unchanged)

Fields 1-12 as defined by specs 019, 028, and 596. No change.

## Mock FOCUS record (reference plugin behavior)

When a FOCUS record is attached (request `billing_account_id` non-empty) and `resource` is set:

| FocusCostRecord field | Source |
| --- | --- |
| `resource_type` | `resource.resource_type` |
| `region_id` | `resource.region` |
| `sku_id` | `resource.sku` |

All other record fields are built as in 585. Cost values never depend on `resource`.
