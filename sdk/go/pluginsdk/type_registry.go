// Copyright 2024-2026 Richard Shade
// Licensed under the Apache License, Version 2.0

package pluginsdk

import (
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// TypeRegistry provides declarative registration of IaC resource type mappings.
// Plugin developers register Terraform/CloudFormation → Pulumi type token mappings
// at initialization time, and the SDK handles ResolveResourceTypes RPC requests
// automatically.
//
// TypeRegistry is designed for initialization-time population followed by
// concurrent read-only access. All Register* methods must be called before the
// gRPC server starts accepting requests. After initialization, Resolve() is
// safe for concurrent use by multiple goroutines without synchronization.
//
// Example usage:
//
//	registry := pluginsdk.NewTypeRegistry()
//	registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
//	    "aws_instance":  "aws:ec2/instance:Instance",
//	    "aws_s3_bucket": "aws:s3/bucket:Bucket",
//	})
//	pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
//	    Plugin:       &MyPlugin{},
//	    TypeRegistry: registry,
//	})
type TypeRegistry struct {
	mappings map[pbc.SourceFormat]map[string]*pbc.ResourceTypeMapping
}

// NewTypeRegistry creates a new empty TypeRegistry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		mappings: make(map[pbc.SourceFormat]map[string]*pbc.ResourceTypeMapping),
	}
}

// RegisterMapping registers a single source type → Pulumi token mapping.
// Must be called during initialization before the gRPC server starts.
func (r *TypeRegistry) RegisterMapping(format pbc.SourceFormat, sourceType, pulumiToken string, supported bool) {
	if r.mappings[format] == nil {
		r.mappings[format] = make(map[string]*pbc.ResourceTypeMapping)
	}
	r.mappings[format][sourceType] = &pbc.ResourceTypeMapping{
		PulumiToken: pulumiToken,
		Supported:   supported,
	}
}

// RegisterMappingWithProperties registers a single source type → Pulumi token mapping
// with optional property name overrides. Must be called during initialization.
func (r *TypeRegistry) RegisterMappingWithProperties(
	format pbc.SourceFormat,
	sourceType, pulumiToken string,
	supported bool,
	propertyMappings map[string]string,
) {
	if r.mappings[format] == nil {
		r.mappings[format] = make(map[string]*pbc.ResourceTypeMapping)
	}
	r.mappings[format][sourceType] = &pbc.ResourceTypeMapping{
		PulumiToken:      pulumiToken,
		Supported:        supported,
		PropertyMappings: propertyMappings,
	}
}

// RegisterMappings batch-registers source type → Pulumi token mappings with supported=true.
// Must be called during initialization before the gRPC server starts.
func (r *TypeRegistry) RegisterMappings(format pbc.SourceFormat, mappings map[string]string) {
	if r.mappings[format] == nil {
		r.mappings[format] = make(map[string]*pbc.ResourceTypeMapping, len(mappings))
	}
	for sourceType, pulumiToken := range mappings {
		r.mappings[format][sourceType] = &pbc.ResourceTypeMapping{
			PulumiToken: pulumiToken,
			Supported:   true,
		}
	}
}

// Resolve translates a batch of source type strings to their Pulumi type mappings.
// Returns only types that have been registered; unknown types are omitted.
// Returns an empty response for SOURCE_FORMAT_UNSPECIFIED or empty source_types.
//
// This method is safe for concurrent use after initialization.
func (r *TypeRegistry) Resolve(req *pbc.ResolveResourceTypesRequest) *pbc.ResolveResourceTypesResponse {
	if req.GetSourceFormat() == pbc.SourceFormat_SOURCE_FORMAT_UNSPECIFIED {
		return &pbc.ResolveResourceTypesResponse{}
	}

	sourceTypes := req.GetSourceTypes()
	if len(sourceTypes) == 0 {
		return &pbc.ResolveResourceTypesResponse{}
	}

	formatMappings, ok := r.mappings[req.GetSourceFormat()]
	if !ok {
		return &pbc.ResolveResourceTypesResponse{}
	}

	result := make(map[string]*pbc.ResourceTypeMapping, len(sourceTypes))
	for _, st := range sourceTypes {
		if mapping, found := formatMappings[st]; found {
			result[st] = mapping
		}
	}

	return &pbc.ResolveResourceTypesResponse{Mappings: result}
}

// Len returns the total number of registered mappings across all source formats.
func (r *TypeRegistry) Len() int {
	total := 0
	for _, m := range r.mappings {
		total += len(m)
	}
	return total
}
