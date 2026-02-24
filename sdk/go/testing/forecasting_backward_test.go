package testing_test

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pricing"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// TestBackwardCompatibility_NoGrowthFields validates that requests without growth fields
// behave exactly as before the forecasting feature was added.
//
// This test ensures FR-009: Backward compatibility - existing clients continue to work
// without modification.
func TestBackwardCompatibility_NoGrowthFields(t *testing.T) {
	baseCost := 100.0
	periods := 12

	// Simulate a request without any growth fields set
	// (the default state for old clients)
	result := pricing.ApplyGrowth(
		baseCost,
		pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED, // Default proto value
		nil,                                    // No rate set
		periods,
	)

	// Without growth fields, the base cost should be returned unchanged
	expected := 100.0
	if result != expected {
		t.Errorf("Backward compatibility broken: with no growth fields, got %v, want %v", result, expected)
	}
}

// TestUnspecifiedEqualsNone validates that GROWTH_TYPE_UNSPECIFIED is treated as GROWTH_TYPE_NONE.
// This ensures that old clients sending zero/default values get consistent behavior.
func TestUnspecifiedEqualsNone(t *testing.T) {
	baseCost := 100.0
	rate := floatPtr(0.10)
	periods := 12

	// UNSPECIFIED should produce the same result as NONE
	unspecifiedResult := pricing.ApplyGrowth(baseCost, pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED, rate, periods)
	noneResult := pricing.ApplyGrowth(baseCost, pbc.GrowthType_GROWTH_TYPE_NONE, rate, periods)

	if unspecifiedResult != noneResult {
		t.Errorf("UNSPECIFIED should equal NONE: UNSPECIFIED=%v, NONE=%v", unspecifiedResult, noneResult)
	}

	// Both should be the unchanged base cost
	expected := 100.0
	if unspecifiedResult != expected {
		t.Errorf("Expected %v for UNSPECIFIED, got %v", expected, unspecifiedResult)
	}
}

// TestResolveGrowthType_UnspecifiedToNone validates the ResolveGrowthType helper.
func TestResolveGrowthType_UnspecifiedToNone(t *testing.T) {
	resolved := pricing.ResolveGrowthType(pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED)
	if resolved != pbc.GrowthType_GROWTH_TYPE_NONE {
		t.Errorf("ResolveGrowthType(UNSPECIFIED) = %v, want NONE", resolved)
	}
}

// TestOldClientNewServer simulates an old client (no growth fields) sending to a new server.
func TestOldClientNewServer(t *testing.T) {
	// Old client creates ResourceDescriptor without growth fields
	oldClientResource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "t3.medium",
		Region:       "us-east-1",
		// GrowthType and GrowthRate intentionally not set (zero values)
	}

	// Verify the proto zero values
	if oldClientResource.GetGrowthType() != pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED {
		t.Errorf("Expected default GrowthType to be UNSPECIFIED, got %v", oldClientResource.GetGrowthType())
	}
	if oldClientResource.GrowthRate != nil {
		t.Errorf("Expected default GrowthRate to be nil, got %v", oldClientResource.GetGrowthRate())
	}

	// Old client creates request without growth fields
	oldClientRequest := &pbc.GetProjectedCostRequest{
		Resource:              oldClientResource,
		UtilizationPercentage: 0.5,
		// GrowthType and GrowthRate intentionally not set
	}

	// Verify request zero values
	if oldClientRequest.GetGrowthType() != pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED {
		t.Errorf("Expected request default GrowthType to be UNSPECIFIED, got %v", oldClientRequest.GetGrowthType())
	}
	if oldClientRequest.GrowthRate != nil {
		t.Errorf("Expected request default GrowthRate to be nil, got %v", oldClientRequest.GetGrowthRate())
	}

	// Server should treat this as "no growth" and return unchanged cost
	// We use direct field access to get *float64 (pointer), GetGrowthRate() returns unwrapped float64
	effectiveType, effectiveRate := pricing.ResolveGrowthParams(
		oldClientRequest.GetGrowthType(), oldClientRequest.GrowthRate,
		oldClientResource.GetGrowthType(), oldClientResource.GrowthRate,
	)

	if effectiveType != pbc.GrowthType_GROWTH_TYPE_NONE {
		t.Errorf("Server should resolve to NONE for old client, got %v", effectiveType)
	}
	if effectiveRate != nil {
		t.Errorf("Server should have nil rate for old client, got %v", effectiveRate)
	}
}

