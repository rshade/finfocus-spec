// Package pluginsdk provides helpers for implementing the DryRun RPC capability.
//
// The DryRun feature allows hosts to query a plugin for its FOCUS field mapping
// logic without performing actual cost data retrieval. This is useful for:
//   - Debugging: Understand which fields a plugin populates for a resource type
//   - Validation: Verify plugin configuration before production deployment
//   - Comparison: Compare capabilities across different plugins
//
// Quick Start:
//
//	// Create field mappings for a supported resource type
//	mappings := pluginsdk.AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED)
//
//	// Customize specific field statuses
//	mappings = pluginsdk.SetFieldStatus(mappings, "availability_zone",
//	    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
//	    pluginsdk.WithConditionDescription("Only for regional resources"))
//
//	// Build the response
//	resp := pluginsdk.NewDryRunResponse(
//	    pluginsdk.WithFieldMappings(mappings),
//	    pluginsdk.WithResourceTypeSupported(true),
//	    pluginsdk.WithConfigurationValid(true),
//	)
package pluginsdk

import (
	"context"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// focusFieldNames contains all FOCUS 1.2/1.3 field names from FocusCostRecord.
// This is a package-level variable for zero-allocation access.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation access
var focusFieldNames = []string{
	// Identity & Hierarchy (FOCUS 1.2 Section 2.1)
	"provider_name",        // deprecated in FOCUS 1.3
	"billing_account_id",   // field 2
	"billing_account_name", // field 3
	"sub_account_id",       // field 24
	"sub_account_name",     // field 25
	"billing_account_type", // field 42
	"sub_account_type",     // field 43

	// Billing Period (FOCUS 1.2 Section 2.2)
	"billing_period_start", // field 26
	"billing_period_end",   // field 27
	"billing_currency",     // field 18

	// Charge Period (FOCUS 1.2 Section 2.3)
	"charge_period_start", // field 4
	"charge_period_end",   // field 5

	// Charge Details (FOCUS 1.2 Section 2.4)
	"charge_category",    // field 8
	"charge_class",       // field 28
	"charge_description", // field 29
	"charge_frequency",   // field 30

	// Pricing Details (FOCUS 1.2 Section 2.5)
	"pricing_category",                       // field 9
	"pricing_quantity",                       // field 31
	"pricing_unit",                           // field 32
	"list_unit_price",                        // field 33
	"pricing_currency",                       // field 51
	"pricing_currency_contracted_unit_price", // field 52
	"pricing_currency_effective_cost",        // field 53
	"pricing_currency_list_unit_price",       // field 54

	// Service & Product (FOCUS 1.2 Section 2.6)
	"service_category",    // field 6
	"service_name",        // field 7
	"service_subcategory", // field 56
	"publisher",           // field 55, deprecated in FOCUS 1.3

	// Resource Details (FOCUS 1.2 Section 2.7)
	"resource_id",   // field 12
	"resource_name", // field 13
	"resource_type", // field 34

	// SKU Details (FOCUS 1.2 Section 2.8)
	"sku_id",            // field 14
	"sku_price_id",      // field 35
	"sku_meter",         // field 57
	"sku_price_details", // field 58

	// Location (FOCUS 1.2 Section 2.9)
	"region_id",         // field 10
	"region_name",       // field 11
	"availability_zone", // field 36

	// Financial Amounts (FOCUS 1.2 Section 2.10)
	"billed_cost",           // field 15
	"list_cost",             // field 16
	"effective_cost",        // field 17
	"contracted_cost",       // field 41
	"contracted_unit_price", // field 50

	// Consumption/Usage (FOCUS 1.2 Section 2.11)
	"consumed_quantity", // field 20
	"consumed_unit",     // field 21

	// Commitment Discounts (FOCUS 1.2 Section 2.12)
	"commitment_discount_category", // field 37
	"commitment_discount_id",       // field 38
	"commitment_discount_name",     // field 39
	"commitment_discount_quantity", // field 46
	"commitment_discount_status",   // field 47
	"commitment_discount_type",     // field 48
	"commitment_discount_unit",     // field 49

	// Capacity Reservation (FOCUS 1.2 Sections 3.6, 3.7)
	"capacity_reservation_id",     // field 44
	"capacity_reservation_status", // field 45

	// Invoice Details (FOCUS 1.2 Section 2.13)
	"invoice_id",     // field 19
	"invoice_issuer", // field 40

	// Metadata & Extension (FOCUS 1.2 Section 2.14)
	"tags",             // field 22
	"extended_columns", // field 23

	// FOCUS 1.3 Provider Identification
	"service_provider_name", // field 59, replaces provider_name
	"host_provider_name",    // field 60, replaces publisher

	// FOCUS 1.3 Split Cost Allocation
	"allocated_method_id",      // field 61
	"allocated_method_details", // field 62
	"allocated_resource_id",    // field 63
	"allocated_resource_name",  // field 64
	"allocated_tags",           // field 65

	// FOCUS 1.3 Contract Commitment Link
	"contract_applied", // field 66
}

// FocusFieldNames returns all FOCUS 1.2/1.3 field names from FocusCostRecord.
// The returned slice contains ~66 field names matching the FocusCostRecord message.
//
// This function is safe for concurrent use and returns a direct reference
// to the package-level slice (zero allocation).
func FocusFieldNames() []string {
	return focusFieldNames
}

// FieldMappingOption is a functional option for configuring a FieldMapping.
type FieldMappingOption func(*pbc.FieldMapping)

// WithConditionDescription sets the condition_description field.
// Use this when support_status is CONDITIONAL or DYNAMIC to explain
// when/why the field is populated.
func WithConditionDescription(description string) FieldMappingOption {
	return func(fm *pbc.FieldMapping) {
		fm.ConditionDescription = description
	}
}

// WithExpectedType sets the expected_type field.
// Values: "string", "double", "timestamp", "enum", "map", "bool".
func WithExpectedType(expectedType string) FieldMappingOption {
	return func(fm *pbc.FieldMapping) {
		fm.ExpectedType = expectedType
	}
}

// NewFieldMapping creates a new FieldMapping with the given field name and status.
// Use functional options to set optional fields.
//
// Example:
//
//	fm := NewFieldMapping("availability_zone",
//	    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
//	    WithConditionDescription("Only for regional resources"),
//	    WithExpectedType("string"),
//	)
func NewFieldMapping(fieldName string, status pbc.FieldSupportStatus, opts ...FieldMappingOption) *pbc.FieldMapping {
	fm := &pbc.FieldMapping{
		FieldName:     fieldName,
		SupportStatus: status,
	}
	for _, opt := range opts {
		opt(fm)
	}
	return fm
}

// AllFieldsWithStatus creates a FieldMapping for each FOCUS field with the given status.
// This is useful for quickly creating a baseline mapping for a resource type.
//
// Example:
//
//	// All fields are supported for this resource type
//	mappings := AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED)
//
//	// Override specific fields
//	mappings = SetFieldStatus(mappings, "availability_zone",
//	    pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
//	    WithConditionDescription("Only for multi-AZ resources"))
func AllFieldsWithStatus(status pbc.FieldSupportStatus) []*pbc.FieldMapping {
	fields := FocusFieldNames()
	mappings := make([]*pbc.FieldMapping, len(fields))
	for i, fieldName := range fields {
		mappings[i] = NewFieldMapping(fieldName, status)
	}
	return mappings
}

// SetFieldStatus finds a field by name in the mappings slice and updates its status.
// If the field is not found, the slice is returned unchanged.
// If mappings is nil, nil is returned.
// Options can be used to set condition_description or expected_type.
func SetFieldStatus(
	mappings []*pbc.FieldMapping,
	fieldName string,
	status pbc.FieldSupportStatus,
	opts ...FieldMappingOption,
) []*pbc.FieldMapping {
	if mappings == nil {
		return nil
	}
	for _, fm := range mappings {
		if fm.GetFieldName() == fieldName {
			fm.SupportStatus = status
			for _, opt := range opts {
				opt(fm)
			}
			break
		}
	}
	return mappings
}

// DryRunResponseOption is a functional option for configuring a DryRunResponse.
type DryRunResponseOption func(*pbc.DryRunResponse)

// WithFieldMappings sets the field_mappings field.
func WithFieldMappings(mappings []*pbc.FieldMapping) DryRunResponseOption {
	return func(resp *pbc.DryRunResponse) {
		resp.FieldMappings = mappings
	}
}

// WithResourceTypeSupported sets the resource_type_supported field.
func WithResourceTypeSupported(supported bool) DryRunResponseOption {
	return func(resp *pbc.DryRunResponse) {
		resp.ResourceTypeSupported = supported
	}
}

// WithConfigurationValid sets the configuration_valid field.
func WithConfigurationValid(valid bool) DryRunResponseOption {
	return func(resp *pbc.DryRunResponse) {
		resp.ConfigurationValid = valid
	}
}

// WithConfigurationErrors sets the configuration_errors field.
// Use when configuration_valid is false to explain what's wrong.
func WithConfigurationErrors(errors []string) DryRunResponseOption {
	return func(resp *pbc.DryRunResponse) {
		resp.ConfigurationErrors = errors
	}
}

// NewDryRunResponse creates a new DryRunResponse with the given options.
//
// Example:
//
//	resp := NewDryRunResponse(
//	    WithFieldMappings(mappings),
//	    WithResourceTypeSupported(true),
//	    WithConfigurationValid(true),
//	)
func NewDryRunResponse(opts ...DryRunResponseOption) *pbc.DryRunResponse {
	resp := &pbc.DryRunResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}

// DryRunHandler is an optional interface that plugins can implement to provide
// DryRun functionality. If a plugin implements this interface, the SDK can
// automatically route DryRun requests to it.
type DryRunHandler interface {
	// HandleDryRun returns field mapping information for the given resource type.
	// Implementations should:
	//   - Return all FOCUS field mappings with appropriate support status
	//   - Not make any external API calls (response time < 100ms)
	//   - Validate configuration and report errors if invalid
	//   - Return resource_type_supported=false for unsupported resources
	HandleDryRun(ctx context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error)
}

// ConfigValidator is an optional interface for plugins to validate their
// configuration during dry-run requests.
type ConfigValidator interface {
	// ValidateConfiguration checks if the plugin configuration is valid.
	// Returns nil if valid, or a slice of error messages if invalid.
	ValidateConfiguration() []string
}
