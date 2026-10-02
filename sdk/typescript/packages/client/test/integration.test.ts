import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from './mocks/handlers.js';
import { create } from '@bufbuild/protobuf';
import { CostSourceClient } from '../src/clients/cost-source.js';
import { ObservabilityClient } from '../src/clients/auxiliary.js';
import { RegistryClient } from '../src/clients/auxiliary.js';
import { ResourceDescriptorBuilder } from '../src/builders/resource-descriptor.js';
import { RecommendationFilterBuilder } from '../src/builders/recommendation-filter.js';
import { FocusRecordBuilder } from '../src/builders/focus-record.js';
import { recommendationsIterator } from '../src/utils/pagination.js';
import { ValidationError } from '../src/errors/validation-error.js';
import {
  GetActualCostRequestSchema,
  GetProjectedCostRequestSchema,
  GetRecommendationsRequestSchema,
  DismissRecommendationRequestSchema,
  NameRequestSchema,
  SupportsRequestSchema,
  EstimateCostRequestSchema,
  GetPricingSpecRequestSchema,
  GetPluginInfoRequestSchema,
  DryRunRequestSchema,
  HealthCheckResponse_Status
} from '../src/generated/finfocus/v1/costsource_pb.js';
import {
  GetBudgetsRequestSchema
} from '../src/generated/finfocus/v1/budget_pb.js';
import { RecommendationPriority } from '../src/generated/finfocus/v1/costsource_pb.js';
import { FocusPricingCategory } from '../src/generated/finfocus/v1/enums_pb.js';
import type { PriceOption } from '../src/generated/finfocus/v1/costsource_pb.js';

function expectPriceOptions(options: PriceOption[]) {
  expect(options).toHaveLength(2);
  expect(options[0].category).toBe(FocusPricingCategory.COMMITTED);
  expect(options[0].model).toBe("Reservation");
  expect(options[0].term).toBe("1 Year");
  expect(options[0].unitPrice).toBe(0.0573);
  expect(options[0].monthlyCost).toBe(41.83);
  expect(options[0].upfrontCost).toBe(502.0);
  expect(options[0].savingsFraction).toBe(0.403125);
  expect(options[1].model).toBe("SavingsPlan");
  expect(options[1].term).toBe("3 Years");
  expect(options[1].upfrontCost).toBe(0);
}

