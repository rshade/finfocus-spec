# Research: Resource Descriptor on Actual Cost Requests

No item in the Technical Context needed clarification. Each decision below records a design
choice and the alternative it beat.

## R1: Field shape and number

- **Decision**: `ResourceDescriptor resource = 11;` on `GetActualCostRequest`, placed after
  `billing_account_id = 9`. Field 10 stays free, as the 585 comment already holds it for a billing
  account name.
- **Rationale**: The issue fixes the number. Reusing the existing message gives the actual path
  every descriptor feature (attributes, tags, limits, builders) with no new type, and the name
  `resource` matches `GetProjectedCostRequest.resource`, so hosts and plugins use the same accessor.
- **Alternatives considered**: `google.protobuf.Struct attributes = 11` alone. Rejected in the
  issue: it leaves provider, type, SKU, and region as injected tags that collide with user labels.

## R2: SDK validator (`pluginsdk.ValidateActualCostRequest`)

- **Decision**: After the time range check, add
  `if resource := req.GetResource(); resource != nil { return ValidateResourceDescriptor(resource) }`
  (the `pluginsdk` descriptor validator in `batch.go`), returning its gRPC `InvalidArgument` error
  unchanged. Update the doc comment's validation order (step 6).
- **Rationale**: Unset is valid (FR-007), so the nil guard sits at the call site; the
  `pluginsdk` descriptor validator rejects nil, which would otherwise break every existing caller.
  The `pluginsdk` validator checks lengths and limits only (no required provider or type), which is
  the rule `ValidateBatchCostRequest` already applies to descriptors. A nil resource adds one load
  and one branch, so the `_Valid` benchmark stays at 0 allocs/op (FR-009). A descriptor with no
  attributes also stays at 0 allocs/op; attributes cost the one 16 B `proto.Size` allocation 596
  documented.
- **Alternatives considered**: Wrapping the error with a `resource:` prefix. Rejected: the
  descriptor errors already name the field (`attributes size ...`, `tag value ...`). The batch path
  prefixes an index only because it has many descriptors, and a `fmt.Errorf` wrap in the function
  body grew a hot validator's frame in 557.

## R3: Contract validator (`plugintesting.ValidateGetActualCostRequest`)

- **Decision**: After `ValidateTags`, add the same nil-guarded call to
  `plugintesting.ValidateResourceDescriptor`, returning its `ContractError`. Add two contract suite
  cases: `GetActualCostRequest_WithResourceAccepted` and
  `GetActualCostRequest_OversizedAttributesRejected`.
- **Rationale**: The contract validator is what conformance and hosts use to check the
  Core-to-plugin contract. Its descriptor rules (provider and resource type required, provider known,
  tags, attributes size) are the ones projected, supports, and batch requests get, so one descriptor
  is valid or invalid the same way on every path (SC-003).
- **Alternatives considered**: Only lengths in the contract layer, to match `pluginsdk`. Rejected:
  the two layers already differ for every descriptor; matching the other contract paths matters more
  than matching the other layer.

## R4: How the mock reads the descriptor

- **Decision**: When the request carries a descriptor and the mock attaches a FOCUS record (only
  when `billing_account_id` is set, per 585), the mock copies the descriptor's `resource_type`,
  `region` (to `region_id`), and `sku` (to `sku_id`) into the record. With no descriptor, the record
  is built exactly as today.
- **Rationale**: This is what a list-price plugin does with the descriptor: it describes the
  resource it priced. It is observable through a public response field, so tests and hosts can see
  the descriptor reached the plugin (FR-011). No FOCUS rule constrains these three columns, so the
  record still validates. It also does not change cost values, keeping Principle III (the spec does
  not calculate).
- **Alternatives considered**:
  - Scaling mock cost by an attribute such as instance count. Rejected: it puts pricing logic in the
    reference plugin and changes the costs that existing tests assert.
  - Recording the last received descriptor in a mock field. Rejected: it shows delivery but not a
    plugin reading pricing dimensions, and it adds mutable shared state to a concurrent mock.

## R5: Conformance check

- **Decision**: Add `RPCCorrectness_GetActualCostWithResource` at Standard level (where the other
  actual cost checks live). It sends an actual cost request with a descriptor carrying tags and the
  shared `conformanceAttributes()` fixture, and passes on a valid response
  (`ValidateActualCostResponse`) or on `NotFound`/`Unavailable` (no data), as the billing account
  check does. It never checks that the plugin used the descriptor.
- **Rationale**: FR-012 asks that a plugin which ignores the descriptor passes. The mock, which only
  copies fields into an optional record, passes either way.
- **Alternatives considered**: Basic level, like `RPCCorrectness_GetProjectedCostWithAttributes`.
  Rejected: actual cost is Standard everywhere else in the suite, and a Basic check would run
  `GetActualCost` against plugins that Basic never asks to implement it.

## R6: TypeScript

- **Decision**: Regenerate bindings only. `actualCostIterator` already clones the whole request with
  `clone(GetActualCostRequestSchema, ...)`, so the descriptor rides on every page. Add a pagination
  test that asserts every page body carries the same `resource`, next to the 585 billing account
  test. Hosts build the descriptor with the existing `ResourceDescriptorBuilder`.
- **Rationale**: FR-013 is met by the existing clone; the test pins it.
- **Alternatives considered**: An iterator option for the descriptor. Rejected: unnecessary, since
  the request already carries it.

## R7: Dry run in the mock

- **Decision**: Unchanged. The mock's dry-run comment says it cannot tell the resource type from
  `resource_id`; it now notes that a plugin may read `resource.resource_type` for dry run, but the
  mock keeps returning its default mappings.
- **Rationale**: Spec says dry run is unchanged; changing mock dry-run output would affect existing
  dry-run conformance tests.
