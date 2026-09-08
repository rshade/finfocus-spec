package jsonld_test

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/jsonld"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestConformance_RequiredContextDeclaration(t *testing.T) {
	serializer := jsonld.NewSerializer()

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ServiceName:     "Amazon EC2",
		BilledCost:      125.50,
		BillingCurrency: "USD",
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Check @context is present
	context, ok := result["@context"]
	if !ok {
		t.Error("Output missing required @context declaration")
		return
	}

	// Check @context is a map or array
	ctxMap, isMap := context.(map[string]interface{})
	if !isMap {
		t.Error("@context should be a map")
		return
	}

	// Check for required vocabulary prefixes
	requiredPrefixes := []string{"schema", "focus"}
	for _, prefix := range requiredPrefixes {
		if _, hasPrefix := ctxMap[prefix]; !hasPrefix {
			t.Errorf("Context missing required prefix: %s", prefix)
		}
	}
}

func TestConformance_IDGeneration_UserProvided(t *testing.T) {
	serializer := jsonld.NewSerializer(
		jsonld.WithUserIDField("invoice_id"),
	)

	record := &pbc.FocusCostRecord{
		InvoiceId:        "INV-2025-001",
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ResourceId:      "i-1234567890abcdef0",
		BilledCost:      100.0,
		BillingCurrency: "USD",
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	id, ok := result["@id"]
	if !ok {
		t.Error("Output missing @id")
		return
	}

	idStr, isString := id.(string)
	if !isString {
		t.Error("@id should be a string")
		return
	}

	if !strings.Contains(idStr, "INV-2025-001") {
		t.Errorf("@id should contain user-provided invoice_id, got: %s", idStr)
	}
}

func TestConformance_IDGeneration_Fallback(t *testing.T) {
	serializer := jsonld.NewSerializer()

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ResourceId:      "i-1234567890abcdef0",
		BilledCost:      100.0,
		BillingCurrency: "USD",
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	id, ok := result["@id"]
	if !ok {
		t.Error("Output missing @id")
		return
	}

	idStr, isString := id.(string)
	if !isString {
		t.Error("@id should be a string")
		return
	}

	if !strings.HasPrefix(idStr, "urn:focus:cost:") {
		t.Errorf("@id should start with 'urn:focus:cost:', got: %s", idStr)
	}

	// Should be 64 hex chars (32 bytes = full SHA256)
	prefix := "urn:focus:cost:"
	idSuffix := idStr[len(prefix):] // Remove prefix
	if len(idSuffix) != 64 {
		t.Errorf("@id suffix should be 64 hex chars (full SHA256), got %d (full id: %s)", len(idSuffix), idStr)
	}
}

func TestConformance_EmptyValueOmission(t *testing.T) {
	serializer := jsonld.NewSerializer()

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ServiceName:     "Amazon EC2",
		BilledCost:      125.50,
		BillingCurrency: "USD",
		// Many optional fields left empty/zero
		ResourceName: "",
		RegionName:   "",
		ListCost:     0,
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Empty string fields should be omitted
	_, hasResourceName := result["resourceName"]
	if hasResourceName {
		t.Error("Empty resourceName should be omitted from output")
	}

	// Zero numeric fields should be omitted
	_, hasListCost := result["listCost"]
	if hasListCost {
		t.Error("Zero listCost should be omitted from output")
	}

	// Non-empty fields should be present
	_, hasServiceName := result["serviceName"]
	if !hasServiceName {
		t.Error("Non-empty serviceName should be present in output")
	}
}

func TestConformance_AllocationFields(t *testing.T) {
	serializer := jsonld.NewSerializer()

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ServiceName:     "Amazon EC2",
		BilledCost:      125.50,
		BillingCurrency: "USD",
		// FOCUS 1.3 allocation fields
		AllocatedMethodId:     "cost-allocation-001",
		AllocatedResourceId:   "i-allocated-123",
		AllocatedResourceName: "Production-Server-1",
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Check allocation fields are present (using proper Go naming convention)
	_, okAllocatedMethodID := result["allocatedMethodId"]
	if !okAllocatedMethodID {
		t.Error("Output missing allocatedMethodId")
	}

	_, okAllocatedResourceID := result["allocatedResourceId"]
	if !okAllocatedResourceID {
		t.Error("Output missing allocatedResourceId")
	}

	_, okAllocatedResourceName := result["allocatedResourceName"]
	if !okAllocatedResourceName {
		t.Error("Output missing allocatedResourceName")
	}
}

func TestConformance_TagsAndExtendedColumns(t *testing.T) {
	serializer := jsonld.NewSerializer()

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ServiceName:     "Amazon EC2",
		BilledCost:      125.50,
		BillingCurrency: "USD",
		Tags: map[string]string{
			"environment": "production",
			"team":        "engineering",
		},
		ExtendedColumns: map[string]string{
			"custom-field-1": "custom-value-1",
			"custom-field-2": "custom-value-2",
		},
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Check tags are present
	tags, hasTags := result["tags"]
	if !hasTags {
		t.Error("Output missing tags field")
		return
	}

	tagsMap, isTagsMap := tags.(map[string]interface{})
	if !isTagsMap {
		t.Error("Tags should be a map")
		return
	}

	// Check extended columns are present
	extCols, hasExtCols := result["extendedColumns"]
	if !hasExtCols {
		t.Error("Output missing extendedColumns field")
		return
	}

	extColsMap, isExtColsMap := extCols.(map[string]interface{})
	if !isExtColsMap {
		t.Error("ExtendedColumns should be a map")
		return
	}

	// Verify tag values
	if _, hasEnv := tagsMap["environment"]; !hasEnv {
		t.Error("Tags missing 'environment' key")
	}
	if _, hasTeam := tagsMap["team"]; !hasTeam {
		t.Error("Tags missing 'team' key")
	}

	// Verify extended column values
	if _, hasField1 := extColsMap["custom-field-1"]; !hasField1 {
		t.Error("ExtendedColumns missing 'custom-field-1' key")
	}
	if _, hasField2 := extColsMap["custom-field-2"]; !hasField2 {
		t.Error("ExtendedColumns missing 'custom-field-2' key")
	}
}

func TestSerializeCommitment(t *testing.T) {
	serializer := jsonld.NewSerializer()

	commitment := &pbc.ContractCommitment{
		ContractCommitmentId:       "commit-001",
		ContractId:                 "contract-001",
		ContractCommitmentCategory: pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND,
		ContractCommitmentType:     "Reserved Instance",
		ContractCommitmentPeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		ContractCommitmentPeriodEnd: &timestamppb.Timestamp{
			Seconds: 1767225600,
		},
		ContractCommitmentCost: 10000.00,
		BillingCurrency:        "USD",
	}

	output, err := serializer.SerializeCommitment(commitment)
	if err != nil {
		t.Fatalf("SerializeCommitment() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Check @context
	if _, hasContext := result["@context"]; !hasContext {
		t.Error("Output missing @context")
	}

	// Check @type
	typeVal, hasType := result["@type"]
	if !hasType {
		t.Error("Output missing @type")
	} else if typeVal != "focus:ContractCommitment" {
		t.Errorf("@type = %v, want focus:ContractCommitment", typeVal)
	}

	// Check @id
	if _, hasID := result["@id"]; !hasID {
		t.Error("Output missing @id")
	}

	// Check contractCommitmentId
	if result["contractCommitmentId"] != "commit-001" {
		t.Errorf("contractCommitmentId = %v, want commit-001", result["contractCommitmentId"])
	}

	// Check contractId
	if result["contractId"] != "contract-001" {
		t.Errorf("contractId = %v, want contract-001", result["contractId"])
	}
}

func TestConformance_CommitmentCostRecordLinking(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Create a commitment
	commitment := &pbc.ContractCommitment{
		ContractCommitmentId: "commit-001",
		ContractId:           "contract-001",
		BillingCurrency:      "USD",
	}

	// Create a cost record that references the commitment
	costRecord := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		BilledCost:      100.0,
		BillingCurrency: "USD",
		ContractApplied: "commit-001", // References the commitment
	}

	// Serialize both
	commitmentOutput, err := serializer.SerializeCommitment(commitment)
	if err != nil {
		t.Fatalf("SerializeCommitment() failed: %v", err)
	}

	costOutput, costErr := serializer.Serialize(costRecord)
	if costErr != nil {
		t.Fatalf("Serialize() failed: %v", costErr)
	}

	var commitmentDoc map[string]interface{}
	if unmarshalErr := json.Unmarshal(commitmentOutput, &commitmentDoc); unmarshalErr != nil {
		t.Fatalf("Commitment output is not valid JSON: %v", unmarshalErr)
	}

	var costDoc map[string]interface{}
	if unmarshalErr := json.Unmarshal(costOutput, &costDoc); unmarshalErr != nil {
		t.Fatalf("Cost output is not valid JSON: %v", unmarshalErr)
	}

	// Check commitment has @id
	commitmentID, hasCommitmentID := commitmentDoc["@id"]
	if !hasCommitmentID {
		t.Error("Commitment missing @id")
		return
	}

	// Check cost record has contractApplied
	contractApplied, hasContractApplied := costDoc["contractApplied"]
	if !hasContractApplied {
		t.Error("Cost record missing contractApplied")
		return
	}

	// Verify the linking value matches the commitment ID field
	if contractApplied != "commit-001" {
		t.Errorf("contractApplied = %v, want commit-001", contractApplied)
	}

	t.Logf("Commitment @id: %v", commitmentID)
	t.Logf("Cost record contractApplied: %v", contractApplied)
}

func TestSerializerOptions_WithOmitEmpty(t *testing.T) {
	// Test with OmitEmpty disabled - zero values should be included
	serializer := jsonld.NewSerializer(jsonld.WithOmitEmpty(false))

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		BilledCost:      0.0, // Zero cost
		BillingCurrency: "USD",
		ListCost:        0.0, // Zero list cost
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// With OmitEmpty false, zero values should still be present
	_, hasBilledCost := result["billedCost"]
	if !hasBilledCost {
		t.Error("With OmitEmpty=false, billedCost should be present even when zero")
	}
}

func TestSerializerOptions_WithIRIEnums(t *testing.T) {
	serializer := jsonld.NewSerializer(jsonld.WithIRIEnums(true))

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		BilledCost:      100.0,
		BillingCurrency: "USD",
		ChargeCategory:  pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	// Verify output is valid JSON
	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Check that chargeCategory is present (IRI enum mode changes format)
	_, hasChargeCategory := result["chargeCategory"]
	if !hasChargeCategory {
		t.Error("Output missing chargeCategory")
	}
}

func TestSerializerOptions_WithDeprecated(t *testing.T) {
	// Test with deprecated fields disabled
	serializer := jsonld.NewSerializer(jsonld.WithDeprecated(false))

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		BilledCost:      100.0,
		BillingCurrency: "USD",
		//nolint:staticcheck // SA1019: Intentionally testing deprecated field for backward compatibility
		ProviderName:        "AWS",
		ServiceProviderName: "AWS", // New field
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// With IncludeDeprecated=false, deprecated fields should be omitted
	_, hasProviderName := result["providerName"]
	if hasProviderName {
		t.Error("With IncludeDeprecated=false, providerName should be omitted")
	}

	// Non-deprecated replacement field should still be present
	_, hasServiceProviderName := result["serviceProviderName"]
	if !hasServiceProviderName {
		t.Error("serviceProviderName should be present")
	}
}

func TestSerializerOptions_WithIDPrefix(t *testing.T) {
	customPrefix := "urn:custom:prefix:"
	serializer := jsonld.NewSerializer(jsonld.WithIDPrefix(customPrefix))

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		BilledCost:      100.0,
		BillingCurrency: "USD",
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	id, hasID := result["@id"].(string)
	if !hasID {
		t.Fatal("Output missing @id")
	}

	if !strings.HasPrefix(id, customPrefix) {
		t.Errorf("@id = %s, want prefix %s", id, customPrefix)
	}
}

func TestSerializerOptions_WithContext(t *testing.T) {
	customContext := jsonld.NewContext().
		WithRemoteContext("https://example.com/custom.jsonld")

	serializer := jsonld.NewSerializer(jsonld.WithContext(customContext))

	record := &pbc.FocusCostRecord{
		BillingAccountId: "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{
			Seconds: 1735689600,
		},
		BilledCost:      100.0,
		BillingCurrency: "USD",
	}

	output, err := serializer.Serialize(record)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	// Verify output is valid JSON with context
	_, hasContext := result["@context"]
	if !hasContext {
		t.Error("Output missing @context")
	}
}

func TestSerializerOptions_WithUserIDField_InvalidField(t *testing.T) {
	// Test that invalid user ID fields cause a panic (fail-fast behavior)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid user ID field, but no panic occurred")
		}
	}()

	// This should panic
	_ = jsonld.WithUserIDField("invalid_field")
}

