# Feature Specification: Stop Mixed-Currency Recommendation Totals

**Feature Branch**: `054-stop-mixed-currency-sum`

**Created**: 2026-09-27

**Status**: Draft

**Input**: User description: "GitHub issue #190, assessed go for Option B only
(`.specify/assessments/multi-currency-segregation/`). Withhold a recommendation-summary
total when its inputs use more than one currency. Do not convert currencies."

## Overview

Someone reading a recommendation summary can add savings that are not in the same currency
and still see one number. Today, 100 US dollars plus 50 euros becomes 150, and the currency
label is cleared. That number is not 150 of anything. A same-currency total (100 plus 50 in
one currency) is a real total and must stay.

This slice stops that one kind of accidental total. When a summary's inputs use two or more
non-empty currencies, the grand total is withheld: the total is zero and the currency is
blank. Counts stay. A bucket that itself mixes currencies stores zero. A bucket that uses
only one non-empty currency keeps its sum. The call does not fail.

This is not currency conversion. It does not add a currency to actual-cost rows, and it does
not return a separate total per currency.

**Relationship to constitution principle III ("The Spec Consumes, It Does Not Calculate")**:
withholding a mixed total is a refusal to add unlike units. It is not an exchange-rate
calculation. Allocation already refuses a mixed currency. This slice brings the
recommendation summary in line with that refusal, without changing allocation.

## Clarifications

### Session 2026-09-27

- Q: When currencies differ, is the summary a failed request or a withheld total? → A:
  Withheld total. The request still succeeds. The summary's shape does not change.
- Q: What does a withheld grand total look like? → A: Total estimated savings is 0 and
  currency is empty. A genuine zero in one currency still names that currency, so zero plus
  a currency is a real zero, and zero plus an empty currency means the grand total was
  withheld.
- Q: Do per-category and per-action savings follow the same rule? → A: A bucket with only
  one non-empty currency keeps its sum. A bucket that itself contains two or more non-empty
  currencies stores 0. Counts are unchanged either way.
- Q: What about an empty currency, and what about the code XXX? → A: They keep today's
  meaning. An empty currency does not count as a second currency, and that amount is still
  included in a single-currency total. XXX is an ordinary non-empty code: XXX alone is one
  currency; XXX together with another non-empty code withholds the affected aggregate.
- Q: Does a cost-category amount in one currency plus a performance-category amount in
  another still keep those bucket sums? → A: Yes. Those buckets each have one non-empty
  currency, so they keep their sums. Only the grand total changes (for the existing
  50-plus-75 example, from 125 to 0). Two rows in the same category and the same action,
  in two currencies, withhold that category bucket, that action bucket, and the grand total.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mixed Grand Total Is Withheld (Priority: P1)

A caller builds a recommendation summary from several savings figures. When those figures
use more than one currency, the caller must not see a single added total. Counts of
recommendations remain, so the caller still knows how many recommendations were considered.

**Why this priority**: The false total (100 in one currency plus 50 in another shown as 150)
is the harm this slice exists to stop. Nothing else in the slice matters if that number
still appears.

**Independent Test**: Summarize 100 in one currency and 50 in another, and summarize 100
and 50 in the same currency. Only the mixed pair withholds the total.

**Acceptance Scenarios**:

1. **Given** savings of 100 USD and 50 EUR, **When** a summary is produced, **Then** the
   total estimated savings is 0, the currency is empty, and the recommendation count is 2.
2. **Given** savings of 100 USD and 50 USD, **When** a summary is produced, **Then** the
   total estimated savings is 150 and the currency is USD.
3. **Given** savings that are genuinely zero in a single currency, **When** a summary is
   produced, **Then** the total is 0 and the currency is that currency.
4. **Given** a mix of non-empty currencies, **When** a summary is produced, **Then** the
   request succeeds. The caller is not given a failure instead of a summary.

---

### User Story 2 - Mixed Buckets Store Zero, Single-Currency Buckets Keep Their Sum (Priority: P2)

The summary also splits savings by category and by action. A split that uses only one
currency is still a meaningful sum. A split that mixes currencies is not, and must not show
an added number.

**Why this priority**: Withholding only the grand total would leave a mixed category or
action showing the same kind of false sum. Keeping single-currency splits preserves the
useful breakdown.

**Independent Test**: Summarize one cost-category amount in USD and one performance-category
amount in EUR, then summarize two cost-category rightsize amounts in USD and EUR.

**Acceptance Scenarios**:

1. **Given** a cost recommendation of 50 USD and a performance recommendation of 75 EUR,
   **When** a summary is produced, **Then** the cost bucket is 50, the performance bucket
   is 75, each action bucket keeps the sum of its own single currency, the grand total is
   0, the currency is empty, and the counts are unchanged.
2. **Given** two cost recommendations that are also rightsize actions, 100 USD and 50 EUR,
   **When** a summary is produced, **Then** the cost bucket is 0, the rightsize bucket is
   0, the grand total is 0, the currency is empty, and the counts still include both
   recommendations.

---

### User Story 3 - Empty Currency Is Not a Second Currency (Priority: P3)

Some savings omit a currency. That omission must not, by itself, cause a total to be
withheld, and the omitted-currency amount must still be included when every stated currency
is the same.

**Why this priority**: Treating a blank as its own currency would withhold totals that
today are intentionally single-currency. Treating XXX as special would change a code that
already means "no currency" in the currency list without being blank.

**Independent Test**: Summarize 100 USD plus 50 with a blank currency, then summarize 100
XXX plus 50 USD, then summarize amounts whose currencies are all blank.

**Acceptance Scenarios**:

