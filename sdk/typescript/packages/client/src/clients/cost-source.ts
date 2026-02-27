import { createClient, Client, Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { create } from "@bufbuild/protobuf";
import {
  BatchCostRequest,
  BatchCostResponse,
  CostSourceService,
  GetActualCostRequest,
  GetActualCostRequestSchema,
  GetActualCostResponse,
  GetProjectedCostRequest,
  GetProjectedCostRequestSchema,
  GetProjectedCostResponse,
  GetRecommendationsRequest,
  GetRecommendationsRequestSchema,
  GetRecommendationsResponse,
  GetPricingSpecRequest,
  GetPricingSpecRequestSchema,
  GetPricingSpecResponse,
  EstimateCostRequest,
  EstimateCostResponse,
  DismissRecommendationRequest,
  DismissRecommendationResponse,
  GetPluginInfoRequest,
  GetPluginInfoRequestSchema,
  GetPluginInfoResponse,
  DryRunRequest,
  DryRunRequestSchema,
  DryRunResponse,
  NameRequest,
  NameRequestSchema,
  NameResponse,
  SupportsRequest,
  SupportsRequestSchema,
  SupportsResponse
} from "../generated/finfocus/v1/costsource_pb.js";
import {
  GetBudgetsRequest,
  GetBudgetsRequestSchema,
  GetBudgetsResponse
} from "../generated/finfocus/v1/budget_pb.js";
import { ValidationError } from "../errors/validation-error.js";
import { MAX_BATCH_SIZE } from "../utils/batch.js";

export interface CostSourceClientConfig {
  baseUrl: string;
  transport?: Transport;
}

export class CostSourceClient {
  private client: Client<typeof CostSourceService>;

  constructor(config: CostSourceClientConfig) {
    const transport = config.transport || createConnectTransport({
      baseUrl: config.baseUrl,
      useBinaryFormat: false,
    });
    this.client = createClient(CostSourceService, transport);
  }

  async name(req: NameRequest = create(NameRequestSchema)): Promise<NameResponse> {
    return this.client.name(req);
  }

  async supports(req: SupportsRequest = create(SupportsRequestSchema)): Promise<SupportsResponse> {
    return this.client.supports(req);
  }

  async getActualCost(req: GetActualCostRequest): Promise<GetActualCostResponse> {
    if (!req.resourceId && !req.arn) {
        throw new ValidationError("Resource ID or ARN is required", "resourceId");
    }
    return this.client.getActualCost(req);
  }

  async getProjectedCost(req: GetProjectedCostRequest): Promise<GetProjectedCostResponse> {
    if (!req.resource) throw new ValidationError("Resource is required", "resource");
    return this.client.getProjectedCost(req);
  }

  async getPricingSpec(req: GetPricingSpecRequest = create(GetPricingSpecRequestSchema)): Promise<GetPricingSpecResponse> {
    return this.client.getPricingSpec(req);
  }

  async estimateCost(req: EstimateCostRequest): Promise<EstimateCostResponse> {
    return this.client.estimateCost(req);
  }

  async getRecommendations(req: GetRecommendationsRequest = create(GetRecommendationsRequestSchema)): Promise<GetRecommendationsResponse> {
    return this.client.getRecommendations(req);
  }

  async dismissRecommendation(req: DismissRecommendationRequest): Promise<DismissRecommendationResponse> {
    if (!req.recommendationId) throw new ValidationError("Recommendation ID is required", "recommendationId");
    return this.client.dismissRecommendation(req);
  }

  async getBudgets(req: GetBudgetsRequest = create(GetBudgetsRequestSchema)): Promise<GetBudgetsResponse> {
    return this.client.getBudgets(req);
  }

  async getPluginInfo(req: GetPluginInfoRequest = create(GetPluginInfoRequestSchema)): Promise<GetPluginInfoResponse> {
    return this.client.getPluginInfo(req);
  }

  async dryRun(req: DryRunRequest = create(DryRunRequestSchema)): Promise<DryRunResponse> {
    return this.client.dryRun(req);
  }

  async batchCost(req: BatchCostRequest): Promise<BatchCostResponse> {
    const resourceCount = req.resources?.length ?? 0;
    if (resourceCount > MAX_BATCH_SIZE) {
      throw new ValidationError(
        `Batch size ${resourceCount} exceeds maximum supported size ${MAX_BATCH_SIZE}`,
        "resources",
      );
    }

    return this.client.batchCost(req);
  }
}
