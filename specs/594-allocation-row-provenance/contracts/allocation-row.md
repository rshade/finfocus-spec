# Contract: AllocationRow provenance

```proto
message AllocationRow {
  // ... fields 1-6 unchanged ...
  string allocated_method_id = 7;
  string allocated_method_details = 8;
  string allocated_resource_id = 9;
  // Field 10 is held for a later LineageNode.
}
```

Validator: `ValidateAllocateResponse` returns an error wrapping `ErrInvalidAllocateResponse` (invalid-argument
semantics) for `rows[i]: allocated_method_id requires allocated_resource_id`.
Compatibility: additive; `buf breaking` against `main` passes.
