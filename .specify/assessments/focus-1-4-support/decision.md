# Decision: Delivery path for FOCUS supplemental datasets

- **Slug**: focus-1-4-support
- **Decided**: 2026-09-28
- **Verdict**: go (option 2, staged)
- **Artifacts reviewed**: intake.md | research.md, issue 544, the FOCUS 1.3 and 1.4 release
  notes, provider export documentation, and the OpenCost plugin protocol, plus
  `.specify/memory/constitution.md` for strategic fit

## Question

How should plugins deliver the FOCUS supplemental datasets to hosts: Contract Commitment (1.3),
and Billing Period and Invoice Detail (1.4)? Today `ContractCommitment` is schema-only. No RPC
returns it, and the phase 2 and phase 3 work would add about 45 more fields with no path onto the
wire.

## Options

1. **Message-only (status quo).** Hosts import the messages for their own pipelines.
2. **A new optional service** registered by `Serve` only when a plugin implements it, following
   `UsageSourceService` and `AllocatorService`.
3. **Response fields on `GetActualCost`.**

## Scorecard

| Criterion | Rating | Justification |
| --- | --- | --- |
| Problem validity | strong | A message that no RPC sends is untested protocol. Conformance can only exercise builders, and phases 2 and 3 would add about 45 fields in the same state. |
| Evidence strength | weak | No consumer exists: finfocus core has no Go references to these datasets and no issue asks for them. No plugin produces them. The FOCUS dataset directory lists AWS, Microsoft and Google Cloud exports at 1.2, and no provider has confirmed support for the 1.3 Contract Commitment dataset. Demand is stated (tracking FOCUS), not observed. |
| Value vs. inaction | adequate | Option 1 is cheap, but it ships dead schema and leaves the conformance gap open. A delivery path makes the datasets testable end to end and gives the first producer somewhere to plug in. |
| Feasibility / appetite | strong | Option 2 reuses a proven shape: optional provider interfaces, `optionalServices` in `Serve`, gRPC and Connect registration, capability inference, and page-token pagination from spec 044. Stage A needs no new data messages, because `ContractCommitment` already exists. |
| Strategic fit | strong | Interface-driven capability discovery (principle XII), additive MINOR changes (VI), SDK parity (XIII), and pass-through of pre-calculated data (III). FOCUS itself models these as separate datasets that join to Cost and Usage by key. |
| Risk posture | adequate | The main risk is building a service before a producer or consumer exists. Staging bounds it: stage A is one RPC over an existing message, and stage B proceeds only with a named producer or consumer. |

## Verdict & Rationale

**Go with option 2, staged.**

- **Option 3 is rejected.** A commitment contract, a billing period and an invoice line do not
  belong to a cost-query time window, pagination over cost rows would duplicate or split them,
  and FOCUS 1.4 gives each dataset its own Correction Handling (Replacement, Delta, Ledger) and
  Delivery Handling (Overwrite, Append). A shared response cannot express that.
- **Option 1 is rejected.** It leaves the "untested protocol" problem that issue 544 was opened
  to prevent, and grows it with every phase.
- **Option 2 matches both FOCUS and this repository.** FOCUS defines supplemental datasets that
  join to cost rows (`InvoiceId`, `InvoiceDetailId`). The repository already delivers optional
  plugin roles as separate, capability-gated services. OpenCost's plugin protocol, the closest
  prior art, has no path for supplemental datasets at all, so this is a gap worth closing.

Evidence strength is weak, which is why the build is staged rather than all at once.

## Handoff to `/speckit-specify`

- **Problem**: FOCUS supplemental datasets have no delivery path, so plugins cannot send them and
  conformance cannot test them.
- **Chosen approach**: a new optional service for FOCUS supplemental datasets.
  - Separate provider interfaces so capabilities stay independent: one for contract commitments
    now, and one for invoice data (Billing Period and Invoice Detail together, since they join)
    in stage B. Each maps to its own automatically discovered `PLUGIN_CAPABILITY_*` value.
  - `Serve` registers the service only when the plugin implements a provider, over gRPC and
    Connect, and adds it to the Connect health checker.
  - Requests take a time window and the existing `page_size` / `page_token` pagination.
- **In scope (stage A, issue 544)**: the service, `GetContractCommitments` over the existing
  `ContractCommitment` message, the provider interface and capability, SDK adapters, request and
  response validation, a mock-plugin reference producer, conformance, a TypeScript client, and
  docs.
- **Out of scope**: `GetBillingPeriods` and `GetInvoiceDetails` (stage B, with issue 543); the
  17 new Contract Commitment columns (issue 542), which travel over this RPC once added; changes
  to `GetActualCost`.
- **Stage B gate**: add the invoice RPCs with issue 543 only if a producer or consumer is named
  by then (for example Azure reservation and invoice exports through the planned azure-costmgmt
  plugin, or a finfocus core consumer). Otherwise Billing Period and Invoice Detail ship as
  messages only.
- **Success metrics**: a plugin implementing the provider interface is discovered with the new
  capability and serves paged commitments over gRPC and Connect; a plugin that does not
  implement it is unchanged; conformance runs the RPC end to end.
- **Carried-forward open questions**:
  - Whether FOCUS 1.4 Correction Handling and Delivery Handling belong on the response.
  - The request filter: billing-period window only, or also commitment status or ID.

## Sources

- [Introducing FOCUS 1.4](https://www.finops.org/insights/introducing-focus-1-4/)
- [Introducing FOCUS 1.3: Contract Commitments](https://www.finops.org/insights/introducing-focus-1-3/)
- [FinOps Foundation launches FOCUS 1.3, expanded vendor support for 1.2](https://www.linuxfoundation.org/press/finops-foundation-launches-focus-1.3-to-deepen-cloud-and-saas-billing-transparency-announces-expanded-vendor-support-for-focus-1.2)
- [AWS Data Exports for FOCUS 1.2 is now generally available](https://aws.amazon.com/about-aws/whats-new/2025/11/aws-data-exports-focus-1-2-available)
- [Microsoft Cost Management exports tutorial](https://learn.microsoft.com/en-us/azure/cost-management-billing/costs/tutorial-improved-exports)
- [opencost/opencost-plugins](https://github.com/opencost/opencost-plugins)
- Local: `rshade/finfocus` (no dataset references), `finfocus-plugin-*` repositories, and
  `specs/051-usage-source-getstats/` and `specs/052-allocator-allocate/` for the service pattern