1. **Given** 100 USD and 50 with an empty currency, **When** a summary is produced,
   **Then** the total is 150 and the currency is USD.
2. **Given** 100 XXX and 50 USD, **When** a summary is produced, **Then** the total is 0
   and the currency is empty, because both codes are non-empty and they differ.
3. **Given** only empty currencies whose amounts add to 150, **When** a summary is
   produced, **Then** the total is 150 and the currency is empty.

---

### Edge Cases

- A summary with no recommendations has a total of 0, an empty currency, and a count of 0.
  That is an empty summary, not evidence that a mixed total was withheld from data that
  was not there.
- A recommendation with no savings impact still counts. It adds nothing to any savings
  figure and does not introduce a currency.
- Two non-empty currencies that differ only by letter case are different currencies.
  Comparison stays exact, as it does today.
- A bucket whose only amounts use an empty currency keeps the sum of those amounts. No
  non-empty currency was seen inside that bucket, so the bucket is not withheld.
- Zero plus an empty currency on the grand total means "withheld" only when two or more
  non-empty currencies were present. An all-zero, all-blank summary can look the same; that
  case has no currency to name, which is today's result, not a new signal.
- A reader-facing guide that says mixed currencies are added and only the currency label
  is cleared is wrong after this slice and must say the total is withheld instead. Guides
  that do not say that are left alone.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When a recommendation summary's inputs contain two or more distinct non-empty
  currencies, the grand total MUST be withheld: total estimated savings is 0 and currency
  is empty.
- **FR-002**: The summary request MUST still succeed. Withholding a total is not a failure
  and does not change who can call the summary or what arguments they pass.
- **FR-003**: Recommendation counts, including counts by category and by action, MUST stay
  the same whether or not a total is withheld.
- **FR-004**: When every non-empty currency in the summary is the same, the grand total
  MUST be the sum of the savings amounts, including amounts whose currency is empty, and
  the currency MUST be that single non-empty currency.
- **FR-005**: A grand total of 0 in one named currency MUST remain distinguishable from a
  withheld total. The named currency is what makes it a real zero.
- **FR-006**: A category bucket or an action bucket that contains only one non-empty
  currency MUST keep the sum of the amounts in that bucket, including amounts in that
  bucket whose currency is empty.
- **FR-007**: A category bucket or an action bucket that contains two or more distinct
  non-empty currencies MUST store 0. Other buckets are unaffected.
- **FR-008**: An empty currency MUST NOT count as a distinct currency. The code XXX MUST
  keep today's meaning: it is non-empty, so it counts as a currency of its own.
- **FR-009**: Same-currency inputs of 100 and 50 MUST still summarize to 150 in that
  currency.
- **FR-010**: Both places that compute this summary MUST apply FR-001 through FR-008.
  Their public calling shape MUST NOT change, and they MUST stay separate copies.
- **FR-011**: If a README or developer guide states that mixed recommendation currencies
  are summed and only the currency label is cleared, that sentence MUST be corrected to
  the withheld-total behavior. No other documentation is rewritten.
- **FR-012**: The slice MUST NOT convert currencies, add a currency to actual-cost rows,
  return one total per currency, change allocation's empty-currency rule, or reject a
  billing record whose billing currency differs from its pricing currency.

### Key Entities

- **Recommendation summary**: The aggregate a caller reads. It has a count, a total
  estimated savings, one currency, counts and savings by category, and counts and savings
  by action.
- **Savings amount**: One recommendation's estimated savings, with a currency that may be
  empty. Empty means "not stated," not a third unit of money.
- **Bucket**: The savings grouped under one category or one action. A bucket is mixed only
  when two or more non-empty currencies appear inside it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A caller who summarizes 100 in one currency and 50 in another no longer sees
  150. They see a withheld grand total (0 and an empty currency) and an unchanged count.
- **SC-002**: A caller who summarizes 100 and 50 in the same currency still sees 150 in
  that currency.
- **SC-003**: A caller can tell a real zero from a withheld total: a real zero names its
  currency; a withheld grand total does not.
- **SC-004**: In a two-currency page where each category and each action uses only one
  currency, those bucket sums are unchanged and only the grand total is withheld.
- **SC-005**: No new way to convert one currency into another is introduced. The number of
  summary paths that add two different non-empty currencies into one presented total is 0.

## Assumptions

- Callers do not depend on the old unitless 150. No such caller is named. The previous
  behavior was a cleared label on top of a sum, which this slice removes.
- "Today's meaning" for an empty currency is: ignore it when deciding whether currencies
  differ, and still add the amount. "Today's meaning" for XXX is: it is a normal non-empty
  code, not a synonym for blank.
- Letter case is significant. "usd" and "USD" are not the same currency.
- An empty recommendation list is unchanged: count 0, total 0, empty currency.
- The broader discovery item (a currency on every actual-cost point, per-currency
  subtotals, or a money type) is a different effort. This slice does not complete it.
- Historical specification samples that show an older summary sketch are not a README or
  developer guide. They are not rewritten unless they are the sentence FR-011 names.

## Out of Scope

- Protocol changes, new fields, and regenerated bindings.
- TypeScript or any other language port.
- Foreign-exchange rates, conversion math, and money libraries.
- Replacing decimal-unaware amounts with fixed-point amounts.
- A currency on each actual-cost point.
- A response that lists one subtotal per currency.
- Changing allocation when every priced currency is empty (that path still uses USD).
- Rejecting one billing record because its billing currency differs from its pricing
  currency.
- Rewriting currency-conversion examples that do not describe this summary.
- Marking a larger multi-currency roadmap item done as if those excluded items had shipped.
