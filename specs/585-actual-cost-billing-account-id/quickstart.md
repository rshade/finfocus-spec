# Quickstart: Validate the Actual Cost Billing Account ID

Run everything from the repository root.

## 1. Generated code is current and the change is additive

```bash
make generate && git diff --exit-code -- sdk/
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'
grep -n "billing_account_id = 9" proto/finfocus/v1/costsource.proto
```

Expected: no diff after generation, no lint or breaking findings, and one match.

## 2. Reference plugin echoes the id into a valid FOCUS record (User Stories 1 and 2)

```bash
go test ./sdk/go/pluginsdk/ -run 'TestActualCostBillingAccount' -v
go test ./sdk/go/testing/ -run 'TestMockActualCostBillingAccount' -v
```

Expected: with an id, every result carries a FOCUS record whose `billing_account_id` matches and
which passes `pluginsdk.ValidateFocusRecord`. Without an id, no result carries a FOCUS record. Costs
are identical in both cases.

## 3. Conformance rule (User Story 3)

```bash
go test ./sdk/go/testing/ -run 'TestValidateActualCostBillingAccount|TestRPCCorrectness' -v
go test -v -run TestConformance ./sdk/go/testing/
```

Expected: the validator passes an echoing plugin and a plugin with no FOCUS records, and fails a
mismatched id with `ErrBillingAccountIDMismatch`. The full conformance suite still passes for the
mock.

## 4. TypeScript (User Story 4)

```bash
cd sdk/typescript && npm ci && npm run build && npm test
```

Expected: the pagination test shows `billingAccountId` on every page request.

## 5. Backward compatibility (SC-002)

```bash
make test
go test -v -tags=integration ./sdk/go/testing/
```

Expected: all existing tests pass unchanged.
