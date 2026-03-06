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
	"sync"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/internal/grpcconv"
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

// DoS-guard limits for ResourceDescriptor field validation.
// These are intentionally more generous than the contract validation limits
// in testing/contract.go (e.g., MaxTagCount=50, MaxResourceIDLength=512),
// which enforce stricter bounds for well-formed data. These limits only
// prevent unbounded allocations from malicious or malformed input.
const (
	// MaxTagsPerResource is the maximum number of tags allowed per ResourceDescriptor.
	MaxTagsPerResource = 256
	// MaxProviderLength is the maximum length for the provider field.
	MaxProviderLength = 32
	// MaxResourceTypeLength is the maximum length for the resource_type field.
	MaxResourceTypeLength = 256
	// MaxIDLength is the maximum length for the id field.
	MaxIDLength = 1024
	// MaxARNLength is the maximum length for the arn field.
	MaxARNLength = 2048
	// MaxSKULength is the maximum length for the sku field.
	MaxSKULength = 128
	// MaxRegionLength is the maximum length for the region field.
	MaxRegionLength = 64
	// MaxTagKeyLength is the maximum length for tag keys.
	MaxTagKeyLength = 128
	// MaxTagValueLength is the maximum length for tag values.
	MaxTagValueLength = 256
)

const (
	fieldBatchResourceCount           = "resource_count"
	fieldBatchQueryType               = "query_type"
	fieldBatchDryRun                  = "dry_run"
	fieldBatchResultCount             = "result_count"
	fieldBatchErrorCount              = "error_count"
	fieldBatchResultIndex             = "result_index"
	fieldBatchResourceID              = "resource_id"
	fieldBatchResourceTypeUnsupported = "resource_type_unsupported"
	fieldBatchMaxBatchSize            = "max_batch_size"
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

// IsValidCostQueryType reports whether queryType is one of the recognized pbc.CostQueryType values.
func IsValidCostQueryType(queryType pbc.CostQueryType) bool {
	for _, valid := range allCostQueryTypes {
		if queryType == valid {
			return true
		}
	}
	return false
}

// NormalizeCostQueryType normalizes query type values for backward compatibility.
// NormalizeCostQueryType maps an UNSPECIFIED CostQueryType to COST_QUERY_TYPE_ESTIMATE.
// It returns the provided queryType unchanged for all other values.
func NormalizeCostQueryType(queryType pbc.CostQueryType) pbc.CostQueryType {
	if queryType == pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED {
		return pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE
	}
	return queryType
}

// ValidateResourceDescriptor validates individual ResourceDescriptor fields against defined limits.
// It checks string field lengths and tag map constraints to prevent denial-of-service scenarios
// from unbounded allocations during batch processing.
//
// This function validates field sizes for DoS prevention only. Semantic validation of field
// values (e.g., recognized provider names, valid ARN formats) is the plugin's responsibility.
// These limits are intentionally more generous than the contract validation limits in the
// testing package (testing/contract.go), which enforce stricter bounds for well-formed data.
//
// Validation rules:
//   - resource must not be nil
//   - provider must not exceed MaxProviderLength (32 characters)
//   - resource_type must not exceed MaxResourceTypeLength (256 characters)
//   - id must not exceed MaxIDLength (1024 characters)
//   - arn must not exceed MaxARNLength (2048 characters)
//   - sku must not exceed MaxSKULength (128 characters)
//   - region must not exceed MaxRegionLength (64 characters)
//   - tags map must not exceed MaxTagsPerResource (256 entries)
//   - tag keys must not exceed MaxTagKeyLength (128 characters)
//   - tag values must not exceed MaxTagValueLength (256 characters)
//
// Returns nil if validation passes, or a gRPC InvalidArgument error describing the violation.
func ValidateResourceDescriptor(resource *pbc.ResourceDescriptor) error {
	if resource == nil {
		return status.Error(codes.InvalidArgument, "resource descriptor must not be nil")
	}

	provider := resource.GetProvider()
	if len(provider) > MaxProviderLength {
		return status.Errorf(
			codes.InvalidArgument,
			"provider length %d exceeds maximum %d",
			len(provider),
			MaxProviderLength,
		)
	}

	resourceType := resource.GetResourceType()
	if len(resourceType) > MaxResourceTypeLength {
		return status.Errorf(
			codes.InvalidArgument,
			"resource_type length %d exceeds maximum %d",
			len(resourceType),
			MaxResourceTypeLength,
		)
	}

	id := resource.GetId()
	if len(id) > MaxIDLength {
		return status.Errorf(
			codes.InvalidArgument,
			"id length %d exceeds maximum %d",
			len(id),
			MaxIDLength,
		)
	}

	arn := resource.GetArn()
	if len(arn) > MaxARNLength {
		return status.Errorf(
			codes.InvalidArgument,
			"arn length %d exceeds maximum %d",
			len(arn),
			MaxARNLength,
		)
	}

	sku := resource.GetSku()
	if len(sku) > MaxSKULength {
		return status.Errorf(
			codes.InvalidArgument,
			"sku length %d exceeds maximum %d",
			len(sku),
			MaxSKULength,
		)
	}

	region := resource.GetRegion()
	if len(region) > MaxRegionLength {
		return status.Errorf(
			codes.InvalidArgument,
			"region length %d exceeds maximum %d",
			len(region),
			MaxRegionLength,
		)
	}

	tags := resource.GetTags()
	if len(tags) > MaxTagsPerResource {
		return status.Errorf(
			codes.InvalidArgument,
			"tag count %d exceeds maximum %d",
			len(tags),
			MaxTagsPerResource,
		)
	}

	for key, value := range tags {
		if len(key) > MaxTagKeyLength {
			return status.Errorf(
				codes.InvalidArgument,
				"tag key %q: length %d exceeds maximum %d",
				key,
				len(key),
				MaxTagKeyLength,
			)
		}
		if len(value) > MaxTagValueLength {
			return status.Errorf(
				codes.InvalidArgument,
				"tag value for key %q: length %d exceeds maximum %d",
				key,
				len(value),
				MaxTagValueLength,
			)
		}
	}

	return nil
}

// BatchCostValidationResult represents the outcome of validating a BatchCostRequest.
// If EarlyResponse is non-nil, the caller should return it immediately without further processing.
// If EarlyResponse is nil and no error occurred, the caller should proceed with normal processing.
type BatchCostValidationResult struct {
	// EarlyResponse is set when validation determines an empty response should be
	// returned immediately (e.g., for empty resource lists).
	EarlyResponse *pbc.BatchCostResponse
}

// ValidateBatchCostRequest validates a BatchCostRequest and returns a BatchCostValidationResult
// indicating whether the request is valid and if an early response should be returned.
//
// Validation rules:
// - req must be non-nil.
// - If maxBatchSize <= 0, DefaultMaxBatchSize is applied.
// - If the resource list is empty, returns a ValidationResult with EarlyResponse set.
// - The number of resources must not exceed maxBatchSize.
// - Each ResourceDescriptor is validated via ValidateResourceDescriptor (field lengths, tag constraints).
// - query_type must be a valid CostQueryType.
// - For ACTUAL queries, both start and end must be present and start must be strictly before end.
//
// Return values:
// - BatchCostValidationResult with EarlyResponse set: when an eager empty response should be returned.
// - BatchCostValidationResult with EarlyResponse nil: when the request is valid and should be processed normally.
// - error: when the request is invalid.
func ValidateBatchCostRequest(req *pbc.BatchCostRequest, maxBatchSize int32) (BatchCostValidationResult, error) {
	if req == nil {
		return BatchCostValidationResult{}, status.Error(codes.InvalidArgument, "request is required")
	}

	if maxBatchSize <= 0 {
		maxBatchSize = DefaultMaxBatchSize
	}

	resourceCount := len(req.GetResources())
	if resourceCount == 0 {
		return BatchCostValidationResult{
			EarlyResponse: NewBatchCostResponse(
				WithBatchResults([]*pbc.ResourceCostResult{}),
				WithMaxBatchSize(maxBatchSize),
			),
		}, nil
	}

	if int64(resourceCount) > int64(maxBatchSize) {
		return BatchCostValidationResult{}, status.Errorf(
			codes.InvalidArgument,
			"batch size %d exceeds max_batch_size %d",
			resourceCount,
			maxBatchSize,
		)
	}

	// Validate each resource descriptor for field length and tag constraints.
	for i, resource := range req.GetResources() {
		if err := ValidateResourceDescriptor(resource); err != nil {
			// ValidateResourceDescriptor always returns gRPC status errors,
			// so status.FromError is guaranteed to succeed here.
			s, _ := status.FromError(err)
			return BatchCostValidationResult{}, status.Errorf(
				codes.InvalidArgument,
				"resource at index %d: %s",
				i,
				s.Message(),
			)
		}
	}

	queryType := req.GetQueryType()
	if !IsValidCostQueryType(queryType) {
		return BatchCostValidationResult{}, status.Errorf(codes.InvalidArgument, "invalid query_type: %d", queryType)
	}

	if NormalizeCostQueryType(queryType) != pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL {
		return BatchCostValidationResult{}, nil
	}

	start := req.GetStart()
	end := req.GetEnd()
	if start == nil || end == nil {
		return BatchCostValidationResult{}, status.Error(
			codes.InvalidArgument,
			"start and end are required for ACTUAL query type",
		)
	}

	if !start.AsTime().Before(end.AsTime()) {
		return BatchCostValidationResult{}, status.Error(
			codes.InvalidArgument,
			"start must be before end for ACTUAL query type",
		)
	}

	return BatchCostValidationResult{}, nil
}

// BatchCostResponseOption is a functional option for BatchCostResponse.
type BatchCostResponseOption func(*pbc.BatchCostResponse)

// WithBatchResults returns a BatchCostResponseOption that sets the Results field of a BatchCostResponse to the provided slice.
func WithBatchResults(results []*pbc.ResourceCostResult) BatchCostResponseOption {
	return func(resp *pbc.BatchCostResponse) {
		resp.Results = results
	}
}

// WithMaxBatchSize returns a BatchCostResponseOption that sets the MaxBatchSize field on a BatchCostResponse to the provided value.
func WithMaxBatchSize(maxBatchSize int32) BatchCostResponseOption {
	return func(resp *pbc.BatchCostResponse) {
		resp.MaxBatchSize = maxBatchSize
	}
}

// NewBatchCostResponse creates a BatchCostResponse and applies the provided functional options.
// Options are applied in the order they are passed; if none are provided an empty response is returned.
func NewBatchCostResponse(opts ...BatchCostResponseOption) *pbc.BatchCostResponse {
	resp := &pbc.BatchCostResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}

// NewResourceError constructs a pbc.ResourceError with the provided gRPC status code, message, and
// resource-type unsupported flag. If message is empty, the code's string representation is used
// for the Message field. The Code field is stored as an int32 suitable for the protobuf representation.
func NewResourceError(code codes.Code, message string, resourceTypeUnsupported bool) *pbc.ResourceError {
	if message == "" {
		message = code.String()
	}
	return &pbc.ResourceError{
		Code:                    grpcconv.CodeToInt32(code),
		Message:                 message,
		ResourceTypeUnsupported: resourceTypeUnsupported,
	}
}

// resourceErrorFromGRPCError converts a Go error into a ResourceError.
// It handles nil errors (mapped to Unknown), context cancellation/deadline errors,
// gRPC status errors, and plain errors by extracting the code and message.
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

// resourceErrorUnsupported constructs a ResourceError indicating the specified resource type is not supported.
// If resourceType is non-empty the message includes the quoted type. The returned error uses the gRPC
// Unimplemented code and sets the resource type unsupported flag to true.
func resourceErrorUnsupported(resourceType string) *pbc.ResourceError {
	message := "resource type is not supported"
	if resourceType != "" {
		message = fmt.Sprintf("resource type %q is not supported", resourceType)
	}
	return NewResourceError(codes.Unimplemented, message, true)
}

// resolveBatchSize returns the configured batch size adjusted to the package limits.
// If configured is less than or equal to zero, DefaultMaxBatchSize is returned.
// If configured is greater than MaxBatchSize, MaxBatchSize is returned.
// Otherwise the configured value is returned as an int32.
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

// resolveBatchWorkers returns the number of workers to use for batching.
// If configured is < MinBatchWorkers it returns DefaultBatchWorkers. Values above
// MaxBatchWorkers are lowered to MaxBatchWorkers; otherwise the configured value
// is returned.
func resolveBatchWorkers(configured int) int {
	switch {
	case configured < MinBatchWorkers:
		return DefaultBatchWorkers
	case configured > MaxBatchWorkers:
		return MaxBatchWorkers
	default:
		return configured
	}
}

// logBatchCostRequest records an incoming batch request with key request attributes.
// It returns the timestamp to use for completion latency logging.
func logBatchCostRequest(logger zerolog.Logger, req *pbc.BatchCostRequest) time.Time {
	queryType := pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED
	resourceCount := 0
	dryRun := false

	if req != nil {
		queryType = NormalizeCostQueryType(req.GetQueryType())
		resourceCount = len(req.GetResources())
		dryRun = req.GetDryRun()
	}

	logger.Info().
		Int(fieldBatchResourceCount, resourceCount).
		Str(fieldBatchQueryType, queryType.String()).
		Bool(fieldBatchDryRun, dryRun).
		Msg("BatchCost request received")

	return time.Now()
}

// logBatchCostResourceErrors logs per-resource errors at warn level and returns the number of error results.
func logBatchCostResourceErrors(logger zerolog.Logger, results []*pbc.ResourceCostResult) int {
	errorCount := 0

	for index, result := range results {
		if result == nil {
			errorCount++
			logger.Warn().
				Int(fieldBatchResultIndex, index).
				Msg("BatchCost resource failed: nil result entry")
			continue
		}

		resultErr := result.GetError()
		if resultErr == nil {
			continue
		}

		errorCount++
		event := logger.Warn().
			Int(fieldBatchResultIndex, index).
			Int32(FieldErrorCode, resultErr.GetCode()).
			Str("error_message", resultErr.GetMessage()).
			Bool(fieldBatchResourceTypeUnsupported, resultErr.GetResourceTypeUnsupported())

		resource := result.GetResource()
		if resource != nil {
			event = event.
				Str(FieldProvider, resource.GetProvider()).
				Str(FieldResourceType, resource.GetResourceType())
			if resource.GetId() != "" {
				event = event.Str(fieldBatchResourceID, resource.GetId())
			}
		}

		event.Msg("BatchCost resource failed")
	}

	return errorCount
}

// logBatchCostCompletion records batch completion with timing and summary metrics.
func logBatchCostCompletion(
	logger zerolog.Logger,
	startedAt time.Time,
	resp *pbc.BatchCostResponse,
	errorCount int,
) {
	if resp == nil {
		return
	}

	logger.Info().
		Int(fieldBatchResultCount, len(resp.GetResults())).
		Int(fieldBatchErrorCount, errorCount).
		Int32(fieldBatchMaxBatchSize, resp.GetMaxBatchSize()).
		Int64(FieldDurationMs, time.Since(startedAt).Milliseconds()).
		Msg("BatchCost completed")
}

// batchCostFallback processes each resource in req in parallel (bounded by workers),
// invoking batchCostForResource for each and collecting per-resource results into a
// BatchCostResponse. It respects ctx cancellation for individual work items by
// recording a ResourceError when the context is done and always returns a response
// populated with the per-resource results and the provided maxBatchSize.
//
// IMPORTANT: req is read-only; the caller skips proto.Clone before passing it here.
// All downstream functions must not modify req. Query type normalization is performed
// into a local variable, never written back to req.
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
		}(idx, resource)
	}

	wg.Wait()

	return NewBatchCostResponse(
		WithBatchResults(results),
		WithMaxBatchSize(maxBatchSize),
	)
}

