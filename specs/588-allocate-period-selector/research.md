# Research: Allocation Period and Partial-Selection Scope

## R1: Fields

- **Decision**: `AllocateRequest`: `google.protobuf.Timestamp start = 5`, `end = 6`,
  `map<string, string> selector = 7`. `AllocateResponse`: `start = 5`, `end = 6`. `allocation.proto`
  imports `google/protobuf/timestamp.proto`.
- **Rationale**: Same names, types, and rules as `GetStatsRequest.start`, `.end`, and `.selector`, so a
  host passes its usage window and selector straight through. Field 5 is the next free tag on both
  messages.
- **Alternatives considered**: An hours count (cannot say which hours); a nested `AllocationScope`
  message (the issue allows either, and flat fields mirror `GetStatsRequest`).

## R2: Request window rule

- **Decision**: `ValidateAllocateRequest` rejects exactly one bound set ("start and end must be set
  together") and `start` after `end`, as `newInvalidArgument(ErrInvalidAllocateRequest, ...)`. It
  compares with the existing `timestampAfter` helper (seconds, then nanos), not `AsTime`.
- **Rationale**: Mirrors `ValidateGetStatsRequest`. The allocation validator already uses
  `invalidArgumentError`, so allocators can return the error as `InvalidArgument` unchanged.

## R3: Response echo rule

- **Decision**: `ValidateAllocateResponse` accepts a response with no window. When either echoed bound
  is set, both must equal the request's bounds (seconds and nanos, nil equals nil). A window on a
  response whose request had none fails. Errors wrap `ErrInvalidAllocateResponse` and name `start` or
  `end`.
- **Rationale**: Hosts run this validator on every response. Requiring the echo would reject every
  allocator built before this change, so the validator demands a correct echo when one is present, and
  conformance demands its presence (R5).

## R4: Selector

- **Decision**: No validation. The reference allocator appends the warning "selector narrows workloads:
  idle and cluster rows include capacity used by unselected workloads" when `len(selector) > 0`, and
  changes no computation. Invariants are unchanged.
- **Rationale**: Chosen by the user in clarification. It keeps conservation and the idle-row rule
  universal, which hosts and `CheckConservation` rely on. The consumer's design already omits idle and
  cluster rows on the host when `--namespace` is set.

## R5: Conformance

- **Decision**: Two scenarios in `RunAllocatorConformance`: `period_echoed` (single-node fixture plus a
  one-day window; the response must carry the identical window) and `selector_keeps_invariants`
  (single-node fixture plus `{"namespace": "payments"}`; `allocateAndVerify` must pass). The warning is
  not checked, because it is a SHOULD.
- **Rationale**: Conformance is where new allocators learn the MUST-echo rule.

## R6: Reference allocator and TypeScript

- **Decision**: `refalloc` clones the request's start and end onto the response (`proto.CloneOf`). The
  TypeScript allocator test sends a window and selector, asserts them in the request body, and reads
  the echoed window.
- **Rationale**: The generated types carry the fields, and the TS client forwards requests whole.
