// Package testing provides a comprehensive testing framework for FinFocus plugins.
// This file implements spec validation for the Plugin Conformance Test Suite.
package testing

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ValidBillingModes contains all valid billing mode values.
// Reference: schemas/pricing_spec.schema.json.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var ValidBillingModes = []string{
	// Time-based billing
	"per_hour", "per_minute", "per_second", "per_day", "per_week", "per_month", "per_year",

	// Storage-based billing
	"per_gb_month", "per_gb_hour", "per_gb_day", "per_tb_month",

	// Usage-based billing
	"per_request", "per_operation", "per_transaction", "per_message", "per_event",

	// Data transfer billing
	"per_gb_transfer", "per_gb_egress", "per_gb_ingress",

	// Compute-based billing
	"per_cpu_hour", "per_cpu_second", "per_core_hour", "per_vcpu_hour",
	"per_memory_gb_hour", "per_memory_gib_hour",

	// Database-specific billing
	"per_rcu", "per_wcu", "per_dtu", "per_iops", "per_io_operation",
	"per_provisioned_iops",

	// Network billing
	"per_endpoint_hour", "per_vpc_hour", "per_nat_gateway_hour",
	"per_load_balancer_hour", "per_connection_hour",

	// Serverless billing
	"per_invocation", "per_execution", "per_gb_second", "per_million_requests",

	// Licensing billing
	"per_license", "per_seat", "per_user",

	// Pricing models
	"on_demand", "reserved", spotResourceKey, "savings_plan",

	// Composite billing
	"flat_rate", "tiered", "volume", "graduated",
}

// isValidBillingMode checks if a billing mode string is valid.
func isValidBillingMode(mode string) bool {
	for _, valid := range ValidBillingModes {
		if mode == valid {
			return true
		}
	}
	return false
}

// validatePricingSpecSchema validates the entire PricingSpec schema compliance.
func validatePricingSpecSchema(spec *pbc.PricingSpec) []ValidationError {
	var errors []ValidationError

	if spec == nil {
		errors = append(errors, NewValidationError("spec", nil, "non-nil", "PricingSpec is nil"))
		return errors
	}

	// Validate required fields
	if err := validateRequiredFields(spec); err != nil {
		errors = append(errors, err...)
	}

	// Validate billing mode enum
	if err := validateBillingModeEnum(spec.GetBillingMode()); err != nil {
		errors = append(errors, *err)
	}

	// Validate currency format
	if err := validateCurrencyFormat(spec.GetCurrency()); err != nil {
		errors = append(errors, *err)
	}

	// Validate numeric constraints
	if err := validateNumericConstraints(spec); err != nil {
		errors = append(errors, err...)
	}

	return errors
}

// validateRequiredFields validates that all required fields are present.
func validateRequiredFields(spec *pbc.PricingSpec) []ValidationError {
	var errors []ValidationError

	if spec.GetProvider() == "" {
		errors = append(errors, NewValidationError(
			"provider", spec.GetProvider(), "non-empty string", "provider is required"))
	}

	if spec.GetResourceType() == "" {
		errors = append(errors, NewValidationError(
			"resource_type", spec.GetResourceType(), "non-empty string", "resource_type is required"))
	}

	if spec.GetBillingMode() == "" {
		errors = append(errors, NewValidationError(
			"billing_mode", spec.GetBillingMode(), "non-empty string", "billing_mode is required"))
	}

	if spec.GetCurrency() == "" {
		errors = append(errors, NewValidationError(
			"currency", spec.GetCurrency(), "3-character ISO code", "currency is required"))
	}

	return errors
}

// validateBillingModeEnum validates that the billing mode is a valid enum value.
func validateBillingModeEnum(mode string) *ValidationError {
	if mode == "" {
		return nil // Required field check handled separately
	}

	if !isValidBillingMode(mode) {
		err := NewValidationError(
			"billing_mode",
			mode,
			fmt.Sprintf("one of %v", ValidBillingModes[:10]), // Show first 10 for brevity
			fmt.Sprintf("invalid billing mode: %s", mode),
		)
		return &err
	}
	return nil
}

