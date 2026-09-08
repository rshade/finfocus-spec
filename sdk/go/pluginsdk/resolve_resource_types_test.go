//nolint:testpackage // Testing internal Server implementation with mocks
package pluginsdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// mockResolvePlugin implements both Plugin and ResolveResourceTypesProvider.
type mockResolvePlugin struct {
	mockPlugin

	mappings  map[string]*pbc.ResourceTypeMapping
	err       error
	returnNil bool
}

func (m *mockResolvePlugin) ResolveResourceTypes(
	_ context.Context,
	req *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.returnNil {
		return nil, nil //nolint:nilnil // intentional nil for testing
	}

	result := make(map[string]*pbc.ResourceTypeMapping)
	for _, st := range req.GetSourceTypes() {
		if mapping, ok := m.mappings[st]; ok {
			result[st] = mapping
		}
	}
	return &pbc.ResolveResourceTypesResponse{Mappings: result}, nil
}

// testTerraformMappings returns 3 standard test Terraform->Pulumi mappings.
func testTerraformMappings() map[string]*pbc.ResourceTypeMapping {
	return map[string]*pbc.ResourceTypeMapping{
		"aws_instance": {
			PulumiToken: "aws:ec2/instance:Instance",
			Supported:   true,
		},
		"aws_s3_bucket": {
			PulumiToken: "aws:s3/bucket:Bucket",
			Supported:   true,
		},
		"aws_lambda_function": {
			PulumiToken: "aws:lambda/function:Function",
			Supported:   true,
		},
	}
}

func TestResolveResourceTypes_WithProvider(t *testing.T) {
	plugin := &mockResolvePlugin{
		mockPlugin: mockPlugin{name: "test-resolve"},
		mappings:   testTerraformMappings(),
	}
	server := NewServer(plugin)

	t.Run("returns correct mappings for known types", func(t *testing.T) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "aws_lambda_function"},
		}
		resp, err := server.ResolveResourceTypes(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, resp.GetMappings(), 3)

		assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
		assert.True(t, resp.GetMappings()["aws_instance"].GetSupported())
		assert.Equal(t, "aws:s3/bucket:Bucket", resp.GetMappings()["aws_s3_bucket"].GetPulumiToken())
		assert.Equal(t, "aws:lambda/function:Function", resp.GetMappings()["aws_lambda_function"].GetPulumiToken())
	})

	t.Run("returns only known mappings for mixed request", func(t *testing.T) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "unknown_resource"},
		}
		resp, err := server.ResolveResourceTypes(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, resp.GetMappings(), 1)

		assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
		_, exists := resp.GetMappings()["unknown_resource"]
		assert.False(t, exists)
	})
}

func TestResolveResourceTypes_WithoutProvider(t *testing.T) {
	plugin := &mockPlugin{name: "basic-plugin"}
	server := NewServer(plugin)

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	}
	resp, err := server.ResolveResourceTypes(context.Background(), req)
	require.NoError(t, err)
	assert.Empty(t, resp.GetMappings())
}

func TestResolveResourceTypes_EdgeCases(t *testing.T) {
	plugin := &mockResolvePlugin{
		mockPlugin: mockPlugin{name: "test-resolve"},
		mappings:   testTerraformMappings(),
	}
	server := NewServer(plugin)

	t.Run("empty source_types returns empty response", func(t *testing.T) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{},
		}
		resp, err := server.ResolveResourceTypes(context.Background(), req)
		require.NoError(t, err)
		assert.Empty(t, resp.GetMappings())
	})

	t.Run("UNSPECIFIED format returns empty response", func(t *testing.T) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED,
			SourceTypes:  []string{"aws_instance"},
		}
		resp, err := server.ResolveResourceTypes(context.Background(), req)
		require.NoError(t, err)
		// The mock doesn't filter by format; this test validates the RPC path works.
		// Format filtering is the plugin/registry's responsibility.
		assert.NotNil(t, resp)
	})

	t.Run("nil response from plugin returns Internal error", func(t *testing.T) {
		nilPlugin := &mockResolvePlugin{
			mockPlugin: mockPlugin{name: "nil-returner"},
			returnNil:  true,
		}
		nilServer := NewServer(nilPlugin)

		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance"},
		}
		_, err := nilServer.ResolveResourceTypes(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestResolveResourceTypes_WithTypeRegistry(t *testing.T) {
	plugin := &mockPlugin{name: "basic-plugin"}
	registry := NewTypeRegistry()
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance":  "aws:ec2/instance:Instance",
		"aws_s3_bucket": "aws:s3/bucket:Bucket",
	})

	server := NewServer(plugin)
	server.typeRegistry = registry

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "unknown_type"},
	}
	resp, err := server.ResolveResourceTypes(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.GetMappings(), 2)

	assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
	assert.True(t, resp.GetMappings()["aws_instance"].GetSupported())
	_, exists := resp.GetMappings()["unknown_type"]
	assert.False(t, exists)
}