// batchCostForResource determines cost information for a single resource according to the requested
// CostQueryType and returns a ResourceCostResult containing either cost data or a ResourceError.
// It returns an InvalidArgument error result if the resource is nil or the query type is invalid.
// If the request requests a dry run, a dry-run result is returned. Query types are handled as:
// ACTUAL -> actual cost data; PROJECTED -> projected cost data; ESTIMATE or UNSPECIFIED -> estimate data.
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
		return batchDryRunForResource(ctx, plugin, resource)
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

// batchEstimateCostForResource calls the plugin's EstimateCost for the given resource type and returns a ResourceCostResult.
//
// Note: Only resource.GetResourceType() is forwarded to EstimateCostRequest.
// ResourceDescriptor attributes (sku, region, tags) are not propagated to the
// underlying EstimateCost call.
//
// On success the result contains CostData with the Estimate returned by the plugin. If the plugin returns an error the result
// contains a ResourceError derived from that gRPC error. If the plugin returns a nil response the result contains an Internal
// ResourceError with the message "EstimateCost returned nil response".
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

// batchProjectedCostForResource requests projected cost for the provided resource from the plugin
// and returns a ResourceCostResult containing either the ProjectedCost data or a ResourceError.
// If the plugin returns an error it is converted to a ResourceError via resourceErrorFromGRPCError;
// a nil response is treated as an internal error.
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