const server = setupServer(...handlers);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('CostSourceClient Integration', () => {
  const client = new CostSourceClient({
    baseUrl: 'https://plugin-aws.example.com'
  });

  it('fetches plugin name successfully', async () => {
    const request = create(NameRequestSchema);
    const response = await client.name(request);
    expect(response.name).toBe("AWS Cost Plugin");
  });

  it('fetches supported capabilities', async () => {
    const request = create(SupportsRequestSchema);
    const response = await client.supports(request);
    expect(response.supported).toBe(true);
    expect(response.capabilities["recommendations"]).toBe(true);
  });

  it('fetches actual cost successfully using ID', async () => {
    const request = create(GetActualCostRequestSchema, { resourceId: 'i-1234567890abcdef0' });
    const response = await client.getActualCost(request);

    expect(response.results).toBeDefined();
    expect(response.results.length).toBeGreaterThan(0);
    expect(response.results[0].cost).toBe(100.0);
  });

  it('reads FOCUS 1.4 cost and usage columns from the actual cost FOCUS record', async () => {
    const request = create(GetActualCostRequestSchema, { resourceId: 'i-1234567890abcdef0' });
    const response = await client.getActualCost(request);

    const record = response.results[0].focusRecord;
    expect(record).toBeDefined();
    expect(record?.serviceProviderName).toBe('AWS');
    expect(record?.providerName).toBe('');
    expect(record?.invoiceId).toBe('INV-2026-09');
    expect(record?.invoiceDetailId).toBe('INV-2026-09-L3');
    expect(JSON.parse(record?.commitmentProgramEligibilityDetails ?? '')).toEqual({
      CommitmentPrograms: [{ ProgramType: 'Savings Plan' }],
    });
  });

  it('fetches projected cost successfully using ResourceDescriptor', async () => {
    const resource = new ResourceDescriptorBuilder()
      .withProvider('AWS')
      .withResourceType('ec2.instance')
      .withRegion('us-east-1')
      .withId('i-1234567890abcdef0')
      .build();

    const request = create(GetProjectedCostRequestSchema, { resource });
    const response = await client.getProjectedCost(request);

    expect(response.costPerMonth).toBe(150.0);
    expect(response.currency).toBe("USD");
    expect(response.costBreakdown.compute).toBe(120.0);
    expect(response.costBreakdown.root_volume).toBe(30.0);
    const componentSum = Object.values(response.costBreakdown).reduce((sum, v) => sum + v, 0);
    expect(componentSum).toBeCloseTo(response.costPerMonth, 9);
    expectPriceOptions(response.priceOptions);

    // region_prices is advisory: rows are read as sent and never change costPerMonth.
    expect(response.regionPrices).toHaveLength(2);
    expect(response.regionPrices[0]).toMatchObject({
      region: "us-west-2", unitPrice: 0.11, monthlyCost: 165.0, currency: "USD",
    });
    expect(response.regionPrices[1]).toMatchObject({
      region: "eu-west-1", unitPrice: 0.12, monthlyCost: 180.0, currency: "EUR",
    });
    expect(response.costPerMonth).toBe(150.0);
  });

  it('fetches pricing specification', async () => {
    const request = create(GetPricingSpecRequestSchema);
    const response = await client.getPricingSpec(request);
    expect(response.spec).toBeDefined();
    expect(response.spec?.provider).toBe("AWS");
  });

  it('estimates cost for a resource', async () => {
    const resource = new ResourceDescriptorBuilder()
      .withProvider('AWS')
      .withResourceType('ec2.instance')
      .build();

    const request = create(EstimateCostRequestSchema, { resource });
    const response = await client.estimateCost(request);

    expect(response.costMonthly).toBe(200.0);
    expect(response.currency).toBe("USD");
    expectPriceOptions(response.priceOptions);
  });

  it('fetches recommendations', async () => {
    const request = create(GetRecommendationsRequestSchema);
    const response = await client.getRecommendations(request);

    expect(response.recommendations).toBeDefined();
    expect(response.recommendations.length).toBeGreaterThan(0);
    expect(response.recommendations[0].description).toBe("Downsize Instance to save costs");
  });

  it('fetches recommendations with filter', async () => {
    const filter = new RecommendationFilterBuilder()
      .forProvider('AWS')
      .withPriority(RecommendationPriority.HIGH)
      .build();

    const request = create(GetRecommendationsRequestSchema, { filter });
    const response = await client.getRecommendations(request);

    expect(response.recommendations).toBeDefined();
  });

  it('iterates through paginated recommendations', async () => {
    const request = create(GetRecommendationsRequestSchema);
    const recommendations: any[] = [];

    for await (const rec of recommendationsIterator(client, request)) {
      recommendations.push(rec);
    }

    expect(recommendations.length).toBeGreaterThan(0);
  });

  it('dismisses a recommendation', async () => {
    const request = create(DismissRecommendationRequestSchema, { recommendationId: 'rec-1' });
    const response = await client.dismissRecommendation(request);
    expect(response.success).toBe(true);
  });

  it('throws ValidationError when dismissing recommendation without ID', async () => {
    const request = create(DismissRecommendationRequestSchema);
    await expect(client.dismissRecommendation(request)).rejects.toThrow(ValidationError);
  });

  it('fetches budgets', async () => {
    const request = create(GetBudgetsRequestSchema);
    const response = await client.getBudgets(request);
    expect(response.budgets).toBeDefined();
    expect(response.budgets.length).toBeGreaterThan(0);
  });

  it('fetches plugin info', async () => {
    const request = create(GetPluginInfoRequestSchema);
    const response = await client.getPluginInfo(request);
    expect(response.name).toBe("AWS Cost Plugin");
    expect(response.version).toBe("1.0.0");
  });

  it('performs DryRun check', async () => {
    const request = create(DryRunRequestSchema);
    const response = await client.dryRun(request);
    expect(response.resourceTypeSupported).toBe(true);
    expect(response.configurationValid).toBe(true);
  });
});

