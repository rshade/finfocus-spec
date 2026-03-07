package testing_test

import (
	"context"
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// TestGetProjectedCostMetadata verifies that metadata field can be populated
// and retrieved correctly from GetProjectedCostResponse.
func TestGetProjectedCostMetadata(t *testing.T) {
	ctx := context.Background()

	// Create a custom mock plugin that populates metadata.
	plugin := &mockPluginWithMetadata{
		MockPlugin: plugintesting.NewMockPlugin(),
	}

	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()

	// Test with metadata
	t.Run("WithMetadata", func(t *testing.T) {
		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resp, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}

		// Verify metadata is present and correct
		meta := resp.GetMetadata()
		if meta == nil {
			t.Fatal("Expected metadata to be non-nil")
		}

		if len(meta) != 2 {
			t.Errorf("Expected 2 metadata entries, got %d", len(meta))
		}

		if meta["defaults_applied"] != "engine=mysql,storageType=gp2" {
			t.Errorf("Unexpected defaults_applied value: %s", meta["defaults_applied"])
		}

		if meta["estimate_quality"] != "medium" {
			t.Errorf("Unexpected estimate_quality value: %s", meta["estimate_quality"])
		}
	})

	// Test without metadata (backward compatibility) using standard mock
	t.Run("WithoutMetadata", func(t *testing.T) {
		standardPlugin := plugintesting.NewMockPlugin()
		standardHarness := plugintesting.NewTestHarness(standardPlugin)
		standardHarness.Start(t)
		defer standardHarness.Stop()

		standardClient := standardHarness.Client()

		resource := plugintesting.CreateResourceDescriptor("aws", "ec2", "t3.micro", "us-east-1")
		resp, err := standardClient.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{
			Resource: resource,
		})
		if err != nil {
			t.Fatalf("GetProjectedCost() failed: %v", err)
		}

		// Verify metadata is an empty map (proto3 default)
		meta := resp.GetMetadata()
		if meta == nil {
			// nil is acceptable for proto3 maps.
			return
		}

		if len(meta) != 0 {
			t.Errorf("Expected empty metadata map, got %d entries", len(meta))
		}
	})
}

// mockPluginWithMetadata is a test helper that extends MockPlugin to populate metadata.
type mockPluginWithMetadata struct {
	*plugintesting.MockPlugin
}

// GetProjectedCost overrides the base implementation to add metadata.
func (m *mockPluginWithMetadata) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	// Get the base response from MockPlugin
	resp, err := m.MockPlugin.GetProjectedCost(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add metadata to the response
	resp.Metadata = map[string]string{
		"defaults_applied": "engine=mysql,storageType=gp2",
		"estimate_quality": "medium",
	}

	return resp, nil
}
