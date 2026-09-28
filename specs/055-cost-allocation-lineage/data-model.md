# Data Model: Standardized Cost Allocation Lineage Metadata

One new message, one new enum, two new optional fields. Nothing is removed, renamed, or
retyped. All additions are optional; absence is the pre-slice behavior.

## LineageNode (`proto/finfocus/v1/costsource.proto`)

| Field | Number | Type | Rule |
|-------|--------|------|------|
| `type` | 1 | `LineageNodeType` | Classification of this level. UNSPECIFIED marks an incomplete implementation. |
| `id` | 2 | `string` | Provider-specific identifier for the level. Opaque; never validated against a provider API. |
| `name` | 3 | `string` | Human-readable display name. May be empty when only the id is known. |
| `parent` | 4 | `LineageNode` | Next level up toward the root. Nil ends the reported chain; a nil parent is never inferred or synthesized. |
| `metadata` | 5 | `map<string, string>` | Free-form per-level attributes (for example `owner_email`, `cost_center`). Keys are not standardized. |

A chain is a singly-linked list. The leaf is the resource; walking `parent` links moves toward
the organization. The node with a nil parent is the top of what was reported, whether or not
its classification is ORGANIZATION.

## LineageNodeType (`proto/finfocus/v1/enums.proto`)

| Value | Number | Example levels |
|-------|--------|----------------|
| UNSPECIFIED | 0 | Default; incomplete implementation. |
| ORGANIZATION | 1 | AWS Organization, Azure Tenant, GCP Organization. |
| ORGANIZATIONAL_UNIT | 2 | AWS OU, Azure Management Group, GCP Folder. |
| BILLING_ACCOUNT | 3 | AWS Payer Account, Azure Billing Profile, GCP Billing Account. |
| SUB_ACCOUNT | 4 | AWS Member Account, Azure Subscription. |
| RESOURCE_GROUP | 5 | Azure Resource Group, GCP Project, Kubernetes Namespace. |
| RESOURCE | 6 | Individual billable resource; the leaf. |
| CUSTOM | 7 | Plugin-defined level; meaning carried by `name` and `metadata`. |

The enum imposes no structural rules: chains may skip levels, repeat a classification, or stop
at any level.

## Field additions

| Message | Field | Number | Notes |
|---------|-------|--------|-------|
| `ResourceDescriptor` | `lineage` | 11 | Optional. Number verified free (fields 1–10 in use). Host-to-plugin input direction. |
| `ActualCostResult` | `lineage` | 9 | Optional. Field 8 is `expires_at` (spec 045) and is untouched. Plugin-to-host output direction. |

## Relationships that do not exist (by design)

- No link between `LineageNode` and `FocusCostRecord.billing_account_id` /
  `sub_account_id`. Neither direction is checked; both are delivered as reported.
- No lineage on `GetProjectedCostResponse`, `EstimateCostResponse`, `BatchCostResponse`
  wrappers, or `FocusCostRecord`.
- No stored depth, no sibling links, no root pointer. Position in the chain is the depth.

## Signals callers can rely on

| What the reader sees | Meaning |
|----------------------|---------|
| `lineage` absent | The plugin reported no ancestry. Normal; not an error. |
| Leaf node only, nil parent | Partial chain of one level. Complete as reported. |
| Node with UNSPECIFIED type | Incomplete plugin implementation, not a new level kind. |
| Chain whose BILLING_ACCOUNT id differs from `billing_account_id` | Both are as the plugin reported; the flat field is canonical for two-level attribution. |
| Two CUSTOM nodes | Distinguished only by their own ids, names, and attributes. |

## Depth and payload bounds

- Nested message encoding imposes practical decoder recursion limits. The comments direct
  implementations to reasonable organizational depth (typically under ten levels) rather than
  promising unlimited depth. The builder accepts any depth the caller constructs.
- Actual-cost pages cap at 1000 results (`MaxPageSize` in `sdk/go/testing/contract.go`). A
  chain per result multiplies payload by depth and attribute count; reporters keep chains to
  what consumers need.