// TestNewClientOldServer simulates a new client (with growth fields) sending to an old server.
// Old servers ignore unknown fields per protobuf wire format - this is handled at the proto level.
func TestNewClientOldServer(t *testing.T) {
	// New client creates ResourceDescriptor with growth fields
	newClientResource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "t3.medium",
		Region:       "us-east-1",
		GrowthType:   pbc.GrowthType_GROWTH_TYPE_LINEAR,
		GrowthRate:   floatPtr(0.10),
	}

	// Serialize to wire format
	data, err := proto.Marshal(newClientResource)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize back (simulating old server with new proto definition)
	// An old server would just ignore fields it doesn't know about
	var restored pbc.ResourceDescriptor
	if unmarshalErr := proto.Unmarshal(data, &restored); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal: %v", unmarshalErr)
	}

	// Verify fields survived round-trip (new server reading new client)
	if restored.GetGrowthType() != pbc.GrowthType_GROWTH_TYPE_LINEAR {
		t.Errorf("GrowthType not preserved: got %v", restored.GetGrowthType())
	}
	if restored.GetGrowthRate() != 0.10 {
		t.Errorf("GrowthRate not preserved: got %v", restored.GetGrowthRate())
	}

	t.Log("Wire format verified: growth fields round-trip correctly")
	t.Log("Note: Actual old servers would ignore unknown fields per protobuf spec")
}

// TestDefaultValuesDoNotAffectProjections ensures default/zero values don't change behavior.
func TestDefaultValuesDoNotAffectProjections(t *testing.T) {
	tests := []struct {
		name       string
		growthType pbc.GrowthType
		growthRate *float64
		baseCost   float64
		periods    int
		expected   float64
	}{
		// All cases where growth is effectively "off" should return base cost
		{"Zero value GrowthType", pbc.GrowthType(0), nil, 100.0, 12, 100.0},
		{"UNSPECIFIED with nil rate", pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED, nil, 100.0, 12, 100.0},
		{"NONE with nil rate", pbc.GrowthType_GROWTH_TYPE_NONE, nil, 100.0, 12, 100.0},
		{"UNSPECIFIED with rate", pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED, floatPtr(0.10), 100.0, 12, 100.0},
		{"NONE with rate", pbc.GrowthType_GROWTH_TYPE_NONE, floatPtr(0.10), 100.0, 12, 100.0},
		{"NONE with zero rate", pbc.GrowthType_GROWTH_TYPE_NONE, floatPtr(0.0), 100.0, 12, 100.0},
		{"LINEAR with zero rate", pbc.GrowthType_GROWTH_TYPE_LINEAR, floatPtr(0.0), 100.0, 12, 100.0},
		{"EXPONENTIAL with zero rate", pbc.GrowthType_GROWTH_TYPE_EXPONENTIAL, floatPtr(0.0), 100.0, 12, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pricing.ApplyGrowth(tt.baseCost, tt.growthType, tt.growthRate, tt.periods)
			if result != tt.expected {
				t.Errorf("ApplyGrowth() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestProtoDefaultsAreZero validates that proto default values are what we expect.
func TestProtoDefaultsAreZero(t *testing.T) {
	// Empty ResourceDescriptor should have zero/nil for growth fields
	rd := &pbc.ResourceDescriptor{}

	if rd.GetGrowthType() != pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED {
		t.Errorf("Expected default GrowthType to be UNSPECIFIED (0), got %v", rd.GetGrowthType())
	}

	if rd.GrowthRate != nil {
		t.Errorf("Expected default GrowthRate to be nil, got %v", rd.GetGrowthRate())
	}

	// Empty request should also have zero/nil
	req := &pbc.GetProjectedCostRequest{}

	if req.GetGrowthType() != pbc.GrowthType_GROWTH_TYPE_UNSPECIFIED {
		t.Errorf("Expected request default GrowthType to be UNSPECIFIED (0), got %v", req.GetGrowthType())
	}

	if req.GrowthRate != nil {
		t.Errorf("Expected request default GrowthRate to be nil, got %v", req.GetGrowthRate())
	}
}
