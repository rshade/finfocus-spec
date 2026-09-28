import { http, HttpResponse } from 'msw';

export const handlers = [
  // CostSourceService endpoints
  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/Name', () => {
    return HttpResponse.json({
      name: "AWS Cost Plugin"
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/Supports', () => {
    return HttpResponse.json({
      supported: true,
      reason: "",
      capabilities: { "recommendations": true, "dry_run": true },
      supportedMetrics: [],
      capabilitiesEnum: [4, 5] // PLUGIN_CAPABILITY_RECOMMENDATIONS, PLUGIN_CAPABILITY_DRY_RUN
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/GetActualCost', () => {
    return HttpResponse.json({
      results: [
        {
          cost: 100.0,
          usageAmount: 720,
          usageUnit: "hours",
          source: "AWS Cost Explorer",
          focusRecord: {
            serviceProviderName: "AWS",
            invoiceId: "INV-2026-09",
            invoiceDetailId: "INV-2026-09-L3",
            commitmentProgramEligibilityDetails: '{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}'
          }
        }
      ],
      fallbackHint: 1 // FALLBACK_HINT_NONE
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/GetProjectedCost', () => {
    return HttpResponse.json({
      unitPrice: 0.10,
      currency: "USD",
      costPerMonth: 150.0,
      billingDetail: "On-demand pricing",
      costBreakdown: { compute: 120.0, root_volume: 30.0 }
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/GetPricingSpec', () => {
    return HttpResponse.json({
      spec: {
        provider: "AWS",
        resourceType: "ec2.instance",
        billingMode: "HOURLY",
        ratePerUnit: 0.10,
        currency: "USD"
      }
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/EstimateCost', () => {
    return HttpResponse.json({
      currency: "USD",
      costMonthly: 200.0
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/GetRecommendations', () => {
    return HttpResponse.json({
      recommendations: [
        {
          id: "rec-1",
          description: "Downsize Instance to save costs",
          category: 1, // RECOMMENDATION_CATEGORY_COST
          actionType: 1, // RECOMMENDATION_ACTION_TYPE_RIGHTSIZE
          impact: {
            estimatedSavings: 50.0,
            currency: "USD",
            projectionPeriod: "monthly"
          },
          priority: 3 // RECOMMENDATION_PRIORITY_HIGH
        }
      ],
      nextPageToken: ""
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/DismissRecommendation', () => {
    return HttpResponse.json({
      success: true
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/GetBudgets', () => {
    return HttpResponse.json({
      budgets: [
        {
          id: "budget-1",
          name: "Monthly Budget",
          source: "aws-budgets",
          amount: {
            limit: 1000.0,
            currency: "USD"
          },
          period: 3, // BUDGET_PERIOD_MONTHLY
          status: {
            currentSpend: 750.0,
            percentageUsed: 75.0,
            currency: "USD",
            health: 1 // BUDGET_HEALTH_STATUS_OK
          }
        }
      ]
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/GetPluginInfo', () => {
    return HttpResponse.json({
      name: "AWS Cost Plugin",
      version: "1.0.0",
      specVersion: "v1",
      providers: ["aws"]
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.CostSourceService/DryRun', () => {
    return HttpResponse.json({
      fieldMappings: [],
      resourceTypeSupported: true,
      configurationValid: true
    });
  }),

  // ObservabilityService endpoints
  http.post('https://plugin-aws.example.com/finfocus.v1.ObservabilityService/HealthCheck', () => {
    return HttpResponse.json({
      status: 1, // HealthCheckResponse.Status.SERVING
      message: "Plugin is healthy"
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.ObservabilityService/GetMetrics', () => {
    return HttpResponse.json({
      metrics: []
    });
  }),

  http.post('https://plugin-aws.example.com/finfocus.v1.ObservabilityService/GetServiceLevelIndicators', () => {
    return HttpResponse.json({
      slis: []
    });
  }),

  // PluginRegistryService endpoints
  http.post('https://plugin-registry.example.com/finfocus.v1.PluginRegistryService/DiscoverPlugins', () => {
    return HttpResponse.json({
      plugins: [
        {
          id: "plugin-aws",
          name: "AWS Cost Plugin",
          version: "1.0.0"
        }
      ]
    });
  }),

  http.post('https://plugin-registry.example.com/finfocus.v1.PluginRegistryService/ListInstalledPlugins', () => {
    return HttpResponse.json({
      plugins: [
        {
          id: "plugin-aws",
          name: "AWS Cost Plugin",
          version: "1.0.0"
        }
      ]
    });
  })
];