func TestSerializerOptions_WithUserIDField_ValidFields(t *testing.T) {
	// Test that valid user ID fields don't panic
	validFields := []string{"invoice_id", "resource_id", "contract_commitment_id", "contract_id"}

	for _, field := range validFields {
		t.Run(field, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Unexpected panic for valid field %q: %v", field, r)
				}
			}()

			// Should not panic
			_ = jsonld.NewSerializer(jsonld.WithUserIDField(field))
		})
	}
}

func TestNewSerializer_InvalidContextPanic(t *testing.T) {
	// Test that invalid context configuration causes a panic (fail-fast behavior)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid context, but no panic occurred")
		}
	}()

	// Create a context with invalid remote URL
	invalidContext := jsonld.NewContext().
		WithRemoteContext("not-a-valid-url")

	// This should panic because context validation fails
	_ = jsonld.NewSerializer(jsonld.WithContext(invalidContext))
}

func TestSchemaOrg_TypeCoercions(t *testing.T) {
	// Test MonetaryAmountTypeCoercion
	monetaryCoercion := jsonld.MonetaryAmountTypeCoercion()

	typeVal, hasType := monetaryCoercion["@type"]
	if !hasType {
		t.Error("MonetaryAmountTypeCoercion missing @type")
	}
	if typeVal != "schema:MonetaryAmount" {
		t.Errorf("MonetaryAmountTypeCoercion @type = %v, want schema:MonetaryAmount", typeVal)
	}

	schemaValue, hasSchemaValue := monetaryCoercion["schema:value"]
	if !hasSchemaValue {
		t.Error("MonetaryAmountTypeCoercion missing schema:value")
	}

	// Check schema:value is a map with @id and @type
	valueMap, isMap := schemaValue.(map[string]interface{})
	if !isMap {
		t.Error("schema:value should be a map")
	} else {
		if _, hasID := valueMap["@id"]; !hasID {
			t.Error("schema:value missing @id")
		}
		if _, hasValueType := valueMap["@type"]; !hasValueType {
			t.Error("schema:value missing @type")
		}
	}

	schemaCurrency, hasCurrency := monetaryCoercion["schema:currency"]
	if !hasCurrency {
		t.Error("MonetaryAmountTypeCoercion missing schema:currency")
	}
	if schemaCurrency != "schema:currency" {
		t.Errorf("MonetaryAmountTypeCoercion schema:currency = %v, want schema:currency", schemaCurrency)
	}

	// Test DateTimeTypeCoercion
	dateTimeCoercion := jsonld.DateTimeTypeCoercion()

	dateID, hasDateID := dateTimeCoercion["@id"]
	if !hasDateID {
		t.Error("DateTimeTypeCoercion missing @id")
	}
	if dateID != "schema:DateTime" {
		t.Errorf("DateTimeTypeCoercion @id = %v, want schema:DateTime", dateID)
	}

	dateType, hasDateType := dateTimeCoercion["@type"]
	if !hasDateType {
		t.Error("DateTimeTypeCoercion missing @type")
	}
	if dateType != "http://www.w3.org/2001/XMLSchema#dateTime" {
		t.Errorf("DateTimeTypeCoercion @type = %v, want xsd:dateTime", dateType)
	}
}

