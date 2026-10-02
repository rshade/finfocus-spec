# Contract: `GetActualCostRequest.billing_account_id`

File: `proto/finfocus/v1/costsource.proto`

```protobuf
message GetActualCostRequest {
  // ... fields 1-8 unchanged ...
  string page_token = 8;
  // billing_account_id is the caller-supplied FOCUS billing account id for this
  // request. It is not a filter and it does not change any cost value.
  //
  // Empty means the caller did not provide one. Plugins MUST NOT invent a value.
  // A plugin that cannot validate a FOCUS record without this id leaves
  // ActualCostResult.focus_record unset and still returns the cost.
  //
  // When non-empty, a plugin that attaches ActualCostResult.focus_record MUST set
  // FocusCostRecord.billing_account_id to this value. Send the same value on every
  // page of one paginated query. Ignored when dry_run is true.
  //
  // Do not carry this id in tags. Field 10 is left free for a later billing
  // account name.
  string billing_account_id = 9;
}
```

## Compatibility

- Additive: `buf breaking --against` `main` must pass.
- Old client to new plugin: the plugin reads `""`.
- New client to old plugin: the plugin ignores unknown tag 9.

## Generated surface

| Language | Accessor |
| --- | --- |
| Go | `GetActualCostRequest.BillingAccountId string`, `GetBillingAccountId() string` |
| TypeScript | `GetActualCostRequest.billingAccountId: string` (default `""`) |

## Go conformance surface (`sdk/go/testing`)

```go
var ErrBillingAccountIDMismatch = errors.New("focus record billing_account_id does not match request")

// ValidateActualCostBillingAccount returns nil when req has no billing_account_id,
// or when every attached FOCUS record in resp carries req's billing_account_id.
func ValidateActualCostBillingAccount(req *pbc.GetActualCostRequest, resp *pbc.GetActualCostResponse) error
```

Conformance test `RPCCorrectness_GetActualCostBillingAccount` (category RPC correctness, Standard
level): sends `billing_account_id` with a one-day range and applies the validator. NotFound or
Unavailable passes. Results without a FOCUS record pass.
