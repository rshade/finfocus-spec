//nolint:testpackage // Testing internal TypeRegistry implementation
package pluginsdk

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestTypeRegistry_Resolve_KnownTypes(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance":        "aws:ec2/instance:Instance",
		"aws_s3_bucket":       "aws:s3/bucket:Bucket",
		"aws_lambda_function": "aws:lambda/function:Function",
	})

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "aws_lambda_function"},
	}
	resp := registry.Resolve(req)
	require.Len(t, resp.GetMappings(), 3)

	assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
	assert.True(t, resp.GetMappings()["aws_instance"].GetSupported())
	assert.Equal(t, "aws:s3/bucket:Bucket", resp.GetMappings()["aws_s3_bucket"].GetPulumiToken())
	assert.Equal(t, "aws:lambda/function:Function", resp.GetMappings()["aws_lambda_function"].GetPulumiToken())
}

func TestTypeRegistry_Resolve_UnknownTypes(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance":        "aws:ec2/instance:Instance",
		"aws_s3_bucket":       "aws:s3/bucket:Bucket",
		"aws_lambda_function": "aws:lambda/function:Function",
	})

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "aws_lambda_function", "unknown_resource"},
	}
	resp := registry.Resolve(req)
	require.Len(t, resp.GetMappings(), 3)

	_, exists := resp.GetMappings()["unknown_resource"]
	assert.False(t, exists)
}

func TestTypeRegistry_Resolve_CrossFormatIsolation(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance": "aws:ec2/instance:Instance",
	})

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_CLOUDFORMATION,
		SourceTypes:  []string{"aws_instance"},
	}
	resp := registry.Resolve(req)
	assert.Empty(t, resp.GetMappings())
}

func TestTypeRegistry_Resolve_EmptyAndUnspecified(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance": "aws:ec2/instance:Instance",
	})

	t.Run("empty source_types returns empty response", func(t *testing.T) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{},
		}
		resp := registry.Resolve(req)
		assert.Empty(t, resp.GetMappings())
	})

	t.Run("UNSPECIFIED format returns empty response", func(t *testing.T) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED,
			SourceTypes:  []string{"aws_instance"},
		}
		resp := registry.Resolve(req)
		assert.Empty(t, resp.GetMappings())
	})
}

func TestTypeRegistry_RegisterMapping_Single(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMapping(
		pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		"aws_rds_instance",
		"aws:rds/instance:Instance",
		false,
	)

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_rds_instance"},
	}
	resp := registry.Resolve(req)
	require.Len(t, resp.GetMappings(), 1)

	mapping := resp.GetMappings()["aws_rds_instance"]
	assert.Equal(t, "aws:rds/instance:Instance", mapping.GetPulumiToken())
	assert.False(t, mapping.GetSupported())
}

func TestTypeRegistry_ConcurrentReads(t *testing.T) {
	registry := NewTypeRegistry()
	// Register 100 mappings
	for i := range 100 {
		sourceType := "aws_resource_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		pulumiToken := "aws:test/resource:Resource" + string(rune('A'+i%26))
		registry.RegisterMapping(
			pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			sourceType,
			pulumiToken,
			true,
		)
	}

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_resource_a0", "aws_resource_b0", "aws_resource_c0"},
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := registry.Resolve(req)
			assert.Len(t, resp.GetMappings(), 3)
		}()
	}
	wg.Wait()
}

func TestTypeRegistry_RegisterMapping_WithPropertyMappings(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMappingWithProperties(
		pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		"aws_instance",
		"aws:ec2/instance:Instance",
		true,
		map[string]string{
			"instance_type":     "instanceType",
			"availability_zone": "availabilityZone",
		},
	)

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	}
	resp := registry.Resolve(req)
	require.Len(t, resp.GetMappings(), 1)

	mapping := resp.GetMappings()["aws_instance"]
	assert.Equal(t, "aws:ec2/instance:Instance", mapping.GetPulumiToken())
	assert.True(t, mapping.GetSupported())
	require.Len(t, mapping.GetPropertyMappings(), 2)
	assert.Equal(t, "instanceType", mapping.GetPropertyMappings()["instance_type"])
	assert.Equal(t, "availabilityZone", mapping.GetPropertyMappings()["availability_zone"])
}

func TestTypeRegistry_PropertyMappings_EmptyByDefault(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMapping(
		pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		"aws_instance",
		"aws:ec2/instance:Instance",
		true,
	)

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	}
	resp := registry.Resolve(req)
	mapping := resp.GetMappings()["aws_instance"]
	assert.Empty(t, mapping.GetPropertyMappings())
}

func BenchmarkTypeRegistry_Resolve(b *testing.B) {
	registry := NewTypeRegistry()
	terraformMappings := make(map[string]string, 1000)
	for i := range 1000 {
		terraformMappings["aws_resource_"+strconv.Itoa(i)] = "aws:test/resource:Resource" + strconv.Itoa(i)
	}
	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, terraformMappings)

	b.Run("single_lookup", func(b *testing.B) {
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_resource_500"},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			registry.Resolve(req)
		}
	})

	b.Run("batch_10", func(b *testing.B) {
		types := make([]string, 10)
		for i := range 10 {
			types[i] = "aws_resource_" + strconv.Itoa(i*100)
		}
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  types,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			registry.Resolve(req)
		}
	})

	b.Run("batch_1000", func(b *testing.B) {
		types := make([]string, 1000)
		for i := range 1000 {
			types[i] = "aws_resource_" + strconv.Itoa(i)
		}
		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  types,
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			registry.Resolve(req)
		}
	})
}

func TestTypeRegistry_Len(t *testing.T) {
	registry := NewTypeRegistry()
	assert.Equal(t, 0, registry.Len())

	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
		"aws_instance":  "aws:ec2/instance:Instance",
		"aws_s3_bucket": "aws:s3/bucket:Bucket",
	})
	assert.Equal(t, 2, registry.Len())

	registry.RegisterMapping(
		pbc.SourceFormat_SOURCE_FORMAT_CLOUDFORMATION,
		"AWS::EC2::Instance",
		"aws:ec2/instance:Instance",
		true,
	)
	assert.Equal(t, 3, registry.Len())
}