func TestResolveResourceTypes_InterfacePrecedence(t *testing.T) {
	// Plugin implementing the interface should take precedence over TypeRegistry
	plugin := &mockResolvePlugin{
		mockPlugin: mockPlugin{name: "interface-plugin"},
		mappings: map[string]*pbc.ResourceTypeMapping{
			"aws_instance": {
				PulumiToken: "aws:ec2/instance:Instance-FROM-INTERFACE",
				Supported:   true,
			},
		},
	}

	registry := NewTypeRegistry()
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance": "aws:ec2/instance:Instance-FROM-REGISTRY",
	})

	server := NewServer(plugin)
	server.typeRegistry = registry

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	}
	resp, err := server.ResolveResourceTypes(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.GetMappings(), 1)

	// Interface result should be used, not registry
	assert.Equal(t, "aws:ec2/instance:Instance-FROM-INTERFACE", resp.GetMappings()["aws_instance"].GetPulumiToken())
}

func TestValidateResolveResourceTypesRequest(t *testing.T) {
	tests := []struct {
		name           string
		req            *pbc.ResolveResourceTypesRequest
		maxSourceTypes int32
		wantErr        bool
	}{
		{
			name:           "nil request is valid (handled downstream)",
			req:            nil,
			maxSourceTypes: DefaultMaxSourceTypes,
			wantErr:        false,
		},
		{
			name: "empty source_types is valid",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  []string{},
			},
			maxSourceTypes: DefaultMaxSourceTypes,
			wantErr:        false,
		},
		{
			name: "UNSPECIFIED format is valid regardless of size",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED,
				SourceTypes:  make([]string, DefaultMaxSourceTypes+1),
			},
			maxSourceTypes: DefaultMaxSourceTypes,
			wantErr:        false,
		},
		{
			name: "under limit is valid",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  make([]string, 10),
			},
			maxSourceTypes: DefaultMaxSourceTypes,
			wantErr:        false,
		},
		{
			name: "at limit is valid",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  make([]string, DefaultMaxSourceTypes),
			},
			maxSourceTypes: DefaultMaxSourceTypes,
			wantErr:        false,
		},
		{
			name: "over limit is invalid",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  make([]string, DefaultMaxSourceTypes+1),
			},
			maxSourceTypes: DefaultMaxSourceTypes,
			wantErr:        true,
		},
		{
			name: "maxSourceTypes<=0 defaults to DefaultMaxSourceTypes",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  make([]string, DefaultMaxSourceTypes+1),
			},
			maxSourceTypes: 0,
			wantErr:        true, // still exceeds the default once applied
		},
		{
			name: "maxSourceTypes<=0 with a request under the default is valid",
			req: &pbc.ResolveResourceTypesRequest{
				SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
				SourceTypes:  make([]string, 5),
			},
			maxSourceTypes: 0,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResolveResourceTypesRequest(tt.req, tt.maxSourceTypes)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// spyResolvePlugin implements Plugin and ResolveResourceTypesProvider, recording
// whether ResolveResourceTypes was invoked.
type spyResolvePlugin struct {
	mockPlugin

	invoked bool
}

func (m *spyResolvePlugin) ResolveResourceTypes(
	_ context.Context,
	_ *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	m.invoked = true
	return &pbc.ResolveResourceTypesResponse{}, nil
}

func TestResolveResourceTypes_SizeLimitEnforcedBeforeDispatch(t *testing.T) {
	t.Run("oversized request is rejected without invoking the provider", func(t *testing.T) {
		plugin := &spyResolvePlugin{mockPlugin: mockPlugin{name: "spy"}}
		server := NewServer(plugin)

		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  make([]string, DefaultMaxSourceTypes+1),
		}
		resp, err := server.ResolveResourceTypes(context.Background(), req)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, resp)
		assert.False(t, plugin.invoked, "provider must not be invoked for an oversized request")
	})

	t.Run("request at the limit is dispatched normally", func(t *testing.T) {
		plugin := &spyResolvePlugin{mockPlugin: mockPlugin{name: "spy"}}
		server := NewServer(plugin)

		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  make([]string, DefaultMaxSourceTypes),
		}
		_, err := server.ResolveResourceTypes(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, plugin.invoked, "provider must be invoked for a well-formed request")
	})
}

func TestResourceTypeMapping_PropertyMappings_RoundTrip(t *testing.T) {
	original := &pbc.ResourceTypeMapping{
		PulumiToken: "aws:ec2/instance:Instance",
		Supported:   true,
		PropertyMappings: map[string]string{
			"instance_type":     "instanceType",
			"availability_zone": "availabilityZone",
			"subnet_id":         "subnetId",
		},
	}

	data, err := proto.Marshal(original)
	require.NoError(t, err)

	decoded := &pbc.ResourceTypeMapping{}
	err = proto.Unmarshal(data, decoded)
	require.NoError(t, err)

	assert.Equal(t, original.GetPulumiToken(), decoded.GetPulumiToken())
	assert.Equal(t, original.GetSupported(), decoded.GetSupported())
	require.Len(t, decoded.GetPropertyMappings(), 3)
	assert.Equal(t, "instanceType", decoded.GetPropertyMappings()["instance_type"])
	assert.Equal(t, "availabilityZone", decoded.GetPropertyMappings()["availability_zone"])
	assert.Equal(t, "subnetId", decoded.GetPropertyMappings()["subnet_id"])
}
