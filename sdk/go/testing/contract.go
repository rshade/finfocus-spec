// Package testing provides contract validation for proto messages.
// This file contains shared contract tests that validate proto message
// compatibility across Core and Plugin repositories.
package testing

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Canonical provider name values shared across contract validation and mock data.
const (
	awsProviderName        = "aws"
	azureProviderName      = "azure"
	gcpProviderName        = "gcp"
	kubernetesProviderName = "kubernetes"
)

// Contract validation constants defining the limits and constraints
// for data exchange between Core and Plugin components. These constants
// ensure resource consumption is bounded and data formats are consistent.
const (
	// MaxResourceIDLength is the maximum allowed resource ID length.
	MaxResourceIDLength = 512

	// MaxTagKeyLength is the maximum allowed tag key length.
	MaxTagKeyLength = 128

	// MaxTagValueLength is the maximum allowed tag value length.
	MaxTagValueLength = 256

	// MaxTagCount is the maximum number of tags per resource.
	MaxTagCount = 50

	// MaxResourceTypeLength is the maximum resource type string length.
	MaxResourceTypeLength = 256

	// MinTimeRangeHours is the minimum time range for cost queries.
	MinTimeRangeHours = 1

	// MaxTimeRangeDays is the maximum time range for cost queries.
	MaxTimeRangeDays = 365

	// MaxPageSize is the maximum page size for paginated requests.
	MaxPageSize = 1000

	// DefaultPageSize is the default page size for paginated requests.
	DefaultPageSize = 50

	// FutureTimeGraceMinutes is the grace period for future time validation.
	FutureTimeGraceMinutes = 5

	// hoursPerDay is the number of hours in a day for time range calculations.
	// Defined locally to keep contract.go self-contained as shared contract tests.
	hoursPerDay = 24

	// MaxTargetResources is the maximum number of target resources allowed
	// in a GetRecommendationsRequest. This limit prevents unbounded memory
	// and processing costs while providing sufficient headroom for typical
	// Pulumi stacks (most have 10-50 resources).
	MaxTargetResources = 100
)

// Validation error types for contract testing.
var (
	ErrNilRequest                  = errors.New("request is nil")
	ErrNilResource                 = errors.New("resource descriptor is nil")
	ErrEmptyProvider               = errors.New("provider is required")
	ErrEmptyResourceType           = errors.New("resource_type is required")
	ErrInvalidProvider             = errors.New("invalid provider value")
	ErrEmptyResourceID             = errors.New("resource_id is required")
	ErrResourceIDTooLong           = errors.New("resource_id exceeds maximum length")
	ErrNilStartTime                = errors.New("start timestamp is required")
	ErrNilEndTime                  = errors.New("end timestamp is required")
	ErrInvalidTimeRange            = errors.New("end time must be after start time")
	ErrTimeRangeTooShort           = errors.New("time range is too short")
	ErrTimeRangeTooLong            = errors.New("time range exceeds maximum")
	ErrFutureStartTime             = errors.New("start time cannot be in the future")
	ErrInvalidPageSize             = errors.New("page_size exceeds maximum")
	ErrEmptyResourceTypeFmt        = errors.New("resource_type format is invalid")
	ErrTagKeyTooLong               = errors.New("tag key exceeds maximum length")
	ErrTagValueTooLong             = errors.New("tag value exceeds maximum length")
	ErrTooManyTags                 = errors.New("tag count exceeds maximum")
	ErrInvalidProjectionPeriod     = errors.New("invalid projection_period value")
	ErrTargetResourcesExceedsLimit = errors.New("target_resources exceeds maximum")
)

// ValidProviders is the list of valid provider values.
//
//nolint:gochecknoglobals // Intentional: shared validation data for contract testing.
var ValidProviders = []string{awsProviderName, azureProviderName, gcpProviderName, kubernetesProviderName, "custom"}

// ValidProjectionPeriods is the list of valid projection period values.
//
//nolint:gochecknoglobals // Intentional: shared validation data for contract testing.
var ValidProjectionPeriods = []string{"", "daily", "monthly", "annual"}

