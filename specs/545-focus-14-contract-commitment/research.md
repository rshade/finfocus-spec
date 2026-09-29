# Research: FOCUS 1.4 Contract Commitment Columns

## Decisions

- **Applicability is a JSON string.** FOCUS defines a JSON object. The SDK already stores JSON
  columns such as allocated method details as strings. A typed message would be a second schema.
- **Billing currency is required for spend only.** FOCUS 1.3 and 1.4 allow null, and they forbid
  null when the category is Spend. The previous validator required it for every category.
- **optional double for three metrics.** Proto3 scalars cannot tell 0 from null. Discount must be
  null for Availability and must be set for Discount, including a real zero.
- **No new RPC.** Issue 552 already serves ContractCommitment. This change extends that message.
- **Per-record rules only.** Superseded chains and one-way status transitions need other rows.
- **Spec number 545.** The highest specs prefix on this branch point is 544.

## Sources

- Issue 542 and `.specify/assessments/focus-1-4-support/research.md`
- FOCUS 1.4 Contract Commitment dataset column list
- FOCUS 1.3 Contract Applied column: Elements must include ContractCommitmentID,
  ContractCommitmentAppliedCost, ContractCommitmentAppliedQuantity, and
  ContractCommitmentAppliedUnit
