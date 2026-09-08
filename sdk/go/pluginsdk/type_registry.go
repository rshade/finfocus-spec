// Copyright 2024-2026 Richard Shade
// Licensed under the Apache License, Version 2.0

package pluginsdk

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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

	// defaultTTL, when > 0, is applied as an expires_at caching hint
	// (now + defaultTTL) on every Resolve() response.
	defaultTTL time.Duration
}

// TypeRegistryOption is a functional option for configuring a TypeRegistry at
// construction time.
type TypeRegistryOption func(*TypeRegistry)

// WithDefaultTTL configures a default expires_at caching hint applied to every
// Resolve() response. Type mappings are near-static, so a long TTL (hours to
// days) is a reasonable default for most plugins. A zero or negative ttl
// disables the default (the pre-existing behavior: no expires_at set).
func WithDefaultTTL(ttl time.Duration) TypeRegistryOption {
	return func(r *TypeRegistry) {
		r.defaultTTL = ttl
	}
}

// NewTypeRegistry creates a new empty TypeRegistry, applying any TypeRegistryOptions.
func NewTypeRegistry(opts ...TypeRegistryOption) *TypeRegistry {
	r := &TypeRegistry{
		mappings: make(map[pbc.SourceFormat]map[string]*pbc.ResourceTypeMapping),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
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

// TypeMapping is one entry for RegisterMappingsWithProperties: the target Pulumi
// token plus optional property-name overrides.
type TypeMapping struct {
	PulumiToken      string
	PropertyMappings map[string]string
}

// RegisterMappingsWithProperties batch-registers source type -> Pulumi token mappings
// that include optional property name overrides, with supported=true (mirrors
// RegisterMappings's existing convention). For per-entry supported=false, use
// RegisterMappingWithProperties instead. Must be called during initialization before
// the gRPC server starts.
func (r *TypeRegistry) RegisterMappingsWithProperties(format pbc.SourceFormat, mappings map[string]TypeMapping) {
	if r.mappings[format] == nil {
		r.mappings[format] = make(map[string]*pbc.ResourceTypeMapping, len(mappings))
	}
	for sourceType, m := range mappings {
		r.mappings[format][sourceType] = &pbc.ResourceTypeMapping{
			PulumiToken:      m.PulumiToken,
			Supported:        true,
			PropertyMappings: m.PropertyMappings,
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

	resp := &pbc.ResolveResourceTypesResponse{Mappings: result}
	if r.defaultTTL > 0 {
		resp.ExpiresAt = timestamppb.New(time.Now().Add(r.defaultTTL))
	}
	return resp
}

// ResolveResourceTypesResponseOption is a functional option for configuring
// ResolveResourceTypesResponse.
type ResolveResourceTypesResponseOption func(*pbc.ResolveResourceTypesResponse)

// NewResolveResourceTypesResponse creates a new ResolveResourceTypesResponse,
// applying any functional options.
func NewResolveResourceTypesResponse(opts ...ResolveResourceTypesResponseOption) *pbc.ResolveResourceTypesResponse {
	resp := &pbc.ResolveResourceTypesResponse{}
	for _, opt := range opts {
		opt(resp)
	}
	return resp
}

// Len returns the total number of registered mappings across all source formats.
func (r *TypeRegistry) Len() int {
	total := 0
	for _, m := range r.mappings {
		total += len(m)
	}
	return total
}