describe('ResourceDescriptorBuilder', () => {
  it('builds descriptor with all properties', () => {
    const descriptor = new ResourceDescriptorBuilder()
      .withProvider('AWS')
      .withResourceType('ec2.instance')
      .withRegion('us-west-2')
      .withSku('m5.large')
      .withArn('arn:aws:ec2:us-west-2:123456789012:instance/i-1234567890abcdef0')
      .withTags({ environment: 'production', team: 'platform' })
      .build();

    expect(descriptor.provider).toBe('AWS');
    expect(descriptor.resourceType).toBe('ec2.instance');
    expect(descriptor.region).toBe('us-west-2');
    expect(descriptor.sku).toBe('m5.large');
    expect(descriptor.arn).toBe('arn:aws:ec2:us-west-2:123456789012:instance/i-1234567890abcdef0');
    expect(descriptor.tags).toEqual({ environment: 'production', team: 'platform' });
  });

  it('supports fluent API chaining', () => {
    const descriptor = new ResourceDescriptorBuilder()
      .withProvider('Azure')
      .withResourceType('virtual_machine')
      .build();

    expect(descriptor.provider).toBe('Azure');
    expect(descriptor.resourceType).toBe('virtual_machine');
  });
});

describe('FocusRecordBuilder', () => {
  it('builds FOCUS record with billing information', () => {
    const now = new Date();
    const record = new FocusRecordBuilder()
      .withBilledCost(100.50, 'USD')
      .withBillingPeriod(new Date(now.getFullYear(), now.getMonth(), 1), now)
      .withResourceId('i-1234567890abcdef0')
      .withProvider('AWS')
      .build();

    expect(record.billedCost).toBe(100.50);
    expect(record.billingCurrency).toBe('USD');
    expect(record.resourceId).toBe('i-1234567890abcdef0');
    expect(record.providerName).toBe('AWS');
  });

  it('sets FOCUS 1.4 invoice detail and commitment program eligibility', () => {
    const details = '{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}';
    const record = new FocusRecordBuilder()
      .withInvoiceDetailId('INV-2026-09-L3')
      .withCommitmentProgramEligibilityDetails(details)
      .build();

    expect(record.invoiceDetailId).toBe('INV-2026-09-L3');
    expect(record.commitmentProgramEligibilityDetails).toBe(details);
  });

  it('rejects an empty invoice detail ID', () => {
    expect(() => new FocusRecordBuilder().withInvoiceDetailId('  ')).toThrow(ValidationError);
  });

  it.each(['[]', 'null', '"x"', '{"CommitmentPrograms":[', '42'])(
    'rejects commitment program eligibility details that are not a JSON object: %s',
    (details) => {
      expect(() => new FocusRecordBuilder().withCommitmentProgramEligibilityDetails(details)).toThrow(
        ValidationError,
      );
    },
  );
});

describe('ObservabilityClient', () => {
  const client = new ObservabilityClient({
    baseUrl: 'https://plugin-aws.example.com'
  });

  it('checks plugin health', async () => {
    const response = await client.healthCheck();
    expect(response.status).toBe(HealthCheckResponse_Status.SERVING);
  });
});

describe('RegistryClient', () => {
  const client = new RegistryClient({
    baseUrl: 'https://plugin-registry.example.com'
  });

  it('discovers available plugins', async () => {
    const response = await client.discoverPlugins();
    expect(response.plugins).toBeDefined();
    expect(response.plugins.length).toBeGreaterThan(0);
  });

  it('lists installed plugins', async () => {
    const response = await client.listInstalledPlugins();
    expect(response.plugins).toBeDefined();
    expect(response.plugins.length).toBeGreaterThan(0);
  });
});
