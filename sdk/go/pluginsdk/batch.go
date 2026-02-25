// Copyright 2024 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pluginsdk

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	// DefaultMaxBatchSize is the default max number of resources per batch request.
	DefaultMaxBatchSize = 100
	// MaxBatchSize is the hard upper limit for max batch size configuration.
	MaxBatchSize = 1000
	// DefaultBatchWorkers is the default number of concurrent workers for fallback processing.
	DefaultBatchWorkers = 10
	// MinBatchWorkers is the minimum allowed fallback worker count.
	MinBatchWorkers = 1
	// MaxBatchWorkers is the maximum allowed fallback worker count.
	MaxBatchWorkers = 50
)

// allCostQueryTypes is allocated once for zero-allocation validation of CostQueryType values.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allCostQueryTypes = []pbc.CostQueryType{
	pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED,
	pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
	pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
	pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
}

// IsValidCostQueryType returns true if queryType is a known CostQueryType enum value.
func IsValidCostQueryType(queryType pbc.CostQueryType) bool {
	for _, valid := range allCostQueryTypes {
		if queryType == valid {
			return true
		}
	}
	return false
}

// NormalizeCostQueryType normalizes query type values for backward compatibility.
// UNSPECIFIED is treated as ESTIMATE.
func NormalizeCostQueryType(queryType pbc.CostQueryType) pbc.CostQueryType {
	if queryType == pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED {
		return pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE
	}
	return queryType
}

// ValidateBatchCostRequest validates a batch request and returns an eager empty response for empty batches.
//
// Returns:
//   - (*BatchCostResponse, nil): for valid empty batch requests
//   - (nil, nil): for valid non-empty requests
//   - (nil, error): for invalid requests
func ValidateBatchCostRequest(req *pbc.BatchCostRequest, maxBatchSize int32) (*pbc.BatchCostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if maxBatchSize <= 0 {
		maxBatchSize = DefaultMaxBatchSize
	}

	resourceCount := len(req.GetResources())
	if resourceCount == 0 {
		return NewBatchCostResponse(
			WithBatchResults([]*pbc.ResourceCostResult{}),
			WithMaxBatchSize(maxBatchSize),
		), nil
	}

	if int64(resourceCount) > int64(maxBatchSize) {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"batch size %d exceeds max_batch_size %d",
			resourceCount,
			maxBatchSize,
		)
	}

	queryType := req.GetQueryType()
	if !IsValidCostQueryType(queryType) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid query_type: %d", queryType)
	}

	if NormalizeCostQueryType(queryType) != pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL {
		//nolint:nilnil // Non-empty valid requests intentionally return (nil, nil)
		return nil, nil
	}

	start := req.GetStart()
	end := req.GetEnd()
	if start == nil || end == nil {
		return nil, status.Error(codes.InvalidArgument, "start and end are required for ACTUAL query type")
	}

	if !start.AsTime().Before(end.AsTime()) {
		return nil, status.Error(codes.InvalidArgument, "start must be before end for ACTUAL query type")
	}

	//nolint:nilnil // Non-empty valid requests intentionally return (nil, nil)
	return nil, nil
}

// BatchCostResponseOption is a functional option for BatchCostResponse.
type BatchCostResponseOption func(*pbc.BatchCostResponse)

// WithBatchResults sets response results.
func WithBatchResults(results []*pbc.ResourceCostResult) BatchCostResponseOption {
	return func(resp *pbc.BatchCostResponse) {
		resp.Results = results
	}
}

// WithMaxBatchSize sets response max_batch_size.
func WithMaxBatchSize(maxBatchSize int32) BatchCostResponseOption {
	return func(resp *pbc.BatchCostResponse) {
		resp.MaxBatchSize = maxBatchSize
	}
}

// NewBatchCostResponse creates a BatchCostResponse using functional options.
func NewBatchCostResponse(opts ...BatchCostResponseOption) *pbc.BatchCostResponse {
	resp := &pbc.BatchCostResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}

// NewResourceError constructs a ResourceError from a gRPC status code and message.
func NewResourceError(code codes.Code, message string, resourceTypeUnsupported bool) *pbc.ResourceError {
	if message == "" {
		message = code.String()
	}
	return &pbc.ResourceError{
		Code:                    grpcCodeToInt32(code),
		Message:                 message,
		ResourceTypeUnsupported: resourceTypeUnsupported,
	}
}