// validateCurrencyFormat validates the currency field format (3-character ISO code).
func validateCurrencyFormat(currency string) *ValidationError {
	if currency == "" {
		return nil // Required field check handled separately
	}

	if len(currency) != CurrencyCodeRequiredLength {
		err := NewValidationError(
			"currency",
			currency,
			"3-character ISO 4217 code (e.g., USD, EUR, GBP)",
			fmt.Sprintf("currency must be %d characters, got %d", CurrencyCodeRequiredLength, len(currency)),
		)
		return &err
	}
	return nil
}

// validateNumericConstraints validates numeric field constraints.
func validateNumericConstraints(spec *pbc.PricingSpec) []ValidationError {
	var errors []ValidationError

	if spec.GetRatePerUnit() < 0 {
		errors = append(errors, NewValidationError(
			"rate_per_unit",
			spec.GetRatePerUnit(),
			">= 0",
			"rate_per_unit cannot be negative",
		))
	}

	return errors
}

// SpecValidationResult contains the result of spec validation.
type SpecValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// RunSpecValidation runs spec validation tests against a plugin.
func RunSpecValidation(impl pbc.CostSourceServiceServer) (*SpecValidationResult, error) {
	harness := NewTestHarness(impl)
	// We need to start the harness, but we don't have a *testing.T here
	// Create a minimal test context
	ctx := context.Background()

	// Create connection manually
	conn, err := createTestConnection(harness)
	if err != nil {
		return nil, fmt.Errorf("failed to create test connection: %w", err)
	}
	defer conn.Close()

	client := pbc.NewCostSourceServiceClient(conn)

	// Get a sample resource to test with
	resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

	// Get the pricing spec
	resp, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: resource})
	if err != nil {
		return nil, fmt.Errorf("GetPricingSpec failed: %w", err)
	}

	// Validate the spec
	validationErrors := validatePricingSpecSchema(resp.GetSpec())

	return &SpecValidationResult{
		Valid:  len(validationErrors) == 0,
		Errors: validationErrors,
	}, nil
}

// createTestConnection creates a gRPC connection for testing.
func createTestConnection(harness *TestHarness) (*grpcConn, error) {
	// Import the grpc package for connection type
	return harness.createClientConnection()
}

// grpcConn is a type alias for grpc.ClientConn for clarity.
type grpcConn = grpc.ClientConn

// createClientConnection creates and returns a client connection.
//
//nolint:staticcheck // grpc.DialContext is deprecated but NewClient doesn't support bufconn dialers
func (h *TestHarness) createClientConnection() (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return h.listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	return conn, err
}

// SpecValidationTests returns the spec validation conformance tests.
func SpecValidationTests() []ConformanceSuiteTest {
	return []ConformanceSuiteTest{
		{
			Name:        "SpecValidation_ValidPricingSpec",
			Description: "Validates that GetPricingSpec returns a schema-compliant response",
			Category:    CategorySpecValidation,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createSpecValidationValidTest(),
		},
		{
			Name:        "SpecValidation_BillingModeEnum",
			Description: "Validates that billing_mode is a valid enum value",
			Category:    CategorySpecValidation,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createBillingModeEnumTest(),
		},
		{
			Name:        "SpecValidation_RequiredFields",
			Description: "Validates that all required fields are present",
			Category:    CategorySpecValidation,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    createRequiredFieldsTest(),
		},
	}
}

// createSpecValidationValidTest creates a test for valid PricingSpec validation.
func createSpecValidationValidTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()
		resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

		resp, err := harness.Client().GetPricingSpec(
			context.Background(),
			&pbc.GetPricingSpecRequest{Resource: resource},
		)
		duration := time.Since(start)

		if err != nil {
			return TestResult{
				Method:   MethodGetPricingSpec,
				Category: CategorySpecValidation,
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  detailsRPCCallFailed,
			}
		}

		validationErrors := validatePricingSpecSchema(resp.GetSpec())
		if len(validationErrors) > 0 {
			return TestResult{
				Method:   MethodGetPricingSpec,
				Category: CategorySpecValidation,
				Success:  false,
				Error:    validationErrors[0], // Return first error
				Duration: duration,
				Details:  fmt.Sprintf("%d validation errors found", len(validationErrors)),
			}
		}

		return TestResult{
			Method:   MethodGetPricingSpec,
			Category: CategorySpecValidation,
			Success:  true,
			Duration: duration,
			Details:  "PricingSpec is schema-compliant",
		}
	}
}

