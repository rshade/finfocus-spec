package jsonld

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ValidationError represents a validation failure during serialization.
type ValidationError struct {
	Field      string
	Message    string
	Suggestion string
}

func (e *ValidationError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("validation error for field %q: %s (suggestion: %s)", e.Field, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("validation error for field %q: %s", e.Field, e.Message)
}

// SerializerOptions configures serialization behavior.
type SerializerOptions struct {
	OmitEmptyFields   bool
	UseIRIEnums       bool
	IncludeDeprecated bool
	PrettyPrint       bool
	DateFormat        string
	UserIDField       string
	IDPrefix          string
	// StreamLimits configures optional limits for streaming serialization.
	// Zero values mean unlimited (no limit enforced).
	StreamLimits StreamLimits
}

// DefaultSerializerOptions returns sensible defaults for serialization.
func DefaultSerializerOptions() *SerializerOptions {
	return &SerializerOptions{
		OmitEmptyFields:   true,
		UseIRIEnums:       false,
		IncludeDeprecated: true,
		PrettyPrint:       false,
		DateFormat:        time.RFC3339,
		UserIDField:       "",
		IDPrefix:          "urn:focus:cost:",
	}
}

// Serializer converts FOCUS cost data to JSON-LD format.
//
// Thread Safety: Serializer instances are safe for concurrent use by multiple
// goroutines after construction. The Serialize and SerializeCommitment methods
// do not modify internal state. However, the Context passed during construction
// should not be modified after the Serializer is created, as this may cause
// data races.
type Serializer struct {
	context     *Context
	idGenerator IDGenerator
	options     *SerializerOptions
}

// SerializerOption is a functional option for configuring the Serializer.
type SerializerOption func(*Serializer)

// NewSerializer creates a new JSON-LD serializer with default configuration.
//
// Panics if the Context configuration is invalid (e.g., invalid remote context URLs).
// NewSerializer creates a Serializer initialized with a default Context, ID generator,
// and default SerializerOptions, then applies the provided SerializerOption functions.
// It validates the configured Context and panics on validation failure. If the
// configured options include a UserIDField, NewSerializer attempts to apply that
// setting to the ID generator via the ConfigurableIDGenerator interface and panics
// if the generator does not support that interface.
func NewSerializer(opts ...SerializerOption) *Serializer {
	s := &Serializer{
		context:     NewContext(),
		idGenerator: NewIDGenerator(),
		options:     DefaultSerializerOptions(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Validate context configuration (fail-fast on invalid config)
	if err := s.context.Validate(); err != nil {
		panic(fmt.Sprintf("invalid context configuration: %v", err))
	}

	// Apply user ID field to generator if specified and supported
	if s.options.UserIDField != "" {
		if gen, ok := s.idGenerator.(ConfigurableIDGenerator); ok {
			s.idGenerator = gen.WithUserIDField(s.options.UserIDField)
		} else {
			panic(
				"IDGenerator does not support WithUserIDField configuration; " +
					"implement ConfigurableIDGenerator interface",
			)
		}
	}

	return s
}

// WithContext sets a custom JSON-LD context.
func WithContext(ctx *Context) SerializerOption {
	return func(s *Serializer) {
		s.context = ctx
	}
}

// validUserIDFields defines the supported fields for user-provided @id generation.
// FocusCostRecord supports: invoice_id, resource_id
// ContractCommitment supports: contract_commitment_id, contract_id
//
//nolint:gochecknoglobals // Intentional package-level validation set for fail-fast behavior
var validUserIDFields = map[string]bool{
	"invoice_id":             true,
	"resource_id":            true,
	"contract_commitment_id": true,
	"contract_id":            true,
}

// WithUserIDField configures the field to use as user-provided @id.
//
// Valid fields for FocusCostRecord: invoice_id, resource_id
// Valid fields for ContractCommitment: contract_commitment_id, contract_id
//
// Panics if an invalid field name is provided (fail-fast behavior).
func WithUserIDField(field string) SerializerOption {
	if !validUserIDFields[field] {
		panic(fmt.Sprintf(
			"invalid user ID field %q; valid fields: invoice_id, resource_id, contract_commitment_id, contract_id",
			field,
		))
	}
	return func(s *Serializer) {
		s.options.UserIDField = field
	}
}

// WithIDPrefix sets a custom prefix for generated IDs.
// If the IDGenerator implements ConfigurableIDGenerator, the prefix is applied
// directly. Otherwise, the prefix is stored in options but may not affect ID generation.
func WithIDPrefix(prefix string) SerializerOption {
	return func(s *Serializer) {
		s.options.IDPrefix = prefix
		if gen, ok := s.idGenerator.(ConfigurableIDGenerator); ok {
			s.idGenerator = gen.WithIDPrefix(prefix)
		}
	}
}

// WithOmitEmpty controls whether empty/zero values are omitted.
func WithOmitEmpty(omit bool) SerializerOption {
	return func(s *Serializer) {
		s.options.OmitEmptyFields = omit
	}
}

// WithIRIEnums controls whether enums are serialized as full IRIs.
func WithIRIEnums(iri bool) SerializerOption {
	return func(s *Serializer) {
		s.options.UseIRIEnums = iri
	}
}

// WithDeprecated controls whether deprecated fields are included.
func WithDeprecated(include bool) SerializerOption {
	return func(s *Serializer) {
		s.options.IncludeDeprecated = include
	}
}

// WithPrettyPrint enables indented JSON output.
func WithPrettyPrint(pretty bool) SerializerOption {
	return func(s *Serializer) {
		s.options.PrettyPrint = pretty
	}
}

// WithStreamLimits configures limits for streaming serialization.
// This helps prevent DoS attacks from unbounded input.
//
// Example:
//
//	s := jsonld.NewSerializer(jsonld.WithStreamLimits(jsonld.StreamLimits{
//	    MaxRecords:    10000,  // Limit to 10K records
//	    MaxRecordSize: 65536,  // Max 64KB per record
//	}))
func WithStreamLimits(limits StreamLimits) SerializerOption {
	return func(s *Serializer) {
		s.options.StreamLimits = limits
	}
}

// fieldWriter is a fail-fast field writer that stops all operations after the first error.
// This prevents partial document corruption and ensures consistent error handling.
type fieldWriter struct {
	doc map[string]interface{}
	s   *Serializer
	err error
}

// newFieldWriter creates a new fieldWriter for the given document.
func (s *Serializer) newFieldWriter(doc map[string]interface{}) *fieldWriter {
	return &fieldWriter{doc: doc, s: s}
}

// Err returns the first error encountered during field writing.
func (fw *fieldWriter) Err() error {
	return fw.err
}

// addString adds a string field if not empty. Stops if an error was already encountered.
func (fw *fieldWriter) addString(name, value string) {
	if fw.err != nil {
		return
	}
	if err := fw.s.addStringField(fw.doc, name, value); err != nil {
		fw.err = err
	}
}

// addFloat adds a float field if not zero. Stops if an error was already encountered.
func (fw *fieldWriter) addFloat(name string, value float64) {
	if fw.err != nil {
		return
	}
	fw.s.addFloatField(fw.doc, name, value)
}

// addTimestamp adds a timestamp field. Stops if an error was already encountered.
func (fw *fieldWriter) addTimestamp(name string, ts *timestamppb.Timestamp) {
	if fw.err != nil {
		return
	}
	fw.s.addTimestampField(fw.doc, name, ts)
}

// addEnum adds an enum field. Stops if an error was already encountered.
func (fw *fieldWriter) addEnum(name, value string) {
	if fw.err != nil {
		return
	}
	fw.s.addEnumField(fw.doc, name, value)
}

// addCost adds a cost field as MonetaryAmount. Stops if an error was already encountered.
func (fw *fieldWriter) addCost(name string, value float64, currency string) {
	if fw.err != nil {
		return
	}
	fw.s.addCostField(fw.doc, name, value, currency)
}

// addMap adds a map field. Stops if an error was already encountered.
func (fw *fieldWriter) addMap(name string, m map[string]string) {
	if fw.err != nil {
		return
	}
	if err := fw.s.addMapField(fw.doc, name, m); err != nil {
		fw.err = err
	}
}

// addDeprecatedString adds a deprecated string field. Stops if an error was already encountered.
func (fw *fieldWriter) addDeprecatedString(name, value string) {
	if fw.err != nil {
		return
	}
	if err := fw.s.addDeprecatedStringField(fw.doc, name, value); err != nil {
		fw.err = err
	}
}

// SerializeCommitment converts a ContractCommitment to JSON-LD format.
//
// Returns an error if:
//   - record is nil
//   - any string field contains invalid UTF-8
//   - JSON marshaling fails
func (s *Serializer) SerializeCommitment(record *pbc.ContractCommitment) ([]byte, error) {
	// Validate input
	if record == nil {
		return nil, &ValidationError{
			Field:      "record",
			Message:    "record cannot be nil",
			Suggestion: "provide a valid ContractCommitment",
		}
	}

	// Build the JSON-LD document
	doc := make(map[string]interface{})

	// Add @context
	doc["@context"] = s.context.Build()

	// Add @type with namespace prefix for proper RDF semantics
	doc["@type"] = ContractCommitmentType

	// Add @id
	doc["@id"] = s.idGenerator.GenerateCommitment(record)

	// Serialize all fields
	if err := s.serializeCommitmentFields(doc, record); err != nil {
		return nil, err
	}

	// Marshal to JSON
	if s.options.PrettyPrint {
		return json.MarshalIndent(doc, "", "  ")
	}
	return json.Marshal(doc)
}

// serializeCommitmentFields adds all ContractCommitment fields to the document.
// Returns an error if any field contains invalid UTF-8.
// Uses fail-fast pattern: stops all field additions after the first error.
func (s *Serializer) serializeCommitmentFields(doc map[string]interface{}, record *pbc.ContractCommitment) error {
	fw := s.newFieldWriter(doc)

	// Identity fields
	fw.addString("contractCommitmentId", record.GetContractCommitmentId())
	fw.addString("contractId", record.GetContractId())

	// Category and type
	fw.addEnum(
		"contractCommitmentCategory",
		record.GetContractCommitmentCategory().String(),
	)
	fw.addString("contractCommitmentType", record.GetContractCommitmentType())

	// Period fields
	fw.addTimestamp("contractCommitmentPeriodStart", record.GetContractCommitmentPeriodStart())
	fw.addTimestamp("contractCommitmentPeriodEnd", record.GetContractCommitmentPeriodEnd())
	fw.addTimestamp("contractPeriodStart", record.GetContractPeriodStart())
	fw.addTimestamp("contractPeriodEnd", record.GetContractPeriodEnd())

	// Cost and quantity
	fw.addCost("contractCommitmentCost", record.GetContractCommitmentCost(), record.GetBillingCurrency())
	fw.addFloat("contractCommitmentQuantity", record.GetContractCommitmentQuantity())
	fw.addString("contractCommitmentUnit", record.GetContractCommitmentUnit())

	// Currency
	fw.addString("billingCurrency", record.GetBillingCurrency())

	return fw.Err()
}

// Serialize converts a FocusCostRecord to JSON-LD format.
//
// Returns an error if:
//   - record is nil
//   - any string field contains invalid UTF-8
//   - JSON marshaling fails
func (s *Serializer) Serialize(record *pbc.FocusCostRecord) ([]byte, error) {
	// Validate input
	if record == nil {
		return nil, &ValidationError{
			Field:      "record",
			Message:    "record cannot be nil",
			Suggestion: "provide a valid FocusCostRecord",
		}
	}

	// Build the JSON-LD document
	doc := make(map[string]interface{})

	// Add @context
	doc["@context"] = s.context.Build()

	// Add @type with namespace prefix for proper RDF semantics
	doc["@type"] = FocusCostRecordType

	// Add @id
	doc["@id"] = s.idGenerator.Generate(record)

	// Serialize all fields
	if err := s.serializeCostRecordFields(doc, record); err != nil {
		return nil, err
	}

	// Marshal to JSON
	if s.options.PrettyPrint {
		return json.MarshalIndent(doc, "", "  ")
	}
	return json.Marshal(doc)
}

// serializeCostRecordFields adds all FocusCostRecord fields to the document.
// Returns an error if any field contains invalid UTF-8.
// Uses fail-fast pattern: stops all field additions after the first error.
//
//nolint:funlen // This function intentionally maps many proto fields to JSON-LD properties.
func (s *Serializer) serializeCostRecordFields(doc map[string]interface{}, record *pbc.FocusCostRecord) error {
	fw := s.newFieldWriter(doc)
	currency := record.GetBillingCurrency()

	// Identity fields
	fw.addString("billingAccountId", record.GetBillingAccountId())
	fw.addString("billingAccountName", record.GetBillingAccountName())
	fw.addString("billingAccountType", record.GetBillingAccountType())
	fw.addString("subAccountId", record.GetSubAccountId())
	fw.addString("subAccountName", record.GetSubAccountName())
	fw.addString("subAccountType", record.GetSubAccountType())

	// Period fields (timestamp to ISO 8601)
	fw.addTimestamp("billingPeriodStart", record.GetBillingPeriodStart())
	fw.addTimestamp("billingPeriodEnd", record.GetBillingPeriodEnd())
	fw.addTimestamp("chargePeriodStart", record.GetChargePeriodStart())
	fw.addTimestamp("chargePeriodEnd", record.GetChargePeriodEnd())

	// Currency
	fw.addString("billingCurrency", currency)
	fw.addString("pricingCurrency", record.GetPricingCurrency())

	// Charge fields
	fw.addEnum("chargeCategory", record.GetChargeCategory().String())
	fw.addEnum("chargeClass", record.GetChargeClass().String())
	fw.addString("chargeDescription", record.GetChargeDescription())
	fw.addEnum("chargeFrequency", record.GetChargeFrequency().String())

	// Pricing fields
	fw.addEnum("pricingCategory", record.GetPricingCategory().String())
	fw.addFloat("pricingQuantity", record.GetPricingQuantity())
	fw.addString("pricingUnit", record.GetPricingUnit())
	fw.addFloat("listUnitPrice", record.GetListUnitPrice())
	fw.addFloat("pricingCurrencyContractedUnitPrice", record.GetPricingCurrencyContractedUnitPrice())
	fw.addFloat("pricingCurrencyEffectiveCost", record.GetPricingCurrencyEffectiveCost())
	fw.addFloat("pricingCurrencyListUnitPrice", record.GetPricingCurrencyListUnitPrice())

	// Service fields
	fw.addEnum("serviceCategory", record.GetServiceCategory().String())
	fw.addString("serviceName", record.GetServiceName())
	fw.addString("serviceSubcategory", record.GetServiceSubcategory())

	// Resource fields
	fw.addString("resourceId", record.GetResourceId())
	fw.addString("resourceName", record.GetResourceName())
	fw.addString("resourceType", record.GetResourceType())

	// SKU fields
	fw.addString("skuId", record.GetSkuId())
	fw.addString("skuPriceId", record.GetSkuPriceId())
	fw.addString("skuMeter", record.GetSkuMeter())
	fw.addString("skuPriceDetails", record.GetSkuPriceDetails())

	// Region fields
	fw.addString("regionId", record.GetRegionId())
	fw.addString("regionName", record.GetRegionName())
	fw.addString("availabilityZone", record.GetAvailabilityZone())

	// Cost fields - serialize as MonetaryAmount when currency is available
	fw.addCost("billedCost", record.GetBilledCost(), currency)
	fw.addCost("listCost", record.GetListCost(), currency)
	fw.addCost("effectiveCost", record.GetEffectiveCost(), currency)
	fw.addCost("contractedCost", record.GetContractedCost(), currency)
	fw.addFloat("contractedUnitPrice", record.GetContractedUnitPrice())

	// Consumption fields
	fw.addFloat("consumedQuantity", record.GetConsumedQuantity())
	fw.addString("consumedUnit", record.GetConsumedUnit())

	// Commitment discount fields
	fw.addEnum(
		"commitmentDiscountCategory",
		record.GetCommitmentDiscountCategory().String(),
	)
	fw.addString("commitmentDiscountId", record.GetCommitmentDiscountId())
	fw.addString("commitmentDiscountName", record.GetCommitmentDiscountName())
	fw.addFloat("commitmentDiscountQuantity", record.GetCommitmentDiscountQuantity())
	fw.addEnum(
		"commitmentDiscountStatus",
		record.GetCommitmentDiscountStatus().String(),
	)
	fw.addString("commitmentDiscountType", record.GetCommitmentDiscountType())
	fw.addString("commitmentDiscountUnit", record.GetCommitmentDiscountUnit())

	// Capacity reservation fields
	fw.addString("capacityReservationId", record.GetCapacityReservationId())
	fw.addEnum(
		"capacityReservationStatus",
		record.GetCapacityReservationStatus().String(),
	)

	// Invoice fields
	fw.addString("invoiceId", record.GetInvoiceId())
	fw.addString("invoiceIssuer", record.GetInvoiceIssuer())

	// Map fields (tags and extended columns)
	fw.addMap("tags", record.GetTags())
	fw.addMap("extendedColumns", record.GetExtendedColumns())

	// FOCUS 1.3 provider fields
	fw.addString("serviceProviderName", record.GetServiceProviderName())
	fw.addString("hostProviderName", record.GetHostProviderName())

	// FOCUS 1.3 allocation fields
	fw.addString("allocatedMethodId", record.GetAllocatedMethodId())
	fw.addString("allocatedMethodDetails", record.GetAllocatedMethodDetails())
	fw.addString("allocatedResourceId", record.GetAllocatedResourceId())
	fw.addString("allocatedResourceName", record.GetAllocatedResourceName())
	fw.addMap("allocatedTags", record.GetAllocatedTags())

	// Contract reference
	fw.addString("contractApplied", record.GetContractApplied())

	// Deprecated fields (with annotation support)
	// These fields are intentionally accessed despite deprecation for backward compatibility.
	if s.options.IncludeDeprecated {
		//nolint:staticcheck // SA1019: Intentional access to deprecated field for backward compatibility
		fw.addDeprecatedString("providerName", record.GetProviderName())
		//nolint:staticcheck // SA1019: Intentional access to deprecated field for backward compatibility
		fw.addDeprecatedString("publisher", record.GetPublisher())
	}

	return fw.Err()
}

// addStringField adds a string field if not empty.
// Returns an error if the value contains invalid UTF-8.
func (s *Serializer) addStringField(doc map[string]interface{}, name, value string) error {
	if !s.options.OmitEmptyFields || value != "" {
		// Validate UTF-8 - fail loudly on invalid data
		if err := s.validateUTF8(name, value); err != nil {
			return err
		}
		doc[name] = value
	}
	return nil
}

// addFloatField adds a float field if not zero.
func (s *Serializer) addFloatField(doc map[string]interface{}, name string, value float64) {
	if !s.options.OmitEmptyFields || value != 0 {
		doc[name] = value
	}
}

// addTimestampField adds a timestamp as ISO 8601 string.
func (s *Serializer) addTimestampField(doc map[string]interface{}, name string, ts *timestamppb.Timestamp) {
	if ts == nil {
		if !s.options.OmitEmptyFields {
			doc[name] = nil
		}
		return
	}
	doc[name] = ts.AsTime().Format(s.options.DateFormat)
}

// addEnumField adds an enum field as string or IRI.
func (s *Serializer) addEnumField(doc map[string]interface{}, name, value string) {
	// Skip UNSPECIFIED values
	if s.options.OmitEmptyFields && isUnspecifiedEnum(value) {
		return
	}

	if s.options.UseIRIEnums {
		// Convert to IRI format: focus:ChargeCategoryUsage
		doc[name] = fmt.Sprintf("focus:%s", toCamelCase(value))
	} else {
		// Use human-readable string
		doc[name] = value
	}
}

// addCostField adds a cost field as Schema.org MonetaryAmount.
func (s *Serializer) addCostField(doc map[string]interface{}, name string, value float64, currency string) {
	if s.options.OmitEmptyFields && value == 0 {
		return
	}

	// If we have a currency, serialize as Schema.org MonetaryAmount
	if currency != "" {
		doc[name] = map[string]interface{}{
			"@type":    "schema:MonetaryAmount",
			"value":    value,
			"currency": currency,
		}
	} else {
		doc[name] = value
	}
}

// addMapField adds a map field (tags, extended columns).
func (s *Serializer) addMapField(doc map[string]interface{}, name string, m map[string]string) error {
	if s.options.OmitEmptyFields && len(m) == 0 {
		return nil
	}
	if m != nil {
		// Convert to interface{} map for JSON serialization
		result := make(map[string]interface{}, len(m))
		for k, v := range m {
			// Validate UTF-8 for keys and values
			if err := s.validateUTF8(name+"."+k, v); err != nil {
				return err
			}
			if err := s.validateUTF8(name+".<key>", k); err != nil {
				return err
			}
			result[k] = v
		}
		doc[name] = result
	}
	return nil
}

// addDeprecatedStringField adds a deprecated string field.
// Returns an error if the value contains invalid UTF-8.
func (s *Serializer) addDeprecatedStringField(doc map[string]interface{}, name, value string) error {
	if value == "" {
		return nil
	}
	// Validate UTF-8 - fail loudly on invalid data
	if err := s.validateUTF8(name, value); err != nil {
		return err
	}
	// Just add the value - the @deprecated annotation is in the context
	doc[name] = value
	return nil
}

// validateUTF8 validates a string field for valid UTF-8.
// Returns an error if the string contains invalid UTF-8 sequences.
func (s *Serializer) validateUTF8(fieldName, value string) error {
	if utf8.ValidString(value) {
		return nil
	}

	return &ValidationError{
		Field:      fieldName,
		Message:    "invalid UTF-8 sequence detected",
		Suggestion: "ensure source data uses valid UTF-8 encoding",
	}
}

// isUnspecifiedEnum checks if an enum value is the UNSPECIFIED sentinel.
func isUnspecifiedEnum(value string) bool {
	// Proto enum UNSPECIFIED values end with _UNSPECIFIED (12 chars)
	if len(value) >= 12 && value[len(value)-12:] == "_UNSPECIFIED" {
		return true
	}
	return false
}

// toCamelCase converts SNAKE_CASE enum values to CamelCase.
func toCamelCase(s string) string {
	result := make([]byte, 0, len(s))
	capitalizeNext := true

	for i := range len(s) {
		c := s[i]
		if c == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			capitalizeNext = false
		} else if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result = append(result, c)
	}
	return string(result)
}
