# Contract: Allocation Window and Selector

File: `proto/finfocus/v1/allocation.proto` (adds `import "google/protobuf/timestamp.proto";`)

```protobuf
message AllocateRequest {
  // ... fields 1-4 unchanged ...

  // Start of the period the priced costs cover, with the same rules as
  // GetStatsRequest.start: set start and end together, or leave both unset for a
  // run-rate allocation. Setting exactly one is INVALID_ARGUMENT. The period labels
  // the result and never changes an allocated cost.
  google.protobuf.Timestamp start = 5;

  // End of the period. Must not be before start.
  google.protobuf.Timestamp end = 6;

  // Workload selector the host used for usage, with the same meaning as
  // GetStatsRequest.selector. A non-empty selector is a partial selection: node
  // capacity still covers every workload, so idle and cluster rows include
  // capacity used by unselected workloads. Every invariant still holds.
  // Allocators SHOULD add a warning. Hosts SHOULD omit or label idle and cluster
  // rows for a partial selection.
  map<string, string> selector = 7;
}

message AllocateResponse {
  // ... fields 1-4 unchanged ...

  // Echo of AllocateRequest.start. Allocators MUST echo the request's period
  // exactly. Hosts accept a response without a period (from allocators built
  // before these fields) and reject one that differs from the request.
  google.protobuf.Timestamp start = 5;

  // Echo of AllocateRequest.end.
  google.protobuf.Timestamp end = 6;
}
```

## Compatibility

Additive. `buf breaking` against `main` must pass. Older allocators ignore fields 5-7 and return no
period, which hosts accept.
