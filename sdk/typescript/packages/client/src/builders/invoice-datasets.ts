import { clone, create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { FocusChargeCategory } from "../generated/finfocus/v1/enums_pb.js";
import {
  BillingPeriod,
  BillingPeriodSchema,
  FocusBillingPeriodStatus,
  FocusInvoiceIssueStatus,
  InvoiceDetail,
  InvoiceDetailSchema,
} from "../generated/finfocus/v1/focus_pb.js";

/**
 * Builds a FOCUS 1.4 BillingPeriod.
 * Build returns a clone and does not re-run the Go validator.
 * Hosts that need those rules call the Go SDK.
 */
export class BillingPeriodBuilder {
  private record: BillingPeriod;

  constructor() {
    this.record = create(BillingPeriodSchema);
  }

  withWindow(start: Date, end: Date): this {
    this.record.billingPeriodStart = timestampFromDate(start);
    this.record.billingPeriodEnd = timestampFromDate(end);
    return this;
  }

  withStatus(status: FocusBillingPeriodStatus): this {
    this.record.billingPeriodStatus = status;
    return this;
  }

  withInvoiceIssuerName(name: string): this {
    this.record.invoiceIssuerName = name;
    return this;
  }

  withCreated(created: Date): this {
    this.record.billingPeriodCreated = timestampFromDate(created);
    return this;
  }

  withLastUpdated(updated: Date): this {
    this.record.billingPeriodLastUpdated = timestampFromDate(updated);
    return this;
  }

  build(): BillingPeriod {
    return clone(BillingPeriodSchema, this.record);
  }
}

/**
 * Builds a FOCUS 1.4 InvoiceDetail.
 * Build returns a clone and does not re-run the Go validator.
 * Hosts that need those rules call the Go SDK.
 * A zero settlement cost is present. Leaving it unset means the column is absent.
 */
export class InvoiceDetailBuilder {
  private record: InvoiceDetail;

  constructor() {
    this.record = create(InvoiceDetailSchema);
  }

  withIdentity(detailId: string, invoiceId: string, issuer: string, accountId: string): this {
    this.record.invoiceDetailId = detailId;
    this.record.invoiceId = invoiceId;
    this.record.invoiceIssuerName = issuer;
    this.record.billingAccountId = accountId;
    return this;
  }

  withBillingPeriod(start: Date, end: Date): this {
    this.record.billingPeriodStart = timestampFromDate(start);
    this.record.billingPeriodEnd = timestampFromDate(end);
    return this;
  }

  withBilledCost(cost: number): this {
    this.record.billedCost = cost;
    return this;
  }

  withBillingCurrency(code: string): this {
    this.record.billingCurrency = code;
    return this;
  }

  withChargeCategory(category: FocusChargeCategory): this {
    this.record.chargeCategory = category;
    return this;
  }

  withIssueStatus(status: FocusInvoiceIssueStatus): this {
    this.record.invoiceIssueStatus = status;
    return this;
  }

  withIssueDate(issued: Date): this {
    this.record.invoiceIssueDate = timestampFromDate(issued);
    return this;
  }

  withCreated(created: Date): this {
    this.record.invoiceDetailCreated = timestampFromDate(created);
    return this;
  }

  withLastUpdated(updated: Date): this {
    this.record.invoiceDetailLastUpdated = timestampFromDate(updated);
    return this;
  }

  withDescription(description: string): this {
    this.record.invoiceDetailDescription = description;
    return this;
  }

  withGrain(grain: Record<string, string>): this {
    this.record.invoiceDetailGrain = copyStringMap(grain);
    return this;
  }

  withPaymentCurrency(code: string): this {
    this.record.paymentCurrency = code;
    return this;
  }

  /** Sets a present settlement cost, including zero. */
  withPaymentCurrencyBilledCost(cost: number): this {
    this.record.paymentCurrencyBilledCost = cost;
    return this;
  }

  withPaymentCurrencyInvoiceDetailId(id: string): this {
    this.record.paymentCurrencyInvoiceDetailId = id;
    return this;
  }

  withPaymentDueDate(due: Date): this {
    this.record.paymentDueDate = timestampFromDate(due);
    return this;
  }

  withPaymentTerms(terms: string): this {
    this.record.paymentTerms = terms;
    return this;
  }

  withPurchaseOrderNumber(number: string): this {
    this.record.purchaseOrderNumber = number;
    return this;
  }

  withReferenceInvoiceId(id: string): this {
    this.record.referenceInvoiceId = id;
    return this;
  }

  withExtendedColumns(columns: Record<string, string>): this {
    this.record.extendedColumns = copyStringMap(columns);
    return this;
  }

  /**
   * Fills the columns that do not allow nulls: a USD usage line for one month,
   * status Issued, terms "Net 30", and a reference id equal to the invoice id.
   * Settlement currency stays absent.
   */
  withBaseline(at: Date): this {
    const end = new Date(at.getTime());
    end.setUTCMonth(end.getUTCMonth() + 1);
    return this.withIdentity("detail-1", "invoice-1", "Example Issuer", "account-1")
      .withBillingPeriod(at, end)
      .withBilledCost(10)
      .withBillingCurrency("USD")
      .withChargeCategory(FocusChargeCategory.USAGE)
      .withIssueStatus(FocusInvoiceIssueStatus.ISSUED)
      .withCreated(at)
      .withLastUpdated(at)
      .withPaymentTerms("Net 30")
      .withReferenceInvoiceId("invoice-1");
  }

  build(): InvoiceDetail {
    return clone(InvoiceDetailSchema, this.record);
  }
}

function copyStringMap(input: Record<string, string>): { [key: string]: string } {
  const out: { [key: string]: string } = {};
  for (const key of Object.keys(input)) {
    out[key] = input[key];
  }
  return out;
}