// createBillingModeEnumTest creates a test for billing mode enum validation.
func createBillingModeEnumTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()
		resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

		resp, err := harness.Client().GetPricingSpec(
			context.Background(),
			&pbc.GetPricingSpecRequest{Resource: resource},
		)
		duration := time.Since(start)

		if err != nil {
			return TestResult{
				Method:   MethodGetPricingSpec,
				Category: CategorySpecValidation,
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  detailsRPCCallFailed,
			}
		}

		billingMode := resp.GetSpec().GetBillingMode()
		if err := validateBillingModeEnum(billingMode); err != nil {
			return TestResult{
				Method:   MethodGetPricingSpec,
				Category: CategorySpecValidation,
				Success:  false,
				Error:    *err,
				Duration: duration,
				Details:  fmt.Sprintf("Invalid billing mode: %s", billingMode),
			}
		}

		return TestResult{
			Method:   MethodGetPricingSpec,
			Category: CategorySpecValidation,
			Success:  true,
			Duration: duration,
			Details:  fmt.Sprintf("Valid billing mode: %s", billingMode),
		}
	}
}

// createRequiredFieldsTest creates a test for required fields validation.
func createRequiredFieldsTest() func(*TestHarness) TestResult {
	return func(harness *TestHarness) TestResult {
		start := time.Now()
		resource := CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")

		resp, err := harness.Client().GetPricingSpec(
			context.Background(),
			&pbc.GetPricingSpecRequest{Resource: resource},
		)
		duration := time.Since(start)

		if err != nil {
			return TestResult{
				Method:   MethodGetPricingSpec,
				Category: CategorySpecValidation,
				Success:  false,
				Error:    err,
				Duration: duration,
				Details:  detailsRPCCallFailed,
			}
		}

		validationErrors := validateRequiredFields(resp.GetSpec())
		if len(validationErrors) > 0 {
			return TestResult{
				Method:   MethodGetPricingSpec,
				Category: CategorySpecValidation,
				Success:  false,
				Error:    validationErrors[0],
				Duration: duration,
				Details:  fmt.Sprintf("Missing required fields: %v", validationErrors),
			}
		}

		return TestResult{
			Method:   MethodGetPricingSpec,
			Category: CategorySpecValidation,
			Success:  true,
			Duration: duration,
			Details:  "All required fields present",
		}
	}
}

// RegisterSpecValidationTests registers spec validation tests with a conformance suite.
func RegisterSpecValidationTests(suite *ConformanceSuite) {
	for _, test := range SpecValidationTests() {
		suite.AddTest(test)
	}
}

// ValidateBillingModePublic is a public wrapper for billing mode validation (for testing).
func ValidateBillingModePublic(mode string) *ValidationError {
	return validateBillingModeEnum(mode)
}

// ValidateCurrencyFormatPublic is a public wrapper for currency format validation (for testing).
func ValidateCurrencyFormatPublic(currency string) *ValidationError {
	return validateCurrencyFormat(currency)
}

// ValidateRequiredFieldsPublic is a public wrapper for required fields validation (for testing).
func ValidateRequiredFieldsPublic(provider, resourceType, billingMode, currency string) []ValidationError {
	// Create a mock spec structure to validate
	// We need to set fields - but PricingSpec is generated proto, we can test with a helper

	var errors []ValidationError

	if provider == "" {
		errors = append(errors, NewValidationError(
			"provider", provider, "non-empty string", "provider is required"))
	}

	if resourceType == "" {
		errors = append(errors, NewValidationError(
			"resource_type", resourceType, "non-empty string", "resource_type is required"))
	}

	if billingMode == "" {
		errors = append(errors, NewValidationError(
			"billing_mode", billingMode, "non-empty string", "billing_mode is required"))
	}

	if currency == "" {
		errors = append(errors, NewValidationError(
			"currency", currency, "3-character ISO code", "currency is required"))
	}

	return errors
}
