# Tasks: FOCUS 1.4 Billing Period and Invoice Detail

**Input**: Design documents from `specs/546-focus-14-billing-invoice/`

**Prerequisites**: plan.md, spec.md

**Tests**: REQUIRED. Conformance and validator cases land with the implementation.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Messages

- [x] T001 Add `BillingPeriod`, `InvoiceDetail`, and the two status enums to
  `proto/finfocus/v1/focus.proto`, then regenerate.
- [x] T002 [US1] Add `ValidateBillingPeriod` and `BillingPeriodBuilder`. Simple setters and the
  status check allocate nothing on the success path.
- [x] T003 [US2] Add `ValidateInvoiceDetail` and `InvoiceDetailBuilder`, including the refund
  rejection, the settlement-currency pair, and zero-versus-absent cost.

## Phase 2: Other SDKs

- [x] T004 [P] [US3] Serialize both records in JSON-LD, including a present zero settlement cost,
  and document the columns in `docs/focus-columns.md`.
- [x] T005 [P] [US3] Add TypeScript builders that return a clone, and cover them with vitest.

## Phase 3: Conformance

- [x] T006 [US1] [US2] Assert field numbers, a valid baseline, refund and unspecified category
  rejection, and the settlement-currency pair in `focus14_conformance_test.go`.