func grpcCodeToInt32(code codes.Code) int32 {
	codeValue := int64(code)
	if codeValue < 0 || codeValue > math.MaxInt32 {
		return int32(codes.Internal)
	}
	return int32(codeValue)
}

func resourceErrorFromGRPCError(err error) *pbc.ResourceError {
	if err == nil {
		return NewResourceError(codes.Unknown, "unknown error", false)
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		ctxStatus := status.FromContextError(err)
		return NewResourceError(ctxStatus.Code(), ctxStatus.Message(), false)
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		return NewResourceError(codes.Internal, err.Error(), false)
	}

	return NewResourceError(grpcStatus.Code(), grpcStatus.Message(), false)
}

func resourceErrorUnsupported(resourceType string) *pbc.ResourceError {
	message := "resource type is not supported"
	if resourceType != "" {
		message = fmt.Sprintf("resource type %q is not supported", resourceType)
	}
	return NewResourceError(codes.Unimplemented, message, true)
}

func resolveBatchSize(configured int) int32 {
	switch {
	case configured <= 0:
		return DefaultMaxBatchSize
	case configured > MaxBatchSize:
		return MaxBatchSize
	default:
		return int32(configured)
	}
}

func resolveBatchWorkers(configured int) int {
	switch {
	case configured <= 0:
		return DefaultBatchWorkers
	case configured < MinBatchWorkers:
		return MinBatchWorkers
	case configured > MaxBatchWorkers:
		return MaxBatchWorkers
	default:
		return configured
	}
}

func batchCostFallback(
	ctx context.Context,
	plugin Plugin,
	req *pbc.BatchCostRequest,
	maxBatchSize int32,
	workers int,
) *pbc.BatchCostResponse {
	resources := req.GetResources()
	results := make([]*pbc.ResourceCostResult, len(resources))
	queryType := NormalizeCostQueryType(req.GetQueryType())

	semaphore := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for idx, resource := range resources {
		wg.Add(1)
		go func(index int, descriptor *pbc.ResourceDescriptor) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = newResourceErrorResult(
					descriptor,
					resourceErrorFromGRPCError(ctx.Err()),
				)
				return
			}

			results[index] = batchCostForResource(ctx, plugin, req, descriptor, queryType)
		}(idx, descriptorClone(resource))
	}

	wg.Wait()

	return NewBatchCostResponse(
		WithBatchResults(results),
		WithMaxBatchSize(maxBatchSize),
	)
}

func batchCostForResource(
	ctx context.Context,
	plugin Plugin,
	req *pbc.BatchCostRequest,
	resource *pbc.ResourceDescriptor,
	queryType pbc.CostQueryType,
) *pbc.ResourceCostResult {
	if resource == nil {
		return newResourceErrorResult(
			resource,
			NewResourceError(codes.InvalidArgument, "resource is required", false),
		)
	}

	if req.GetDryRun() {
		return batchDryRunForResource(plugin, resource)
	}

	switch queryType {
	case pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL:
		return batchActualCostForResource(ctx, plugin, req, resource)
	case pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED:
		return batchProjectedCostForResource(ctx, plugin, resource)
	case pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED:
		// NormalizeCostQueryType maps UNSPECIFIED → ESTIMATE before this function is called;
		// UNSPECIFIED is listed here for exhaustive switch compliance.
		return batchEstimateCostForResource(ctx, plugin, resource)
	default:
		return newResourceErrorResult(
			resource,
			NewResourceError(codes.InvalidArgument, "invalid query type", false),
		)
	}
}

func batchEstimateCostForResource(
	ctx context.Context,
	plugin Plugin,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	resp, err := plugin.EstimateCost(ctx, &pbc.EstimateCostRequest{
		ResourceType: resource.GetResourceType(),
	})
	if err != nil {
		return newResourceErrorResult(resource, resourceErrorFromGRPCError(err))
	}
	if resp == nil {
		return newResourceErrorResult(
			resource,
			NewResourceError(codes.Internal, "EstimateCost returned nil response", false),
		)
	}

	return newResourceDataResult(resource, &pbc.CostData{
		Data: &pbc.CostData_Estimate{
			Estimate: resp,
		},
	})
}