// resourceTypePattern validates resource type format.
// Accepts formats like: "ec2", "s3", "k8s-namespace", "aws:ec2/instance:Instance".
var resourceTypePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-:/]*$`)

// ContractError wraps a validation error with additional context.
type ContractError struct {
	Field   string
	Value   any
	Wrapped error
}

// NewContractError creates a new ContractError.
func NewContractError(field string, value any, err error) *ContractError {
	return &ContractError{
		Field:   field,
		Value:   value,
		Wrapped: err,
	}
}

// Error implements the error interface.
func (e *ContractError) Error() string {
	return fmt.Sprintf("contract violation: field=%s value=%v: %s", e.Field, e.Value, e.Wrapped.Error())
}

// Unwrap returns the underlying error.
func (e *ContractError) Unwrap() error {
	return e.Wrapped
}

// =============================================================================
// Request Validation Functions
// =============================================================================

// ValidateSupportsRequest validates a SupportsRequest message.
// This validates the contract between Core and Plugin for the Supports RPC.
func ValidateSupportsRequest(req *pbc.SupportsRequest) error {
	if req == nil {
		return ErrNilRequest
	}
	return ValidateResourceDescriptor(req.GetResource())
}

// ValidateGetActualCostRequest validates a GetActualCostRequest message.
// This validates the contract between Core and Plugin for the GetActualCost RPC.
func ValidateGetActualCostRequest(req *pbc.GetActualCostRequest) error {
	if req == nil {
		return ErrNilRequest
	}

	// Validate resource_id
	if req.GetResourceId() == "" {
		return NewContractError("resource_id", req.GetResourceId(), ErrEmptyResourceID)
	}
	if len(req.GetResourceId()) > MaxResourceIDLength {
		return NewContractError("resource_id", len(req.GetResourceId()), ErrResourceIDTooLong)
	}

	// Validate time range
	if err := ValidateTimeRange(req.GetStart(), req.GetEnd()); err != nil {
		return err
	}

	// Validate tags if present
	if err := ValidateTags(req.GetTags()); err != nil {
		return err
	}

	return nil
}

// ValidateGetProjectedCostRequest validates a GetProjectedCostRequest message.
// This validates the contract between Core and Plugin for the GetProjectedCost RPC.
func ValidateGetProjectedCostRequest(req *pbc.GetProjectedCostRequest) error {
	if req == nil {
		return ErrNilRequest
	}
	return ValidateResourceDescriptor(req.GetResource())
}

// ValidateGetPricingSpecRequest validates a GetPricingSpecRequest message.
// This validates the contract between Core and Plugin for the GetPricingSpec RPC.
func ValidateGetPricingSpecRequest(req *pbc.GetPricingSpecRequest) error {
	if req == nil {
		return ErrNilRequest
	}
	return ValidateResourceDescriptor(req.GetResource())
}

// ValidateEstimateCostRequest validates an EstimateCostRequest message.
// This validates the contract between Core and Plugin for the EstimateCost RPC.
func ValidateEstimateCostRequest(req *pbc.EstimateCostRequest) error {
	if req == nil {
		return ErrNilRequest
	}

	// Validate resource_type (required)
	if req.GetResourceType() == "" {
		return NewContractError("resource_type", req.GetResourceType(), ErrEmptyResourceType)
	}
	if len(req.GetResourceType()) > MaxResourceTypeLength {
		return NewContractError("resource_type", len(req.GetResourceType()),
			fmt.Errorf("resource_type exceeds maximum length of %d", MaxResourceTypeLength))
	}

	// Validate resource_type format (Pulumi format: provider:module/resource:Type)
	if !resourceTypePattern.MatchString(req.GetResourceType()) {
		return NewContractError("resource_type", req.GetResourceType(), ErrEmptyResourceTypeFmt)
	}

	// attributes is optional (can be nil or empty)
	return nil
}

// ValidateGetRecommendationsRequest validates a GetRecommendationsRequest message.
// This validates the contract between Core and Plugin for the GetRecommendations RPC.
func ValidateGetRecommendationsRequest(req *pbc.GetRecommendationsRequest) error {
	if req == nil {
		return ErrNilRequest
	}

	// Validate page_size if specified
	if req.GetPageSize() > MaxPageSize {
		return NewContractError("page_size", req.GetPageSize(), ErrInvalidPageSize)
	}

	// Validate projection_period if specified
	if req.GetProjectionPeriod() != "" {
		valid := false
		for _, p := range ValidProjectionPeriods {
			if req.GetProjectionPeriod() == p {
				valid = true
				break
			}
		}
		if !valid {
			return NewContractError("projection_period", req.GetProjectionPeriod(), ErrInvalidProjectionPeriod)
		}
	}

	// Validate filter if present
	if req.GetFilter() != nil {
		if err := ValidateRecommendationFilter(req.GetFilter()); err != nil {
			return err
		}
	}

	// Validate target_resources if present
	if err := ValidateTargetResources(req.GetTargetResources()); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// Component Validation Functions
// =============================================================================

// ValidateTargetResources validates the target_resources field of a GetRecommendationsRequest.
// Returns nil if the list is empty/nil (preserves existing behavior) or all entries are valid.
// Returns an error if the list exceeds MaxTargetResources or contains invalid entries.
func ValidateTargetResources(targets []*pbc.ResourceDescriptor) error {
	// Empty or nil is valid - preserves existing behavior (analyze all resources)
	if len(targets) == 0 {
		return nil
	}

	// Check maximum limit
	if len(targets) > MaxTargetResources {
		return NewContractError("target_resources", len(targets), ErrTargetResourcesExceedsLimit)
	}

	// Validate each resource descriptor
	for i, resource := range targets {
		if err := ValidateResourceDescriptor(resource); err != nil {
			return fmt.Errorf("target_resources[%d]: %w", i, err)
		}
	}

	return nil
}

// ValidateResourceDescriptor validates a ResourceDescriptor message.
// This is the core validation used by multiple RPC requests.
func ValidateResourceDescriptor(resource *pbc.ResourceDescriptor) error {
	if resource == nil {
		return ErrNilResource
	}

	// Validate provider (required)
	if resource.GetProvider() == "" {
		return NewContractError("provider", resource.GetProvider(), ErrEmptyProvider)
	}
	if !isValidProvider(resource.GetProvider()) {
		return NewContractError("provider", resource.GetProvider(), ErrInvalidProvider)
	}

	// Validate resource_type (required)
	if resource.GetResourceType() == "" {
		return NewContractError("resource_type", resource.GetResourceType(), ErrEmptyResourceType)
	}
	if len(resource.GetResourceType()) > MaxResourceTypeLength {
		return NewContractError("resource_type", len(resource.GetResourceType()),
			fmt.Errorf("resource_type exceeds maximum length of %d", MaxResourceTypeLength))
	}

	// Validate tags if present
	if err := ValidateTags(resource.GetTags()); err != nil {
		return err
	}

	// sku and region are optional - no validation required
	return nil
}

// ValidateTimeRange validates start and end timestamps for cost queries.
func ValidateTimeRange(start, end *timestamppb.Timestamp) error {
	if start == nil {
		return NewContractError("start", nil, ErrNilStartTime)
	}
	if end == nil {
		return NewContractError("end", nil, ErrNilEndTime)
	}

	startTime := start.AsTime()
	endTime := end.AsTime()

	// End must be after start
	if !endTime.After(startTime) {
		return NewContractError("time_range", fmt.Sprintf("%v to %v", startTime, endTime), ErrInvalidTimeRange)
	}

	// Check minimum time range (1 hour)
	minDuration := time.Duration(MinTimeRangeHours) * time.Hour
	if endTime.Sub(startTime) < minDuration {
		return NewContractError("time_range", endTime.Sub(startTime), ErrTimeRangeTooShort)
	}

	// Check maximum time range (365 days)
	maxDuration := time.Duration(MaxTimeRangeDays) * hoursPerDay * time.Hour
	if endTime.Sub(startTime) > maxDuration {
		return NewContractError("time_range", endTime.Sub(startTime), ErrTimeRangeTooLong)
	}

	// Start time should not be in the future (with grace period)
	// Use UTC explicitly since protobuf timestamps are always UTC, avoiding
	// validation failures in non-UTC deployments.
	if startTime.After(time.Now().UTC().Add(FutureTimeGraceMinutes * time.Minute)) {
		return NewContractError("start", startTime, ErrFutureStartTime)
	}

	return nil
}

// ValidateTags validates a tags map.
func ValidateTags(tags map[string]string) error {
	if tags == nil {
		return nil
	}

	if len(tags) > MaxTagCount {
		return NewContractError("tags", len(tags), ErrTooManyTags)
	}

	for key, value := range tags {
		if len(key) > MaxTagKeyLength {
			return NewContractError("tags.key", key, ErrTagKeyTooLong)
		}
		if len(value) > MaxTagValueLength {
			return NewContractError("tags.value", value, ErrTagValueTooLong)
		}
	}

	return nil
}

// ValidateRecommendationFilter validates a RecommendationFilter message.
func ValidateRecommendationFilter(filter *pbc.RecommendationFilter) error {
	if filter == nil {
		return nil
	}

	// Validate provider if specified
	if filter.GetProvider() != "" && !isValidProvider(filter.GetProvider()) {
		return NewContractError("filter.provider", filter.GetProvider(), ErrInvalidProvider)
	}

	// Validate resource_type length if specified
	if filter.GetResourceType() != "" && len(filter.GetResourceType()) > MaxResourceTypeLength {
		return NewContractError("filter.resource_type", len(filter.GetResourceType()),
			fmt.Errorf("resource_type exceeds maximum length of %d", MaxResourceTypeLength))
	}

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// isValidProvider checks if a provider string is valid.
func isValidProvider(provider string) bool {
	for _, valid := range ValidProviders {
		if provider == valid {
			return true
		}
	}
	return false
}

// =============================================================================
// Contract Test Suite
// =============================================================================

// ContractTestCase defines a single contract test case.
type ContractTestCase struct {
	Name        string
	Description string
	TestFunc    func() error
}

// ContractTestResult contains the result of a contract test.
type ContractTestResult struct {
	Name    string
	Passed  bool
	Error   error
	Details string
}

// ContractTestSuite provides a collection of contract tests for validating
// proto message compatibility between Core and Plugin implementations.
type ContractTestSuite struct {
	tests []ContractTestCase
}

// NewContractTestSuite creates a new contract test suite.
func NewContractTestSuite() *ContractTestSuite {
	return &ContractTestSuite{
		tests: make([]ContractTestCase, 0),
	}
}

// AddTest adds a contract test to the suite.
func (s *ContractTestSuite) AddTest(test ContractTestCase) {
	s.tests = append(s.tests, test)
}

// Run executes all contract tests and returns the results.
func (s *ContractTestSuite) Run() []ContractTestResult {
	results := make([]ContractTestResult, 0, len(s.tests))

	for _, test := range s.tests {
		result := ContractTestResult{
			Name: test.Name,
		}

		err := test.TestFunc()
		if err != nil {
			result.Passed = false
			result.Error = err
			result.Details = err.Error()
		} else {
			result.Passed = true
		}

		results = append(results, result)
	}

	return results
}

// RegisterStandardContractTests registers all standard contract validation tests.
func RegisterStandardContractTests(suite *ContractTestSuite) {
	registerResourceDescriptorTests(suite)
	registerTimeRangeTests(suite)
	registerRequestTests(suite)
}

// registerResourceDescriptorTests adds ResourceDescriptor validation tests.
func registerResourceDescriptorTests(suite *ContractTestSuite) {
	suite.AddTest(ContractTestCase{
		Name:        "ResourceDescriptor_NilRejected",
		Description: "Nil ResourceDescriptor should be rejected",
		TestFunc: func() error {
			err := ValidateResourceDescriptor(nil)
			if err == nil {
				return errors.New("expected error for nil resource descriptor")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "ResourceDescriptor_EmptyProviderRejected",
		Description: "Empty provider should be rejected",
		TestFunc: func() error {
			err := ValidateResourceDescriptor(&pbc.ResourceDescriptor{
				Provider:     "",
				ResourceType: ec2ResourceType,
			})
			if err == nil {
				return errors.New("expected error for empty provider")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "ResourceDescriptor_InvalidProviderRejected",
		Description: "Invalid provider should be rejected",
		TestFunc: func() error {
			err := ValidateResourceDescriptor(&pbc.ResourceDescriptor{
				Provider:     "invalid-provider",
				ResourceType: ec2ResourceType,
			})
			if err == nil {
				return errors.New("expected error for invalid provider")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "ResourceDescriptor_ValidAccepted",
		Description: "Valid ResourceDescriptor should be accepted",
		TestFunc: func() error {
			return ValidateResourceDescriptor(&pbc.ResourceDescriptor{
				Provider:     awsProviderName,
				ResourceType: ec2ResourceType,
				Sku:          "t3.micro",
				Region:       "us-east-1",
			})
		},
	})
}

// registerTimeRangeTests adds TimeRange validation tests.
func registerTimeRangeTests(suite *ContractTestSuite) {
	suite.AddTest(ContractTestCase{
		Name:        "TimeRange_NilStartRejected",
		Description: "Nil start timestamp should be rejected",
		TestFunc: func() error {
			err := ValidateTimeRange(nil, timestamppb.Now())
			if err == nil {
				return errors.New("expected error for nil start")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "TimeRange_InvalidRangeRejected",
		Description: "End before start should be rejected",
		TestFunc: func() error {
			now := time.Now()
			start := timestamppb.New(now)
			end := timestamppb.New(now.Add(-1 * time.Hour))
			err := ValidateTimeRange(start, end)
			if err == nil {
				return errors.New("expected error for invalid time range")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "TimeRange_ValidAccepted",
		Description: "Valid time range should be accepted",
		TestFunc: func() error {
			now := time.Now()
			start := timestamppb.New(now.Add(-hoursPerDay * time.Hour))
			end := timestamppb.New(now)
			return ValidateTimeRange(start, end)
		},
	})
}

// registerRequestTests adds RPC request validation tests.
func registerRequestTests(suite *ContractTestSuite) {
	suite.AddTest(ContractTestCase{
		Name:        "GetActualCostRequest_EmptyResourceIDRejected",
		Description: "Empty resource_id should be rejected",
		TestFunc: func() error {
			now := time.Now()
			err := ValidateGetActualCostRequest(&pbc.GetActualCostRequest{
				ResourceId: "",
				Start:      timestamppb.New(now.Add(-hoursPerDay * time.Hour)),
				End:        timestamppb.New(now),
			})
			if err == nil {
				return errors.New("expected error for empty resource_id")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "GetActualCostRequest_ValidAccepted",
		Description: "Valid GetActualCostRequest should be accepted",
		TestFunc: func() error {
			now := time.Now()
			return ValidateGetActualCostRequest(&pbc.GetActualCostRequest{
				ResourceId: "i-abc123",
				Start:      timestamppb.New(now.Add(-hoursPerDay * time.Hour)),
				End:        timestamppb.New(now),
			})
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "EstimateCostRequest_EmptyResourceTypeRejected",
		Description: "Empty resource_type should be rejected",
		TestFunc: func() error {
			err := ValidateEstimateCostRequest(&pbc.EstimateCostRequest{
				ResourceType: "",
			})
			if err == nil {
				return errors.New("expected error for empty resource_type")
			}
			return nil
		},
	})

	suite.AddTest(ContractTestCase{
		Name:        "EstimateCostRequest_ValidAccepted",
		Description: "Valid EstimateCostRequest should be accepted",
		TestFunc: func() error {
			return ValidateEstimateCostRequest(&pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
			})
		},
	})
}

// RunStandardContractTests runs the standard contract test suite and returns results.
func RunStandardContractTests() []ContractTestResult {
	suite := NewContractTestSuite()
	RegisterStandardContractTests(suite)
	return suite.Run()
}
