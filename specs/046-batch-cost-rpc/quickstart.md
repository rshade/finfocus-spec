# Quickstart: Batch Cost RPC

**Feature**: 046-batch-cost-rpc | **Date**: 2026-02-24

## Plugin Developer

### Option 1: Automatic Fallback (Zero Code Changes)

Plugins that do not implement `BatchCostHandler` automatically support batch requests
via the SDK's concurrent fallback. No code changes required.

```go
package main

import (
    "context"

    pbc "github.com/rshade/finfocus-spec/sdk/go/proto"
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

// MyPlugin implements the required Plugin interface only.
// BatchCost requests are automatically handled by the SDK fallback.
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }

func (p *MyPlugin) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (
    *pbc.GetActualCostResponse, error) {
    // Existing implementation - called per-resource during batch fallback
    return &pbc.GetActualCostResponse{}, nil
}

// ... other required methods ...

func main() {
    ctx := context.Background()
    info := pluginsdk.NewPluginInfo("my-plugin", "v1.0.0")
    // BatchCost capability is NOT reported; SDK handles requests via fallback
    pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin:     &MyPlugin{},
        PluginInfo: info,
    })
}
```

### Option 2: Optimized Batch Handler

Implement `BatchCostHandler` for plugins that can optimize multi-resource queries
(e.g., batching cloud provider API calls).

```go
package main

import (
    "context"

    pbc "github.com/rshade/finfocus-spec/sdk/go/proto"
    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

type MyOptimizedPlugin struct{}

func (p *MyOptimizedPlugin) Name() string { return "my-optimized-plugin" }

// BatchCost implements pluginsdk.BatchCostHandler for optimized batch processing.
func (p *MyOptimizedPlugin) BatchCost(ctx context.Context, req *pbc.BatchCostRequest) (
    *pbc.BatchCostResponse, error) {
    results := make([]*pbc.ResourceCostResult, len(req.Resources))

    for i, resource := range req.Resources {
        result := &pbc.ResourceCostResult{
            Resource: resource,
        }

        // Check if resource type is supported
        if !p.supportsResourceType(resource.ResourceType) {
            result.Result = &pbc.ResourceCostResult_Error{
                Error: &pbc.ResourceError{
                    Code:                    12, // UNIMPLEMENTED
                    Message:                 "unsupported resource type: " + resource.ResourceType,
                    ResourceTypeUnsupported: true,
                },
            }
        } else {
            // Perform optimized batch query to cloud provider
            estimate, err := p.queryCost(ctx, resource, req.QueryType)
            if err != nil {
                result.Result = &pbc.ResourceCostResult_Error{
                    Error: &pbc.ResourceError{
                        Code:    13, // INTERNAL
                        Message: err.Error(),
                    },
                }
            } else {
                result.Result = &pbc.ResourceCostResult_CostData{
                    CostData: &pbc.CostData{
                        Data: &pbc.CostData_Estimate{
                            Estimate: estimate,
                        },
                    },
                }
            }
        }

        results[i] = result
    }

    return &pbc.BatchCostResponse{
        Results:      results,
        MaxBatchSize: 100,
    }, nil
}

// ... required Plugin interface methods ...

func main() {
    ctx := context.Background()
    info := pluginsdk.NewPluginInfo("my-optimized-plugin", "v1.0.0")
    // PLUGIN_CAPABILITY_BATCH_COST auto-discovered from BatchCostHandler
    pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin:       &MyOptimizedPlugin{},
        PluginInfo:   info,
        MaxBatchSize: 200, // Override default of 100
    })
}
```

### Configuring Fallback Workers

```go
pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
    Plugin:       &MyPlugin{},
    PluginInfo:   info,
    BatchWorkers: 20, // Increase concurrent workers from default 10
})
```

## Host Developer

### Basic Batch Query

