# Data Model: Allocation Period and Partial-Selection Scope

## AllocateRequest (changed)

| Field | Number | Type | Rule |
| --- | --- | --- | --- |
| `usage`, `priced`, `policy_json`, `mode` | 1-4 | unchanged | unchanged |
| `start` | 5 | Timestamp | **new**. Set together with `end`, or not at all |
| `end` | 6 | Timestamp | **new**. Not before `start` |
| `selector` | 7 | map<string,string> | **new**. Same meaning as `GetStatsRequest.selector`; not validated |

## AllocateResponse (changed)

| Field | Number | Type | Rule |
| --- | --- | --- | --- |
| `rows`, `effective_policy_json`, `policy_digest`, `warnings` | 1-4 | unchanged | unchanged |
| `start` | 5 | Timestamp | **new**. Echo of the request; absent is accepted |
| `end` | 6 | Timestamp | **new**. Echo of the request; absent is accepted |

## Validation outcomes

| Case | Result |
| --- | --- |
| Request without a window | Valid, as before |
| Request with only `start` or only `end` | `ErrInvalidAllocateRequest`, InvalidArgument |
| Request `start` after `end` | `ErrInvalidAllocateRequest`, InvalidArgument |
| Request `start` equal to `end` | Valid |
| Response without a window | Valid (older allocator) |
| Response window equal to the request's | Valid |
| Response window differs, or exists when the request had none | `ErrInvalidAllocateResponse`, names `start`/`end` |
| Non-empty selector | Invariants unchanged; the reference allocator warns |
