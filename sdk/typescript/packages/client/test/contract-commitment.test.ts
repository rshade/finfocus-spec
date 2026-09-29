import { describe, expect, it } from "vitest";
import {
  ContractCommitmentBuilder,
  FocusRecordBuilder,
  formatContractApplied,
} from "../src/index.js";
import {
  FocusContractCommitmentBenefitCategory,
  FocusContractCommitmentCategory,
} from "../src/generated/finfocus/v1/focus_pb.js";
import { ValidationError } from "../src/errors/validation-error.js";

describe("FOCUS 1.4 contract commitment", () => {
  it("keeps a zero discount distinct from an omitted one", () => {
    const present = new ContractCommitmentBuilder()
      .withIdentity("commit-1", "contract-1")
      .withCategory(FocusContractCommitmentCategory.SPEND)
      .withBillingCurrency("USD")
      .withDiscountPercentage(0)
      .build();
    const absent = new ContractCommitmentBuilder().withIdentity("commit-2", "contract-1").build();

    expect(present.contractCommitmentDiscountPercentage).toBe(0);
    expect(absent.contractCommitmentDiscountPercentage).toBeUndefined();
  });

  it("rejects applicability that is not a JSON object", () => {
    expect(() => new ContractCommitmentBuilder().withApplicability("[1]")).toThrow(ValidationError);
  });

  it("emits a ContractApplied object from the cost builder", () => {
    const record = new FocusRecordBuilder().withContractAppliedObject([
      { contractId: "contract-1", commitmentId: "commit-1", appliedCost: 12.5 },
    ]);
    expect(record.build().contractApplied).toBe(
      formatContractApplied([
        { contractId: "contract-1", commitmentId: "commit-1", appliedCost: 12.5 },
      ]),
    );
    expect(record.build().contractApplied).toContain('"ContractCommitmentId":"commit-1"');
    expect(record.build().contractApplied).toContain('"ContractId":"contract-1"');
    expect(record.build().contractApplied).toContain('"ContractCommitmentAppliedCost":12.5');
  });

  it("rejects an element without a contract ID or an applied metric", () => {
    expect(() =>
      formatContractApplied([{ contractId: "", commitmentId: "commit-1", appliedCost: 1 }]),
    ).toThrow(ValidationError);
    expect(() =>
      formatContractApplied([{ contractId: "contract-1", commitmentId: "commit-1" }]),
    ).toThrow(ValidationError);
    expect(() =>
      formatContractApplied([
        { contractId: "contract-1", commitmentId: "commit-1", appliedQuantity: 1 },
      ]),
    ).toThrow(ValidationError);
  });

  it("writes a zero cost and keeps both a cost and a quantity", () => {
    expect(
      formatContractApplied([{ contractId: "contract-1", commitmentId: "commit-1", appliedCost: 0 }]),
    ).toContain('"ContractCommitmentAppliedCost":0');
    const both = formatContractApplied([
      {
        contractId: "contract-1",
        commitmentId: "commit-1",
        appliedCost: 1,
        appliedQuantity: 2,
        appliedUnit: "Hours",
      },
    ]);
    expect(both).toContain('"ContractCommitmentAppliedQuantity":2');
    expect(both).toContain('"ContractCommitmentAppliedUnit":"Hours"');
  });

  it("returns a clone so a later setter does not change a built record", () => {
    const builder = new ContractCommitmentBuilder().withIdentity("commit-1", "contract-1");
    const built = builder.build();
    builder.withIdentity("commit-2", "contract-2");
    expect(built.contractCommitmentId).toBe("commit-1");
    expect(built.contractId).toBe("contract-1");
  });

  it("exports the benefit category enum", () => {
    expect(FocusContractCommitmentBenefitCategory.DISCOUNT).toBeGreaterThan(0);
  });
});