```go
package main

import (
    "context"
    "fmt"

    pbc "github.com/rshade/finfocus-spec/sdk/go/proto"
    "google.golang.org/grpc/codes"
)

func queryBatchCosts(ctx context.Context, client pbc.CostSourceServiceClient) error {
    resp, err := client.BatchCost(ctx, &pbc.BatchCostRequest{
        QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
        Resources: []*pbc.ResourceDescriptor{
            {Provider: "aws", ResourceType: "ec2", Id: "web-server-1"},
            {Provider: "aws", ResourceType: "rds", Id: "db-primary"},
            {Provider: "gcp", ResourceType: "compute_engine", Id: "worker-1"},
        },
    })
    if err != nil {
        return fmt.Errorf("batch cost query failed: %w", err)
    }

    for i, result := range resp.Results {
        switch r := result.Result.(type) {
        case *pbc.ResourceCostResult_CostData:
            estimate := r.CostData.GetEstimate()
            fmt.Printf("Resource %d (%s): $%.2f/month\n",
                i, result.Resource.Id, estimate.MonthlyCost)
        case *pbc.ResourceCostResult_Error:
            fmt.Printf("Resource %d (%s): ERROR [%d] %s\n",
                i, result.Resource.Id, r.Error.Code, r.Error.Message)
        }
    }

    return nil
}
```

### Batch Actual Cost Query with Time Range

```go
func queryBatchActualCosts(ctx context.Context, client pbc.CostSourceServiceClient,
    start, end *timestamppb.Timestamp) error {

    resp, err := client.BatchCost(ctx, &pbc.BatchCostRequest{
        QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
        Start:     start,
        End:       end,
        Resources: []*pbc.ResourceDescriptor{
            {Provider: "aws", ResourceType: "ec2", Id: "web-1"},
            {Provider: "aws", ResourceType: "s3", Id: "bucket-1"},
        },
    })
    if err != nil {
        return err
    }

    for _, result := range resp.Results {
        if data := result.GetCostData(); data != nil {
            actual := data.GetActualCost()
            fmt.Printf("%s: %d data points, total_count=%d\n",
                result.Resource.Id, len(actual.Results), actual.TotalCount)
        }
    }
    return nil
}
```

### Capability Discovery Before Batch

```go
func checkBatchSupport(ctx context.Context, client pbc.CostSourceServiceClient) (bool, int32) {
    info, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
    if err != nil {
        return false, 0
    }

    for _, cap := range info.Capabilities {
        if cap == pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST {
            // Check max batch size from metadata
            maxSize := int32(100) // default
            if v, ok := info.Metadata["max_batch_size"]; ok {
                // parse v to int32
                _ = v
            }
            return true, maxSize
        }
    }
    return false, 0
}
```

### Batch DryRun

```go
func batchDryRun(ctx context.Context, client pbc.CostSourceServiceClient) error {
    resp, err := client.BatchCost(ctx, &pbc.BatchCostRequest{
        QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
        DryRun:    true,
        Resources: []*pbc.ResourceDescriptor{
            {Provider: "aws", ResourceType: "ec2"},
            {Provider: "aws", ResourceType: "s3"},
        },
    })
    if err != nil {
        return err
    }

    for _, result := range resp.Results {
        if data := result.GetCostData(); data != nil {
            dryRun := data.GetDryRunResult()
            fmt.Printf("%s/%s: %d field mappings\n",
                result.Resource.Provider,
                result.Resource.ResourceType,
                len(dryRun.FieldMappings))
        }
    }
    return nil
}
```

## TypeScript Client

```typescript
import { createCostSourceClient } from '@finfocus/client';

const client = createCostSourceClient({ baseUrl: 'http://localhost:50051' });

// Basic batch estimate
const response = await client.batchCost({
  queryType: CostQueryType.ESTIMATE,
  resources: [
    { provider: 'aws', resourceType: 'ec2', id: 'web-1' },
    { provider: 'aws', resourceType: 'rds', id: 'db-1' },
  ],
});

for (const result of response.results) {
  if (result.costData) {
    const estimate = result.costData.estimate;
    console.log(`${result.resource?.id}: $${estimate?.monthlyCost}/month`);
  } else if (result.error) {
    console.error(`${result.resource?.id}: ${result.error.message}`);
  }
}
```
