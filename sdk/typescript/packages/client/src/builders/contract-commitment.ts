import { clone, create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { ValidationError } from "../errors/validation-error.js";
import {
  ContractCommitment,
  ContractCommitmentSchema,
  FocusContractCommitmentBenefitCategory,
  FocusContractCommitmentCategory,
  FocusContractCommitmentFulfillmentInterval,
  FocusContractCommitmentLifecycleStatus,
  FocusContractCommitmentModel,
  FocusContractCommitmentOfferCategory,
  FocusContractCommitmentPaymentInterval,
  FocusContractCommitmentPaymentModel,
} from "../generated/finfocus/v1/focus_pb.js";

/**
 * One entry in a FOCUS 1.4 ContractApplied Elements array.
 * A null or omitted applied cost or quantity is left out of the JSON.
 * Zero is present and is written as 0.
 */
export interface ContractAppliedElement {
  contractId: string;
  commitmentId: string;
  appliedCost?: number | null;
  appliedQuantity?: number | null;
  appliedUnit?: string | null;
}

/** Builds the FOCUS 1.4 ContractAppliedObject JSON: {"Elements":[...]}. */
export function formatContractApplied(elements: ContractAppliedElement[]): string {
  if (elements.length === 0) {
    throw new ValidationError("Contract applied requires at least one element", "contractApplied");
  }
  const payload = {
    Elements: elements.map((element, index) => appliedElement(element, index)),
  };
  return JSON.stringify(payload);
}

function appliedElement(element: ContractAppliedElement, index: number): Record<string, unknown> {
  if (!element.contractId || element.contractId.trim() === "") {
    throw new ValidationError(
      `Contract applied element ${index} requires a contract ID`,
      "contractApplied",
    );
  }
  if (!element.commitmentId || element.commitmentId.trim() === "") {
    throw new ValidationError(
      `Contract applied element ${index} requires a commitment ID`,
      "contractApplied",
    );
  }
  const costPresent = typeof element.appliedCost === "number";
  const quantityPresent = typeof element.appliedQuantity === "number";
  const unitPresent = typeof element.appliedUnit === "string" && element.appliedUnit.trim() !== "";
  const unitProvided = element.appliedUnit !== undefined && element.appliedUnit !== null;
  if (!costPresent && !quantityPresent) {
    throw new ValidationError(
      `Contract applied element ${index} requires a cost or a quantity`,
      "contractApplied",
    );
  }
  if (costPresent && !Number.isFinite(element.appliedCost)) {
    throw new ValidationError(
      `Contract applied element ${index} cost must be a finite number`,
      "contractApplied",
    );
  }
  if (quantityPresent && !Number.isFinite(element.appliedQuantity)) {
    throw new ValidationError(
      `Contract applied element ${index} quantity must be a finite number`,
      "contractApplied",
    );
  }
  if (quantityPresent && !unitPresent) {
    throw new ValidationError(
      `Contract applied element ${index} requires a unit when quantity is set`,
      "contractApplied",
    );
  }
  if (!quantityPresent && unitProvided) {
    throw new ValidationError(
      `Contract applied element ${index} unit requires a quantity`,
      "contractApplied",
    );
  }
  return {
    ContractId: element.contractId,
    ContractCommitmentId: element.commitmentId,
    ...(costPresent ? { ContractCommitmentAppliedCost: element.appliedCost } : {}),
    ...(quantityPresent
      ? {
          ContractCommitmentAppliedQuantity: element.appliedQuantity,
          ContractCommitmentAppliedUnit: element.appliedUnit,
        }
      : {}),
  };
}

/**
 * Builds a FOCUS 1.4 ContractCommitment. Build does not re-run the Go validator;
 * hosts that need those rules call the Go SDK.
 */
export class ContractCommitmentBuilder {
  private record: ContractCommitment;

  constructor() {
    this.record = create(ContractCommitmentSchema);
  }

  withIdentity(commitmentId: string, contractId: string): this {
    this.record.contractCommitmentId = commitmentId;
    this.record.contractId = contractId;
    return this;
  }

  withCategory(category: FocusContractCommitmentCategory): this {
    this.record.contractCommitmentCategory = category;
    return this;
  }

  withBillingCurrency(currencyCode: string): this {
    this.record.billingCurrency = currencyCode;
    return this;
  }

  withApplicability(applicabilityJson: string): this {
    let parsed: unknown;
    try {
      parsed = JSON.parse(applicabilityJson);
    } catch {
      throw new ValidationError(
        "Contract commitment applicability must be well-formed JSON",
        "contractCommitmentApplicability",
      );
    }
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new ValidationError(
        "Contract commitment applicability must be a JSON object",
        "contractCommitmentApplicability",
      );
    }
    this.record.contractCommitmentApplicability = applicabilityJson;
    return this;
  }

  withBenefitCategory(category: FocusContractCommitmentBenefitCategory): this {
    this.record.contractCommitmentBenefitCategory = category;
    return this;
  }

  /** Sets a present discount fraction. Zero is not null. */
  withDiscountPercentage(percentage: number): this {
    this.record.contractCommitmentDiscountPercentage = percentage;
    return this;
  }

  withDurationType(duration: string): this {
    this.record.contractCommitmentDurationType = duration;
    return this;
  }

  withFulfillmentInterval(interval: FocusContractCommitmentFulfillmentInterval): this {
    this.record.contractCommitmentFulfillmentInterval = interval;
    return this;
  }

  withLifecycleStatus(status: FocusContractCommitmentLifecycleStatus): this {
    this.record.contractCommitmentLifecycleStatus = status;
    return this;
  }

  withModel(model: FocusContractCommitmentModel): this {
    this.record.contractCommitmentModel = model;
    return this;
  }

  withOfferCategory(category: FocusContractCommitmentOfferCategory): this {
    this.record.contractCommitmentOfferCategory = category;
    return this;
  }

  withPaymentInterval(interval: FocusContractCommitmentPaymentInterval): this {
    this.record.contractCommitmentPaymentInterval = interval;
    return this;
  }

  withPaymentModel(model: FocusContractCommitmentPaymentModel): this {
    this.record.contractCommitmentPaymentModel = model;
    return this;
  }

  /** Sets a present upfront fraction. Zero is not null. */
  withPaymentUpfrontPercentage(percentage: number): this {
    this.record.contractCommitmentPaymentUpfrontPercentage = percentage;
    return this;
  }

  withCreated(created: Date): this {
    this.record.contractCommitmentCreated = timestampFromDate(created);
    return this;
  }

  withLastUpdated(updated: Date): this {
    this.record.contractCommitmentLastUpdated = timestampFromDate(updated);
    return this;
  }

  withInvoiceIssuerName(name: string): this {
    this.record.invoiceIssuerName = name;
    return this;
  }

  withServiceProviderName(name: string): this {
    this.record.serviceProviderName = name;
    return this;
  }

  withDescription(description: string): this {
    this.record.contractCommitmentDescription = description;
    return this;
  }

  build(): ContractCommitment {
    return clone(ContractCommitmentSchema, this.record);
  }
}
