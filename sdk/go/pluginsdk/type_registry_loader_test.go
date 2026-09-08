// Copyright 2024-2026 Richard Shade
// Licensed under the Apache License, Version 2.0

//nolint:testpackage // Testing internal TypeRegistry implementation
package pluginsdk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const validMappingFileJSON = `{
  "source_format": "TERRAFORM",
  "mappings": {
    "aws_instance": {
      "pulumi_token": "aws:ec2/instance:Instance",
      "supported": true,
      "property_mappings": {
        "instance_type": "instanceType"
      }
    },
    "aws_s3_bucket": {
      "pulumi_token": "aws:s3/bucket:Bucket",
      "supported": true
    }
  }
}`

func TestLoadMappingsFromJSON_ValidRoundTrip(t *testing.T) {
	registry := NewTypeRegistry()
	err := registry.LoadMappingsFromJSON(strings.NewReader(validMappingFileJSON))
	require.NoError(t, err)

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "unknown_type"},
	}
	resp := registry.Resolve(req)
	require.Len(t, resp.GetMappings(), 2)

	instance := resp.GetMappings()["aws_instance"]
	assert.Equal(t, "aws:ec2/instance:Instance", instance.GetPulumiToken())
	assert.True(t, instance.GetSupported())
	assert.Equal(t, "instanceType", instance.GetPropertyMappings()["instance_type"])

	bucket := resp.GetMappings()["aws_s3_bucket"]
	assert.Equal(t, "aws:s3/bucket:Bucket", bucket.GetPulumiToken())
	assert.True(t, bucket.GetSupported())

	_, exists := resp.GetMappings()["unknown_type"]
	assert.False(t, exists)
}

func TestLoadMappingsFromJSON_MalformedJSON(t *testing.T) {
	registry := NewTypeRegistry()
	err := registry.LoadMappingsFromJSON(strings.NewReader(`{not valid json`))
	require.Error(t, err)
	assert.Equal(t, 0, registry.Len())
}

func TestLoadMappingsFromJSON_UnknownSourceFormat(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "unrecognized format string",
			json: `{"source_format": "PULUMI_YAML", "mappings": {}}`,
		},
		{
			name: "missing source_format",
			json: `{"mappings": {}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewTypeRegistry()
			err := registry.LoadMappingsFromJSON(strings.NewReader(tt.json))
			require.Error(t, err)
			assert.Equal(t, 0, registry.Len())
		})
	}
}

func TestLoadMappingsFromJSON_MissingPulumiToken(t *testing.T) {
	registry := NewTypeRegistry()
	err := registry.LoadMappingsFromJSON(strings.NewReader(`{
		"source_format": "TERRAFORM",
		"mappings": {
			"aws_instance": { "supported": true }
		}
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pulumi_token")
}

func TestLoadMappingsFromFile_FileNotFound(t *testing.T) {
	registry := NewTypeRegistry()
	err := registry.LoadMappingsFromFile(filepath.Join(t.TempDir(), "does-not-exist.json"))
	require.Error(t, err)
}

func TestLoadMappingsFromFile_ValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mappings.json")
	require.NoError(t, os.WriteFile(path, []byte(validMappingFileJSON), 0o600))

	registry := NewTypeRegistry()
	err := registry.LoadMappingsFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, 2, registry.Len())
}

func TestLoadMappingsFromJSON_MergeWithExisting(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterMapping(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		"aws_instance", "aws:ec2/instance:Instance-OLD", true)
	registry.RegisterMapping(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		"aws_lambda_function", "aws:lambda/function:Function", true)

	err := registry.LoadMappingsFromJSON(strings.NewReader(validMappingFileJSON))
	require.NoError(t, err)

	req := &pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "aws_lambda_function"},
	}
	resp := registry.Resolve(req)
	require.Len(t, resp.GetMappings(), 3)

	// File entry overwrites the pre-existing key (last write wins).
	assert.Equal(t, "aws:ec2/instance:Instance", resp.GetMappings()["aws_instance"].GetPulumiToken())
	// Non-overlapping keys from both sources are preserved.
	assert.Equal(t, "aws:s3/bucket:Bucket", resp.GetMappings()["aws_s3_bucket"].GetPulumiToken())
	assert.Equal(t, "aws:lambda/function:Function", resp.GetMappings()["aws_lambda_function"].GetPulumiToken())
}