func batchProjectedCostForResource(
	ctx context.Context,
	plugin Plugin,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	resp, err := plugin.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
		Resource: descriptorClone(resource),
	})
	if err != nil {
		return newResourceErrorResult(resource, resourceErrorFromGRPCError(err))
	}
	if resp == nil {
		return newResourceErrorResult(
			resource,
			NewResourceError(codes.Internal, "GetProjectedCost returned nil response", false),
		)
	}

	return newResourceDataResult(resource, &pbc.CostData{
		Data: &pbc.CostData_ProjectedCost{
			ProjectedCost: resp,
		},
	})
}

func batchActualCostForResource(
	ctx context.Context,
	plugin Plugin,
	req *pbc.BatchCostRequest,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	resp, err := plugin.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: defaultActualResourceID(resource),
		Start:      req.GetStart(),
		End:        req.GetEnd(),
		Tags:       copyStringMap(resource.GetTags()),
		Arn:        resource.GetArn(),
	})
	if err != nil {
		return newResourceErrorResult(resource, resourceErrorFromGRPCError(err))
	}
	if resp == nil {
		return newResourceErrorResult(
			resource,
			NewResourceError(codes.Internal, "GetActualCost returned nil response", false),
		)
	}

	return newResourceDataResult(resource, &pbc.CostData{
		Data: &pbc.CostData_ActualCost{
			ActualCost: &pbc.ActualCostData{
				Results:       resp.GetResults(),
				FallbackHint:  resp.GetFallbackHint(),
				NextPageToken: resp.GetNextPageToken(),
				TotalCount:    resp.GetTotalCount(),
			},
		},
	})
}

func batchDryRunForResource(plugin Plugin, resource *pbc.ResourceDescriptor) *pbc.ResourceCostResult {
	handler, ok := plugin.(DryRunHandler)
	if !ok {
		// Plugin does not implement DryRunHandler: return all-unsupported mappings.
		// configuration_valid is true because the plugin itself is functional;
		// it simply has no dry-run introspection capability.
		return newResourceDataResult(resource, &pbc.CostData{
			Data: &pbc.CostData_DryRunResult{
				DryRunResult: NewDryRunResponse(
					WithFieldMappings(AllFieldsWithStatus(
						pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_UNSUPPORTED,
					)),
					WithResourceTypeSupported(false),
					WithConfigurationValid(true),
				),
			},
		})
	}

	resp, err := handler.HandleDryRun(&pbc.DryRunRequest{
		Resource: descriptorClone(resource),
	})
	if err != nil {
		return newResourceErrorResult(resource, resourceErrorFromGRPCError(err))
	}
	if resp == nil {
		return newResourceErrorResult(
			resource,
			NewResourceError(codes.Internal, "HandleDryRun returned nil response", false),
		)
	}

	return newResourceDataResult(resource, &pbc.CostData{
		Data: &pbc.CostData_DryRunResult{
			DryRunResult: resp,
		},
	})
}

func defaultActualResourceID(resource *pbc.ResourceDescriptor) string {
	if resource.GetId() != "" {
		return resource.GetId()
	}
	if resource.GetArn() != "" {
		return resource.GetArn()
	}
	if resource.GetResourceType() != "" {
		return resource.GetResourceType()
	}
	return "unknown-resource"
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func descriptorClone(resource *pbc.ResourceDescriptor) *pbc.ResourceDescriptor {
	if resource == nil {
		return nil
	}
	// proto.Clone preserves the concrete type, so this assertion is safe.
	//nolint:errcheck // Type assertion guaranteed by proto.Clone contract.
	return proto.Clone(resource).(*pbc.ResourceDescriptor)
}

func newResourceDataResult(resource *pbc.ResourceDescriptor, data *pbc.CostData) *pbc.ResourceCostResult {
	return &pbc.ResourceCostResult{
		Resource: descriptorClone(resource),
		Result: &pbc.ResourceCostResult_CostData{
			CostData: data,
		},
	}
}

func newResourceErrorResult(resource *pbc.ResourceDescriptor, resultErr *pbc.ResourceError) *pbc.ResourceCostResult {
	return &pbc.ResourceCostResult{
		Resource: descriptorClone(resource),
		Result: &pbc.ResourceCostResult_Error{
			Error: resultErr,
		},
	}
}
