# Contract: Lineage Chain

This contract governs the two new optional fields and the SDK builder. Names, numbers, and
behavior below are normative for both `ResourceDescriptor.lineage` (field 11) and
`ActualCostResult.lineage` (field 9), and for
`pluginsdk.LineageBuilder` (`NewLineageBuilder` / `WithParent` / `WithMetadata` / `Build`).

## Reporting (plugin side)

- A plugin MAY attach one chain per cost result and per resource descriptor. It MAY attach
  none.
- The chain's leaf is the resource (type RESOURCE). Each `WithParent` call, or each nested
  `parent` assignment, adds exactly one level above the current top.
- Any level MAY carry `metadata` attributes. Attributes stay on the level the reporter put
  them on.
- A plugin MAY stop at any level, skip classifications, repeat classifications, and use
  CUSTOM levels with plugin-defined meaning.

## Transport (host side)

A host MUST:

1. Store and forward the chain exactly as received.
2. Treat a nil `parent` as the top of the reported chain.
3. Serialize the chain to JSON and back with order, ids, names, classifications, and
   per-level attributes preserved.

A host MUST NOT:

1. Infer or synthesize a parent the plugin did not report.
2. Auto-populate lineage from other fields (for example deriving a parent from
   `billing_account_id`).
3. Validate node ids against provider APIs.
4. Warn, reject, or fail conformance when the chain omits or disagrees with
   `billing_account_id` or `sub_account_id`. The flat FOCUS account fields remain canonical.
5. Require a complete chain, in any context.

## Builder semantics

- `NewLineageBuilder(resourceID, resourceName)` creates the leaf node (type RESOURCE).
- `WithParent(nodeType, id, name)` links a new node above the current top and makes it the
  new top. O(1) per call.
- `WithMetadata(key, value)` sets the attribute on the most recently added node — the leaf
  before any `WithParent`, the newest parent afterward.
- `Build()` returns the leaf. The chain walks upward through `parent` links. `Build()`
  performs no validation; partial chains are valid output.
- The builder assembles caller-supplied data only. It does not infer, validate, or complete
  anything.

## Round-trip matrix (locked by tests)

| Input shape | Guaranteed outcome |
|-------------|--------------------|
| Four-level chain with attributes on leaf and a middle level, on `ActualCostResult` | All four levels, exact order, ids, names, classifications, and per-level attributes survive protojson. |
| Same chain on `ResourceDescriptor` | Same fidelity. |
| Leaf only (no parents) | Non-nil leaf, nil parent, accepted. |
| Ten-level chain with CUSTOM nodes | All levels survive in order. |
| Chain with no billing-account node plus a `FocusCostRecord` with account ids | Both survive unchanged; the flat ids are untouched. |

## Non-behavior

- No error path exists for lineage content. There is nothing to reject.
- No depth limit is enforced by the contract; nested encoding imposes practical recursion
  limits, documented on the message.
- No RPC, capability, or conformance fixture is added.