func TestSerialize_NilRecord(t *testing.T) {
	serializer := jsonld.NewSerializer()

	_, err := serializer.Serialize(nil)
	if err == nil {
		t.Error("Serialize(nil) should return an error")
	}
}

func TestSerializeCommitment_NilRecord(t *testing.T) {
	serializer := jsonld.NewSerializer()

	_, err := serializer.SerializeCommitment(nil)
	if err == nil {
		t.Error("SerializeCommitment(nil) should return an error")
	}
}

func TestSerializeCommitment_WithUserIDField(t *testing.T) {
	serializer := jsonld.NewSerializer(
		jsonld.WithUserIDField("contract_commitment_id"),
	)

	commitment := &pbc.ContractCommitment{
		ContractCommitmentId: "custom-commitment-123",
		ContractId:           "contract-456",
		BillingCurrency:      "USD",
	}

	output, err := serializer.SerializeCommitment(commitment)
	if err != nil {
		t.Fatalf("SerializeCommitment() failed: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(output, &result); unmarshalErr != nil {
		t.Fatalf("Output is not valid JSON: %v", unmarshalErr)
	}

	id, hasID := result["@id"].(string)
	if !hasID {
		t.Fatal("Output missing @id")
	}

	// Should contain the user-provided ID
	if !strings.Contains(id, "custom-commitment-123") {
		t.Errorf("@id should contain user-provided ID, got: %s", id)
	}
}
