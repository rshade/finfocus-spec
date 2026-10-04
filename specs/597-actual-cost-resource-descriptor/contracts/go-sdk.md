# Contract: Go SDK surface

No new exported identifiers. Two exported validators change behavior for requests that set the
new field; requests without it behave exactly as before.

## `pluginsdk`

```go
// ValidateActualCostRequest validation order (fail-fast):
//  1. Request nil check
//  2. ResourceId empty check
//  3. StartTime nil check
//  4. EndTime nil check
//  5. TimeRange validation (EndTime must be after StartTime)
//  6. Resource descriptor (only when set): ValidateResourceDescriptor
func ValidateActualCostRequest(req *pbc.GetActualCostRequest) error
```

Step 6 returns `ValidateResourceDescriptor`'s `codes.InvalidArgument` status error unchanged.
A valid request without a descriptor stays at 0 allocs/op.

## `testing` (plugintesting)

```go
// ValidateGetActualCostRequest additionally runs ValidateResourceDescriptor on
// req.Resource when it is set, after the tags check, returning its ContractError
// (for example NewContractError("attributes", size, ErrAttributesTooLarge)).
func ValidateGetActualCostRequest(req *pbc.GetActualCostRequest) error
```

New contract-suite cases: `GetActualCostRequest_WithResourceAccepted`,
`GetActualCostRequest_OversizedAttributesRejected`.

New conformance test: `RPCCorrectness_GetActualCostWithResource` (Standard). Passes on a valid
response or on `NotFound`/`Unavailable`; never checks that the plugin used the descriptor.

`MockPlugin.GetActualCost`: when a FOCUS record is attached and `resource` is set, the record's
`resource_type`, `region_id`, and `sku_id` come from the descriptor.

## TypeScript

`GetActualCostRequest.resource?: ResourceDescriptor` (generated). There is no new hand-written API;
`actualCostIterator` sends it on every page through its existing request clone.
