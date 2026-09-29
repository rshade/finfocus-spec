# Quickstart: FOCUS 1.4 Contract Commitment Columns

Build a spend commitment that satisfies the columns that do not allow nulls:

```go
at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
record, err := pluginsdk.NewContractCommitmentBuilder().
    WithIdentity("commit-1", "contract-1").
    WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
    WithFinancials(12000, 0, "", "USD").
    WithBaselineTerms(at).
    BuildFocus14()
```

`Build` keeps the pre-1.4 rules and does not require the 1.4 columns, so existing builder chains
keep working. `BuildFocus14` also requires them.

Override a column by calling its setter after `WithBaselineTerms`. Clear a discount with
`ClearDiscountPercentage` when the benefit is Availability.

Attach the commitment to a cost row:

```go
cost := 12.5
object, err := pluginsdk.FormatContractApplied([]pluginsdk.ContractAppliedElement{{
    ContractID:   "contract-1",
    CommitmentID: record.GetContractCommitmentId(),
    AppliedCost:  &cost,
}})
builder.WithContractAppliedObject(object)
```

`WithContractApplied` still stores a bare ID and is deprecated.
