package jsonld

// Schema.org vocabulary type definitions for JSON-LD context.
//
// Schema.org provides widely-understood vocabulary types for linked data.
// This package defines constants for Schema.org types and properties used in
// FOCUS cost data serialization where natural mappings exist.
const (
	// JSONLDTypeKey is the reserved JSON-LD keyword for a node's type.
	JSONLDTypeKey = "@type"

	// SchemaNamespace is the Schema.org namespace IRI.
	SchemaNamespace = "https://schema.org/"

	// Schema.org types for JSON-LD @type declarations.
	MonetaryAmountType = "schema:MonetaryAmount"
	DateTimeType       = "schema:DateTime"
	PropertyValueType  = "schema:PropertyValue"
	ServiceType        = "schema:Service"
	ThingType          = "schema:Thing"
	PlaceType          = "schema:Place"

	// Schema.org properties for compact IRIs.
	SchemaValue        = "schema:value"
	SchemaCurrency     = "schema:currency"
	SchemaName         = "schema:name"
	SchemaSupersededBy = "schema:supersededBy"
)

// MonetaryAmountTypeCoercion returns a JSON-LD type coercion for Schema.org MonetaryAmount.
func MonetaryAmountTypeCoercion() map[string]interface{} {
	return map[string]interface{}{
		JSONLDTypeKey: MonetaryAmountType,
		"schema:value": map[string]interface{}{
			"@id":         SchemaValue,
			JSONLDTypeKey: "http://www.w3.org/2001/XMLSchema#decimal",
		},
		"schema:currency": SchemaCurrency,
	}
}

// DateTimeTypeCoercion returns a JSON-LD type coercion for ISO 8601 dates.
func DateTimeTypeCoercion() map[string]interface{} {
	return map[string]interface{}{
		"@id":         "schema:DateTime",
		JSONLDTypeKey: "http://www.w3.org/2001/XMLSchema#dateTime",
	}
}

//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var schemaMappedFields = map[string]bool{
	"billed_cost":          true,
	"list_cost":            true,
	"effective_cost":       true,
	"contracted_cost":      true,
	"charge_period_start":  true,
	"charge_period_end":    true,
	"billing_period_start": true,
	"billing_period_end":   true,
	"service_name":         true,
	"resource_name":        true,
	"region_name":          true,
}

// IsSchemaMapped checks if a field has a natural Schema.org mapping.
//
// Returns true for fields that should use Schema.org types (MonetaryAmount, DateTime, etc.)
// instead of FOCUS namespace.
// Uses zero-allocation lookup via package-level map.
func IsSchemaMapped(fieldName string) bool {
	return schemaMappedFields[fieldName]
}
