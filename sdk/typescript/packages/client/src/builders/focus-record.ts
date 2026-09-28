import { create, clone } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { FocusCostRecord, FocusCostRecordSchema } from "../generated/finfocus/v1/focus_pb.js";
import { ValidationError } from "../errors/validation-error.js";

export class FocusRecordBuilder {
  private record: FocusCostRecord;

  constructor() {
    this.record = create(FocusCostRecordSchema);
  }

  withBilledCost(amount: number, currencyCode: string): this {
    if (!currencyCode || currencyCode.trim() === '') {
      throw new ValidationError("Currency code cannot be empty", "currencyCode");
    }
    this.record.billedCost = amount;
    this.record.billingCurrency = currencyCode;
    return this;
  }

  withBillingPeriod(start: Date, end: Date): this {
    if (end < start) {
      throw new ValidationError("Billing period end cannot be before start", "billingPeriod");
    }
    this.record.billingPeriodStart = timestampFromDate(start);
    this.record.billingPeriodEnd = timestampFromDate(end);
    return this;
  }

  withResourceId(id: string): this {
    if (!id || id.trim() === '') {
      throw new ValidationError("Resource ID cannot be empty", "resourceId");
    }
    this.record.resourceId = id;
    return this;
  }

  withProvider(provider: string): this {
    if (!provider || provider.trim() === '') {
      throw new ValidationError("Provider cannot be empty", "provider");
    }
    this.record.providerName = provider;
    return this;
  }

  /**
   * Sets the FOCUS 1.4 InvoiceDetailId: the invoice line item this cost row
   * contributes to. It is unique within an InvoiceId, so set invoiceId as well.
   */
  withInvoiceDetailId(id: string): this {
    if (!id || id.trim() === '') {
      throw new ValidationError("Invoice detail ID cannot be empty", "invoiceDetailId");
    }
    this.record.invoiceDetailId = id;
    return this;
  }

  /**
   * Sets the FOCUS 1.4 CommitmentProgramEligibilityDetails column, a JSON object such as
   * {"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}. Throws a ValidationError
   * unless the value parses to a JSON object (not an array, null, or a scalar).
   */
  withCommitmentProgramEligibilityDetails(detailsJson: string): this {
    let parsed: unknown;
    try {
      parsed = JSON.parse(detailsJson);
    } catch {
      throw new ValidationError(
        "Commitment program eligibility details must be well-formed JSON",
        "commitmentProgramEligibilityDetails",
      );
    }
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new ValidationError(
        "Commitment program eligibility details must be a JSON object",
        "commitmentProgramEligibilityDetails",
      );
    }
    this.record.commitmentProgramEligibilityDetails = detailsJson;
    return this;
  }

  build(): FocusCostRecord {
    // Create a copy to prevent mutation issues when reusing the builder
    return clone(FocusCostRecordSchema, this.record);
  }
}