// batchActualCostForResource retrieves actual cost data for the given resource by calling the plugin's GetActualCost
// and returns a ResourceCostResult containing ActualCost on success.
//
// On plugin errors the result contains a ResourceError derived from the gRPC/error context; if the plugin returns a nil
// response the result contains an internal ResourceError indicating the nil response. The returned ResourceCostResult
// always includes a clone of the provided resource descriptor.
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

// batchDryRunForResource produces a ResourceCostResult containing dry-run information for the given resource.
// If the plugin implements DryRunHandler, it invokes HandleDryRun and returns the handler's DryRunResult; if the handler returns an error or a nil response, the result contains a corresponding ResourceError.
// If the plugin does not implement DryRunHandler, the result indicates all fields are unsupported, resourceTypeSupported is false, and configurationValid is true.
// The context is checked for cancellation before invoking the handler.
func batchDryRunForResource(
	ctx context.Context,
	plugin Plugin,
	resource *pbc.ResourceDescriptor,
) *pbc.ResourceCostResult {
	if err := ctx.Err(); err != nil {
		return newResourceErrorResult(resource, resourceErrorFromGRPCError(err))
	}

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

	resp, err := handler.HandleDryRun(ctx, &pbc.DryRunRequest{
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

// defaultActualResourceID returns a best-effort identifier for the given resource descriptor,
// preferring Id, then Arn, then ResourceType, and falling back to "unknown-resource" if none are set.
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

// copyStringMap returns a shallow copy of in, or nil if in is nil or has no entries.
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

// descriptorClone returns a deep copy of the given ResourceDescriptor.
// If resource is nil, descriptorClone returns nil.
func descriptorClone(resource *pbc.ResourceDescriptor) *pbc.ResourceDescriptor {
	if resource == nil {
		return nil
	}
	// proto.Clone preserves the concrete type, so this assertion is safe.
	//nolint:errcheck // Type assertion guaranteed by proto.Clone contract.
	return proto.Clone(resource).(*pbc.ResourceDescriptor)
}

// newResourceDataResult creates a ResourceCostResult that contains a deep-cloned ResourceDescriptor and the given CostData.
// The input descriptor is cloned to avoid mutating the original; the CostData is stored in the Result field.
func newResourceDataResult(resource *pbc.ResourceDescriptor, data *pbc.CostData) *pbc.ResourceCostResult {
	return &pbc.ResourceCostResult{
		Resource: descriptorClone(resource),
		Result: &pbc.ResourceCostResult_CostData{
			CostData: data,
		},
	}
}

// newResourceErrorResult creates a ResourceCostResult containing a deep-cloned
// ResourceDescriptor and the given ResourceError as the result.
func newResourceErrorResult(resource *pbc.ResourceDescriptor, resultErr *pbc.ResourceError) *pbc.ResourceCostResult {
	return &pbc.ResourceCostResult{
		Resource: descriptorClone(resource),
		Result: &pbc.ResourceCostResult_Error{
			Error: resultErr,
		},
	}
}
