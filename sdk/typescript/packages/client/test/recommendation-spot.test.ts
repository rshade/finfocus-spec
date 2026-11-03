import { describe, it, expect } from 'vitest';
import { create, toJson, fromJson } from '@bufbuild/protobuf';
import { RecommendationSchema } from '../src/generated/finfocus/v1/costsource_pb.js';
import { FocusPricingCategory } from '../src/generated/finfocus/v1/enums_pb.js';

describe('Recommendation spot fields', () => {
  it('defaults to an unspecified category and zero risk', () => {
    const rec = create(RecommendationSchema);
    expect(rec.pricingCategory).toBe(FocusPricingCategory.UNSPECIFIED);
    expect(rec.spotInterruptionRiskScore).toBe(0);
  });

  it('round trips a spot recommendation through proto3 JSON', () => {
    const rec = create(RecommendationSchema, {
      id: 'rec-1',
      pricingCategory: FocusPricingCategory.DYNAMIC,
      spotInterruptionRiskScore: 0.12,
    });
    const decoded = fromJson(RecommendationSchema, toJson(RecommendationSchema, rec));
    expect(decoded.pricingCategory).toBe(FocusPricingCategory.DYNAMIC);
    expect(decoded.spotInterruptionRiskScore).toBe(0.12);
  });
});
