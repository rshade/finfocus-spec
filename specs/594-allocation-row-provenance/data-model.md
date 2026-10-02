# Data Model: Allocation Row Provenance

## AllocationRow (extended)

| Field | Number | Type | Maps to | Rule |
|-------|--------|------|---------|------|
| allocated_method_id | 7 | string | FOCUS 1.3 AllocatedMethodId | Optional. Non-empty requires allocated_resource_id. |
| allocated_method_details | 8 | string | FOCUS 1.3 AllocatedMethodDetails | Optional, free-form. |
| allocated_resource_id | 9 | string | FOCUS 1.3 AllocatedResourceId | Optional, opaque. |
| (held) | 10 | - | later LineageNode | Held by comment, not reserved. |

Fields 1-6 are unchanged. Conservation, totals, and currency rules are unchanged.
