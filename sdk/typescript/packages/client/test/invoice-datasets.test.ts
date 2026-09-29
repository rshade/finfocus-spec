import { describe, expect, it } from "vitest";
import { BillingPeriodBuilder, InvoiceDetailBuilder } from "../src/index.js";
import { FocusChargeCategory } from "../src/generated/finfocus/v1/enums_pb.js";
import {
  FocusBillingPeriodStatus,
  FocusInvoiceIssueStatus,
} from "../src/generated/finfocus/v1/focus_pb.js";

describe("FOCUS 1.4 billing period and invoice detail", () => {
  const at = new Date(Date.UTC(2026, 0, 1));

  it("builds a billing period and keeps the clone isolated", () => {
    const end = new Date(Date.UTC(2026, 1, 1));
    const builder = new BillingPeriodBuilder()
      .withWindow(at, end)
      .withStatus(FocusBillingPeriodStatus.OPEN)
      .withInvoiceIssuerName("Example Issuer")
      .withCreated(at)
      .withLastUpdated(at);
    const built = builder.build();
    builder.withInvoiceIssuerName("Other Issuer");

    expect(built.invoiceIssuerName).toBe("Example Issuer");
    expect(built.billingPeriodStatus).toBe(FocusBillingPeriodStatus.OPEN);
    expect(built.billingPeriodStart).toBeDefined();
  });

  it("keeps a zero settlement cost distinct from an omitted one", () => {
    const present = new InvoiceDetailBuilder()
      .withBaseline(at)
      .withPaymentCurrency("EUR")
      .withPaymentCurrencyBilledCost(0)
      .build();
    const absent = new InvoiceDetailBuilder().withBaseline(at).build();

    expect(present.paymentCurrencyBilledCost).toBe(0);
    expect(present.billedCost).toBe(10);
    expect(absent.paymentCurrencyBilledCost).toBeUndefined();
    expect(absent.chargeCategory).toBe(FocusChargeCategory.USAGE);
    expect(absent.invoiceIssueStatus).toBe(FocusInvoiceIssueStatus.ISSUED);
  });

  it("returns a clone so a later setter does not change a built invoice line", () => {
    const builder = new InvoiceDetailBuilder().withBaseline(at);
    const built = builder.build();
    builder.withIdentity("detail-2", "invoice-2", "Other Issuer", "account-2");
    expect(built.invoiceDetailId).toBe("detail-1");
    expect(built.invoiceId).toBe("invoice-1");
  });

  it("copies grain so a later mutation of the input does not change the record", () => {
    const grain = { ServiceName: "Compute", x_BillingMode: "on-demand" };
    const built = new InvoiceDetailBuilder().withBaseline(at).withGrain(grain).build();
    grain.ServiceName = "Storage";
    expect(built.invoiceDetailGrain.ServiceName).toBe("Compute");
    expect(built.invoiceDetailGrain.x_BillingMode).toBe("on-demand");
  });
});
